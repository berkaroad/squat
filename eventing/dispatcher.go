package eventing

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/internal/counter"
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
	notifier        messaging.MessageHandleResultNotifier

	initOnce        sync.Once
	initialized     bool
	handlers        map[string][]messaging.MessageHandler[EventData]
	proxies         []messaging.MessageHandlerProxy[EventData]
	proxiedHandlers map[string][]messaging.MessageHandler[EventData]
	locker          sync.Mutex
}

func (ed *DefaultEventDispatcher) Initialize(mailboxProvider messaging.MailboxProvider[EventData], notifier messaging.MessageHandleResultNotifier) *DefaultEventDispatcher {
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
		for eventTypeName, handler := range handlerGroup.EventHandlers() {
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

	counter.Begin()
	defer counter.End()

	resultCh := make(chan messaging.MessageHandleResult, 1)
	mail := messaging.MailsWithResult[EventData]{
		Category: MailCategory,
		Mails:    make([]messaging.Mail[EventData], len(data.Events)),
		ResultCh: resultCh,
	}
	for i, event := range data.Events {
		mail.Mails[i] = CreateEventMail(&EventData{
			DomainEvent:   event,
			AggregateID:   data.AggregateID,
			AggregateType: data.AggregateType,
			StreamVersion: data.StreamVersion,
			CommandID:     data.CommandID,
			CommandType:   data.CommandType,
			Extensions: messaging.Extensions(data.Extensions).
				Set(messaging.ExtensionKeyFromMessageID, data.CommandID).
				Set(messaging.ExtensionKeyFromMessageType, data.CommandType),
		})
	}
	mb := ed.mailboxProvider.GetMailbox(data.AggregateID, data.AggregateType, ed.proxiedHandlers)
	err := mb.SendMail(mail)
	for err != nil {
		time.Sleep(time.Millisecond)
		mb = ed.mailboxProvider.GetMailbox(data.AggregateID, data.AggregateType, ed.proxiedHandlers)
		err = mb.SendMail(mail)
	}

	if ed.notifier != nil {
		// notify event bus
		if noticeServiceEndpoint, ok := messaging.Extensions(data.Extensions).Get(messaging.ExtensionKeyNoticeServiceEndpoint); ok {
			go func() {
				logger := logging.Get(context.Background())
				result := <-resultCh
				ed.notifier.Notify(noticeServiceEndpoint, data.CommandID, CommandHandleResultProvider, result)
				logger.Info(fmt.Sprintf("notify event handle result from %s", CommandHandleResultProvider),
					slog.String("command-id", data.CommandID),
					slog.String("command-type", data.CommandType),
					slog.String("aggregate-id", data.AggregateID),
					slog.String("aggregate-type", data.AggregateType),
					slog.Int("stream-version", data.StreamVersion),
				)
			}()
		}
	}
}
