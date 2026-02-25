package eventing

import (
	"context"
	"strings"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/messaging"
)

const MailCategory string = "event"

type EventBus interface {
	Publish(es domain.EventStream) error
}

type EventProcessor interface {
	Start()
	Stop()
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

func (data *EventData) SetCustomExtension(ctx context.Context, key string, val string) {
	metadata := messaging.FromContext(ctx)
	if metadata != nil && metadata.Category == MailCategory && !strings.HasPrefix(key, messaging.SysExtensionKeyPrefix) {
		metadata.Extensions = metadata.Extensions.Clone().
			Set(messaging.ExtensionKey(key), val)
		data.Extensions = metadata.Extensions
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
