package eventing

import "github.com/berkaroad/squat/domain"

type EventBus interface {
	Publish(es domain.EventStream)
}

type EventHandler func(aggregateID string, streamVersion int, event domain.DomainEvent)

type EventStreamHandler func(es domain.EventStream)
