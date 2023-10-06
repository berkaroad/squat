package messaging

import (
	"context"
)

type ctxKey int

var messageKey ctxKey

func NewContext(ctx context.Context, message MessageMetadata) context.Context {
	return context.WithValue(ctx, messageKey, message)
}

func FromContext(ctx context.Context) MessageMetadata {
	message, ok := ctx.Value(messageKey).(MessageMetadata)
	if ok {
		return message
	}
	return MessageMetadata{}
}
