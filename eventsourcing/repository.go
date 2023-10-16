package eventsourcing

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/messaging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/snapshotstore"
)

var _ domain.Repository[EventSourcedAggregate] = (*EventSourcedRepositoryBase[EventSourcedAggregate])(nil)

type EventSourcedRepositoryBase[T EventSourcedAggregate] struct {
	SnapshotEnabled  bool
	CacheEnabled     bool
	CacheExpiration  time.Duration
	es               eventstore.EventStore
	ess              eventstore.EventStoreSaver
	ss               snapshotstore.SnapshotStore
	sss              snapshotstore.SnapshotStoreSaver
	ep               EventPublisher
	cache            AggregateSnapshotCache
	serializer       serialization.TextSerializer
	binarySerializer serialization.BinarySerializer

	initOnce            sync.Once
	initialized         bool
	aggregateStructType reflect.Type
	snapshotTypeName    string
}

func (r *EventSourcedRepositoryBase[T]) Initialize(
	eventStore eventstore.EventStore,
	eventStoreSaver eventstore.EventStoreSaver,
	snapshotStore snapshotstore.SnapshotStore,
	snapshotStoreSaver snapshotstore.SnapshotStoreSaver,
	eventPublisher EventPublisher,
	cache AggregateSnapshotCache,
	serializer serialization.TextSerializer,
	binarySerializer serialization.BinarySerializer,
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
		r.sss = snapshotStoreSaver
		r.ep = eventPublisher
		r.cache = cache
		r.serializer = serializer
		r.binarySerializer = binarySerializer

		r.aggregateStructType = typ
		r.snapshotTypeName = reflect.New(typ).Interface().(T).Snapshot().TypeName()
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
	if r.CacheEnabled && r.cache != nil {
		dataInterface, loaded := r.cache.Get(aggregateID, r.snapshotTypeName)
		if loaded {
			newAggregate := reflect.New(r.aggregateStructType).Interface().(T)
			data := dataInterface.(AggregateSnapshot)
			err := newAggregate.Restore(data, nil)
			if err != nil {
				r.cache.Remove(aggregateID)
			}
			logger.Debug("load aggregate from cache",
				slog.String("aggregate-id", aggregateID),
			)
			return newAggregate, nil
		}
	}

	var aggregate T
	eventStreams := make(domain.EventStreamSlice, 0)
	if r.SnapshotEnabled && r.ss != nil {
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

	for _, eventStreamData := range eventStreamDatas {
		eventStream, err := ToEventStream(r.serializer, eventStreamData)
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
		newAggregate := reflect.New(r.aggregateStructType).Interface().(T)
		if err = newAggregate.Restore(snapshot, eventStreams); err != nil {
			return aggregate, err
		}
		aggregate = newAggregate
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
	eventStreamData, err := ToEventStreamData(r.serializer, eventStream)
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
	if r.CacheEnabled && r.cache != nil {
		r.cache.Set(aggregate.AggregateID(), aggregate.Snapshot(), r.CacheExpiration)
	}
	r.ep.Publish(ctx, eventStream)

	// save snapshot
	if r.SnapshotEnabled && r.sss != nil {
		snapshotData, err := ToAggregateSnapshotData(r.serializer, aggregate.Snapshot())
		if err == nil {
			r.sss.SaveSnapshot(ctx, snapshotData)
		}
	}
	return nil
}
