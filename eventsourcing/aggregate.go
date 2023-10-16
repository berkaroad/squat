package eventsourcing

import (
	"fmt"
	"sort"

	"github.com/berkaroad/squat/caching"
	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/serialization"
)

type EventSourcedAggregate interface {
	domain.Aggregate
	Changes() []domain.DomainEvent
	HasChanged() bool
	Apply(e domain.DomainEvent)
	AcceptChanges()
	Snapshot() AggregateSnapshot
	Restore(snapshot AggregateSnapshot, eventStreams domain.EventStreamSlice) error
}

type AggregateSnapshot interface {
	serialization.Serializable
	AggregateID() string
	AggregateTypeName() string
	SnapshotVersion() int
}

type AggregateSnapshotCache interface {
	caching.Cache
}

type EventSourcedAggregateBase struct {
	version          int
	publishingEvents []domain.DomainEvent
}

func (s *EventSourcedAggregateBase) AggregateVersion() int {
	return s.version
}

func (s *EventSourcedAggregateBase) Changes() []domain.DomainEvent {
	return s.publishingEvents
}

func (s *EventSourcedAggregateBase) HasChanged() bool {
	return len(s.publishingEvents) > 0
}

func (s *EventSourcedAggregateBase) Apply(e domain.DomainEvent, mutate func(domain.DomainEvent)) {
	if e == nil {
		panic("param 'e' is null")
	}
	if mutate == nil {
		panic("param 'mutate' is null")
	}

	mutate(e)
	if len(s.publishingEvents) == 0 {
		s.version++
		s.publishingEvents = make([]domain.DomainEvent, 0)
	}
	s.publishingEvents = append(s.publishingEvents, e)
}

func (s *EventSourcedAggregateBase) AcceptChanges() {
	if len(s.publishingEvents) > 0 {
		s.publishingEvents = make([]domain.DomainEvent, 0)
	}
}

func (s *EventSourcedAggregateBase) Restore(snapshot AggregateSnapshot, restoreSnapshot func(AggregateSnapshot) error, eventStreams domain.EventStreamSlice, mutate func(domain.DomainEvent)) error {
	if restoreSnapshot == nil {
		panic("param 'restoreSnapshot' is null")
	}
	if mutate == nil {
		panic("param 'mutate' is null")
	}
	if s.HasChanged() {
		return ErrAggregateHasChanged
	}

	if snapshot != nil {
		if snapshot.SnapshotVersion() < 1 {
			return fmt.Errorf("%w: 'SnapshotVersion' should greater than or equal to 1", ErrInvalidSnapshot)
		}
		if err := restoreSnapshot(snapshot); err != nil {
			return err
		}
		s.version = snapshot.SnapshotVersion()
	}

	if len(eventStreams) > 0 {
		sort.Sort(eventStreams)
	}

	for i, es := range eventStreams {
		if es.StreamVersion != s.version+1 {
			return fmt.Errorf("%w: 'eventStreams[%d].StreamVersion' should be equal to %d", ErrInvalidEventStream, i, s.version+1)
		}
		s.version = es.StreamVersion
		for _, e := range es.Events {
			mutate(e)
		}
	}

	return nil
}
