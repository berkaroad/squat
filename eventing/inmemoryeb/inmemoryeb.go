package inmemoryeb

import (
	"context"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/utilities/goroutine"
)

var _ eventing.EventBus = (*InMemoryEventBus)(nil)
var _ eventing.EventProcessor = (*InMemoryEventBus)(nil)
var _ eventing.EventDispatcher = (*InMemoryEventBus)(nil)

type InMemoryEventBus struct {
	eventing.EventDispatcher

	receiverCh chan *eventing.EventData
}

func (eb *InMemoryEventBus) Publish(es domain.EventStream) error {
	if eb.receiverCh == nil {
		panic("field 'receiver' is null")
	}

	for _, event := range es.Events {
		eb.receiverCh <- &eventing.EventData{
			EventSourceID:       es.AggregateID,
			EventSourceTypeName: es.AggregateTypeName,
			StreamVersion:       es.StreamVersion,
			Event:               event,
		}
	}
	return nil
}

func (eb *InMemoryEventBus) Process(ctx context.Context) <-chan struct{} {
	if eb.receiverCh == nil {
		eb.receiverCh = make(chan *eventing.EventData, 1)
	}

	dispatcher := eb.EventDispatcher
	if dispatcher == nil {
		dispatcher = &eventing.DefaultEventDispatcher{}
	}

	doneCh := make(chan struct{})
	go func() {
	loop:
		for {
			select {
			case data := <-eb.receiverCh:
				dispatcher.Dispatch(data)
			case <-ctx.Done():
				time.Sleep(time.Second * 3)
				if len(eb.receiverCh) > 0 {
					continue
				}
				<-goroutine.Wait()
				close(eb.receiverCh)
				break loop
			}
		}
		close(doneCh)
	}()
	return doneCh
}
