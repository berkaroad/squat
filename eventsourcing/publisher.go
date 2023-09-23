package eventsourcing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/internal/goroutine"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/publishedstore"
	"github.com/berkaroad/squat/utilities/retrying"
)

type eventPublisher struct {
	EventBus       eventing.EventBus
	EventStore     eventstore.EventStore
	PublishedStore publishedstore.PublishedStore
	Serializer     serialization.Serializer

	receiverCh chan domain.EventStream
	initOnce   sync.Once
}

func (p *eventPublisher) Publish(ctx context.Context, eventStream domain.EventStream) {
	p.receiverCh <- eventStream
}

func (p *eventPublisher) Initialize(ctx context.Context) *eventPublisher {
	p.initOnce.Do(func() {
		p.receiverCh = make(chan domain.EventStream, 1)

		goroutine.Go(ctx, func(ctx context.Context) {
			baseLogger := logging.Get(ctx)
			eventBus := p.EventBus
			eventStore := p.EventStore
			publishedStore := p.PublishedStore
			serializer := p.Serializer
		loop:
			for {
				select {
				case eventStream := <-p.receiverCh:
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
				case <-ctx.Done():
					time.Sleep(time.Second * 3)
					if len(p.receiverCh) > 0 {
						continue
					}
					close(p.receiverCh)
					break loop
				}
			}
		})
	})
	return p
}
