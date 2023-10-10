package snapshotstore

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

type SnapshotStoreSaver interface {
	SaveSnapshot(ctx context.Context, data AggregateSnapshotData)
	Start()
	Stop()
}

var _ SnapshotStoreSaver = (*DefaultSnapshotStoreSaver)(nil)

type DefaultSnapshotStoreSaver struct {
	TakeSnapshotMinVersion int
	BatchSize              int
	BatchInterval          time.Duration
	ss                     SnapshotStore

	initOnce    sync.Once
	initialized bool
	receiverCh  chan *AggregateSnapshotData
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (saver *DefaultSnapshotStoreSaver) Initialize(snapshotStore SnapshotStore) *DefaultSnapshotStoreSaver {
	saver.initOnce.Do(func() {
		if snapshotStore == nil {
			panic("param 'snapshotStore' is null")
		}
		batchSize := saver.BatchSize
		if batchSize <= 0 {
			batchSize = 100
		}
		saver.ss = snapshotStore
		saver.receiverCh = make(chan *AggregateSnapshotData, batchSize)
		saver.initialized = true
	})
	return saver
}

func (saver *DefaultSnapshotStoreSaver) SaveSnapshot(ctx context.Context, data AggregateSnapshotData) {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.Load() != 1 {
		panic("batch saving snapshot store has stopped")
	}

	saver.receiverCh <- &data
}

func (saver *DefaultSnapshotStoreSaver) Start() {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.CompareAndSwap(0, 1) {
		go func() {
			batchSize := saver.BatchSize
			batchInterval := saver.BatchInterval
			if batchInterval <= 0 {
				batchInterval = time.Second * 3
			}
			takeSnapshotMinVersion := saver.TakeSnapshotMinVersion
			if takeSnapshotMinVersion <= 0 {
				takeSnapshotMinVersion = 10
			}
			store := saver.ss
			datas := make(map[string]*AggregateSnapshotData)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data := <-saver.receiverCh:
					if data.SnapshotVersion < takeSnapshotMinVersion {
						continue
					}
					if exists, ok := datas[data.AggregateID]; !ok || exists.SnapshotVersion < data.SnapshotVersion {
						datas[data.AggregateID] = data
					}
					if len(datas) >= batchSize {
						saver.batchSave(bgCtx, store, datas)
						datas = make(map[string]*AggregateSnapshotData)
					}
				case <-time.After(batchInterval):
					if len(datas) > 0 {
						saver.batchSave(bgCtx, store, datas)
						datas = make(map[string]*AggregateSnapshotData)
					} else if saver.status.Load() != 1 {
						break loop
					}
				}
			}
		}()
	}
}

func (saver *DefaultSnapshotStoreSaver) Stop() {
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

func (saver *DefaultSnapshotStoreSaver) batchSave(ctx context.Context, store SnapshotStore, dataMapping map[string]*AggregateSnapshotData) {
	datas := make([]AggregateSnapshotData, 0, len(dataMapping))
	logger := logging.Get(ctx)
	for _, data := range dataMapping {
		datas = append(datas, *data)
	}
	err := retrying.Retry(func() error {
		return store.SaveSnapshot(ctx, datas)
	}, time.Second, func(retryCount int, err error) bool {
		if _, ok := err.(net.Error); ok {
			return true
		}
		return false
	}, -1)
	if err != nil {
		logger.Error(fmt.Sprintf("batch save snapshot eventstream fail: %v", err))
	}
}
