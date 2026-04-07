package inmemorycs

import (
	"context"
	"sync"

	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/messaging"
)

var instance = &InMemoryCommandService{}

func Default() *InMemoryCommandService {
	return instance
}

var _ commanding.CommandService = (*InMemoryCommandService)(nil)

type InMemoryCommandService struct {
	NoticeServiceEndpoint string
	dispatcher            commanding.CommandDispatcher
	watcher               messaging.MessageHandleResultWatcher

	initOnce    sync.Once
	initialized bool
}

func (cs *InMemoryCommandService) Initialize(dispatcher commanding.CommandDispatcher, watcher messaging.MessageHandleResultWatcher) *InMemoryCommandService {
	cs.initOnce.Do(func() {
		if dispatcher == nil {
			dispatcher = &commanding.DefaultCommandDispatcher{}
		}
		if watcher == nil {
			panic("param 'watcher' is null")
		}
		cs.dispatcher = dispatcher
		cs.watcher = watcher
		cs.initialized = true
	})
	return cs
}

func (cs *InMemoryCommandService) Send(ctx context.Context, cmd commanding.Command) error {
	if !cs.initialized {
		panic("not initialized")
	}

	eventMetadata := messaging.FromContext(ctx)
	if eventMetadata == nil {
		cs.dispatcher.Dispatch(&commanding.CommandData{
			Command: cmd,
		}, nil)
		return nil
	} else {
		if eventMetadata.Category == commanding.MailCategory {
			panic("'InMemoryCommandService.Send(context.Context, commanding.Command)' couldn't be invoked in CommandHandler")
		}
		cs.dispatcher.Dispatch(&commanding.CommandData{
			Command: cmd,
			Extensions: eventMetadata.Extensions.Clone().
				Remove(messaging.ExtensionKeyNoticeServiceEndpoint).
				Set(messaging.ExtensionKeyFromMessageID, eventMetadata.MessageID).
				Set(messaging.ExtensionKeyFromMessageType, eventMetadata.MessageType),
		}, nil)
		return nil
	}
}

func (cs *InMemoryCommandService) Execute(ctx context.Context, cmd commanding.Command) (*commanding.CommandHandleResult, error) {
	if !cs.initialized {
		panic("not initialized")
	}

	eventMetadata := messaging.FromContext(ctx)
	if eventMetadata == nil {
		fromCommandWatchItem := cs.watcher.Watch(cmd.CommandID(), commanding.CommandHandleResultProvider)
		fromEventWatchItem := cs.watcher.Watch(cmd.CommandID(), eventing.CommandHandleResultProvider)

		cs.dispatcher.Dispatch(&commanding.CommandData{
			Command:    cmd,
			Extensions: map[string]string{string(messaging.ExtensionKeyNoticeServiceEndpoint): cs.NoticeServiceEndpoint},
		}, nil)
		return commanding.NewCommandHandleResult(fromCommandWatchItem, fromEventWatchItem), nil
	} else {
		if eventMetadata.Category == commanding.MailCategory {
			panic("'InMemoryCommandService.Execute(context.Context, commanding.Command)' couldn't be invoked in CommandHandler")
		}

		cs.dispatcher.Dispatch(&commanding.CommandData{
			Command: cmd,
			Extensions: eventMetadata.Extensions.Clone().
				Set(messaging.ExtensionKeyNoticeServiceEndpoint, cs.NoticeServiceEndpoint).
				Set(messaging.ExtensionKeyFromMessageID, eventMetadata.MessageID).
				Set(messaging.ExtensionKeyFromMessageType, eventMetadata.MessageType),
		}, nil)
		return commanding.NewCommandHandleResult(nil, nil), nil
	}
}
