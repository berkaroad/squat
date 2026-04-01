package inmemoryes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/berkaroad/squat/store/eventstore"
)

var instance *InMemoryEventStore = &InMemoryEventStore{SimulateTimeout: time.Millisecond * 10 /* Simulate disk 100 iops */}

func Default() *InMemoryEventStore {
	return instance
}

var _ eventstore.EventStore = (*InMemoryEventStore)(nil)

type InMemoryEventStore struct {
	SimulateTimeout time.Duration

	eventstreamMapper sync.Map
	commandIDs        sync.Map
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
	localCommandIDs := make(map[string]struct{})
	localAggregateIDs := make(map[string]struct{})
	for _, data := range datas {
		if _, ok := s.commandIDs.Load(data.CommandID); ok {
			return eventstore.NewErrDuplicateCommandID(data.CommandID)
		}

		if _, ok := localCommandIDs[data.CommandID]; ok {
			return fmt.Errorf("%w: command-id %s", eventstore.ErrEventStreamConcurrencyConflict, data.CommandID)
		}
		localCommandIDs[data.CommandID] = struct{}{}

		if _, ok := localAggregateIDs[data.AggregateID]; ok {
			return fmt.Errorf("%w: aggregate-id %s", eventstore.ErrEventStreamConcurrencyConflict, data.AggregateID)
		}
		localAggregateIDs[data.AggregateID] = struct{}{}
	}

	for _, data := range datas {
		if _, loaded := s.commandIDs.LoadOrStore(data.CommandID, struct{}{}); loaded {
			return eventstore.NewErrDuplicateCommandID(data.CommandID)
		}
		eventStreamDatas := make(eventstore.EventStreamDataSlice, 0)
		eventstreamDatasObj, ok := s.eventstreamMapper.Load(data.AggregateID)
		if ok {
			eventStreamDatas = eventstreamDatasObj.(eventstore.EventStreamDataSlice)
		}
		if len(eventStreamDatas)+1 != data.StreamVersion {
			s.commandIDs.Delete(data.CommandID)
			return fmt.Errorf("%w: expected is %d, actual is %d", eventstore.ErrUnexpectedVersion, len(eventStreamDatas)+1, data.StreamVersion)
		}
		eventStreamDatas = append(eventStreamDatas, data)
		s.eventstreamMapper.Store(data.AggregateID, eventStreamDatas)
	}
	return nil
}
