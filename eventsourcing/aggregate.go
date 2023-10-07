package eventsourcing

import (
	"fmt"
	"sort"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/serialization"
)

type EventSourcedAggregate interface {
	domain.Aggregate
	Changes() []domain.DomainEvent
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

func (s *EventSourcedAggregateBase) Apply(e domain.DomainEvent, mutate func(domain.DomainEvent)) {
	if e == nil {
		panic("param 'e' must not nil")
	}
	if mutate == nil {
		panic("param 'mutate' must not nil")
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
		panic("param 'restoreSnapshot' must not nil")
	}
	if mutate == nil {
		panic("param 'mutate' must not nil")
	}

	if snapshot != nil {
		if snapshot.SnapshotVersion() < 1 {
			return fmt.Errorf("%w: 'SnapshotVersion' must greater than or equal to 1", ErrInvalidSnapshot)
		}
		if err := restoreSnapshot(snapshot); err != nil {
			return err
		}
		s.version = snapshot.SnapshotVersion()
	}

	if len(eventStreams) > 0 {
		sort.Sort(eventStreams)
		if eventStreams[0].StreamVersion < 1 {
			return fmt.Errorf("%w: 'streamversion' must greater than or equal to 1", ErrInvalidEventStream)
		}
	}

	for _, es := range eventStreams {
		s.version = es.StreamVersion
		for _, e := range es.Events {
			mutate(e)
		}
	}

	return nil
}
