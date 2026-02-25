package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/errors"
	"github.com/berkaroad/squat/internal/counter"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/utilities/retrying"
)

var (
	errMailboxDestoryed error = errors.NewWithCode(errors.SysErrCodePrefix+"MailboxDestoryed", "mailbox has been destoryed")
)

type Mailbox[TMessageBody any] interface {
	Name() string
	SendMail(data MailsWithResult[TMessageBody]) error
}

type MailboxProvider[TMessageBody any] interface {
	GetMailbox(aggregateID string, aggregateTypeName string, handlers map[string][]MessageHandler[TMessageBody]) Mailbox[TMessageBody]
	Stats() MailBoxStatistic
}

type MailBoxStatistic struct {
	CreatedCount int64
	RemovedCount int64
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
	Err        error
	Extensions Extensions
}

var _ MailboxProvider[any] = (*DefaultMailboxProvider[any])(nil)

type DefaultMailboxProvider[TMessage any] struct {
	MailboxCapacity    int
	GetMailboxName     func(aggregateID string, aggregateTypeName string) string
	AutoReleaseTimeout time.Duration

	mailboxes sync.Map

	// stats
	createdCounter atomic.Int64
	removedCounter atomic.Int64
}

func (p *DefaultMailboxProvider[TMessage]) GetMailbox(aggregateID string, aggregateTypeName string, handlers map[string][]MessageHandler[TMessage]) Mailbox[TMessage] {
	mailboxName := aggregateTypeName
	if p.GetMailboxName != nil {
		mailboxName = p.GetMailboxName(aggregateID, aggregateTypeName)
	}

	actual, loaded := p.mailboxes.LoadOrStore(mailboxName, &defaultMailbox[TMessage]{
		name:               mailboxName,
		autoReleaseTimeout: p.AutoReleaseTimeout,
		handlers:           handlers,
	})
	mb := actual.(*defaultMailbox[TMessage])
	if !loaded {
		mb.initialize(p.MailboxCapacity, func(s string) {
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
	statusDisactived int32 = iota
	statusActived
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
	for mb.status.Load() == statusDisactived {
		time.Sleep(time.Millisecond)
	}
	if mb.status.Load() == statusDestoryed {
		mb.logger.Warn(errMailboxDestoryed.Error())
		return errMailboxDestoryed
	}
	mb.receiverCh <- data
	return nil
}

func (mb *defaultMailbox[TMessage]) initialize(mailboxCapacity int, removeSelf func(string)) {
	if mailboxCapacity <= 0 {
		mailboxCapacity = 1000
	}
	mb.receiverCh = make(chan MailsWithResult[TMessage], mailboxCapacity)
	mb.logger = logging.Get(context.Background()).With(
		slog.String("mailbox", mb.name),
	)
	mb.status.Swap(statusActived)
	mb.logger.Debug("mailbox initialized")

	go func() {
		autoReleaseTimeout := mb.autoReleaseTimeout
		if autoReleaseTimeout <= 0 {
			autoReleaseTimeout = time.Second * 30
		}
	loop:
		for {
			select {
			case data, ok := <-mb.receiverCh:
				if !ok {
					break loop
				}
				mb.processMails(data)
			case <-time.After(autoReleaseTimeout):
				if mb.status.CompareAndSwap(statusActived, statusDisactived) {
					for i := 0; i < len(mb.receiverCh); i++ {
						mb.processMails(<-mb.receiverCh)
					}
					close(mb.receiverCh)
					break loop
				}
			}
		}
		removeSelf(mb.name)
		mb.status.CompareAndSwap(statusDisactived, statusDestoryed)
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
		metadata := mail.Metadata()
		logger := mb.logger.With(
			slog.String("message-id", metadata.MessageID),
			slog.String("message-type", metadata.MessageType),
			slog.String("aggregate-id", metadata.AggregateID),
			slog.String("aggregate-type", metadata.AggregateType),
		)
		handleCtx := NewContext(logging.NewContext(context.Background(), logger), &metadata)

		var handleErr error
		var noHandler bool
		if handlers, ok := allHandlers[metadata.MessageType]; ok {
			if len(handlers) == 0 {
				noHandler = true
			}
			for _, handler := range handlers {
				hasPanic := false
				err := retrying.Retry(func() (err error) {
					defer func() {
						if r := recover(); r != nil {
							hasPanic = true
							if panicErr, ok := r.(error); ok {
								err = fmt.Errorf("panic: %w", panicErr)
							} else {
								err = fmt.Errorf("panic: %v", r)
							}
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
				} else if errors.Is(err, domain.ErrAggregateNoChange) {
					logger.Debug(fmt.Sprintf("handle %s success, but %v", data.Category, err),
						slog.String(fmt.Sprintf("%s-handler", data.Category), handler.FuncName),
					)
				} else {
					handleErr = errors.Join(err)
					if hasPanic {
						logger.Error(fmt.Sprintf("handle %s %v", data.Category, err),
							slog.String(fmt.Sprintf("%s-handler", data.Category), handler.FuncName),
						)
					}
				}
			}
		} else {
			noHandler = true
		}
		handleResult.Extensions = handleResult.Extensions.Merge(metadata.Extensions)
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
		handleResult.Err = handlerErrs
	}
}
