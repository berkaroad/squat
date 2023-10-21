package commanding

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/berkaroad/squat/internal/counter"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/messaging"
)

const CommandHandleResultProvider string = "commanding.dispatcher"

type CommandDispatcher interface {
	Subscribe(commandTypeName string, handler CommandHandler)
	SubscribeMulti(handlerGroup CommandHandlerGroup)
	AddProxy(proxies ...CommandHandlerProxy)
	Dispatch(data *CommandData)
}

var _ CommandDispatcher = (*DefaultCommandDispatcher)(nil)

type DefaultCommandDispatcher struct {
	mailboxProvider messaging.MailboxProvider[CommandData]
	notifier        messaging.MessageHandleResultNotifier

	initOnce        sync.Once
	initialized     bool
	handlers        map[string][]messaging.MessageHandler[CommandData]
	proxies         []messaging.MessageHandlerProxy[CommandData]
	proxiedHandlers map[string][]messaging.MessageHandler[CommandData]
	locker          sync.Mutex
}

func (cd *DefaultCommandDispatcher) Initialize(mailboxProvider messaging.MailboxProvider[CommandData], notifier messaging.MessageHandleResultNotifier) *DefaultCommandDispatcher {
	cd.initOnce.Do(func() {
		if mailboxProvider == nil {
			mailboxProvider = &messaging.DefaultMailboxProvider[CommandData]{}
		}
		if notifier == nil {
			panic("param 'notifier' is null")
		}
		cd.mailboxProvider = mailboxProvider
		cd.notifier = notifier
		cd.initialized = true
	})
	return cd
}

func (cd *DefaultCommandDispatcher) Subscribe(commandTypeName string, handler CommandHandler) {
	if !cd.initialized {
		panic("not initialized")
	}

	defer cd.locker.Unlock()
	cd.locker.Lock()

	if commandTypeName == "" {
		panic("param 'commandTypeName' is empty")
	}
	if handler.Handle == nil {
		panic("param 'handler.Handle' is null")
	}
	if handler.FuncName == "" {
		handler.FuncName = runtime.FuncForPC(reflect.ValueOf(handler.Handle).Pointer()).Name()
	}

	if cd.handlers == nil {
		cd.handlers = map[string][]messaging.MessageHandler[CommandData]{commandTypeName: {messaging.MessageHandler[CommandData](handler)}}
	} else if _, ok := cd.handlers[commandTypeName]; ok {
		return
	} else {
		cd.handlers[commandTypeName] = []messaging.MessageHandler[CommandData]{messaging.MessageHandler[CommandData](handler)}
	}
	proxiedHande := handler.Handle
	for _, proxy := range cd.proxies {
		proxiedHande = proxy.Wrap(handler.FuncName, proxiedHande)
	}
	proxiedHandler := messaging.MessageHandler[CommandData]{
		FuncName: handler.FuncName,
		Handle:   proxiedHande,
	}

	if cd.proxiedHandlers == nil {
		cd.proxiedHandlers = map[string][]messaging.MessageHandler[CommandData]{commandTypeName: {proxiedHandler}}
	} else if _, ok := cd.proxiedHandlers[commandTypeName]; ok {
		return
	} else {
		cd.proxiedHandlers[commandTypeName] = []messaging.MessageHandler[CommandData]{proxiedHandler}
	}
}

func (cd *DefaultCommandDispatcher) SubscribeMulti(handlerGroup CommandHandlerGroup) {
	if !cd.initialized {
		panic("not initialized")
	}

	if handlerGroup != nil {
		for commandTypeName, handler := range handlerGroup.CommandHandlers() {
			cd.Subscribe(commandTypeName, handler)
		}
	}
}

func (cd *DefaultCommandDispatcher) AddProxy(proxies ...CommandHandlerProxy) {
	if !cd.initialized {
		panic("not initialized")
	}

	defer cd.locker.Unlock()
	cd.locker.Lock()

	if cd.proxies == nil {
		cd.proxies = make([]messaging.MessageHandlerProxy[CommandData], 0)
	}
	for _, proxy := range proxies {
		if proxy == nil {
			continue
		}
		cd.proxies = append(cd.proxies, proxy)
	}

	for _, handlers := range cd.handlers {
		for i, handler := range handlers {
			proxiedHande := handler.Handle
			for _, proxy := range cd.proxies {
				proxiedHande = proxy.Wrap(handler.FuncName, proxiedHande)
			}
			proxiedHandler := messaging.MessageHandler[CommandData]{
				FuncName: handler.FuncName,
				Handle:   proxiedHande,
			}
			handlers[i] = proxiedHandler
		}
	}
}

func (cd *DefaultCommandDispatcher) Dispatch(data *CommandData) {
	if !cd.initialized {
		panic("not initialized")
	}

	counter.Begin()
	defer counter.End()

	resultCh := make(chan messaging.MessageHandleResult, 1)
	msg := messaging.MailsWithResult[CommandData]{
		Category: MailCategory,
		Mails:    []messaging.Mail[CommandData]{CreateCommandMail(data)},
		ResultCh: resultCh,
	}
	mb := cd.mailboxProvider.GetMailbox(data.AggregateID(), data.AggregateTypeName(), cd.proxiedHandlers)
	err := mb.SendMail(msg)
	for err != nil {
		time.Sleep(time.Millisecond)
		mb = cd.mailboxProvider.GetMailbox(data.AggregateID(), data.AggregateTypeName(), cd.proxiedHandlers)
		err = mb.SendMail(msg)
	}

	if cd.notifier != nil {
		// notify event bus
		if noticeServiceEndpoint, ok := messaging.Extensions(data.Extensions).Get(messaging.ExtensionKeyNoticeServiceEndpoint); ok {
			go func() {
				logger := logging.Get(context.Background())
				result := <-resultCh
				cd.notifier.Notify(noticeServiceEndpoint, data.CommandID(), CommandHandleResultProvider, result)
				logger.Info(fmt.Sprintf("notify command handle result from %s", CommandHandleResultProvider),
					slog.String("command-id", data.CommandID()),
					slog.String("command-type", data.TypeName()),
					slog.String("aggregate-id", data.AggregateID()),
					slog.String("aggregate-type", data.AggregateTypeName()),
				)
			}()
		}
	}
}
