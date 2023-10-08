package eventsourcing

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"sync"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/messaging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/snapshotstore"
)

var _ domain.Repository[EventSourcedAggregate] = (*EventSourcedRepositoryBase[EventSourcedAggregate])(nil)
var _ AggregateSnapshoter[EventSourcedAggregate] = (*EventSourcedRepositoryBase[EventSourcedAggregate])(nil)

type EventSourcedRepositoryBase[T EventSourcedAggregate] struct {
	es         eventstore.EventStore
	ess        eventstore.EventStoreSaver
	ss         snapshotstore.SnapshotStore
	ep         eventing.EventPublisher
	serializer serialization.Serializer

	initOnce            sync.Once
	initialized         bool
	aggregateStructType reflect.Type
}

func (r *EventSourcedRepositoryBase[T]) Initialize(
	eventStore eventstore.EventStore,
	eventStoreSaver eventstore.EventStoreSaver,
	snapshotStore snapshotstore.SnapshotStore,
	eventPublisher eventing.EventPublisher,
	serializer serialization.Serializer,
) *EventSourcedRepositoryBase[T] {
	r.initOnce.Do(func() {
		typ := reflect.TypeOf((*T)(nil)).Elem()
		if typ.Kind() != reflect.Pointer {
			panic("typeparam 'T' from 'EventSourcedRepositoryBase' must be a pointer to struct")
		}
		typ = typ.Elem()
		if typ.Kind() != reflect.Struct {
			panic("typeparam 'T' from 'EventSourcedRepositoryBase' must be a pointer to struct")
		}

		if eventStore == nil {
			panic("param 'eventStore' is null")
		}
		if eventPublisher == nil {
			panic("param 'eventPublisher' is null")
		}

		r.es = eventStore
		r.ess = eventStoreSaver
		r.ss = snapshotStore
		r.ep = eventPublisher
		r.serializer = serializer

		r.aggregateStructType = typ
		r.initialized = true
	})
	return r
}

func (r *EventSourcedRepositoryBase[T]) Get(ctx context.Context, aggregateID string) (T, error) {
	if !r.initialized {
		panic("not initialized")
	}

	logger := logging.Get(ctx)
	var snapshot AggregateSnapshot
	var startVersion int
	if r.ss != nil {
		snapshotData, err := r.ss.GetSnapshot(ctx, aggregateID)
		if err != nil {
			err = fmt.Errorf("%w: %v", ErrGetSnapshotFail, err)
			logger.Warn(err.Error(),
				slog.String("aggregate-id", aggregateID),
			)
		} else if snapshotData.AggregateID != "" && snapshotData.SnapshotVersion > 0 {
			snapshot, err = ToAggregateSnapshot(r.serializer, snapshotData)
			if err != nil {
				logger.Warn(err.Error(),
					slog.String("aggregate-id", snapshotData.AggregateID),
					slog.Int("snapshot-version", snapshotData.SnapshotVersion),
				)
			} else {
				startVersion = snapshotData.SnapshotVersion
			}
		}
	}

	var aggregate T
	eventStreamDatas, err := r.es.QueryEventStreamList(ctx, aggregateID, startVersion+1, math.MaxInt32)
	if err != nil {
		err = fmt.Errorf("%w: %v", ErrQueryEventStreamListFail, err)
		logger.Error(err.Error(),
			slog.String("aggregate-id", aggregateID),
			slog.Int("start-version", startVersion+1),
			slog.Int("end-version", math.MaxInt32),
		)
		return aggregate, err
	}

	eventStreams := make(domain.EventStreamSlice, 0)
	for _, eventStreamData := range eventStreamDatas {
		eventStream, err := eventstore.ToEventStream(r.serializer, eventStreamData)
		if err != nil {
			logger.Error(err.Error(),
				slog.String("aggregate-id", eventStreamData.AggregateID),
				slog.Int("stream-version", eventStreamData.StreamVersion),
			)
			return aggregate, err
		}
		eventStreams = append(eventStreams, eventStream)
	}

	if snapshot != nil || len(eventStreams) > 0 {
		aggregate = reflect.New(r.aggregateStructType).Interface().(T)
		aggregate.Restore(snapshot, eventStreams)
	}
	return aggregate, nil
}

func (r *EventSourcedRepositoryBase[T]) Save(ctx context.Context, aggregate T) error {
	if !r.initialized {
		panic("not initialized")
	}
	if len(aggregate.Changes()) == 0 {
		return ErrAggregateNoChange
	}

	logger := logging.Get(ctx)
	if aggregate.AggregateID() == "" {
		return ErrEmptyAggregateID
	}

	eventStream := domain.EventStream{
		AggregateID:       aggregate.AggregateID(),
		AggregateTypeName: aggregate.AggregateTypeName(),
		StreamVersion:     aggregate.AggregateVersion(),
		Events:            aggregate.Changes(),
		CommandID:         messaging.FromContext(ctx).ID,
	}
	eventStreamData, err := eventstore.ToEventStreamData(r.serializer, eventStream)
	if err != nil {
		logger.Error(err.Error(),
			slog.String("aggregate-id", eventStream.AggregateID),
			slog.Int("stream-version", eventStream.StreamVersion),
		)
		return err
	}
	if r.ess != nil {
		err = <-r.ess.AppendEventStream(ctx, eventStreamData)
	} else {
		err = r.es.AppendEventStream(ctx, eventstore.EventStreamDataSlice{eventStreamData})
	}
	if err != nil {
		err = fmt.Errorf("%w: %v", ErrAppendEventStreamFail, err)
		logger.Error(err.Error(),
			slog.String("aggregate-id", aggregate.AggregateID()),
			slog.Int("stream-version", aggregate.AggregateVersion()),
		)
		return err
	}
	aggregate.AcceptChanges()
	r.ep.Publish(ctx, eventStream)
	return nil
}

func (r *EventSourcedRepositoryBase[T]) TaskSnapshot(ctx context.Context, aggregateID string) error {
	if !r.initialized {
		panic("not initialized")
	}
	if r.ss == nil {
		return nil
	}

	logger := logging.Get(ctx)
	aggregate, err := r.Get(ctx, aggregateID)
	if err != nil {
		return err
	}
	if reflect.ValueOf(aggregate).IsNil() {
		return nil
	}

	snapshot := aggregate.Snapshot()
	snapshotData, err := ToAggregateSnapshotData(r.serializer, snapshot)
	if err != nil {
		logger.Error(err.Error(),
			slog.String("aggregate-id", snapshot.AggregateID()),
			slog.Int("snapshot-version", snapshot.SnapshotVersion()),
		)
		return err
	}
	err = r.ss.SaveSnapshot(ctx, snapshotData)
	if err != nil {
		err = fmt.Errorf("%w: %v", ErrSaveSnapshotFail, err)
		logger.Error(err.Error(),
			slog.String("aggregate-id", snapshotData.AggregateID),
			slog.Int("snapshot-version", snapshotData.SnapshotVersion),
		)
		return err
	}
	return nil
}

func (r *EventSourcedRepositoryBase[T]) GetSnapshot(ctx context.Context, aggregateID string, endVersion int) (T, error) {
	if !r.initialized {
		panic("not initialized")
	}

	logger := logging.Get(ctx)
	var snapshot AggregateSnapshot
	var startVersion int
	if r.ss != nil {
		snapshotData, err := r.ss.GetSnapshot(ctx, aggregateID)
		if err != nil {
			logger.Warn(fmt.Sprintf("get snapshot fail: %v", err),
				slog.String("aggregate-id", aggregateID),
			)
		} else if snapshotData.AggregateID != "" && snapshotData.SnapshotVersion > 0 && snapshotData.SnapshotVersion <= endVersion {
			snapshot, err = ToAggregateSnapshot(r.serializer, snapshotData)
			if err != nil {
				logger.Warn(err.Error(),
					slog.String("aggregate-id", snapshotData.AggregateID),
					slog.Int("snapshot-version", snapshotData.SnapshotVersion),
				)
			} else {
				startVersion = snapshotData.SnapshotVersion
			}
		}
	}

	var aggregate T
	eventStreams := make(domain.EventStreamSlice, 0)
	if startVersion < endVersion {
		eventStreamDatas, err := r.es.QueryEventStreamList(ctx, aggregateID, startVersion+1, endVersion)
		if err != nil {
			err = fmt.Errorf("%w: %v", ErrQueryEventStreamListFail, err)
			logger.Error(err.Error(),
				slog.String("aggregate-id", aggregateID),
				slog.Int("start-version", startVersion+1),
				slog.Int("end-version", endVersion),
			)
			return aggregate, err
		}

		for _, eventStreamData := range eventStreamDatas {
			eventStream, err := eventstore.ToEventStream(r.serializer, eventStreamData)
			if err != nil {
				logger.Error(err.Error(),
					slog.String("aggregate-id", eventStream.AggregateID),
					slog.Int("stream-version", eventStream.StreamVersion),
				)
				return aggregate, err
			}
			eventStreams = append(eventStreams, eventStream)
		}
	}

	if snapshot != nil || len(eventStreams) > 0 {
		aggregate = reflect.New(r.aggregateStructType).Interface().(T)
		aggregate.Restore(snapshot, eventStreams)
	}
	return aggregate, nil
}
