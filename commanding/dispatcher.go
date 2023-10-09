package commanding

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/berkaroad/squat/internal/goroutine"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/messaging"
)

const CommandHandleResultProvider string = "commanding.dispatcher"

type CommandDispatcher interface {
	Subscribe(commandTypeName string, handler CommandHandler)
	SubscribeMulti(handlerGroup CommandHandlerGroup)
	AddProxy(proxies ...CommandHandlerProxy)
	Dispatch(data Command)
}

var _ CommandDispatcher = (*DefaultCommandDispatcher)(nil)

type DefaultCommandDispatcher struct {
	mailboxProvider messaging.MailboxProvider[Command]
	notifier        messaging.MessageHandleResultNotifier[Command]

	initOnce        sync.Once
	initialized     bool
	handlers        map[string][]messaging.MessageHandler[Command]
	proxies         []messaging.MessageHandlerProxy[Command]
	proxiedHandlers map[string][]messaging.MessageHandler[Command]
	locker          sync.Mutex
}

func (cd *DefaultCommandDispatcher) Initialize(mailboxProvider messaging.MailboxProvider[Command], notifier messaging.MessageHandleResultNotifier[Command]) *DefaultCommandDispatcher {
	cd.initOnce.Do(func() {
		if mailboxProvider == nil {
			mailboxProvider = &messaging.DefaultMailboxProvider[Command]{}
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
		cd.handlers = map[string][]messaging.MessageHandler[Command]{commandTypeName: {messaging.MessageHandler[Command](handler)}}
	} else if _, ok := cd.handlers[commandTypeName]; ok {
		return
	} else {
		cd.handlers[commandTypeName] = []messaging.MessageHandler[Command]{messaging.MessageHandler[Command](handler)}
	}
	proxiedHande := handler.Handle
	for _, proxy := range cd.proxies {
		proxiedHande = proxy.Wrap(handler.FuncName, proxiedHande)
	}
	proxiedHandler := messaging.MessageHandler[Command]{
		FuncName: handler.FuncName,
		Handle:   proxiedHande,
	}

	if cd.proxiedHandlers == nil {
		cd.proxiedHandlers = map[string][]messaging.MessageHandler[Command]{commandTypeName: {proxiedHandler}}
	} else if _, ok := cd.proxiedHandlers[commandTypeName]; ok {
		return
	} else {
		cd.proxiedHandlers[commandTypeName] = []messaging.MessageHandler[Command]{proxiedHandler}
	}
}

func (cd *DefaultCommandDispatcher) SubscribeMulti(handlerGroup CommandHandlerGroup) {
	if !cd.initialized {
		panic("not initialized")
	}

	if handlerGroup != nil {
		for commandTypeName, handler := range handlerGroup.Handlers() {
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
		cd.proxies = make([]messaging.MessageHandlerProxy[Command], 0)
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
			proxiedHandler := messaging.MessageHandler[Command]{
				FuncName: handler.FuncName,
				Handle:   proxiedHande,
			}
			handlers[i] = proxiedHandler
		}
	}
}

func (cd *DefaultCommandDispatcher) Dispatch(data Command) {
	if !cd.initialized {
		panic("not initialized")
	}

	resultCh := make(chan messaging.MessageHandleResult, 1)
	msg := messaging.MailsWithResult[Command]{
		Category: MailCategory,
		Mails:    []messaging.Mail[Command]{CreateCommandMail(data)},
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
		goroutine.Go(context.Background(), func(ctx context.Context) {
			logger := logging.Get(ctx)
			result := <-resultCh
			logger.Info(fmt.Sprintf("notify command handle result from %s", CommandHandleResultProvider),
				slog.String("command-id", data.CommandID()),
			)
			cd.notifier.Notify(data.CommandID(), CommandHandleResultProvider, result)
		})
	}
}
