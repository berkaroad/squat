package inmemoryss

import (
	"context"
	"sync"
	"time"

	"github.com/berkaroad/squat/store/snapshotstore"
)

var instance *InMemorySnapshotStore = &InMemorySnapshotStore{SimulateTimeout: time.Millisecond * 10 /* Simulate disk 100 iops */}

func Default() *InMemorySnapshotStore {
	return instance
}

var _ snapshotstore.SnapshotStore = (*InMemorySnapshotStore)(nil)

type InMemorySnapshotStore struct {
	SimulateTimeout time.Duration

	snapshotMapper sync.Map
}

func (s *InMemorySnapshotStore) GetSnapshot(ctx context.Context, aggregateID string) (snapshotstore.AggregateSnapshotData, error) {
	time.Sleep(s.SimulateTimeout)
	var snapshotData snapshotstore.AggregateSnapshotData
	snapshotDataObj, ok := s.snapshotMapper.Load(aggregateID)
	if !ok {
		return snapshotData, nil
	}
	snapshotData = snapshotDataObj.(snapshotstore.AggregateSnapshotData)
	return snapshotData, nil
}

func (s *InMemorySnapshotStore) SaveSnapshot(ctx context.Context, datas []snapshotstore.AggregateSnapshotData) error {
	time.Sleep(s.SimulateTimeout)
	for _, data := range datas {
		s.snapshotMapper.Store(data.AggregateID, data)
	}
	return nil
}
