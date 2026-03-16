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
	SaveSnapshot(data AggregateSnapshotData)
	Start()
	Stop()
}

const (
	defaultTakeSnapshotMinVersionDiff int           = 10
	defaultBatchSize                  int           = 100
	defaultBatchInterval              time.Duration = time.Second
)

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
		saver.ss = snapshotStore
		saver.initialized = true
	})
	return saver
}

func (saver *DefaultSnapshotStoreSaver) SaveSnapshot(data AggregateSnapshotData) {
	if !saver.initialized {
		panic("not initialized")
	}

	if saver.status.Load() != 1 {
		logger := logging.Get(context.TODO())
		logger.Warn("'DefaultSnapshotStoreSaver' has stopped")
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
				minVersionDiff = defaultTakeSnapshotMinVersionDiff
			}
			batchSize := saver.BatchSize
			if batchSize <= 0 {
				batchSize = defaultBatchSize
			}
			saver.receiverCh = make(chan *AggregateSnapshotData, batchSize*2)
			batchInterval := saver.BatchInterval
			if batchInterval <= 0 {
				batchInterval = defaultBatchInterval
			}
			shardingAlgorithm := saver.ShardingAlgorithm
			if shardingAlgorithm == nil {
				shardingAlgorithm = func(aggregateID string) uint8 { return 0 }
			}
			store := saver.ss
			shardingMapping := make(map[uint8]map[string]*AggregateSnapshotData)
			shardingTimeMapping := make(map[uint8]time.Time)
			snapshotVersionDiffMapping := make(map[string]int)
			bgCtx := context.Background()
		loop:
			for {
				select {
				case data, ok := <-saver.receiverCh:
					if !ok {
						break loop
					}
					if data.SnapshotVersion < minVersionDiff {
						continue
					}
					shardKey := shardingAlgorithm(data.AggregateID)
					if _, ok := shardingMapping[shardKey]; !ok {
						shardingMapping[shardKey] = make(map[string]*AggregateSnapshotData, batchSize)
					}
					if _, ok := shardingTimeMapping[shardKey]; !ok {
						shardingTimeMapping[shardKey] = time.Now()
					}
					if _, ok := snapshotVersionDiffMapping[data.AggregateID]; !ok {
						snapshotVersionDiffMapping[data.AggregateID] = 0
					}
					if snapshotVersionDiffMapping[data.AggregateID] < data.SnapshotVersion {
						snapshotVersionDiffMapping[data.AggregateID] += 1
						if snapshotVersionDiffMapping[data.AggregateID] >= minVersionDiff {
							shardingMapping[shardKey][data.AggregateID] = data
						}
					}
					if len(shardingMapping[shardKey]) >= batchSize {
						delete(shardingTimeMapping, shardKey)
						saver.batchSave(bgCtx, store, shardKey, shardingMapping[shardKey])
						for aggrID := range shardingMapping[shardKey] {
							delete(snapshotVersionDiffMapping, aggrID)
						}
						shardingMapping[shardKey] = make(map[string]*AggregateSnapshotData, batchSize)
						time.Sleep(batchInterval)
					}
				case <-time.After(batchInterval / 10):
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
								for aggrID := range shardingMapping[shardKey] {
									delete(snapshotVersionDiffMapping, aggrID)
								}
								shardingMapping[shardKey] = make(map[string]*AggregateSnapshotData, batchSize)
								wg.Add(1)
								go func(shardKey uint8, datas map[string]*AggregateSnapshotData) {
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

func (saver *DefaultSnapshotStoreSaver) Stop() {
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

func (saver *DefaultSnapshotStoreSaver) batchSave(ctx context.Context, store SnapshotStore, shardKey uint8, dataMapping map[string]*AggregateSnapshotData) {
	datas := make([]AggregateSnapshotData, 0, len(dataMapping))
	logger := logging.Get(ctx)
	for _, data := range dataMapping {
		datas = append(datas, *data)
	}
	err := retrying.Retry(func() error {
		return store.SaveSnapshot(ctx, datas)
	}, time.Second, func(retryCount int, err error) bool {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true
		}
		return false
	}, -1)
	if err != nil {
		logger.Error(fmt.Sprintf("batch save to snapshotstore fail: %v", err),
			slog.Uint64("shard_key", uint64(shardKey)),
			slog.Uint64("data_count", uint64(len(dataMapping))),
		)
	} else {
		logger.Debug("batch save to snapshotstore success",
			slog.Uint64("shard_key", uint64(shardKey)),
			slog.Uint64("data_count", uint64(len(dataMapping))),
		)
	}
}
