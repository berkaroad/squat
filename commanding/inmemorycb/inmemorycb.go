package inmemorycb

import (
	"errors"
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
	dispatcher commanding.CommandDispatcher
	subscriber messaging.MessageHandleResultSubscriber[commanding.Command]

	initOnce    sync.Once
	initialized bool
	receiverCh  chan commanding.Command
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (cb *InMemoryCommandBus) Initialize(dispatcher commanding.CommandDispatcher, subscriber messaging.MessageHandleResultSubscriber[commanding.Command]) *InMemoryCommandBus {
	cb.initOnce.Do(func() {
		if dispatcher == nil {
			dispatcher = &commanding.DefaultCommandDispatcher{}
		}
		if subscriber == nil {
			panic("param 'subscriber' is null")
		}
		cb.dispatcher = dispatcher
		cb.subscriber = subscriber
		cb.receiverCh = make(chan commanding.Command, 1)
		cb.initialized = true
	})
	return cb
}

func (cb *InMemoryCommandBus) Send(cmd commanding.Command) (<-chan messaging.MessageHandleResult, error) {
	if !cb.initialized {
		panic("not initialized")
	}

	if cb.status.Load() != 1 {
		return nil, errors.New("command processor has stopped")
	}

	resultCh := make(chan messaging.MessageHandleResult, 2)
	cb.subscriber.Subscribe(cmd.CommandID(), commanding.CommandHandleResultProvider, resultCh)
	cb.subscriber.Subscribe(cmd.CommandID(), eventing.CommandHandleResultProvider, resultCh)
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
