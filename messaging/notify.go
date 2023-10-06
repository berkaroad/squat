package messaging

type MessageHandleResultNotifier[TMessage any] interface {
	Notify(messageID string, resultProvider string, data MessageHandleResult)
}

type MessageHandleResultSubscriber[TMessage any] interface {
	Subscribe(messageID string, resultProvider string, resultCh chan<- MessageHandleResult)
}
