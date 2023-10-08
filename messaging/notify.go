package messaging

type MessageHandleResultNotifier[TMessage any] interface {
	Notify(messageID string, resultProvider string, result MessageHandleResult)
}

type MessageHandleResultWatcher[TMessage any] interface {
	Watch(messageID string, resultProvider string) MessageHandleResultWatchItem
}

type MessageHandleResultWatchItem interface {
	Result() <-chan MessageHandleResult
	Unwatch()
}
