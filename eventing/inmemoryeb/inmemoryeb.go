package inmemoryeb

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/utilities/goroutine"
)

var instance = &InMemoryEventBus{}

func Default() *InMemoryEventBus {
	return instance
}

var _ eventing.EventBus = (*InMemoryEventBus)(nil)
var _ eventing.EventProcessor = (*InMemoryEventBus)(nil)

type InMemoryEventBus struct {
	BufferSize int
	dispatcher eventing.EventDispatcher

	initOnce    sync.Once
	initialized bool
	receiverCh  chan *domain.EventStream
	status      atomic.Int32 // 0: stop, 1: running, 2: stopping
}

func (eb *InMemoryEventBus) Initialize(dispatcher eventing.EventDispatcher) *InMemoryEventBus {
	eb.initOnce.Do(func() {
		if dispatcher == nil {
			dispatcher = &eventing.DefaultEventDispatcher{}
		}
		bufferSize := eb.BufferSize
		if bufferSize <= 0 {
			bufferSize = 1000
		}
		eb.dispatcher = dispatcher
		eb.receiverCh = make(chan *domain.EventStream, bufferSize)
		eb.initialized = true
	})
	return eb
}

func (eb *InMemoryEventBus) Publish(es domain.EventStream) error {
	if !eb.initialized {
		panic("not initialized")
	}

	eb.receiverCh <- &es
	return nil
}

func (eb *InMemoryEventBus) Start() {
	if !eb.initialized {
		panic("not initialized")
	}

	if eb.status.CompareAndSwap(0, 1) {
		go func() {
		loop:
			for {
				select {
				case data := <-eb.receiverCh:
					eb.dispatcher.Dispatch(data)
				case <-time.After(time.Second):
					if eb.status.Load() != 1 {
						break loop
					}
				}
			}
		}()
	}
}

func (eb *InMemoryEventBus) Stop() {
	if !eb.initialized {
		panic("not initialized")
	}

	eb.status.CompareAndSwap(1, 2)
	time.Sleep(time.Second)
	for len(eb.receiverCh) > 0 {
		time.Sleep(time.Second)
	}
	<-goroutine.Wait()
	eb.status.CompareAndSwap(2, 0)
}
