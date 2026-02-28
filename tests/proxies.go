package tests

import (
	"context"
	"fmt"
	"sync"

	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/eventing"
)

var _ commanding.CommandHandlerProxy = (*IdempotentCommandHandlerProxy)(nil)

type IdempotentCommandHandlerProxy struct {
	store sync.Map
}

func (p *IdempotentCommandHandlerProxy) Name() string {
	return "idempotent"
}

func (p *IdempotentCommandHandlerProxy) Wrap(handleFuncName string, previousHandle commanding.CommandHandleFunc) commanding.CommandHandleFunc {
	return func(ctx context.Context, data commanding.CommandData) error {
		key := data.CommandID()
		if _, ok := p.store.Load(key); ok {
			return nil
		}
		err := previousHandle(ctx, data)
		if err != nil {
			return err
		}

		p.store.Store(key, struct{}{})
		return nil
	}
}

var _ eventing.EventHandlerProxy = (*IdempotentEventHandlerProxy)(nil)

type IdempotentEventHandlerProxy struct {
	store sync.Map
}

func (p *IdempotentEventHandlerProxy) Name() string {
	return "idempotent"
}

func (p *IdempotentEventHandlerProxy) Wrap(handleFuncName string, previousHandle eventing.EventHandleFunc) eventing.EventHandleFunc {
	return func(ctx context.Context, data eventing.EventData) error {
		key := fmt.Sprintf("%s:%s", data.EventID(), handleFuncName)
		if _, ok := p.store.Load(key); ok {
			return nil
		}
		err := previousHandle(ctx, data)
		if err != nil {
			return err
		}

		p.store.Store(key, struct{}{})
		return nil
	}
}
