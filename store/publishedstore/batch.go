package publishedstore

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/utilities/goroutine"
	"github.com/berkaroad/squat/utilities/retrying"
)

type BatchSavingPublishedStore interface {
	PublishedStore
	Start()
	Stop()
}

var _ BatchSavingPublishedStore = (*DefaultBatchSavingPublishedStore)(nil)

type DefaultBatchSavingPublishedStore struct {
	BatchSize int
	ps        PublishedStore

	initOnce              sync.Once
	initialized           bool
	receiverCh            chan PublishedEventStreamRef
	publishedVersionCache sync.Map
	status                atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (ps *DefaultBatchSavingPublishedStore) Initialize(publishedStore PublishedStore) *DefaultBatchSavingPublishedStore {
	ps.initOnce.Do(func() {
		if publishedStore == nil {
			panic("param 'publishedStore' is null")
		}
		batchSize := ps.BatchSize
		if batchSize <= 0 {
			batchSize = 1
		}
		ps.ps = publishedStore
		ps.receiverCh = make(chan PublishedEventStreamRef, batchSize)
		ps.initialized = true
	})
	return ps
}

func (ps *DefaultBatchSavingPublishedStore) GetPublishedVersion(ctx context.Context, aggregateID string) (int, error) {
	if !ps.initialized {
		panic("not initialized")
	}

	if ps.status.Load() != 1 {
		panic("batch saving published store has stopped")
	}

	if val, ok := ps.publishedVersionCache.Load(aggregateID); ok {
		return val.(int), nil
	}

	return ps.ps.GetPublishedVersion(ctx, aggregateID)
}

func (ps *DefaultBatchSavingPublishedStore) Save(ctx context.Context, data PublishedEventStreamRef) error {
	if !ps.initialized {
		panic("not initialized")
	}

	if ps.status.Load() != 1 {
		panic("batch saving published store has stopped")
	}

	ps.receiverCh <- data
	return nil
}

func (ps *DefaultBatchSavingPublishedStore) Start() {
	if !ps.initialized {
		panic("not initialized")
	}

	if ps.status.CompareAndSwap(0, 1) {
		go func() {
			batchSize := ps.BatchSize
			store := ps.ps
			datas := make(map[string]PublishedEventStreamRef)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data := <-ps.receiverCh:
					if exists, ok := datas[data.AggregateID]; !ok || exists.PublishedVersion < data.PublishedVersion {
						datas[data.AggregateID] = data
					}
					ps.publishedVersionCache.Store(data.AggregateID, datas[data.AggregateID].PublishedVersion)
					if len(datas) >= batchSize {
						ps.batchSave(bgCtx, store, datas)
						datas = make(map[string]PublishedEventStreamRef)
					}
				case <-time.After(time.Second * 3):
					if len(datas) > 0 {
						ps.batchSave(bgCtx, store, datas)
						datas = make(map[string]PublishedEventStreamRef)
					} else if ps.status.Load() != 1 {
						break loop
					}
				}
			}
		}()
	}
}

func (ps *DefaultBatchSavingPublishedStore) Stop() {
	if !ps.initialized {
		panic("not initialized")
	}

	ps.status.CompareAndSwap(1, 2)
	time.Sleep(time.Second * 3)
	for len(ps.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	ps.status.CompareAndSwap(2, 0)
}

func (ps *DefaultBatchSavingPublishedStore) batchSave(ctx context.Context, store PublishedStore, datas map[string]PublishedEventStreamRef) {
	baseLogger := logging.Get(ctx)
	for _, data := range datas {
		logger := baseLogger.With(
			slog.String("aggregate-id", data.AggregateID),
			slog.String("aggregate-type", data.AggregateTypeName),
			slog.Int("published-version", data.PublishedVersion),
		)
		retrying.RetryForever(func() error {
			return store.Save(ctx, data)
		}, time.Second, func(retryCount int, err error) bool {
			if retryCount == 0 {
				logger.Error(fmt.Sprintf("batch save published eventstream fail: %v", err))
			}
			return true
		})
		ps.publishedVersionCache.CompareAndDelete(data.AggregateID, data.PublishedVersion)
	}
}
