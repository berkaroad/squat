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

type Message struct {
	MessageMetadata
	Body string `json:"body"`
}

type MessageMetadata struct {
	MessageID     string     `json:"message_id"`
	MessageType   string     `json:"message_type"`
	AggregateID   string     `json:"aggregate_id"`
	AggregateType string     `json:"aggregate_type"`
	Category      string     `json:"category"`
	Extensions    Extensions `json:"extensions"`
}
