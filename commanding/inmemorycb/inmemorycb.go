package inmemorycb

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/eventing"
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
	watcher               messaging.MessageHandleResultWatcher

	initOnce    sync.Once
	initialized bool
	mqCh        chan *commanding.CommandData // simulate a real message queue.
	status      atomic.Int32                 // 0: stop, 1: running, 2: stopping

	// statistics
	fetchedCount int64
}

func (cb *InMemoryCommandBus) Initialize(dispatcher commanding.CommandDispatcher, watcher messaging.MessageHandleResultWatcher) *InMemoryCommandBus {
	cb.initOnce.Do(func() {
		if dispatcher == nil {
			dispatcher = &commanding.DefaultCommandDispatcher{}
		}
		if watcher == nil {
			panic("param 'watcher' is null")
		}
		bufferSize := cb.BufferSize
		if bufferSize <= 0 {
			bufferSize = 1000
		}
		cb.dispatcher = dispatcher
		cb.watcher = watcher
		cb.mqCh = make(chan *commanding.CommandData, bufferSize)
		cb.initialized = true
	})
	return cb
}

func (cb *InMemoryCommandBus) Send(ctx context.Context, cmd commanding.Command) error {
	if !cb.initialized {
		panic("not initialized")
	}

	eventMetadata := messaging.FromContext(ctx)
	if eventMetadata == nil {
		cb.mqCh <- &commanding.CommandData{
			Command: cmd,
		}
		return nil
	} else {
		if eventMetadata.Category == commanding.MailCategory {
			panic("'InMemoryCommandBus.Send(context.Context, commanding.Command)' couldn't be invoked in CommandHandler")
		}
		cb.mqCh <- &commanding.CommandData{
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

	eventMetadata := messaging.FromContext(ctx)
	if eventMetadata == nil {
		fromCommandWatchItem := cb.watcher.Watch(cmd.CommandID(), commanding.CommandHandleResultProvider)
		fromEventWatchItem := cb.watcher.Watch(cmd.CommandID(), eventing.CommandHandleResultProvider)

		cb.mqCh <- &commanding.CommandData{
			Command:    cmd,
			Extensions: map[string]string{string(messaging.ExtensionKeyNoticeServiceEndpoint): cb.NoticeServiceEndpoint},
		}
		return commanding.NewCommandHandleResult(fromCommandWatchItem, fromEventWatchItem), nil
	} else {
		if eventMetadata.Category == commanding.MailCategory {
			panic("'InMemoryCommandBus.Execute(context.Context, commanding.Command)' couldn't be invoked in CommandHandler")
		}

		cb.mqCh <- &commanding.CommandData{
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
		bufferSize := cb.BufferSize
		if bufferSize <= 0 {
			bufferSize = 1000
		}
		go func() {
		loop:
			for {
				select {
				case data, ok := <-cb.mqCh:
					if !ok {
						break loop
					}
					atomic.AddInt64(&cb.fetchedCount, 1)
					cb.dispatcher.Dispatch(data, nil)
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
	for len(cb.mqCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	cb.status.CompareAndSwap(2, 0)
}

func (cb *InMemoryCommandBus) Stats() commanding.CommandProcessorStatistic {
	return commanding.CommandProcessorStatistic{
		FetchedCount: atomic.LoadInt64(&cb.fetchedCount),
	}
}
