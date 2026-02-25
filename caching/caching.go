package caching

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
)

type Cache interface {
	SetCacheName(name string)
	Get(cacheKey string, typeName string) (data any, loaded bool)
	Set(cacheKey string, data any, expiration time.Duration) error
	Remove(cacheKey string)
	Stats() CacheStatistic
}

type CacheStatistic struct {
	ReadCount   int64
	WriteCount  int64
	HitCount    int64
	RemoveCount int64
}

func (stat CacheStatistic) HitRate() float64 {
	if stat.ReadCount == 0 {
		return 0
	}
	return float64(stat.HitCount) / float64(stat.ReadCount)
}

var _ Cache = (*MemoryCache)(nil)

type MemoryCache struct {
	CleanInterval time.Duration
	serializer    serialization.BinarySerializer

	initOnce    sync.Once
	initialized bool
	items       sync.Map
	cleaning    atomic.Int32
	logger      *slog.Logger
	name        string

	readCounter   atomic.Int64
	writeCounter  atomic.Int64
	hitCounter    atomic.Int64
	removeCounter atomic.Int64
}

func (cache *MemoryCache) Initialize(serializer serialization.BinarySerializer) *MemoryCache {
	cache.initOnce.Do(func() {
		cache.serializer = serializer
		cache.logger = logging.Get(context.Background())
		cache.cleanExpired()
		cache.initialized = true
	})
	return cache
}

func (cache *MemoryCache) SetCacheName(name string) {
	cache.name = name
}

func (cache *MemoryCache) Get(cacheKey string, typeName string) (data any, loaded bool) {
	if !cache.initialized {
		panic("not initialized")
	}

	cache.readCounter.Add(1)
	logger := logging.Get(context.Background())
	if val, ok := cache.items.Load(cacheKey); ok {
		exists := val.(cacheItem)
		if !exists.IsExpired() {
			dataInterface, err := serialization.Deserialize(cache.serializer, typeName, bytes.NewReader(exists.Data))
			if err != nil {
				logger.Error(fmt.Sprintf("deserialize cache item fail: %v", err),
					slog.String("cache-key", cacheKey),
					slog.String("type-name", typeName),
				)
				return
			}
			data = dataInterface
			loaded = ok
			exists.Refresh()
			cache.hitCounter.Add(1)
		} else {
			logger.Debug("cache item has expired",
				slog.String("cache-key", cacheKey),
				slog.String("type-name", typeName),
			)
		}
	}
	return
}

func (cache *MemoryCache) Set(cacheKey string, data any, expiration time.Duration) error {
	if !cache.initialized {
		panic("not initialized")
	}

	cache.writeCounter.Add(1)
	logger := logging.Get(context.Background())
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	if err := serialization.Serialize(cache.serializer, buf, data); err != nil {
		logger.Error(fmt.Sprintf("serialize cache item fail: %v", err),
			slog.String("cache-key", cacheKey),
		)
		return err
	} else {
		cache.items.Store(cacheKey, cacheItem{
			Key:        cacheKey,
			Data:       buf.Bytes(),
			CreateTime: time.Now(),
			Expiration: expiration,
		})
		return nil
	}
}

func (cache *MemoryCache) Remove(cacheKey string) {
	if !cache.initialized {
		panic("not initialized")
	}

	cache.items.Delete(cacheKey)
}

func (cache *MemoryCache) Stats() CacheStatistic {
	return CacheStatistic{
		ReadCount:   cache.readCounter.Load(),
		WriteCount:  cache.writeCounter.Load(),
		HitCount:    cache.hitCounter.Load(),
		RemoveCount: cache.removeCounter.Load(),
	}
}

func (cache *MemoryCache) cleanExpired() {
	if !cache.cleaning.CompareAndSwap(0, 1) {
		return
	}
	go func() {
		cleanInterval := cache.CleanInterval
		if cleanInterval < time.Second*3 {
			cleanInterval = time.Second * 3
		}
		<-time.After(cleanInterval)

		stat := cache.Stats()
		hitRate := stat.HitRate()
		cache.logger.Debug(fmt.Sprintf("cache '%s' hit rate %.4f", cache.name, hitRate),
			slog.Float64("read-count", float64(stat.ReadCount)),
			slog.Float64("write-count", float64(stat.WriteCount)),
			slog.Float64("hit-count", float64(stat.HitCount)),
			slog.Float64("remove-count", float64(stat.RemoveCount)),
		)
		if stat.WriteCount > 0 && hitRate < 0.8 {
			cache.logger.Warn(fmt.Sprintf("cache '%s' hit rate %.4f too low", cache.name, hitRate),
				slog.Float64("read-count", float64(stat.ReadCount)),
				slog.Float64("write-count", float64(stat.WriteCount)),
				slog.Float64("hit-count", float64(stat.HitCount)),
				slog.Float64("remove-count", float64(stat.RemoveCount)),
			)
		}

		keysToRemove := make([]string, 0, 1024)
		cache.items.Range(func(key, value any) bool {
			cacheKey := key.(string)
			cacheValue := value.(cacheItem)
			if cacheValue.IsExpired() {
				keysToRemove = append(keysToRemove, cacheKey)
			}
			if len(keysToRemove) >= 1024 {
				return false
			}
			return true
		})

		if len(keysToRemove) > 0 {
			for _, key := range keysToRemove {
				cache.items.Delete(key)
			}
			cache.removeCounter.Add(int64(len(keysToRemove)))
			cache.logger.Debug(fmt.Sprintf("cache '%s' has cleaned %d items", cache.name, len(keysToRemove)),
				slog.Float64("read-count", float64(stat.ReadCount)),
				slog.Float64("write-count", float64(stat.WriteCount)),
				slog.Float64("hit-count", float64(stat.HitCount)),
				slog.Float64("remove-count", float64(stat.RemoveCount)))
		}
		cache.cleaning.CompareAndSwap(1, 0)
		cache.cleanExpired()
	}()
}

type cacheItem struct {
	Key        string
	Data       []byte
	CreateTime time.Time
	Expiration time.Duration
}

func (item *cacheItem) IsExpired() bool {
	return item.Expiration > 0 && time.Since(item.CreateTime) > item.Expiration
}

func (item *cacheItem) Refresh() {
	item.CreateTime = time.Now()
}
