package eventsourcing

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/publishedstore"
	"github.com/berkaroad/squat/utilities/retrying"
)

type eventPublisher struct {
	eb         eventing.EventBus
	es         eventstore.EventStore
	ps         publishedstore.PublishedStore
	pss        publishedstore.PublishedStoreSaver
	serializer serialization.TextSerializer
}

func (ep *eventPublisher) Publish(eventStream domain.EventStream) {
	baseLogger := logging.Get(context.Background())
	eventBus := ep.eb
	eventStore := ep.es
	publishedStore := ep.ps
	publishedStoreSaver := ep.pss
	serializer := ep.serializer
	logger := baseLogger.With(
		slog.String("aggregate-id", eventStream.AggregateID),
		slog.String("aggregate-type", eventStream.AggregateType),
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
			textData, _ := serialization.SerializeToText(ep.serializer, unpublishedEventStreamDatas)
			logger.Error(fmt.Sprintf("convert EventStreamDataSlice to EventStreamSlice fail: %v", err),
				slog.String("unpublished-eventstream", textData),
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
			textData, _ := serialization.SerializeToText(ep.serializer, unpublishedEventStreamDatas)
			logger.Error(fmt.Sprintf("convert EventStreamDataSlice to EventStreamSlice fail: %v", err),
				slog.String("unpublished-eventstream", textData),
			)
			return
		}
	}

	if len(unpublishedEventStreams) > 0 {
		unpublishedLogger := baseLogger.With(
			slog.String("aggregate-id", eventStream.AggregateID),
			slog.String("aggregate-type", eventStream.AggregateType),
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
					AggregateID:      eventStream.AggregateID,
					AggregateType:    eventStream.AggregateType,
					PublishedVersion: publishedVersion,
				})
				return nil
			} else {
				return publishedStore.SavePublished(bgCtx2, []publishedstore.PublishedEventStreamRef{{
					AggregateID:      eventStream.AggregateID,
					AggregateType:    eventStream.AggregateType,
					PublishedVersion: publishedVersion,
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
			return eventBus.Publish(eventStream)
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
					AggregateID:      eventStream.AggregateID,
					AggregateType:    eventStream.AggregateType,
					PublishedVersion: publishedVersion,
				})
				return nil
			} else {
				return publishedStore.SavePublished(bgCtx, []publishedstore.PublishedEventStreamRef{{
					AggregateID:      eventStream.AggregateID,
					AggregateType:    eventStream.AggregateType,
					PublishedVersion: publishedVersion,
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
