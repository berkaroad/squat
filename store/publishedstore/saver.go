package publishedstore

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

type PublishedStoreSaver interface {
	Save(ctx context.Context, data PublishedEventStreamRef) error
	Start()
	Stop()
}

var _ PublishedStoreSaver = (*DefaultPublishedStoreSaver)(nil)

type DefaultPublishedStoreSaver struct {
	BatchSize     int
	BatchInterval time.Duration
	ps            PublishedStore

	initOnce    sync.Once
	initialized bool
	receiverCh  chan *PublishedEventStreamRef
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
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

func (saver *DefaultPublishedStoreSaver) Save(ctx context.Context, data PublishedEventStreamRef) error {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.Load() != 1 {
		panic("batch saving published store has stopped")
	}

	saver.receiverCh <- &data
	return nil
}

func (saver *DefaultPublishedStoreSaver) Start() {
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
			store := saver.ps
			datas := make(map[string]*PublishedEventStreamRef)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data := <-saver.receiverCh:
					if exists, ok := datas[data.AggregateID]; !ok || exists.PublishedVersion < data.PublishedVersion {
						datas[data.AggregateID] = data
					}
					if len(datas) >= batchSize {
						saver.batchSave(bgCtx, store, datas)
						datas = make(map[string]*PublishedEventStreamRef)
					}
				case <-time.After(batchInterval):
					if len(datas) > 0 {
						saver.batchSave(bgCtx, store, datas)
						datas = make(map[string]*PublishedEventStreamRef)
					} else if saver.status.Load() != 1 {
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
	time.Sleep(time.Second * 3)
	for len(saver.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	saver.status.CompareAndSwap(2, 0)
}

func (saver *DefaultPublishedStoreSaver) batchSave(ctx context.Context, store PublishedStore, dataMapping map[string]*PublishedEventStreamRef) {
	datas := make([]PublishedEventStreamRef, 0, len(dataMapping))
	logger := logging.Get(ctx)
	for _, data := range dataMapping {
		datas = append(datas, *data)
	}
	err := retrying.Retry(func() error {
		return store.Save(ctx, datas)
	}, time.Second, func(retryCount int, err error) bool {
		if _, ok := err.(net.Error); ok {
			return true
		}
		return false
	}, -1)
	if err != nil {
		logger.Error(fmt.Sprintf("batch save published eventstream fail: %v", err))
	}
}
