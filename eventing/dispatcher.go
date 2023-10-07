package eventing

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/errors"
	"github.com/berkaroad/squat/internal/goroutine"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/messaging"
)

const CommandHandleResultProvider string = "eventing.dispatcher"

type EventDispatcher interface {
	Subscribe(eventTypeName string, handler EventHandler)
	SubscribeMulti(handlerGroup EventHandlerGroup)
	AddProxy(proxies ...EventHandlerProxy)
	Dispatch(data *domain.EventStream)
}

var _ EventDispatcher = (*DefaultEventDispatcher)(nil)

type DefaultEventDispatcher struct {
	mailboxProvider messaging.MailboxProvider[EventData]
	notifier        messaging.MessageHandleResultNotifier[EventData]

	initOnce        sync.Once
	initialized     bool
	handlers        map[string][]messaging.MessageHandler[EventData]
	proxies         []messaging.MessageHandlerProxy[EventData]
	proxiedHandlers map[string][]messaging.MessageHandler[EventData]
	locker          sync.Mutex
}

func (ed *DefaultEventDispatcher) Initialize(mailboxProvider messaging.MailboxProvider[EventData], notifier messaging.MessageHandleResultNotifier[commanding.Command]) *DefaultEventDispatcher {
	ed.initOnce.Do(func() {
		if mailboxProvider == nil {
			mailboxProvider = &messaging.DefaultMailboxProvider[EventData]{}
		}
		if notifier == nil {
			panic("param 'notifier' is null")
		}
		ed.mailboxProvider = mailboxProvider
		ed.notifier = notifier
		ed.initialized = true
	})
	return ed
}

func (ed *DefaultEventDispatcher) Subscribe(eventTypeName string, handler EventHandler) {
	if !ed.initialized {
		panic("not initialized")
	}

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
		ed.handlers = map[string][]messaging.MessageHandler[EventData]{eventTypeName: {messaging.MessageHandler[EventData](handler)}}
	} else if existsHandlers, ok := ed.handlers[eventTypeName]; ok {
		existsHandlers = append(existsHandlers, messaging.MessageHandler[EventData](handler))
		ed.handlers[eventTypeName] = existsHandlers
	} else {
		ed.handlers[eventTypeName] = []messaging.MessageHandler[EventData]{messaging.MessageHandler[EventData](handler)}
	}

	proxiedHande := handler.Handle
	for _, proxy := range ed.proxies {
		proxiedHande = proxy.Wrap(handler.FuncName, proxiedHande)
	}
	proxiedHandler := messaging.MessageHandler[EventData]{
		FuncName: handler.FuncName,
		Handle:   proxiedHande,
	}

	if ed.proxiedHandlers == nil {
		ed.proxiedHandlers = map[string][]messaging.MessageHandler[EventData]{eventTypeName: {proxiedHandler}}
	} else if existsProxiedHandlers, ok := ed.proxiedHandlers[eventTypeName]; ok {
		existsProxiedHandlers = append(existsProxiedHandlers, proxiedHandler)
		ed.proxiedHandlers[eventTypeName] = existsProxiedHandlers
	} else {
		ed.proxiedHandlers[eventTypeName] = []messaging.MessageHandler[EventData]{proxiedHandler}
	}
}

func (ed *DefaultEventDispatcher) SubscribeMulti(handlerGroup EventHandlerGroup) {
	if !ed.initialized {
		panic("not initialized")
	}

	if handlerGroup != nil {
		for eventTypeName, handler := range handlerGroup.Handlers() {
			ed.Subscribe(eventTypeName, EventHandler(handler))
		}
	}
}

func (ed *DefaultEventDispatcher) AddProxy(proxies ...EventHandlerProxy) {
	if !ed.initialized {
		panic("not initialized")
	}

	defer ed.locker.Unlock()
	ed.locker.Lock()

	if ed.proxies == nil {
		ed.proxies = make([]messaging.MessageHandlerProxy[EventData], 0)
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
				proxiedHande = proxy.Wrap(handler.FuncName, proxiedHande)
			}
			proxiedHandler := messaging.MessageHandler[EventData]{
				FuncName: handler.FuncName,
				Handle:   proxiedHande,
			}
			handlers[i] = proxiedHandler
		}
	}
}

func (ed *DefaultEventDispatcher) Dispatch(data *domain.EventStream) {
	if !ed.initialized {
		panic("not initialized")
	}

	if data == nil || len(data.Events) == 0 {
		return
	}

	resultCh := make(chan messaging.MessageHandleResult, len(data.Events))
	for _, event := range data.Events {
		msg := messaging.MailWithResult[EventData]{
			Mail: CreateEventMail(&EventData{
				AggregateID:       data.AggregateID,
				AggregateTypeName: data.AggregateTypeName,
				StreamVersion:     data.StreamVersion,
				Event:             event,
			}),
			ResultCh: resultCh,
		}
		mb := ed.mailboxProvider.GetMailbox(data.AggregateID, data.AggregateTypeName, ed.proxiedHandlers)
		err := mb.SendMail(msg)
		for err != nil {
			time.Sleep(time.Millisecond)
			mb = ed.mailboxProvider.GetMailbox(data.AggregateID, data.AggregateTypeName, ed.proxiedHandlers)
			err = mb.SendMail(msg)
		}
	}

	if ed.notifier != nil {
		// notify event bus
		goroutine.Go(context.Background(), func(ctx context.Context) {
			logger := logging.Get(ctx)
			var aggrErr error
			for i := 0; i < cap(resultCh); i++ {
				err := (<-resultCh).Err
				if err != nil {
					if aggrErr == nil {
						aggrErr = err
					} else {
						aggrErr = errors.Join(err)
					}
				}
			}
			logger.Info(fmt.Sprintf("notify event handle result from %s", CommandHandleResultProvider),
				slog.String("command-id", data.CommandID),
			)
			ed.notifier.Notify(data.CommandID, CommandHandleResultProvider, messaging.MessageHandleResult{Err: aggrErr})
		})
	}
}
