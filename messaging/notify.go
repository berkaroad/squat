package messaging

type MessageHandleResultNotifier interface {
	Notify(endpoint string, messageID string, resultProvider string, result MessageHandleResult)
}

type MessageHandleResultWatcher interface {
	Watch(messageID string, resultProvider string) MessageHandleResultWatchItem
}

type MessageHandleResultWatchItem interface {
	Result() <-chan MessageHandleResult
	Unwatch()
}

var _ MessageHandleResultWatcher = (*NullMessageHandleResultWatcher)(nil)

type NullMessageHandleResultWatcher struct{}

func (watcher *NullMessageHandleResultWatcher) Watch(messageID string, resultProvider string) MessageHandleResultWatchItem {
	return nil
}
