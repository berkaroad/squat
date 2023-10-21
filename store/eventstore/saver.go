package eventstore

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

type EventStoreSaver interface {
	AppendEventStream(ctx context.Context, data EventStreamData) <-chan error
	Start()
	Stop()
}

type DefaultEventStoreSaver struct {
	BatchSize         int
	BatchInterval     time.Duration
	ShardingAlgorithm func(aggregateID string) uint8
	es                EventStore

	initOnce    sync.Once
	initialized bool
	receiverCh  chan eventStreamDataWithResult
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (saver *DefaultEventStoreSaver) Initialize(eventStore EventStore) *DefaultEventStoreSaver {
	saver.initOnce.Do(func() {
		if eventStore == nil {
			panic("field 'eventStore' is null")
		}
		batchSize := saver.BatchSize
		if batchSize <= 0 {
			batchSize = 100
		}
		saver.es = eventStore
		saver.receiverCh = make(chan eventStreamDataWithResult, batchSize)
		saver.initialized = true
	})
	return saver
}

func (saver *DefaultEventStoreSaver) AppendEventStream(ctx context.Context, data EventStreamData) <-chan error {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.Load() != 1 {
		logger := logging.Get(ctx)
		logger.Warn("'DefaultEventStoreSaver' has stopped")
	}

	resultCh := make(chan error, 1)
	saver.receiverCh <- eventStreamDataWithResult{
		Data:     &data,
		ResultCh: resultCh,
	}
	return resultCh
}

func (saver *DefaultEventStoreSaver) Start() {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.CompareAndSwap(0, 1) {
		go func() {
			batchSize := cap(saver.receiverCh)
			batchInterval := saver.BatchInterval
			if batchInterval <= 0 {
				batchInterval = time.Millisecond * 100
			}
			shardingAlgorithm := saver.ShardingAlgorithm
			if shardingAlgorithm == nil {
				shardingAlgorithm = func(aggregateID string) uint8 { return 0 }
			}
			store := saver.es
			shardingMapping := make(map[uint8]map[string]eventStreamDataWithResult)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data := <-saver.receiverCh:
					shardKey := shardingAlgorithm(data.Data.AggregateID)
					if _, ok := shardingMapping[shardKey]; !ok {
						shardingMapping[shardKey] = make(map[string]eventStreamDataWithResult)
					}
					if _, ok := shardingMapping[shardKey][data.Data.AggregateID]; !ok {
						shardingMapping[shardKey][data.Data.AggregateID] = data
					} else {
						data.ResultCh <- ErrEventStreamConcurrencyConflict
					}
					if len(shardingMapping[shardKey]) >= batchSize {
						saver.batchSave(bgCtx, store, shardKey, shardingMapping[shardKey])
						shardingMapping[shardKey] = make(map[string]eventStreamDataWithResult)
					}
				case <-time.After(batchInterval):
					hasData := false
					var wg sync.WaitGroup
					for shardKey, datas := range shardingMapping {
						if len(datas) > 0 {
							hasData = true
							shardingMapping[shardKey] = make(map[string]eventStreamDataWithResult)
							wg.Add(1)
							go func(shardKey uint8, datas map[string]eventStreamDataWithResult) {
								defer wg.Done()

								saver.batchSave(bgCtx, store, shardKey, datas)
							}(shardKey, datas)
						}
					}
					wg.Wait()
					if !hasData && saver.status.Load() != 1 {
						break loop
					}
				}
			}
		}()
	}
}

func (saver *DefaultEventStoreSaver) Stop() {
	if !saver.initialized {
		panic("not initialized")
	}

	saver.status.CompareAndSwap(1, 2)
	time.Sleep(time.Second)
	for len(saver.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	saver.status.CompareAndSwap(2, 0)
}

func (saver *DefaultEventStoreSaver) batchSave(ctx context.Context, store EventStore, shardKey uint8, dataMapping map[string]eventStreamDataWithResult) {
	datas := make([]EventStreamData, 0, len(dataMapping))
	logger := logging.Get(ctx)
	for _, data := range dataMapping {
		datas = append(datas, *data.Data)
	}
	err := retrying.Retry(func() error {
		return store.AppendEventStream(ctx, datas)
	}, time.Second, func(retryCount int, err error) bool {
		if _, ok := err.(net.Error); ok {
			return true
		}
		return false
	}, -1)
	for _, data := range dataMapping {
		data.ResultCh <- err
	}
	if err != nil {
		logger.Error(fmt.Sprintf("batch save to eventstore fail: %v", err),
			slog.Uint64("shard-key", uint64(shardKey)),
			slog.Uint64("data-count", uint64(len(dataMapping))),
		)
	} else {
		logger.Debug("batch save to eventstore success",
			slog.Uint64("shard-key", uint64(shardKey)),
			slog.Uint64("data-count", uint64(len(dataMapping))),
		)
	}
}

type eventStreamDataWithResult struct {
	Data     *EventStreamData
	ResultCh chan error
}
