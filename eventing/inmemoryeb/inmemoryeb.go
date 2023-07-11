package inmemoryeb

import (
	"container/list"
	"sync"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/eventing"
)

var _ eventing.EventBus = (*InMemoryEventBus)(nil)

type InMemoryEventBus struct {
	eventHandlerMapper sync.Map
	eventStreams       *list.List
	eventStreamsLocker sync.Mutex
}

func New() *InMemoryEventBus {
	eb := &InMemoryEventBus{
		eventStreams: list.New().Init(),
	}

	return eb
}

func (eb *InMemoryEventBus) RegisterEventHandler(eventType string, handler eventing.EventHandler) {
	receiverObj, _ := eb.eventHandlerMapper.LoadOrStore(eventType, &eventHandlerReceiver{
		Handlers: make([]eventing.EventHandler, 0),
		Receiver: make(chan eventHandlerReceiveData),
	})
	receiver := receiverObj.(*eventHandlerReceiver)
	receiver.Handlers = append(receiver.Handlers, handler)
	eb.eventHandlerMapper.Store(eventType, receiver)
}

func (eb *InMemoryEventBus) Publish(es domain.EventStream) {
	for _, event := range es.Events {
		if receiverObj, ok := eb.eventHandlerMapper.Load(event.TypeName()); ok {
			receiver := receiverObj.(*eventHandlerReceiver)
			receiver.Receiver <- eventHandlerReceiveData{
				AggregateID:   es.AggregateID,
				StreamVersion: es.StreamVersion,
				Event:         event,
			}
		}

	}

	defer eb.eventStreamsLocker.Unlock()
	eb.eventStreamsLocker.Lock()

	//eb.eventStreams.PushBack(es)

}

type eventHandlerReceiver struct {
	Handlers []eventing.EventHandler
	Receiver chan eventHandlerReceiveData
}

func (r *eventHandlerReceiver) Process() {
	go func() {
		for {
			data := <-r.Receiver
			for _, handler := range r.Handlers {
				handler(data.AggregateID, data.StreamVersion, data.Event)
			}
		}
	}()
}

type eventHandlerReceiveData struct {
	AggregateID   string
	StreamVersion int
	Event         domain.DomainEvent
}
