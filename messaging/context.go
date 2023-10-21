package messaging

import (
	"context"
)

type ctxKey int

var messageKey ctxKey

func NewContext(ctx context.Context, message *MessageMetadata) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if message == nil {
		return ctx
	}
	return context.WithValue(ctx, messageKey, message)
}

func FromContext(ctx context.Context) *MessageMetadata {
	if ctx == nil {
		return nil
	}
	message, ok := ctx.Value(messageKey).(*MessageMetadata)
	if ok {
		return message
	}
	return nil
}
