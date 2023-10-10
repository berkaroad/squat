package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/errors"
	"github.com/berkaroad/squat/internal/counter"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/utilities/retrying"
)

var (
	errMailboxDestoryed error = errors.NewWithCode("S:MailboxDestoryed", "mailbox has been destoryed")
)

type Mailbox[TMessageBody any] interface {
	Name() string
	SendMail(data MailsWithResult[TMessageBody]) error
}

type MailboxProvider[TMessageBody any] interface {
	GetMailbox(aggregateID string, aggregateTypeName string, handlers map[string][]MessageHandler[TMessageBody]) Mailbox[TMessageBody]
	Stats() MailBoxStatistic
}

type Mail[TMessage any] interface {
	serialization.Serializable
	Metadata() MessageMetadata
	Unwrap() TMessage
}

type MailsWithResult[TMessage any] struct {
	Category string
	Mails    []Mail[TMessage]
	ResultCh chan MessageHandleResult
}

type MessageHandleResult struct {
	Err  error
	Code string
}

type MailBoxStatistic struct {
	CreatedCount int64
	RemovedCount int64
}

var _ MailboxProvider[any] = (*DefaultMailboxProvider[any])(nil)

type DefaultMailboxProvider[TMessage any] struct {
	MailboxCapacity    int
	GetMailboxName     func(aggregateID string, aggregateTypeName string) string
	AutoReleaseTimeout time.Duration

	mailboxes      sync.Map
	recreateLocker sync.Mutex

	// stats
	createdCounter atomic.Int64
	removedCounter atomic.Int64
}

func (p *DefaultMailboxProvider[TMessage]) GetMailbox(aggregateID string, aggregateTypeName string, handlers map[string][]MessageHandler[TMessage]) Mailbox[TMessage] {
	mailboxName := aggregateTypeName
	if p.GetMailboxName != nil {
		mailboxName = p.GetMailboxName(aggregateID, aggregateTypeName)
	}
	mailboxCapacity := p.MailboxCapacity
	if mailboxCapacity <= 0 {
		mailboxCapacity = 1000
	}

	actual, loaded := p.mailboxes.LoadOrStore(mailboxName, &defaultMailbox[TMessage]{
		name:               mailboxName,
		autoReleaseTimeout: p.AutoReleaseTimeout,
		receiverCh:         make(chan MailsWithResult[TMessage], mailboxCapacity),
		handlers:           handlers,
		logger: slog.Default().With(
			slog.String("mailbox", mailboxName),
		),
	})
	mb := actual.(*defaultMailbox[TMessage])
	needInitialize := !loaded
	if loaded && mb.status.Load() == statusDestoryed {
		defer p.recreateLocker.Unlock()
		p.recreateLocker.Lock()
		p.mailboxes.Delete(mailboxName)
		p.removedCounter.Add(1)
		actual, loaded = p.mailboxes.LoadOrStore(mailboxName, &defaultMailbox[TMessage]{
			name:               mailboxName,
			autoReleaseTimeout: p.AutoReleaseTimeout,
			receiverCh:         make(chan MailsWithResult[TMessage], mailboxCapacity),
			handlers:           handlers,
			logger: slog.Default().With(
				slog.String("mailbox", mailboxName),
			),
		})
		mb = actual.(*defaultMailbox[TMessage])
		needInitialize = !loaded
	}

	if needInitialize {
		mb.initialize(func(s string) {
			p.mailboxes.Delete(mailboxName)
			p.removedCounter.Add(1)
		})
		p.createdCounter.Add(1)
	}
	return mb
}

func (p *DefaultMailboxProvider[TMessage]) Stats() MailBoxStatistic {
	return MailBoxStatistic{
		CreatedCount: p.createdCounter.Load(),
		RemovedCount: p.removedCounter.Load(),
	}
}

const (
	statusActived int32 = iota
	statusActivating
	statusDisactived
	statusDestoryed
)

var _ Mailbox[any] = (*defaultMailbox[any])(nil)

type defaultMailbox[TMessage any] struct {
	name               string
	autoReleaseTimeout time.Duration
	receiverCh         chan MailsWithResult[TMessage]
	handlers           map[string][]MessageHandler[TMessage]
	logger             *slog.Logger

	status atomic.Int32
}

func (mb *defaultMailbox[TMessage]) Name() string {
	return mb.name
}

func (mb *defaultMailbox[TMessage]) SendMail(data MailsWithResult[TMessage]) error {
	mb.status.CompareAndSwap(statusActived, statusActivating)
	mb.status.CompareAndSwap(statusDisactived, statusActivating)
	if mb.status.Load() == statusDestoryed {
		mb.logger.Error(errMailboxDestoryed.Error())
		return errMailboxDestoryed
	}
	mb.receiverCh <- data
	mb.status.CompareAndSwap(statusActivating, statusActived)
	return nil
}

func (mb *defaultMailbox[TMessage]) initialize(removeSelf func(string)) {
	mb.logger.Debug("mailbox initialized")
	go func() {
		autoReleaseTimeout := mb.autoReleaseTimeout
		if autoReleaseTimeout <= 0 {
			autoReleaseTimeout = time.Second * 10
		}
	loop:
		for {
			select {
			case data := <-mb.receiverCh:
				mb.processMails(data)
			case <-time.After(autoReleaseTimeout):
				mb.status.CompareAndSwap(statusActived, statusDisactived)
				mb.status.CompareAndSwap(statusDisactived, statusDestoryed)
				if mb.status.Load() == statusDestoryed {
					time.Sleep(time.Second)
					if len(mb.receiverCh) > 0 {
						continue
					}
					close(mb.receiverCh)
					break loop
				}
			}
		}
		removeSelf(mb.name)
		mb.logger.Debug("mailbox removed")
	}()
}

func (mb *defaultMailbox[TMessage]) processMails(data MailsWithResult[TMessage]) {
	counter.Begin()
	defer counter.End()

	handleResult := MessageHandleResult{}
	defer func() {
		data.ResultCh <- handleResult
	}()

	allHandlers := mb.handlers
	var handlerErrs error
	for _, mail := range data.Mails {
		messageMetadata := mail.Metadata()
		messageID := messageMetadata.ID
		messageTypeName := mail.TypeName()
		aggregateID := messageMetadata.AggregateID
		aggregateTypeName := messageMetadata.AggregateTypeName
		logger := mb.logger.With(
			slog.String("message-id", messageID),
			slog.String("message-type", messageTypeName),
			slog.String("aggregate-id", aggregateID),
			slog.String("aggregate-type", aggregateTypeName),
		)
		handleCtx := NewContext(logging.NewContext(context.Background(), logger), messageMetadata)

		var handleErr error
		var noHandler bool
		if handlers, ok := allHandlers[messageTypeName]; ok {
			if len(handlers) == 0 {
				noHandler = true
			}
			for _, handler := range handlers {
				err := retrying.Retry(func() (err error) {
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("%v", r)
						}
					}()
					err = handler.Handle(handleCtx, mail.Unwrap())
					return
				}, time.Second, func(retryCount int, err error) bool {
					if _, ok := err.(net.Error); ok {
						return true
					}
					return false
				}, -1)
				if err == nil {
					logger.Debug(fmt.Sprintf("handle %s success", data.Category),
						slog.String(fmt.Sprintf("%s-handler", data.Category), handler.FuncName),
					)
				} else {
					handleErr = errors.Join(err)
				}
			}
		} else {
			noHandler = true
		}
		if noHandler {
			handleErr = ErrMissingMessageHandler
			logger.Warn(fmt.Sprintf("no %s handler", data.Category))
		}
		if handleErr != nil {
			if handlerErrs == nil {
				handlerErrs = handleErr
			} else {
				handlerErrs = errors.Join(handleErr)
			}
		}
	}

	if handlerErrs != nil {
		handleResult.Code = errors.GetErrorCode(handlerErrs)
		handleResult.Err = handlerErrs
	}
}
