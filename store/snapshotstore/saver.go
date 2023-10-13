package snapshotstore

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

type SnapshotStoreSaver interface {
	SaveSnapshot(ctx context.Context, data AggregateSnapshotData)
	Start()
	Stop()
}

var _ SnapshotStoreSaver = (*DefaultSnapshotStoreSaver)(nil)

type DefaultSnapshotStoreSaver struct {
	TakeSnapshotMinVersionDiff int
	BatchSize                  int
	BatchInterval              time.Duration
	ShardingAlgorithm          func(aggregateID string) uint8
	ss                         SnapshotStore

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
			minVersionDiff := saver.TakeSnapshotMinVersionDiff
			if minVersionDiff <= 0 {
				minVersionDiff = 10
			}
			batchSize := cap(saver.receiverCh)
			batchInterval := saver.BatchInterval
			if batchInterval <= 0 {
				batchInterval = time.Second * 30
			}
			shardingAlgorithm := saver.ShardingAlgorithm
			if shardingAlgorithm == nil {
				shardingAlgorithm = func(aggregateID string) uint8 { return 0 }
			}
			store := saver.ss
			shardingMapping := make(map[uint8]map[string]*AggregateSnapshotData)
			snapshotVersionDiffMapping := make(map[string]*snapshotVersionDiff)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data := <-saver.receiverCh:
					if data.SnapshotVersion < minVersionDiff {
						continue
					}
					shardKey := shardingAlgorithm(data.AggregateID)
					if _, ok := shardingMapping[shardKey]; !ok {
						shardingMapping[shardKey] = make(map[string]*AggregateSnapshotData)
					}
					if exists, ok := snapshotVersionDiffMapping[data.AggregateID]; !ok {
						snapshotVersionDiffMapping[data.AggregateID] = &snapshotVersionDiff{StartVersion: 1, EndVersion: 1}
					} else if exists.EndVersion < data.SnapshotVersion {
						snapshotVersionDiffMapping[data.AggregateID].EndVersion++
						if snapshotVersionDiffMapping[data.AggregateID].Diff() >= minVersionDiff {
							shardingMapping[shardKey][data.AggregateID] = data
						}
					}
					if len(shardingMapping[shardKey]) >= batchSize {
						saver.batchSave(bgCtx, store, shardKey, shardingMapping[shardKey])
						for aggrID := range shardingMapping[shardKey] {
							delete(snapshotVersionDiffMapping, aggrID)
						}
						shardingMapping[shardKey] = make(map[string]*AggregateSnapshotData)
					}
				case <-time.After(batchInterval):
					hasData := false
					for shardKey, datas := range shardingMapping {
						if len(datas) > 0 {
							saver.batchSave(bgCtx, store, shardKey, datas)
							for aggrID := range shardingMapping[shardKey] {
								delete(snapshotVersionDiffMapping, aggrID)
							}
							shardingMapping[shardKey] = make(map[string]*AggregateSnapshotData)
						}
					}
					if !hasData && saver.status.Load() != 1 {
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

func (saver *DefaultSnapshotStoreSaver) batchSave(ctx context.Context, store SnapshotStore, shardKey uint8, dataMapping map[string]*AggregateSnapshotData) {
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
		logger.Error(fmt.Sprintf("batch save to snapshotstore fail: %v", err),
			slog.Uint64("shard-key", uint64(shardKey)),
			slog.Uint64("data-count", uint64(len(dataMapping))),
		)
	} else {
		logger.Info("batch save to snapshotstore success",
			slog.Uint64("shard-key", uint64(shardKey)),
			slog.Uint64("data-count", uint64(len(dataMapping))),
		)
	}
}

type snapshotVersionDiff struct {
	StartVersion int
	EndVersion   int
}

func (diff *snapshotVersionDiff) Diff() int {
	return diff.EndVersion - diff.StartVersion
}
