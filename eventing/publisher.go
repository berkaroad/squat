package eventing

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
	eb         EventBus
	es         eventstore.EventStore
	ps         publishedstore.PublishedStore
	serializer serialization.Serializer

	initOnce    sync.Once
	initialized bool
	receiverCh  chan domain.EventStream
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (ep *DefaultEventPublisher) Initialize(
	eventBus EventBus,
	eventStore eventstore.EventStore,
	publishedStore publishedstore.BatchSavingPublishedStore,
	serializer serialization.Serializer,
) *DefaultEventPublisher {
	ep.initOnce.Do(func() {
		ep.eb = eventBus
		ep.es = eventStore
		ep.ps = publishedStore
		ep.serializer = serializer

		ep.receiverCh = make(chan domain.EventStream, 1)
		ep.initialized = true
	})
	return ep
}

func (ep *DefaultEventPublisher) Publish(ctx context.Context, eventStream domain.EventStream) {
	ep.receiverCh <- eventStream
}

func (ep *DefaultEventPublisher) Start() {
	if ep.status.CompareAndSwap(0, 1) {
		go func() {
			baseLogger := logging.Get(context.Background())
			eventBus := ep.eb
			eventStore := ep.es
			publishedStore := ep.ps
			serializer := ep.serializer

			for {
				select {
				case eventStream := <-ep.receiverCh:
					if eventStream.AggregateID == "" {
						baseLogger.Error("aggregate-id of from eventstream is empty")
						continue
					}
					logger := baseLogger.With(
						slog.String("aggregate-id", eventStream.AggregateID),
						slog.String("aggregate-type", eventStream.AggregateTypeName),
						slog.Int("stream-version", eventStream.StreamVersion),
					)
					bgCtx := logging.NewContext(context.Background(), logger)
					publishedVersion := retrying.RetryWithResultForever[int](func() (int, error) {
						return publishedStore.GetPublishedVersion(bgCtx, eventStream.AggregateID)
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
						unpublishedEventStreams, err = eventstore.ToEventStreamSlice(serializer, unpublishedEventStreamDatas)
						if err != nil {
							dataBytes, _ := json.Marshal(unpublishedEventStreamDatas)
							logger.Error(fmt.Sprintf("convert EventStreamDataSlice to EventStreamSlice fail: %v", err),
								slog.Any("unpublished-eventstream-data", dataBytes),
							)
							continue
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
						unpublishedEventStreams, err = eventstore.ToEventStreamSlice(serializer, unpublishedEventStreamDatas)
						if err != nil {
							dataBytes, _ := json.Marshal(unpublishedEventStreamDatas)
							logger.Error(fmt.Sprintf("convert EventStreamDataSlice to EventStreamSlice fail: %v", err),
								slog.Any("unpublished-eventstream-data", dataBytes),
							)
							continue
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
						}

						retrying.RetryForever(func() error {
							return publishedStore.Save(bgCtx2, publishedstore.PublishedEventStreamRef{
								AggregateID:       eventStream.AggregateID,
								AggregateTypeName: eventStream.AggregateTypeName,
								PublishedVersion:  publishedVersion,
							})
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
							return eventBus.Publish(eventStream)
						}, time.Second, func(retryCount int, err error) bool {
							if retryCount == 0 {
								logger.Error(fmt.Sprintf("publish eventstream fail: %v", err))
							}
							return true
						})
						publishedVersion = eventStream.StreamVersion

						retrying.RetryForever(func() error {
							return publishedStore.Save(bgCtx, publishedstore.PublishedEventStreamRef{
								AggregateID:       eventStream.AggregateID,
								AggregateTypeName: eventStream.AggregateTypeName,
								PublishedVersion:  publishedVersion,
							})
						}, time.Second, func(retryCount int, err error) bool {
							if retryCount == 0 {
								logger.Error(fmt.Sprintf("save published eventstream fail: %v", err),
									slog.Int("published-version", publishedVersion),
								)
							}
							return true
						})
					}
				case <-time.After(time.Second):
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
	time.Sleep(time.Second * 3)
	for len(ep.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	ep.status.CompareAndSwap(2, 0)
}