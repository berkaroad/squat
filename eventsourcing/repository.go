package eventsourcing

import (
	"context"
	"math"
	"reflect"
	"sync"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/publishedstore"
	"github.com/berkaroad/squat/store/snapshotstore"
)

var _ domain.Repository[*NullEventSourcedAggregate] = (*EventSourcedRepositoryBase[*NullEventSourcedAggregate])(nil)
var _ AggregateSnapshoter[*NullEventSourcedAggregate] = (*EventSourcedRepositoryBase[*NullEventSourcedAggregate])(nil)

type EventSourcedRepositoryBase[T EventSourcedAggregate] struct {
	EventStore     eventstore.EventStore         // must: specific one eventstore.EventStore
	EventBus       eventing.EventBus             // must: specific one eventing.EventBus
	PublishedStore publishedstore.PublishedStore // must: specific one publishedstore.PublishedEventStreamStore

	Serializer    serialization.Serializer    // optional: specific one serialization.Serializer
	SnapshotStore snapshotstore.SnapshotStore // optional: specific one snapshotstore.SnapshotStore

	publisher *eventPublisher
	initOnce  sync.Once
}

func (r *EventSourcedRepositoryBase[T]) Initialize(ctx context.Context) *EventSourcedRepositoryBase[T] {
	r.initOnce.Do(func() {
		if r.EventStore == nil {
			panic("field 'EventStore' is null")
		}
		if r.EventBus == nil {
			panic("field 'EventBus' is null")
		}
		if r.PublishedStore == nil {
			panic("field 'PublishedStore' is null")
		}

		r.publisher = (&eventPublisher{
			EventBus:       r.EventBus,
			EventStore:     r.EventStore,
			PublishedStore: r.PublishedStore,
			Serializer:     r.Serializer,
		}).Initialize(ctx)
	})
	return r
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
	if r.EventStore == nil {
		panic("field 'EventStore' is null")
	}

	var snapshot AggregateSnapshot
	var startVersion int
	if r.SnapshotStore != nil {
		snapshotData, err := r.SnapshotStore.GetSnapshot(ctx, aggregateID)
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

	eventStreamDatas, err := r.EventStore.QueryEventStreamList(ctx, aggregateID, startVersion+1, math.MaxInt32)
	if err != nil {
		return aggregate, err
	}

	eventStreams := make(domain.EventStreamSlice, 0)
	for _, eventStreamData := range eventStreamDatas {
		eventStream, err := eventstore.ToEventStream(r.Serializer, eventStreamData)
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
	if r.EventStore == nil {
		panic("field 'EventStore' is null")
	}
	if r.publisher == nil {
		panic("field 'publisher' is null")
	}
	if len(aggregate.Changes()) == 0 {
		return nil
	}

	eventStream := domain.EventStream{
		AggregateID:       aggregate.AggregateID(),
		AggregateTypeName: aggregate.AggregateTypeName(),
		StreamVersion:     aggregate.AggregateVersion(),
		Events:            aggregate.Changes(),
	}
	eventStreamData, err := eventstore.ToEventStreamData(r.Serializer, eventStream)
	if err != nil {
		return err
	}
	err = r.EventStore.AppendEventStream(ctx, eventStreamData)
	if err != nil {
		return err
	}
	aggregate.AcceptChanges()
	r.publisher.Publish(ctx, eventStream)
	return nil
}

func (r *EventSourcedRepositoryBase[T]) TaskSnapshot(ctx context.Context, aggregateID string) error {
	if r.SnapshotStore == nil {
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
	err = r.SnapshotStore.SaveSnapshot(ctx, snapshotData)
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
	if r.EventStore == nil {
		panic("field 'EventStore' is null")
	}

	var snapshot AggregateSnapshot
	var startVersion int
	if r.SnapshotStore != nil {
		snapshotData, err := r.SnapshotStore.GetSnapshot(ctx, aggregateID)
		if err != nil {
			return aggregate, err
		}
		if snapshotData.AggregateID != "" && snapshotData.SnapshotVersion > 0 && snapshotData.SnapshotVersion <= endVersion {
			snapshot, err = ToAggregateSnapshot(r.Serializer, snapshotData)
			if err != nil {
				return aggregate, err
			}

			startVersion = snapshotData.SnapshotVersion
		}
	}

	eventStreams := make(domain.EventStreamSlice, 0)
	if startVersion < endVersion {
		eventStreamDatas, err := r.EventStore.QueryEventStreamList(ctx, aggregateID, startVersion+1, endVersion)
		if err != nil {
			return aggregate, err
		}

		for _, eventStreamData := range eventStreamDatas {
			eventStream, err := eventstore.ToEventStream(r.Serializer, eventStreamData)
			if err != nil {
				return aggregate, err
			}
			eventStreams = append(eventStreams, eventStream)
		}
	}

	if snapshot != nil || len(eventStreams) > 0 {
		aggregate = reflect.New(typ).Interface().(T)
		aggregate.Restore(snapshot, eventStreams)
	}
	return aggregate, nil
}
