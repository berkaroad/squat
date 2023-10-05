package inmemorymhrn

import (
	"fmt"
	"sync"

	"github.com/berkaroad/squat/messaging"
)

var instance *InMemoryMessageHandleResultNotifier = &InMemoryMessageHandleResultNotifier{}

func Default() *InMemoryMessageHandleResultNotifier {
	return instance
}

var _ messaging.MessageHandleResultNotifier[any] = (*InMemoryMessageHandleResultNotifier)(nil)
var _ messaging.MessageHandleResultSubscriber[any] = (*InMemoryMessageHandleResultNotifier)(nil)

type InMemoryMessageHandleResultNotifier struct {
	resultMapper sync.Map
}

func (notifier *InMemoryMessageHandleResultNotifier) Notify(messageID string, resultProvider string, result messaging.MessageHandleResult) {
	key := fmt.Sprintf("%s:%s", messageID, resultProvider)
	if val, ok := notifier.resultMapper.LoadAndDelete(key); ok {
		resultCh := val.(chan<- messaging.MessageHandleResult)
		resultCh <- result
	}
}

func (notifier *InMemoryMessageHandleResultNotifier) Subscribe(messageID string, resultProvider string, resultCh chan<- messaging.MessageHandleResult) {
	if resultCh != nil {
		key := fmt.Sprintf("%s:%s", messageID, resultProvider)
		notifier.resultMapper.Store(key, resultCh)
	}
}
