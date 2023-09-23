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

	receiver chan *eventing.EventData
}

func (eb *InMemoryEventBus) Publish(es domain.EventStream) error {
	if eb.receiver == nil {
		panic("field 'receiver' is null")
	}

	for _, event := range es.Events {
		eb.receiver <- &eventing.EventData{
			EventSourceID:       es.AggregateID,
			EventSourceTypeName: es.AggregateTypeName,
			StreamVersion:       es.StreamVersion,
			Event:               event,
		}
	}
	return nil
}

func (eb *InMemoryEventBus) Process(ctx context.Context) {
	if eb.receiver == nil {
		eb.receiver = make(chan *eventing.EventData)
	}

	dispatcher := eb.EventDispatcher
	if dispatcher == nil {
		dispatcher = &eventing.DefaultEventDispatcher{}
	}

	go func() {
	loop:
		for {
			select {
			case data := <-eb.receiver:
				dispatcher.Dispatch(data)
			case <-ctx.Done():
				time.Sleep(time.Second * 3)
				if len(eb.receiver) > 0 {
					continue
				}
				<-goroutine.Wait()
				close(eb.receiver)
				break loop
			}
		}
	}()
}
