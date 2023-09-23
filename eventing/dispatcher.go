package eventing

import (
	"reflect"
	"runtime"
	"sync"
	"time"
)

type EventDispatcher interface {
	Subscribe(eventTypeName string, handler EventHandler)
	SubscribeMulti(handlerGroup EventHandlerGroup)
	AddProxy(proxies ...EventHandlerProxy)
	Dispatch(data *EventData)
}

var _ EventDispatcher = (*DefaultEventDispatcher)(nil)

type DefaultEventDispatcher struct {
	MailboxProvider MailboxProvider
	Callback        func(data EventData, resultCh chan EventHandleResult)

	handlers        map[string][]EventHandler
	proxies         []EventHandlerProxy
	proxiedHandlers map[string][]EventHandler
	locker          sync.Mutex
}

func (ed *DefaultEventDispatcher) Subscribe(eventTypeName string, handler EventHandler) {
	defer ed.locker.Unlock()
	ed.locker.Lock()

	if eventTypeName == "" {
		panic("param 'eventTypeName' is empty")
	}
	if handler.Handle == nil {
		panic("param 'handler.Handle' is null")
	}
	if handler.FuncName == "" {
		handler.FuncName = runtime.FuncForPC(reflect.ValueOf(handler.Handle).Pointer()).Name()
	}

	if ed.handlers == nil {
		ed.handlers = map[string][]EventHandler{eventTypeName: {handler}}
	} else if existsHandlers, ok := ed.handlers[eventTypeName]; ok {
		existsHandlers = append(existsHandlers, handler)
		ed.handlers[eventTypeName] = existsHandlers
	} else {
		ed.handlers[eventTypeName] = []EventHandler{handler}
	}

	proxiedHande := handler.Handle
	for _, proxy := range ed.proxies {
		proxiedHande = proxy.Wrap(proxiedHande)
	}
	proxiedHandler := EventHandler{
		FuncName: handler.FuncName,
		Handle:   proxiedHande,
	}

	if ed.proxiedHandlers == nil {
		ed.proxiedHandlers = map[string][]EventHandler{eventTypeName: {proxiedHandler}}
	} else if existsProxiedHandlers, ok := ed.proxiedHandlers[eventTypeName]; ok {
		existsProxiedHandlers = append(existsProxiedHandlers, proxiedHandler)
		ed.proxiedHandlers[eventTypeName] = existsProxiedHandlers
	} else {
		ed.proxiedHandlers[eventTypeName] = []EventHandler{proxiedHandler}
	}
}

func (ed *DefaultEventDispatcher) SubscribeMulti(handlerGroup EventHandlerGroup) {
	if handlerGroup != nil {
		for eventTypeName, handler := range handlerGroup.Handlers() {
			ed.Subscribe(eventTypeName, handler)
		}
	}
}

func (ed *DefaultEventDispatcher) AddProxy(proxies ...EventHandlerProxy) {
	defer ed.locker.Unlock()
	ed.locker.Lock()

	if ed.proxies == nil {
		ed.proxies = make([]EventHandlerProxy, 0)
	}
	for _, proxy := range proxies {
		if proxy == nil {
			continue
		}
		ed.proxies = append(ed.proxies, proxy)
	}

	for _, handlers := range ed.handlers {
		for i, handler := range handlers {
			proxiedHande := handler.Handle
			for _, proxy := range ed.proxies {
				proxiedHande = proxy.Wrap(proxiedHande)
			}
			proxiedHandler := EventHandler{
				FuncName: handler.FuncName,
				Handle:   proxiedHande,
			}
			handlers[i] = proxiedHandler
		}
	}
}

func (ed *DefaultEventDispatcher) Dispatch(data *EventData) {
	mbp := ed.MailboxProvider
	if mbp == nil {
		mbp = &DefaultMailboxProvider{}
	}

	mb := mbp.GetMailbox(data.EventSourceID, data.EventSourceTypeName, ed.proxiedHandlers)
	resultCh := make(chan EventHandleResult, 1)
	err := mb.Receive(data, resultCh)
	for err != nil {
		time.Sleep(time.Millisecond)
		mb = mbp.GetMailbox(data.EventSourceID, data.EventSourceTypeName, ed.proxiedHandlers)
		err = mb.Receive(data, resultCh)
	}

	callback := ed.Callback
	if callback != nil {
		callback(*data, resultCh)
	}
}
