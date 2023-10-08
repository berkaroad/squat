package eventstore

import (
	"context"
	"fmt"
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
	BatchSize     int
	BatchInterval time.Duration
	es            EventStore

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
			batchSize := saver.BatchSize
			batchInterval := saver.BatchInterval
			if batchInterval <= 0 {
				batchInterval = time.Millisecond * 100
			}
			store := saver.es
			datas := make(map[string]eventStreamDataWithResult)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data := <-saver.receiverCh:
					if _, ok := datas[data.Data.AggregateID]; !ok {
						datas[data.Data.AggregateID] = data
					} else {
						data.ResultCh <- ErrEventStreamConcurrencyConflict
					}
					if len(datas) >= batchSize {
						saver.batchSave(bgCtx, store, datas)
						datas = make(map[string]eventStreamDataWithResult)
					}
				case <-time.After(batchInterval):
					if len(datas) > 0 {
						saver.batchSave(bgCtx, store, datas)
						datas = make(map[string]eventStreamDataWithResult)
					} else if saver.status.Load() != 1 {
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
	time.Sleep(time.Second * 3)
	for len(saver.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	saver.status.CompareAndSwap(2, 0)
}

func (saver *DefaultEventStoreSaver) batchSave(ctx context.Context, store EventStore, dataMapping map[string]eventStreamDataWithResult) {
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
	if err != nil {
		logger.Error(fmt.Sprintf("batch save eventstream fail: %v", err))
	}
	for _, data := range dataMapping {
		data.ResultCh <- err
	}
}

type eventStreamDataWithResult struct {
	Data     *EventStreamData
	ResultCh chan error
}
