package inmemoryes

import (
	"context"
	"fmt"
	"sync"

	"github.com/berkaroad/squat/store/eventstore"
)

var _ eventstore.EventStore = (*InMemoryEventStore)(nil)

type InMemoryEventStore struct {
	eventstreamMapper sync.Map
}

func (s *InMemoryEventStore) QueryEventStreamList(ctx context.Context, aggregateID string, startVersion, endVersion int) (eventstore.EventStreamDataSlice, error) {
	eventStreamDatas := make(eventstore.EventStreamDataSlice, 0)
	eventstreamDatasObj, ok := s.eventstreamMapper.Load(aggregateID)
	if !ok {
		return eventStreamDatas, nil
	}
	eventStreamDatas = eventstreamDatasObj.(eventstore.EventStreamDataSlice)
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

func (s *InMemoryEventStore) AppendEventStream(ctx context.Context, data eventstore.EventStreamData) error {
	eventStreamDatas := make(eventstore.EventStreamDataSlice, 0)
	eventstreamDatasObj, ok := s.eventstreamMapper.Load(data.AggregateID)
	if ok {
		eventStreamDatas = eventstreamDatasObj.(eventstore.EventStreamDataSlice)
	}
	if len(eventStreamDatas)+1 != data.StreamVersion {
		return fmt.Errorf("%w: expected is %d, actual is %d", eventstore.ErrUnexpectedVersion, len(eventStreamDatas)+1, data.StreamVersion)
	}
	eventStreamDatas = append(eventStreamDatas, data)
	s.eventstreamMapper.Store(data.AggregateID, eventStreamDatas)
	return nil
}