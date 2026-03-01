package publishedstore

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/utilities/goroutine"
	"github.com/berkaroad/squat/utilities/retrying"
)

type PublishedStoreSaver interface {
	GetPublishedVersion(ctx context.Context, aggregateID string) (int, error)
	SavePublished(ctx context.Context, data PublishedEventStreamRef)
	Start()
	Stop()
}

const checkInterval time.Duration = time.Millisecond * 10

var _ PublishedStoreSaver = (*DefaultPublishedStoreSaver)(nil)

type DefaultPublishedStoreSaver struct {
	BatchSize         int
	BatchInterval     time.Duration
	ShardingAlgorithm func(aggregateID string) uint8
	ps                PublishedStore

	initOnce              sync.Once
	initialized           bool
	receiverCh            chan *PublishedEventStreamRef
	publishedVersionCache sync.Map
	status                atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (saver *DefaultPublishedStoreSaver) Initialize(publishedStore PublishedStore) *DefaultPublishedStoreSaver {
	saver.initOnce.Do(func() {
		if publishedStore == nil {
			panic("param 'publishedStore' is null")
		}
		batchSize := saver.BatchSize
		if batchSize <= 0 {
			batchSize = 100
		}
		saver.ps = publishedStore
		saver.receiverCh = make(chan *PublishedEventStreamRef, batchSize)
		saver.initialized = true
	})
	return saver
}

func (saver *DefaultPublishedStoreSaver) GetPublishedVersion(ctx context.Context, aggregateID string) (int, error) {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.Load() != 1 {
		logger := logging.Get(ctx)
		logger.Warn("'DefaultPublishedStoreSaver' has stopped")
	}

	if val, ok := saver.publishedVersionCache.Load(aggregateID); ok {
		return val.(int), nil
	}

	return saver.ps.GetPublishedVersion(ctx, aggregateID)
}

func (saver *DefaultPublishedStoreSaver) SavePublished(ctx context.Context, data PublishedEventStreamRef) {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.Load() != 1 {
		logger := logging.Get(ctx)
		logger.Warn("'DefaultPublishedStoreSaver' has stopped")
	}

	saver.receiverCh <- &data
	if _, loaded := saver.publishedVersionCache.LoadOrStore(data.AggregateID, data.PublishedVersion); loaded {
		saver.publishedVersionCache.CompareAndSwap(data.AggregateID, data.PublishedVersion-1, data.PublishedVersion)
	}
}

func (saver *DefaultPublishedStoreSaver) Start() {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.CompareAndSwap(0, 1) {
		batchSize := saver.BatchSize
		if batchSize <= 0 {
			batchSize = 100
		}
		saver.receiverCh = make(chan *PublishedEventStreamRef, batchSize)
		go func() {
			batchInterval := saver.BatchInterval
			if batchInterval <= 0 {
				batchInterval = 100 * time.Millisecond
			}
			shardingAlgorithm := saver.ShardingAlgorithm
			if shardingAlgorithm == nil {
				shardingAlgorithm = func(aggregateID string) uint8 { return 0 }
			}
			store := saver.ps
			shardingMapping := make(map[uint8]map[string]*PublishedEventStreamRef)
			shardingTimeMapping := make(map[uint8]time.Time)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data, ok := <-saver.receiverCh:
					if !ok {
						break loop
					}
					shardKey := shardingAlgorithm(data.AggregateID)
					if _, ok := shardingMapping[shardKey]; !ok {
						shardingMapping[shardKey] = make(map[string]*PublishedEventStreamRef, batchSize)
					}
					if _, ok := shardingTimeMapping[shardKey]; !ok {
						shardingTimeMapping[shardKey] = time.Now()
					}
					if exists, ok := shardingMapping[shardKey][data.AggregateID]; !ok || exists.PublishedVersion < data.PublishedVersion {
						shardingMapping[shardKey][data.AggregateID] = data
					}
					if len(shardingMapping[shardKey]) >= batchSize {
						delete(shardingTimeMapping, shardKey)
						saver.batchSave(bgCtx, store, shardKey, shardingMapping[shardKey])
						shardingMapping[shardKey] = make(map[string]*PublishedEventStreamRef, batchSize)
					}
				case <-time.After(checkInterval):
					timeoutShardKeys := make([]uint8, 0, len(shardingTimeMapping))
					for shardKey, timestamp := range shardingTimeMapping {
						if time.Since(timestamp) >= batchInterval {
							timeoutShardKeys = append(timeoutShardKeys, shardKey)
						}
					}
					if len(timeoutShardKeys) > 0 {
						var wg sync.WaitGroup
						for _, shardKey := range timeoutShardKeys {
							delete(shardingTimeMapping, shardKey)
							datas := shardingMapping[shardKey]
							if len(datas) > 0 {
								shardingMapping[shardKey] = make(map[string]*PublishedEventStreamRef, batchSize)
								wg.Add(1)
								go func(shardKey uint8, datas map[string]*PublishedEventStreamRef) {
									defer wg.Done()

									saver.batchSave(bgCtx, store, shardKey, datas)
								}(shardKey, datas)
							}
						}
						wg.Wait()
					}
					if saver.status.Load() != 1 {
						break loop
					}
				}
			}
		}()
	}
}

func (saver *DefaultPublishedStoreSaver) Stop() {
	if !saver.initialized {
		panic("not initialized")
	}

	saver.status.CompareAndSwap(1, 2)
	time.Sleep(time.Second)
	for len(saver.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	close(saver.receiverCh)
	<-goroutine.Wait()
	saver.status.CompareAndSwap(2, 0)
}

func (saver *DefaultPublishedStoreSaver) batchSave(ctx context.Context, store PublishedStore, shardKey uint8, dataMapping map[string]*PublishedEventStreamRef) {
	datas := make([]PublishedEventStreamRef, 0, len(dataMapping))
	logger := logging.Get(ctx)
	for _, data := range dataMapping {
		datas = append(datas, *data)
	}
	err := retrying.Retry(func() error {
		return store.SavePublished(ctx, datas)
	}, time.Second, func(retryCount int, err error) bool {
		if _, ok := err.(net.Error); ok {
			return true
		}
		return false
	}, -1)
	if err != nil {
		logger.Error(fmt.Sprintf("batch save to publishedstore fail: %v", err),
			slog.Uint64("shard-key", uint64(shardKey)),
			slog.Uint64("data-count", uint64(len(dataMapping))),
		)
	} else {
		logger.Debug("batch save to publishedstore success",
			slog.Uint64("shard-key", uint64(shardKey)),
			slog.Uint64("data-count", uint64(len(dataMapping))),
		)
	}
}
