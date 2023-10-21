package commanding

import (
	"context"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/messaging"
)

func NewCommandHandleResult(fromCommandWatchItem, fromEventWatchItem messaging.MessageHandleResultWatchItem) *CommandHandleResult {
	return &CommandHandleResult{
		fromCommandWatchItem: fromCommandWatchItem,
		fromEventWatchItem:   fromEventWatchItem,
	}
}

type CommandHandleResult struct {
	fromCommandWatchItem messaging.MessageHandleResultWatchItem
	fromEventWatchItem   messaging.MessageHandleResultWatchItem
}

func (chr *CommandHandleResult) FromCommandHandle(ctx context.Context, timeout time.Duration) messaging.MessageHandleResult {
	if timeout == 0 {
		timeout = time.Second * 30
	}
	if chr.fromCommandWatchItem == nil {
		return messaging.MessageHandleResult{}
	}
	select {
	case fromCommand := <-chr.fromCommandWatchItem.Result():
		return fromCommand
	case <-time.After(timeout):
		return messaging.MessageHandleResult{
			Err: ErrWaitFromCommandHandlerTimeout,
		}
	case <-ctx.Done():
		return messaging.MessageHandleResult{
			Err: ErrWaitCommandResultCancelled,
		}
	}
}

func (chr *CommandHandleResult) FromEventHandle(ctx context.Context, timeout time.Duration) messaging.MessageHandleResult {
	if timeout == 0 {
		timeout = time.Second * 30
	}
	if chr.fromCommandWatchItem == nil {
		return messaging.MessageHandleResult{}
	}
	waitStartTime := time.Now()
	select {
	case fromCommand := <-chr.fromCommandWatchItem.Result():
		if fromCommand.Err == nil {
			waitFromCommandDuration := time.Since(waitStartTime)
			if val, _ := fromCommand.Extensions.Get(messaging.ExtensionKeyAggregateChanged); val != "true" {
				fromCommand.Extensions = fromCommand.Extensions.CustomExtensions()
				fromCommand.Err = domain.ErrAggregateNoChange
				return fromCommand
			} else if chr.fromEventWatchItem == nil {
				fromCommand.Extensions = fromCommand.Extensions.CustomExtensions()
				return fromCommand
			}
			select {
			case fromEvent := <-chr.fromEventWatchItem.Result():
				fromEvent.Extensions = fromEvent.Extensions.CustomExtensions()
				return fromEvent
			case <-time.After(timeout - waitFromCommandDuration):
				return messaging.MessageHandleResult{
					Err: ErrWaitFromEventHandlerTimeout,
				}
			case <-ctx.Done():
				return messaging.MessageHandleResult{
					Err: ErrWaitCommandResultCancelled,
				}
			}
		} else {
			chr.fromEventWatchItem.Unwatch()
			fromCommand.Extensions = fromCommand.Extensions.CustomExtensions()
			return fromCommand
		}
	case <-time.After(timeout):
		return messaging.MessageHandleResult{
			Err: ErrWaitFromCommandHandlerTimeout,
		}
	case <-ctx.Done():
		return messaging.MessageHandleResult{
			Err: ErrWaitCommandResultCancelled,
		}
	}
}
