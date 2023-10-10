package inmemoryss

import (
	"context"
	"sync"

	"github.com/berkaroad/squat/store/snapshotstore"
)

var instance *InMemorySnapshotStore = &InMemorySnapshotStore{}

func Default() *InMemorySnapshotStore {
	return instance
}

var _ snapshotstore.SnapshotStore = (*InMemorySnapshotStore)(nil)

type InMemorySnapshotStore struct {
	snapshotMapper sync.Map
}

func (s *InMemorySnapshotStore) GetSnapshot(ctx context.Context, aggregateID string) (snapshotstore.AggregateSnapshotData, error) {
	var snapshotData snapshotstore.AggregateSnapshotData
	snapshotDataObj, ok := s.snapshotMapper.Load(aggregateID)
	if !ok {
		return snapshotData, nil
	}
	snapshotData = snapshotDataObj.(snapshotstore.AggregateSnapshotData)
	return snapshotData, nil
}

func (s *InMemorySnapshotStore) SaveSnapshot(ctx context.Context, datas []snapshotstore.AggregateSnapshotData) error {
	for _, data := range datas {
		s.snapshotMapper.Store(data.AggregateID, data)
	}
	return nil
}
