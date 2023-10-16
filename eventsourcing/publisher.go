package eventsourcing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/publishedstore"
	"github.com/berkaroad/squat/utilities/goroutine"
	"github.com/berkaroad/squat/utilities/retrying"
)

type EventPublisher interface {
	Publish(ctx context.Context, eventStream domain.EventStream)
	Start()
	Stop()
}

var _ EventPublisher = (*DefaultEventPublisher)(nil)

type DefaultEventPublisher struct {
	BufferSize int
	eb         eventing.EventBus
	es         eventstore.EventStore
	ps         publishedstore.PublishedStore
	pss        publishedstore.PublishedStoreSaver
	serializer serialization.TextSerializer

	initOnce    sync.Once
	initialized bool
	receiverCh  chan *domain.EventStream
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (ep *DefaultEventPublisher) Initialize(
	eventBus eventing.EventBus,
	eventStore eventstore.EventStore,
	publishedStore publishedstore.PublishedStore,
	publishedStoreSaver publishedstore.PublishedStoreSaver,
	serializer serialization.TextSerializer,
) *DefaultEventPublisher {
	ep.initOnce.Do(func() {
		if eventBus == nil {
			panic("field 'eventBus' is null")
		}
		if eventStore == nil {
			panic("field 'eventStore' is null")
		}
		if publishedStore == nil {
			panic("field 'publishedStore' is null")
		}
		bufferSize := ep.BufferSize
		if bufferSize <= 0 {
			bufferSize = 1000
		}
		ep.eb = eventBus
		ep.es = eventStore
		ep.ps = publishedStore
		ep.pss = publishedStoreSaver
		ep.serializer = serializer

		ep.receiverCh = make(chan *domain.EventStream, bufferSize)
		ep.initialized = true
	})
	return ep
}

func (ep *DefaultEventPublisher) Publish(ctx context.Context, eventStream domain.EventStream) {
	if !ep.initialized {
		panic("not initialized")
	}

	if ep.status.Load() != 1 {
		logger := logging.Get(ctx)
		logger.Warn("'DefaultEventPublisher' has stopped")
	}

	ep.receiverCh <- &eventStream
}

func (ep *DefaultEventPublisher) Start() {
	if !ep.initialized {
		panic("not initialized")
	}

	if ep.status.CompareAndSwap(0, 1) {
		go func() {

		loop:
			for {
				select {
				case eventStream := <-ep.receiverCh:
					ep.processEventStream(eventStream)
				case <-time.After(time.Second):
					if ep.status.Load() != 1 {
						break loop
					}
				}
			}
		}()
	}
}

func (ep *DefaultEventPublisher) Stop() {
	if !ep.initialized {
		panic("not initialized")
	}

	ep.status.CompareAndSwap(1, 2)
	time.Sleep(time.Second)
	for len(ep.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	ep.status.CompareAndSwap(2, 0)
}

func (ep *DefaultEventPublisher) processEventStream(eventStream *domain.EventStream) {
	baseLogger := logging.Get(context.Background())
	eventBus := ep.eb
	eventStore := ep.es
	publishedStore := ep.ps
	publishedStoreSaver := ep.pss
	serializer := ep.serializer
	logger := baseLogger.With(
		slog.String("aggregate-id", eventStream.AggregateID),
		slog.String("aggregate-type", eventStream.AggregateTypeName),
		slog.Int("stream-version", eventStream.StreamVersion),
	)
	bgCtx := logging.NewContext(context.Background(), logger)
	publishedVersion := retrying.RetryWithResultForever[int](func() (int, error) {
		if publishedStoreSaver != nil {
			return publishedStoreSaver.GetPublishedVersion(bgCtx, eventStream.AggregateID)
		} else {
			return publishedStore.GetPublishedVersion(bgCtx, eventStream.AggregateID)
		}
	}, time.Second, func(retryCount int, err error) bool {
		if retryCount == 0 {
			logger.Error(fmt.Sprintf("get published version fail: %v", err))
		}
		return true
	})

	var unpublishedEventStreams domain.EventStreamSlice
	if eventStream.StreamVersion == 0 && len(eventStream.Events) == 0 {
		unpublishedEventStreamDatas := retrying.RetryWithResultForever[eventstore.EventStreamDataSlice](func() (eventstore.EventStreamDataSlice, error) {
			return eventStore.QueryEventStreamList(bgCtx, eventStream.AggregateID, publishedVersion+1, math.MaxInt)
		}, time.Second, func(retryCount int, err error) bool {
			if retryCount == 0 {
				logger.Error(fmt.Sprintf("query eventstream fail: %v", err),
					slog.Int("start-version", publishedVersion+1),
					slog.Int("end-version", math.MaxInt),
				)
			}
			return true
		})

		var err error
		unpublishedEventStreams, err = ToEventStreamSlice(serializer, unpublishedEventStreamDatas)
		if err != nil {
			dataBytes, _ := json.Marshal(unpublishedEventStreamDatas)
			logger.Error(fmt.Sprintf("convert EventStreamDataSlice to EventStreamSlice fail: %v", err),
				slog.Any("unpublished-eventstream-data", dataBytes),
			)
			return
		}
	} else if publishedVersion < eventStream.StreamVersion-1 {
		unpublishedEventStreamDatas := retrying.RetryWithResultForever[eventstore.EventStreamDataSlice](func() (eventstore.EventStreamDataSlice, error) {
			return eventStore.QueryEventStreamList(bgCtx, eventStream.AggregateID, publishedVersion+1, eventStream.StreamVersion-1)
		}, time.Second, func(retryCount int, err error) bool {
			if retryCount == 0 {
				logger.Error(fmt.Sprintf("query eventstream fail: %v", err),
					slog.Int("start-version", publishedVersion+1),
					slog.Int("end-version", eventStream.StreamVersion-1),
				)
			}
			return true
		})

		var err error
		unpublishedEventStreams, err = ToEventStreamSlice(serializer, unpublishedEventStreamDatas)
		if err != nil {
			dataBytes, _ := json.Marshal(unpublishedEventStreamDatas)
			logger.Error(fmt.Sprintf("convert EventStreamDataSlice to EventStreamSlice fail: %v", err),
				slog.Any("unpublished-eventstream-data", dataBytes),
			)
			return
		}
	}

	if len(unpublishedEventStreams) > 0 {
		unpublishedLogger := baseLogger.With(
			slog.String("aggregate-id", eventStream.AggregateID),
			slog.String("aggregate-type", eventStream.AggregateTypeName),
		)
		bgCtx2 := logging.NewContext(context.Background(), logger)
		for _, unpublishedEventStream := range unpublishedEventStreams {
			retrying.RetryForever(func() error {
				return eventBus.Publish(unpublishedEventStream)
			}, time.Second, func(retryCount int, err error) bool {
				if retryCount == 0 {
					unpublishedLogger.Error(fmt.Sprintf("publish eventstream fail: %v", err),
						slog.Int("stream-version", unpublishedEventStream.StreamVersion),
					)
				}
				return true
			})
			publishedVersion = unpublishedEventStream.StreamVersion
			logger.Debug("publish unpublished eventstreams",
				slog.Int("unpublished-version", unpublishedEventStream.StreamVersion),
			)
		}

		retrying.RetryForever(func() error {
			if publishedStoreSaver != nil {
				publishedStoreSaver.SavePublished(bgCtx2, publishedstore.PublishedEventStreamRef{
					AggregateID:       eventStream.AggregateID,
					AggregateTypeName: eventStream.AggregateTypeName,
					PublishedVersion:  publishedVersion,
				})
				return nil
			} else {
				return publishedStore.SavePublished(bgCtx2, []publishedstore.PublishedEventStreamRef{{
					AggregateID:       eventStream.AggregateID,
					AggregateTypeName: eventStream.AggregateTypeName,
					PublishedVersion:  publishedVersion,
				}})
			}
		}, time.Second, func(retryCount int, err error) bool {
			if retryCount == 0 {
				unpublishedLogger.Error(fmt.Sprintf("save published eventstream fail: %v", err),
					slog.Int("published-version", publishedVersion),
				)
			}
			return true
		})
	}

	if publishedVersion == eventStream.StreamVersion-1 {
		retrying.RetryForever(func() error {
			return eventBus.Publish(*eventStream)
		}, time.Second, func(retryCount int, err error) bool {
			if retryCount == 0 {
				logger.Error(fmt.Sprintf("publish eventstream fail: %v", err))
			}
			return true
		})
		publishedVersion = eventStream.StreamVersion

		retrying.RetryForever(func() error {
			if publishedStoreSaver != nil {
				publishedStoreSaver.SavePublished(bgCtx, publishedstore.PublishedEventStreamRef{
					AggregateID:       eventStream.AggregateID,
					AggregateTypeName: eventStream.AggregateTypeName,
					PublishedVersion:  publishedVersion,
				})
				return nil
			} else {
				return publishedStore.SavePublished(bgCtx, []publishedstore.PublishedEventStreamRef{{
					AggregateID:       eventStream.AggregateID,
					AggregateTypeName: eventStream.AggregateTypeName,
					PublishedVersion:  publishedVersion,
				}})
			}
		}, time.Second, func(retryCount int, err error) bool {
			if retryCount == 0 {
				logger.Error(fmt.Sprintf("save published eventstream fail: %v", err),
					slog.Int("published-version", publishedVersion),
				)
			}
			return true
		})
	} else {
		logger.Warn("skip published eventstream",
			slog.Int("published-version", publishedVersion),
		)
	}
}
