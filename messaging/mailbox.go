package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/errors"
	"github.com/berkaroad/squat/internal/goroutine"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/utilities/retrying"
)

var (
	errMailboxDestoryed error = errors.New("S:MailboxDestoryed", "mailbox has been destoryed")
)

type Mailbox[TMessageBody any] interface {
	Name() string
	SendMail(data MailWithResult[TMessageBody]) error
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

type MailWithResult[TMessage any] struct {
	Mail     Mail[TMessage]
	ResultCh chan MessageHandleResult
}

type MessageHandleResult struct {
	MessageCategory string
	Err             error
	Code            string
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

	actual, loaded := p.mailboxes.LoadOrStore(mailboxName, &defaultMailbox[TMessage]{
		name:               mailboxName,
		autoReleaseTimeout: p.AutoReleaseTimeout,
		receiverCh:         make(chan MailWithResult[TMessage], p.MailboxCapacity),
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
			receiverCh:         make(chan MailWithResult[TMessage], p.MailboxCapacity),
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
	receiverCh         chan MailWithResult[TMessage]
	handlers           map[string][]MessageHandler[TMessage]
	logger             *slog.Logger

	status atomic.Int32
}

func (mb *defaultMailbox[TMessage]) Name() string {
	return mb.name
}

func (mb *defaultMailbox[TMessage]) SendMail(data MailWithResult[TMessage]) error {
	mb.status.CompareAndSwap(statusActived, statusActivating)
	mb.status.CompareAndSwap(statusDisactived, statusActivating)
	if mb.status.Load() == statusDestoryed {
		mb.logger.Error(errMailboxDestoryed.Error())
		return errMailboxDestoryed
	}
	mb.receiverCh <- data
	mb.status.CompareAndSwap(statusActivating, statusActived)
	if mb.logger.Enabled(context.Background(), slog.LevelDebug) {
		dataBytes, _ := json.Marshal(data.Mail.Unwrap())
		mb.logger.Debug(fmt.Sprintf("receive %s", data.Mail.Metadata().Category),
			slog.Any("message-id", data.Mail.Metadata().ID),
			slog.Any("message-body", dataBytes),
		)
	}
	return nil
}

func (mb *defaultMailbox[TMessage]) initialize(removeSelf func(string)) {
	mb.logger.Debug("mailbox initialized")
	goroutine.Go(context.Background(), func(ctx context.Context) {
		autoReleaseTimeout := mb.autoReleaseTimeout
		if autoReleaseTimeout <= 0 {
			autoReleaseTimeout = time.Second * 30
		}

		allHandlers := mb.handlers
	loop:
		for {
			select {
			case data := <-mb.receiverCh:
				messageMetadata := data.Mail.Metadata()
				messageID := messageMetadata.ID
				messageTypeName := data.Mail.TypeName()
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
							err = handler.Handle(handleCtx, data.Mail.Unwrap())
							return
						}, time.Second*3, func(retryCount int, err error) bool {
							if _, ok := err.(net.Error); ok || errors.Is(err, ErrNetworkError) {
								return true
							}
							return false
						}, -1)
						if err == nil {
							logger.Debug(fmt.Sprintf("handle %s success", messageMetadata.Category),
								slog.String(fmt.Sprintf("%s-handler", messageMetadata.Category), handler.FuncName),
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
					logger.Warn(fmt.Sprintf("no %s handler", messageMetadata.Category))
				}
				handleResult := MessageHandleResult{
					MessageCategory: messageMetadata.Category,
				}
				if handleErr != nil {
					handleResult.Code = errors.GetErrorCode(handleErr)
					handleResult.Err = fmt.Errorf("handle %s '%s(id=%s)' fail : %w", messageMetadata.Category, data.Mail.TypeName(), messageMetadata.ID, handleErr)
				}
				data.ResultCh <- handleResult
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
	})
}
