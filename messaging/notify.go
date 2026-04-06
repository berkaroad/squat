package messaging

type MessageHandleResultNotifier interface {
	Notify(endpoint string, messageID string, resultProvider string, result MessageHandleResult)
}

type MessageHandleResultWatcher interface {
	Watch(messageID string, resultProvider string) MessageHandleResultWatchItem
	Receive(messageID string, resultProvider string, result MessageHandleResult)
}

type MessageHandleResultWatchItem interface {
	Result() <-chan MessageHandleResult
	Unwatch()
}
