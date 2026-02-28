package eventsourcing

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/messaging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/publishedstore"
	"github.com/berkaroad/squat/store/snapshotstore"
)

var _ domain.Repository[EventSourcedAggregate] = (*EventSourcedRepository[EventSourcedAggregate])(nil)

type EventSourcedRepository[T EventSourcedAggregate] struct {
	SnapshotEnabled  bool
	CacheEnabled     bool
	CacheExpiration  time.Duration
	es               eventstore.EventStore
	ess              eventstore.EventStoreSaver
	ss               snapshotstore.SnapshotStore
	sss              snapshotstore.SnapshotStoreSaver
	ep               *eventPublisher
	cache            AggregateSnapshotCache
	serializer       serialization.TextSerializer
	binarySerializer serialization.BinarySerializer

	initOnce            sync.Once
	initialized         bool
	aggregateStructType reflect.Type
	snapshotTypeName    string
}

func (r *EventSourcedRepository[T]) Initialize(
	eventBus eventing.EventBus,
	eventStore eventstore.EventStore,
	eventStoreSaver eventstore.EventStoreSaver,
	publishedStore publishedstore.PublishedStore,
	publishedStoreSaver publishedstore.PublishedStoreSaver,
	snapshotStore snapshotstore.SnapshotStore,
	snapshotStoreSaver snapshotstore.SnapshotStoreSaver,
	cache AggregateSnapshotCache,
	serializer serialization.TextSerializer,
	binarySerializer serialization.BinarySerializer,

) *EventSourcedRepository[T] {
	r.initOnce.Do(func() {
		typ := reflect.TypeOf((*T)(nil)).Elem()
		if typ.Kind() != reflect.Pointer {
			panic("typeparam 'T' from 'EventSourcedRepository[T]' should be a pointer to struct")
		}
		typ = typ.Elem()
		if typ.Kind() != reflect.Struct {
			panic("typeparam 'T' from 'EventSourcedRepository[T]' should be a pointer to struct")
		}

		if eventBus == nil {
			panic("param 'eventBus' is null")
		}
		if eventStore == nil {
			panic("param 'eventStore' is null")
		}
		if publishedStore == nil {
			panic("param 'publishedStore' is null")
		}

		r.es = eventStore
		r.ess = eventStoreSaver
		r.ss = snapshotStore
		r.sss = snapshotStoreSaver
		r.ep = &eventPublisher{
			eb:         eventBus,
			es:         eventStore,
			ps:         publishedStore,
			pss:        publishedStoreSaver,
			serializer: serializer,
		}
		r.cache = cache
		r.serializer = serializer
		r.binarySerializer = binarySerializer

		r.aggregateStructType = typ
		r.snapshotTypeName = reflect.New(typ).Interface().(T).Snapshot().TypeName()
		if r.cache != nil {
			cache.SetCacheName(r.snapshotTypeName)
		}
		r.initialized = true
	})
	return r
}

func (r *EventSourcedRepository[T]) Get(ctx context.Context, aggregateID string) (T, error) {
	if !r.initialized {
		panic("not initialized")
	}
	if aggregateID == "" {
		panic(domain.ErrEmptyAggregateID)
	}
	commandMeta := messaging.FromContext(ctx)
	if commandMeta == nil || commandMeta.Category != commanding.MailCategory {
		panic("'EventSourcedRepository[T].Get(context.Context, string)' should be invoked in CommandHandler")
	}

	logger := logging.Get(ctx)
	if r.CacheEnabled && r.cache != nil {
		dataInterface, loaded := r.cache.Get(aggregateID, r.snapshotTypeName)
		if loaded {
			newAggregate := reflect.New(r.aggregateStructType).Interface().(T)
			data := dataInterface.(AggregateSnapshot)
			err := newAggregate.Restore(data, nil)
			if err != nil {
				r.cache.Remove(aggregateID)
			} else {
				logger.Debug("load aggregate from cache",
					slog.String("aggregate-id", aggregateID),
				)
				return newAggregate, nil
			}
		}
	}

	var aggregate T
	var snapshot AggregateSnapshot
	var startVersion int
	if r.SnapshotEnabled && r.ss != nil {
		snapshotData, err := r.ss.GetSnapshot(ctx, aggregateID)
		if err != nil {
			err = fmt.Errorf("%w: %v", domain.ErrGetAggregateFail, err)
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
		err = fmt.Errorf("%w: %v", domain.ErrGetAggregateFail, err)
		logger.Error(err.Error(),
			slog.String("aggregate-id", aggregateID),
			slog.Int("start-version", startVersion+1),
			slog.Int("end-version", math.MaxInt32),
		)
		return aggregate, err
	}

	eventStreams := make(domain.EventStreamSlice, 0)
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
		if r.CacheEnabled && r.cache != nil {
			r.cache.Set(aggregate.AggregateID(), aggregate.Snapshot(), r.CacheExpiration)
		}
	}
	return aggregate, nil
}

func (r *EventSourcedRepository[T]) Save(ctx context.Context, aggregate T) error {
	if !r.initialized {
		panic("not initialized")
	}
	if aggregate.AggregateID() == "" {
		panic(domain.ErrEmptyAggregateID)
	}
	commandMeta := messaging.FromContext(ctx)
	if commandMeta == nil || commandMeta.Category != commanding.MailCategory {
		panic("'EventSourcedRepository[T].Save(context.Context, T)' should be invoked in CommandHandler")
	}

	if !aggregate.HasChanged() {
		return domain.ErrAggregateNoChange
	}
	commandMeta.Extensions = commandMeta.Extensions.Clone().
		Set(messaging.ExtensionKeyAggregateChanged, "true")

	logger := logging.Get(ctx)
	eventStream := domain.EventStream{
		AggregateID:   aggregate.AggregateID(),
		AggregateType: aggregate.AggregateTypeName(),
		StreamVersion: aggregate.AggregateVersion(),
		Events:        aggregate.Changes(),
		CommandID:     commandMeta.MessageID,
		CommandType:   commandMeta.MessageType,
		Extensions: commandMeta.Extensions.Clone().
			Remove(messaging.ExtensionKeyFromMessageID).
			Remove(messaging.ExtensionKeyFromMessageType).
			Remove(messaging.ExtensionKeyAggregateChanged),
	}
	eventStreamData, err := ToEventStreamData(r.serializer, eventStream)
	if err != nil {
		err = fmt.Errorf("%w: %v", domain.ErrSaveAggregateFail, err)
		logger.Error(err.Error(),
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
		err = fmt.Errorf("%w: %v", domain.ErrSaveAggregateFail, err)
		logger.Error(err.Error(),
			slog.Int("stream-version", eventStream.StreamVersion),
		)
		return err
	}
	aggregate.AcceptChanges()
	if r.CacheEnabled && r.cache != nil {
		r.cache.Set(aggregate.AggregateID(), aggregate.Snapshot(), r.CacheExpiration)
	}

	// publish eventstream
	r.ep.Publish(eventStream)

	// save snapshot
	if r.SnapshotEnabled && (r.ss != nil || r.sss != nil) {
		snapshotData, err := ToAggregateSnapshotData(r.serializer, aggregate.Snapshot())
		if err == nil {
			if r.sss != nil {
				r.sss.SaveSnapshot(snapshotData)
			} else {
				r.ss.SaveSnapshot(ctx, []snapshotstore.AggregateSnapshotData{snapshotData})
			}
		}
	}
	return nil
}
