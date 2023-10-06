package messaging

import (
	"context"
)

type MessageHandleFunc[TMessage any] func(ctx context.Context, data TMessage) error

type MessageHandler[TMessage any] struct {
	Handle   MessageHandleFunc[TMessage]
	FuncName string
}

type MessageHandlerProxy[TMessage any] interface {
	Name() string
	Wrap(handleFuncName string, previousHandle MessageHandleFunc[TMessage]) MessageHandleFunc[TMessage]
}

type MessageMetadata struct {
	ID                string
	AggregateID       string
	AggregateTypeName string
	Category          string
}
