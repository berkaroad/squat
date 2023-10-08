package eventing

import (
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
	AggregateID       string
	AggregateTypeName string
	StreamVersion     int
	Event             domain.DomainEvent
}

type EventHandler messaging.MessageHandler[EventData]

type EventHandlerGroup interface {
	Handlers() map[string]EventHandler
}

type EventHandlerProxy messaging.MessageHandlerProxy[EventData]

func CreateEventMail(data *EventData) messaging.Mail[EventData] {
	return &eventMail{
		eventData:   data,
		DomainEvent: data.Event,
	}
}

var _ messaging.Mail[EventData] = (*eventMail)(nil)

type eventMail struct {
	domain.DomainEvent
	eventData *EventData
}

func (m *eventMail) Metadata() messaging.MessageMetadata {
	return messaging.MessageMetadata{
		ID:                m.DomainEvent.EventID(),
		AggregateID:       m.eventData.AggregateID,
		AggregateTypeName: m.eventData.AggregateTypeName,
		Category:          MailCategory,
	}
}

func (m *eventMail) Unwrap() EventData {
	return *m.eventData
}
