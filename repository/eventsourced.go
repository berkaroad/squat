package repository

import (
	"context"
	"fmt"
	"math"
	"reflect"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/domain/eventsourcing"
	"github.com/berkaroad/squat/eventstore"
	"github.com/berkaroad/squat/serialization"
)

var _ domain.Repository[*eventsourcing.NullEventSourcedAggregate] = (*EventSourcedRepositoryBase[*eventsourcing.NullEventSourcedAggregate])(nil)
var _ eventsourcing.AggregateSnapshoter[*eventsourcing.NullEventSourcedAggregate] = (*EventSourcedRepositoryBase[*eventsourcing.NullEventSourcedAggregate])(nil)

type EventSourcedRepositoryBase[T eventsourcing.EventSourcedAggregate] struct {
	Serializer      serialization.Serializer
	Store           eventstore.EventStore
	SnapshotEnabled bool
}

func (r *EventSourcedRepositoryBase[T]) Get(ctx context.Context, aggregateID string) (T, error) {
	var aggregate T
	typ := reflect.TypeOf(aggregate)
	if typ.Kind() != reflect.Pointer {
		panic("typeparam 'T' from 'EventSourcedRepositoryBase' must be a pointer to struct")
	}
	typ = typ.Elem()
	if typ.Kind() != reflect.Struct {
		panic("typeparam 'T' from 'EventSourcedRepositoryBase' must be a pointer to struct")
	}
	if r.Store == nil {
		panic("field 'Store' is null")
	}

	var snapshot eventsourcing.AggregateSnapshot
	var startVersion int
	if r.SnapshotEnabled {
		snapshotData, err := r.Store.GetSnapshot(ctx, aggregateID)
		if err != nil {
			return aggregate, err
		}
		if snapshotData.AggregateID != "" && snapshotData.SnapshotVersion > 0 {
			snapshot, err = ToAggregateSnapshot(r.Serializer, snapshotData)
			if err != nil {
				return aggregate, err
			}
			startVersion = snapshotData.SnapshotVersion
		}
	}

	eventStreamDatas, err := r.Store.QueryEventStreamList(ctx, aggregateID, startVersion+1, math.MaxInt32)
	if err != nil {
		return aggregate, err
	}

	eventStreams := make(domain.EventStreamSlice, 0)
	for _, eventStreamData := range eventStreamDatas {
		eventStream, err := ToEventStream(r.Serializer, eventStreamData)
		if err != nil {
			return aggregate, err
		}
		eventStreams = append(eventStreams, eventStream)
	}

	if snapshot != nil || len(eventStreams) > 0 {
		aggregate = reflect.New(typ).Interface().(T)
		aggregate.Restore(snapshot, eventStreams)
	}
	return aggregate, nil
}

func (r *EventSourcedRepositoryBase[T]) Save(ctx context.Context, aggregate T) error {
	if r.Store == nil {
		panic("field 'Store' is null")
	}

	if len(aggregate.Changes()) == 0 {
		return nil
	}

	eventStream := domain.EventStream{
		AggregateID:   aggregate.AggregateID(),
		StreamVersion: aggregate.AggregateVersion(),
		Events:        aggregate.Changes(),
	}
	eventStreamData, err := ToEventStreamData(r.Serializer, eventStream)
	if err != nil {
		return err
	}
	err = r.Store.AppendEventStream(ctx, eventStreamData)
	if err != nil {
		return err
	}
	aggregate.AcceptChanges()
	return nil
}

func (r *EventSourcedRepositoryBase[T]) TaskSnapshot(ctx context.Context, aggregateID string) error {
	if !r.SnapshotEnabled {
		return nil
	}

	aggregate, err := r.Get(ctx, aggregateID)
	if err != nil {
		return err
	}
	snapshot := aggregate.Snapshot()
	snapshotData, err := ToAggregateSnapshotData(r.Serializer, snapshot)
	if err != nil {
		return err
	}
	err = r.Store.SaveSnapshot(ctx, snapshotData)
	if err != nil {
		return err
	}
	return nil
}

func (r *EventSourcedRepositoryBase[T]) GetSnapshot(ctx context.Context, aggregateID string, endVersion int) (T, error) {
	var aggregate T
	typ := reflect.TypeOf(aggregate)
	if typ.Kind() != reflect.Pointer {
		panic("typeparam 'T' from 'EventSourcedRepositoryBase' must be a pointer to struct")
	}
	typ = typ.Elem()
	if typ.Kind() != reflect.Struct {
		panic("typeparam 'T' from 'EventSourcedRepositoryBase' must be a pointer to struct")
	}
	if r.Store == nil {
		panic("field 'Store' is null")
	}

	eventStreamDatas, err := r.Store.QueryEventStreamList(ctx, aggregateID, 1, endVersion)
	if err != nil {
		return aggregate, err
	}

	eventStreams := make(domain.EventStreamSlice, 0)
	for _, eventStreamData := range eventStreamDatas {
		eventStream, err := ToEventStream(r.Serializer, eventStreamData)
		if err != nil {
			return aggregate, err
		}
		eventStreams = append(eventStreams, eventStream)
	}

	if len(eventStreams) > 0 {
		aggregate = reflect.New(typ).Interface().(T)
		aggregate.Restore(nil, eventStreams)
	}
	return aggregate, nil
}

func ToAggregateSnapshot(serializer serialization.Serializer, asd eventstore.AggregateSnapshotData) (eventsourcing.AggregateSnapshot, error) {
	snapshotObj, err := serialization.Deserialize(serializer, asd.SnapshotType, []byte(asd.Body))
	if err != nil {
		return nil, err
	}
	snapshot, ok := snapshotObj.(eventsourcing.AggregateSnapshot)
	if !ok {
		return nil, fmt.Errorf("cann't cast '%#v' to 'eventsourcing.AggregateSnapshot'", snapshotObj)
	}
	return snapshot, nil
}

func ToAggregateSnapshotData(serializer serialization.Serializer, as eventsourcing.AggregateSnapshot) (eventstore.AggregateSnapshotData, error) {
	asd := eventstore.AggregateSnapshotData{
		AggregateID:     as.AggregateID(),
		SnapshotVersion: as.SnapshotVersion(),
		SnapshotType:    as.TypeName(),
	}
	body, err := serialization.Serialize(serializer, as)
	if err != nil {
		return asd, err
	}
	asd.Body = string(body)
	return asd, nil
}

func ToEventStream(serializer serialization.Serializer, esd eventstore.EventStreamData) (domain.EventStream, error) {
	es := domain.EventStream{
		AggregateID:   esd.AggregateID,
		StreamVersion: esd.StreamVersion,
		Events:        make([]domain.DomainEvent, len(esd.Events)),
	}
	for i, eventData := range esd.Events {
		eventObj, err := serialization.Deserialize(serializer, eventData.EventType, []byte(eventData.Body))
		if err != nil {
			return es, err
		}
		event, ok := eventObj.(domain.DomainEvent)
		if !ok {
			return es, fmt.Errorf("cann't cast '%#v' to 'domain.DomainEvent'", eventObj)
		}
		es.Events[i] = event
	}
	return es, nil
}

func ToEventStreamData(serializer serialization.Serializer, es domain.EventStream) (eventstore.EventStreamData, error) {
	esd := eventstore.EventStreamData{
		AggregateID:   es.AggregateID,
		StreamVersion: es.StreamVersion,
		Events:        make([]eventstore.DomainEventData, len(es.Events)),
	}
	for i, event := range es.Events {
		body, err := serialization.Serialize(serializer, event)
		if err != nil {
			return esd, err
		}
		esd.Events[i] = eventstore.DomainEventData{
			EventID:   event.EventID(),
			EventType: event.TypeName(),
			OccurTime: event.OccurTime().Unix(),
			Body:      string(body),
		}
	}
	return esd, nil
}
