package eventing

import (
	"context"

	"github.com/berkaroad/squat/domain"
)

type EventBus interface {
	Publish(es domain.EventStream) error
}

type EventProcessor interface {
	Process(ctx context.Context) <-chan struct{}
}

type EventHandleFunc func(ctx context.Context, data EventData) error

type EventData struct {
	EventSourceID       string
	EventSourceTypeName string
	StreamVersion       int
	Event               domain.DomainEvent
}

type EventHandler struct {
	Handle   EventHandleFunc
	FuncName string
}

type EventHandlerGroup interface {
	Handlers() map[string]EventHandler
}

type EventHandlerProxy interface {
	Name() string
	Wrap(handleFuncName string, previousHandle EventHandleFunc) EventHandleFunc
}
