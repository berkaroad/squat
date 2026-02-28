package inmemorycb

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/messaging"
	"github.com/berkaroad/squat/utilities/goroutine"
)

var instance = &InMemoryCommandBus{}

func Default() *InMemoryCommandBus {
	return instance
}

var _ commanding.CommandBus = (*InMemoryCommandBus)(nil)
var _ commanding.CommandProcessor = (*InMemoryCommandBus)(nil)

type InMemoryCommandBus struct {
	BufferSize            int
	NoticeServiceEndpoint string
	dispatcher            commanding.CommandDispatcher
	resultWatcher         messaging.MessageHandleResultWatcher

	initOnce    sync.Once
	initialized bool
	receiverCh  chan *commanding.CommandData
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (cb *InMemoryCommandBus) Initialize(dispatcher commanding.CommandDispatcher, resultWatcher messaging.MessageHandleResultWatcher) *InMemoryCommandBus {
	cb.initOnce.Do(func() {
		if dispatcher == nil {
			dispatcher = &commanding.DefaultCommandDispatcher{}
		}
		if resultWatcher == nil {
			panic("param 'resultWatcher' is null")
		}
		bufferSize := cb.BufferSize
		if bufferSize <= 0 {
			bufferSize = 1000
		}
		cb.dispatcher = dispatcher
		cb.resultWatcher = resultWatcher
		cb.receiverCh = make(chan *commanding.CommandData, bufferSize)
		cb.initialized = true
	})
	return cb
}

func (cb *InMemoryCommandBus) Send(ctx context.Context, cmd commanding.Command) error {
	if !cb.initialized {
		panic("not initialized")
	}

	if cb.status.Load() != 1 {
		logger := logging.Get(ctx)
		logger.Warn("'InMemoryCommandBus' has stopped")
		return commanding.ErrStoppedCommandBus
	}

	defer func() error {
		if recover() != nil {
			logger := logging.Get(ctx)
			logger.Warn("'InMemoryCommandBus' has stopped")
			return commanding.ErrStoppedCommandBus
		}
		return nil
	}()

	eventMetadata := messaging.FromContext(ctx)
	if eventMetadata == nil {
		cb.receiverCh <- &commanding.CommandData{
			Command: cmd,
		}
		return nil
	} else {
		if eventMetadata.Category == commanding.MailCategory {
			panic("'InMemoryCommandBus.Send(context.Context, commanding.Command)' couldn't be invoked in CommandHandler")
		}
		cb.receiverCh <- &commanding.CommandData{
			Command: cmd,
			Extensions: eventMetadata.Extensions.Clone().
				Remove(messaging.ExtensionKeyNoticeServiceEndpoint).
				Set(messaging.ExtensionKeyFromMessageID, eventMetadata.MessageID).
				Set(messaging.ExtensionKeyFromMessageType, eventMetadata.MessageType),
		}
		return nil
	}
}

func (cb *InMemoryCommandBus) Execute(ctx context.Context, cmd commanding.Command) (*commanding.CommandHandleResult, error) {
	if !cb.initialized {
		panic("not initialized")
	}

	if cb.status.Load() != 1 {
		logger := logging.Get(ctx)
		logger.Warn("'InMemoryCommandBus' has stopped")
		return nil, commanding.ErrStoppedCommandBus
	}

	eventMetadata := messaging.FromContext(ctx)
	if eventMetadata == nil {
		fromCommandWatchItem := cb.resultWatcher.Watch(cmd.CommandID(), commanding.CommandHandleResultProvider)
		fromEventWatchItem := cb.resultWatcher.Watch(cmd.CommandID(), eventing.CommandHandleResultProvider)

		defer func() (*commanding.CommandHandleResult, error) {
			if recover() != nil {
				logger := logging.Get(ctx)
				logger.Warn("'InMemoryCommandBus' has stopped")
				return nil, commanding.ErrStoppedCommandBus
			}
			return nil, nil
		}()
		cb.receiverCh <- &commanding.CommandData{
			Command:    cmd,
			Extensions: map[string]string{string(messaging.ExtensionKeyNoticeServiceEndpoint): cb.NoticeServiceEndpoint},
		}
		return commanding.NewCommandHandleResult(fromCommandWatchItem, fromEventWatchItem), nil
	} else {
		if eventMetadata.Category == commanding.MailCategory {
			panic("'InMemoryCommandBus.Execute(context.Context, commanding.Command)' couldn't be invoked in CommandHandler")
		}
		defer func() (*commanding.CommandHandleResult, error) {
			if recover() != nil {
				logger := logging.Get(ctx)
				logger.Warn("'InMemoryCommandBus' has stopped")
				return nil, commanding.ErrStoppedCommandBus
			}
			return nil, nil
		}()
		cb.receiverCh <- &commanding.CommandData{
			Command: cmd,
			Extensions: eventMetadata.Extensions.Clone().
				Remove(messaging.ExtensionKeyNoticeServiceEndpoint).
				Set(messaging.ExtensionKeyFromMessageID, eventMetadata.MessageID).
				Set(messaging.ExtensionKeyFromMessageType, eventMetadata.MessageType),
		}
		return commanding.NewCommandHandleResult(nil, nil), nil
	}
}

func (cb *InMemoryCommandBus) Start() {
	if !cb.initialized {
		panic("not initialized")
	}

	if cb.status.CompareAndSwap(0, 1) {
		go func() {
		loop:
			for {
				select {
				case data, ok := <-cb.receiverCh:
					if !ok {
						break loop
					}
					cb.dispatcher.Dispatch(data)
				case <-time.After(time.Second):
					if cb.status.Load() != 1 {
						break loop
					}
				}
			}
		}()
	}
}

func (cb *InMemoryCommandBus) Stop() {
	if !cb.initialized {
		panic("not initialized")
	}

	cb.status.CompareAndSwap(1, 2)
	time.Sleep(time.Second)
	for len(cb.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	close(cb.receiverCh)
	cb.status.CompareAndSwap(2, 0)
}
