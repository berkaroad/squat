package inmemorycb

import (
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
var _ commanding.CommandProcess = (*InMemoryCommandBus)(nil)

type InMemoryCommandBus struct {
	BufferSize    int
	dispatcher    commanding.CommandDispatcher
	resultWatcher messaging.MessageHandleResultWatcher[commanding.Command]

	initOnce    sync.Once
	initialized bool
	receiverCh  chan commanding.Command
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (cb *InMemoryCommandBus) Initialize(dispatcher commanding.CommandDispatcher, resultWatcher messaging.MessageHandleResultWatcher[commanding.Command]) *InMemoryCommandBus {
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
		cb.receiverCh = make(chan commanding.Command, bufferSize)
		cb.initialized = true
	})
	return cb
}

func (cb *InMemoryCommandBus) Send(cmd commanding.Command) (<-chan commanding.CommandHandleResult, error) {
	if !cb.initialized {
		panic("not initialized")
	}

	resultCh := make(chan commanding.CommandHandleResult, 1)
	commandHandleResultWatchItem := cb.resultWatcher.Watch(cmd.CommandID(), commanding.CommandHandleResultProvider)
	eventHandleResultWatchItem := cb.resultWatcher.Watch(cmd.CommandID(), eventing.CommandHandleResultProvider)
	go func() {
		result := commanding.CommandHandleResult{
			FromCommandHandle: <-commandHandleResultWatchItem.Result(),
		}
		if result.FromCommandHandle.Err == nil {
			result.FromEventHandleCh = eventHandleResultWatchItem.Result()
		} else {
			eventHandleResultWatchItem.Unwatch()
		}
		resultCh <- result
	}()
	cb.receiverCh <- cmd
	return resultCh, nil
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
				case data := <-cb.receiverCh:
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
	time.Sleep(time.Second * 3)
	for len(cb.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	cb.status.CompareAndSwap(2, 0)
}
