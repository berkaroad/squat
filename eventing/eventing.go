package eventing

import (
	"context"
	"log/slog"
	"strings"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/messaging"
)

const MailCategory string = "event"

type EventBus interface {
	Publish(es domain.EventStream) error
}

type EventProcessorStatistic struct {
	FetchedCount int64
}

type EventProcessor interface {
	Start()
	Stop()
	Stats() EventProcessorStatistic
}

type EventHandleFunc = messaging.MessageHandleFunc[EventData]

type EventData struct {
	domain.DomainEvent
	AggregateID   string
	AggregateType string
	StreamVersion int
	CommandID     string
	CommandType   string
	Extensions    map[string]string
}

func (data *EventData) SetCustomExtension(ctx context.Context, key messaging.ExtensionKey, val string) {
	metadata := messaging.FromContext(ctx)
	if metadata != nil && metadata.Category == MailCategory && !strings.HasPrefix(string(key), messaging.SysExtensionKeyPrefix) {
		metadata.Extensions = metadata.Extensions.Clone().
			Set(key, val)
		data.Extensions = metadata.Extensions
		slog.Debug("EventData.SetCustomExtension",
			slog.String("key", string(key)),
			slog.String("val", val))
	}
}

type EventHandler messaging.MessageHandler[EventData]

type EventHandlerGroup interface {
	EventHandlers() map[string]EventHandler
}

type EventHandlerProxy messaging.MessageHandlerProxy[EventData]

func CreateEventMail(data *EventData) messaging.Mail[EventData] {
	return &eventMail{
		DomainEvent: data.DomainEvent,
		eventData:   data,
	}
}

var _ messaging.Mail[EventData] = (*eventMail)(nil)

type eventMail struct {
	domain.DomainEvent
	eventData *EventData
}

func (m *eventMail) Metadata() messaging.MessageMetadata {
	return messaging.MessageMetadata{
		MessageID:     m.DomainEvent.EventID(),
		MessageType:   m.DomainEvent.TypeName(),
		AggregateID:   m.eventData.AggregateID,
		AggregateType: m.eventData.AggregateType,
		Category:      MailCategory,
		Extensions:    m.eventData.Extensions,
	}
}

func (m *eventMail) Unwrap() EventData {
	return *m.eventData
}
