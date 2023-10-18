package caching

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
)

type Cache interface {
	Get(cacheKey string, typeName string) (data any, loaded bool)
	Set(cacheKey string, data any, expiration time.Duration) error
	Remove(cacheKey string)
}

var _ Cache = (*MemoryCache)(nil)

type MemoryCache struct {
	serializer serialization.BinarySerializer

	initOnce    sync.Once
	initialized bool
	items       sync.Map
}

func (cache *MemoryCache) Initialize(serializer serialization.BinarySerializer) *MemoryCache {
	cache.initOnce.Do(func() {
		cache.serializer = serializer
		cache.initialized = true
	})
	return cache
}

func (cache *MemoryCache) Get(cacheKey string, typeName string) (data any, loaded bool) {
	if !cache.initialized {
		panic("not initialized")
	}

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

	logger := logging.Get(context.Background())
	buf := bytes.NewBuffer(make([]byte, 0, 64))
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

type cacheItem struct {
	Key        string
	Data       []byte
	CreateTime time.Time
	Expiration time.Duration
}

func (item cacheItem) IsExpired() bool {
	return item.Expiration > 0 && time.Since(item.CreateTime) > item.Expiration
}
