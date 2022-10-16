package inmemoryes

import (
	"context"
	"fmt"
	"sync"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/domain/eventsourcing"
)

var _ eventsourcing.EventStore = (*InMemoryEventStore)(nil)

type InMemoryEventStore struct {
	snapshotMapper    sync.Map
	eventstreamMapper sync.Map
}

func (s *InMemoryEventStore) GetSnapshot(ctx context.Context, aggregateID string) (eventsourcing.AggregateSnapshotData, error) {
	var snapshotData eventsourcing.AggregateSnapshotData
	snapshotDataObj, ok := s.snapshotMapper.Load(aggregateID)
	if !ok {
		return snapshotData, nil
	}
	snapshotData = snapshotDataObj.(eventsourcing.AggregateSnapshotData)
	return snapshotData, nil
}

func (s *InMemoryEventStore) SaveSnapshot(ctx context.Context, data eventsourcing.AggregateSnapshotData) error {
	s.snapshotMapper.Store(data.AggregateID, data)
	return nil
}

func (s *InMemoryEventStore) QueryEventStreamList(ctx context.Context, aggregateID string, startVersion, endVersion int) (eventsourcing.EventStreamDataSlice, error) {
	eventStreamDatas := make(eventsourcing.EventStreamDataSlice, 0)
	eventstreamDatasObj, ok := s.eventstreamMapper.Load(aggregateID)
	if !ok {
		return eventStreamDatas, nil
	}
	eventStreamDatas = eventstreamDatasObj.(eventsourcing.EventStreamDataSlice)
	if len(eventStreamDatas) == 0 {
		return eventStreamDatas, nil
	}
	if startVersion < 1 {
		startVersion = 1
	}
	if endVersion > len(eventStreamDatas) {
		endVersion = len(eventStreamDatas)
	}
	eventStreamDatas = eventStreamDatas[startVersion-1 : endVersion]
	return eventStreamDatas, nil
}

func (s *InMemoryEventStore) AppendEventStream(ctx context.Context, data eventsourcing.EventStreamData) error {
	eventStreamDatas := make(eventsourcing.EventStreamDataSlice, 0)
	eventstreamDatasObj, ok := s.eventstreamMapper.Load(data.AggregateID)
	if ok {
		eventStreamDatas = eventstreamDatasObj.(eventsourcing.EventStreamDataSlice)
	}
	if len(eventStreamDatas)+1 != data.StreamVersion {
		return fmt.Errorf("%w: expected is %d, actual is %d", domain.ErrUnexpectedVersion, len(eventStreamDatas)+1, data.StreamVersion)
	}
	eventStreamDatas = append(eventStreamDatas, data)
	s.eventstreamMapper.Store(data.AggregateID, eventStreamDatas)
	return nil
}
