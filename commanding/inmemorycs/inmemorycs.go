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

func (cs *InMemoryCommandService) Send(ctx context.Context, cmd commanding.Command) (*commanding.CommandHandleResult, error) {
	if !cs.initialized {
		panic("not initialized")
	}

	var extensions messaging.Extensions
	eventMetadata := messaging.FromContext(ctx)
	if eventMetadata == nil {
		extensions = map[string]string{string(messaging.ExtensionKeyNoticeServiceEndpoint): cs.NoticeServiceEndpoint}
	} else {
		if eventMetadata.Category == commanding.MailCategory {
			panic("'InMemoryCommandService.Send(context.Context, commanding.Command)' couldn't be invoked in CommandHandler")
		}
		extensions = eventMetadata.Extensions.Clone().
			Set(messaging.ExtensionKeyNoticeServiceEndpoint, cs.NoticeServiceEndpoint).
			Set(messaging.ExtensionKeyFromMessageID, eventMetadata.MessageID).
			Set(messaging.ExtensionKeyFromMessageType, eventMetadata.MessageType)
	}
	fromCommandWatchItem := cs.watcher.Watch(cmd.CommandID(), commanding.CommandHandleResultProvider)
	fromEventWatchItem := cs.watcher.Watch(cmd.CommandID(), eventing.CommandHandleResultProvider)

	cs.dispatcher.Dispatch(&commanding.CommandData{
		Command:    cmd,
		Extensions: extensions,
	}, nil)
	return commanding.NewCommandHandleResult(fromCommandWatchItem, fromEventWatchItem), nil
}
