package eventing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/berkaroad/squat/internal/goroutine"
	"github.com/berkaroad/squat/logging"
	"github.com/berkaroad/squat/utilities/retrying"
)

var (
	ErrMailboxDestoryed error = errors.New("mailbox has been destoryed")
)

type Mailbox interface {
	Name() string
	Receive(data *EventData) (chan EventHandleResult, error)
}

type EventHandleResult struct {
	Err error
}

type MailboxProvider interface {
	GetMailbox(eventSourceID string, eventSourceTypeName string, handlers map[string][]EventHandler) Mailbox
	Stats() MailBoxStatistic
}

type MailBoxStatistic struct {
	CreatedCount int64
	RemovedCount int64
}

var _ MailboxProvider = (*DefaultMailboxProvider)(nil)

type DefaultMailboxProvider struct {
	MailboxCapacity          int
	GetMailboxName           func(eventSourceID string, eventSourceTypeName string) string
	GetAutoReleaseTimeout    func(eventSourceTypeName string) time.Duration
	HandlerFailRetryInterval time.Duration
	HandlerFailMaxRetryCount int

	mailboxes      sync.Map
	recreateLocker sync.Mutex

	// stats
	createdCounter atomic.Int64
	removedCounter atomic.Int64
}

func (p *DefaultMailboxProvider) GetMailbox(eventSourceID string, eventSourceTypeName string, handlers map[string][]EventHandler) Mailbox {
	mailboxName := eventSourceTypeName
	if p.GetMailboxName != nil {
		mailboxName = p.GetMailboxName(eventSourceID, eventSourceTypeName)
	}

	var autoReleaseTimeout time.Duration
	if p.GetAutoReleaseTimeout != nil {
		autoReleaseTimeout = p.GetAutoReleaseTimeout(eventSourceTypeName)
	}

	actual, loaded := p.mailboxes.LoadOrStore(mailboxName, &defaultMailbox{
		name:                     mailboxName,
		handlerFailRetryInterval: p.HandlerFailRetryInterval,
		handlerFailMaxRetryCount: p.HandlerFailMaxRetryCount,
		autoReleaseTimeout:       autoReleaseTimeout,
		receiver:                 make(chan eventDataWithResult, p.MailboxCapacity),
		handlers:                 handlers,
		logger: slog.Default().With(
			slog.String("mailbox", mailboxName),
		),
	})
	mb := actual.(*defaultMailbox)
	needInitialize := !loaded
	if loaded && mb.status.Load() == statusDestoryed {
		defer p.recreateLocker.Unlock()
		p.recreateLocker.Lock()
		p.mailboxes.Delete(mailboxName)
		p.removedCounter.Add(1)
		actual, loaded = p.mailboxes.LoadOrStore(mailboxName, &defaultMailbox{
			name:                     mailboxName,
			handlerFailRetryInterval: p.HandlerFailRetryInterval,
			handlerFailMaxRetryCount: p.HandlerFailMaxRetryCount,
			autoReleaseTimeout:       autoReleaseTimeout,
			receiver:                 make(chan eventDataWithResult, p.MailboxCapacity),
			handlers:                 handlers,
			logger: slog.Default().With(
				slog.String("mailbox", mailboxName),
			),
		})
		mb = actual.(*defaultMailbox)
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

func (p *DefaultMailboxProvider) Stats() MailBoxStatistic {
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

var _ Mailbox = (*defaultMailbox)(nil)

type defaultMailbox struct {
	name                     string
	handlerFailRetryInterval time.Duration
	handlerFailMaxRetryCount int
	autoReleaseTimeout       time.Duration
	receiver                 chan eventDataWithResult
	handlers                 map[string][]EventHandler
	logger                   *slog.Logger

	status atomic.Int32
}

func (mb *defaultMailbox) Name() string {
	return mb.name
}

func (mb *defaultMailbox) Receive(data *EventData) (chan EventHandleResult, error) {
	mb.status.CompareAndSwap(statusActived, statusActivating)
	mb.status.CompareAndSwap(statusDisactived, statusActivating)
	if mb.status.Load() == statusDestoryed {
		err := ErrMailboxDestoryed
		mb.logger.Error(err.Error())
		return nil, err
	}
	result := make(chan EventHandleResult, 1)
	mb.receiver <- eventDataWithResult{
		EventData: data,
		Result:    result,
	}
	mb.status.CompareAndSwap(statusActivating, statusActived)
	if mb.logger.Enabled(context.Background(), slog.LevelDebug) {
		dataBytes, _ := json.Marshal(data)
		mb.logger.Debug("receive event handler data", slog.Any("eventhandler-data", dataBytes))
	}
	return result, nil
}

func (mb *defaultMailbox) initialize(removeSelf func(string)) {
	mb.logger.Info("mailbox initialized")
	goroutine.Go(context.Background(), func(ctx context.Context) {
		handlerFailRetryInterval := mb.handlerFailRetryInterval
		if handlerFailRetryInterval <= 0 {
			handlerFailRetryInterval = time.Second * 3
		}
		handlerFailMaxRetryCount := mb.handlerFailMaxRetryCount

		autoReleaseTimeout := mb.autoReleaseTimeout
		if autoReleaseTimeout <= 0 {
			autoReleaseTimeout = time.Second * 30
		}

		allHandlers := mb.handlers
	loop:
		for {
			select {
			case data := <-mb.receiver:
				eventID := data.EventData.Event.EventID()
				eventTypeName := data.EventData.Event.TypeName()
				eventSourceID := data.EventData.EventSourceID
				eventSourceTypeName := data.EventData.EventSourceTypeName
				logger := mb.logger.With(
					slog.String("event-id", eventID),
					slog.String("event-type", eventTypeName),
					slog.String("eventsource-id", eventSourceID),
					slog.String("eventsource-type", eventSourceTypeName),
				)
				bgCtx := logging.NewContext(context.Background(), logger)

				var handleErr error
				if handlers, ok := allHandlers[eventTypeName]; ok {
					for _, handler := range handlers {
						err := retrying.Retry(func() (err error) {
							defer func() {
								if r := recover(); r != nil {
									err = fmt.Errorf("%v", r)
								}
							}()
							err = handler.Handle(bgCtx, *data.EventData)
							return
						}, handlerFailRetryInterval, func(retryCount int, err error) bool {
							if retryCount == 0 {
								logger.Error(fmt.Sprintf("handle event fail: %v", err), slog.String("event-handler", handler.FuncName))
							}
							return true
						}, handlerFailMaxRetryCount)
						if err == nil {
							logger.Debug("handle event success", slog.String("event-handler", handler.FuncName))
						} else {
							handleErr = errors.Join(err)
						}
					}
				}
				data.Result <- EventHandleResult{
					Err: handleErr,
				}
			case <-time.After(autoReleaseTimeout):
				mb.status.CompareAndSwap(statusActived, statusDisactived)
				mb.status.CompareAndSwap(statusDisactived, statusDestoryed)
				if mb.status.Load() == statusDestoryed {
					time.Sleep(time.Second)
					if len(mb.receiver) > 0 {
						continue
					}
					close(mb.receiver)
					break loop
				}
			}
		}
		removeSelf(mb.name)
		mb.logger.Info("mailbox removed")
	})
}

type eventDataWithResult struct {
	EventData *EventData
	Result    chan EventHandleResult
}
