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

var _ messaging.MessageHandleResultNotifier = (*InMemoryMessageHandleResultNotifier)(nil)
var _ messaging.MessageHandleResultWatcher = (*InMemoryMessageHandleResultNotifier)(nil)

type InMemoryMessageHandleResultNotifier struct {
	resultMapper sync.Map
}

func (notifier *InMemoryMessageHandleResultNotifier) Notify(endpoint string, messageID string, resultProvider string, result messaging.MessageHandleResult) {
	key := fmt.Sprintf("%s:%s", messageID, resultProvider)
	if val, ok := notifier.resultMapper.LoadAndDelete(key); ok {
		writeResultCh := val.(chan messaging.MessageHandleResult)
		writeResultCh <- result
	}
}

func (notifier *InMemoryMessageHandleResultNotifier) Watch(messageID string, resultProvider string) messaging.MessageHandleResultWatchItem {
	resultCh := make(chan messaging.MessageHandleResult, 1)
	key := fmt.Sprintf("%s:%s", messageID, resultProvider)
	notifier.resultMapper.Store(key, resultCh)
	return &watchResult{
		resultCh: resultCh,
		unwatch: func() {
			notifier.resultMapper.Delete(key)
		},
	}
}

var _ messaging.MessageHandleResultWatchItem = (*watchResult)(nil)

type watchResult struct {
	resultCh <-chan messaging.MessageHandleResult
	unwatch  func()
}

func (wr *watchResult) Result() <-chan messaging.MessageHandleResult {
	return wr.resultCh
}

func (wr *watchResult) Unwatch() {
	wr.unwatch()
}
