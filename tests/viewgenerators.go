package tests

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/logging"
)

var _ eventing.EventHandlerGroup = (*AccountViewGenerator)(nil)

type AccountViewGenerator struct {
	counter int64
}

func (vg *AccountViewGenerator) EventHandlers() map[string]eventing.EventHandler {
	return map[string]eventing.EventHandler{
		Event_AccountCreated:        {Handle: vg.handleAccountCreated},
		Event_AccountBalanceChanged: {Handle: vg.handleAccountBalanceChanged},
		Event_AccountRemoved:        {Handle: vg.handleAccountRemoved},
	}
}

func (vg *AccountViewGenerator) handleAccountCreated(ctx context.Context, data eventing.EventData) error {
	currentCounter := atomic.AddInt64(&vg.counter, 1)
	logger := logging.Get(ctx)
	logger.Info("generate device view by 'AccountCreated'",
		slog.Int64("counter", currentCounter),
	)
	return nil
}

func (vg *AccountViewGenerator) handleAccountBalanceChanged(ctx context.Context, data eventing.EventData) error {
	currentCounter := atomic.AddInt64(&vg.counter, 1)
	logger := logging.Get(ctx)
	logger.Info("generate device view by 'AccountBalanceChanged'",
		slog.Int64("counter", currentCounter),
	)
	return nil
}

func (vg *AccountViewGenerator) handleAccountRemoved(ctx context.Context, data eventing.EventData) error {
	currentCounter := atomic.AddInt64(&vg.counter, 1)
	logger := logging.Get(ctx)
	logger.Info("generate device view by 'AccountRemoved'",
		slog.Int64("counter", currentCounter),
	)
	return nil
}
