package inmemoryes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/berkaroad/squat/store/eventstore"
)

var instance *InMemoryEventStore = &InMemoryEventStore{SimulateTimeout: time.Millisecond * 20}

func Default() *InMemoryEventStore {
	return instance
}

var _ eventstore.EventStore = (*InMemoryEventStore)(nil)

type InMemoryEventStore struct {
	SimulateTimeout time.Duration

	eventstreamMapper sync.Map
}

func (s *InMemoryEventStore) QueryEventStreamList(ctx context.Context, aggregateID string, startVersion, endVersion int) (eventstore.EventStreamDataSlice, error) {
	time.Sleep(s.SimulateTimeout)
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

func (s *InMemoryEventStore) AppendEventStream(ctx context.Context, datas eventstore.EventStreamDataSlice) error {
	time.Sleep(s.SimulateTimeout)
	for _, data := range datas {
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
	}
	return nil
}
