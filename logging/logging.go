package logging

import (
	"context"
	"log/slog"
)

type ctxKey int

var loggerKey ctxKey

func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return nil
	}
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if ok {
		return logger
	}
	return nil
}

func Get(ctx context.Context) *slog.Logger {
	logger := FromContext(ctx)
	if logger == nil {
		logger = slog.Default()
	}
	return logger
}
