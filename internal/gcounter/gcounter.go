package gcounter

import (
	"sync/atomic"
	"time"
)

var counter atomic.Int64

func Begin() {
	counter.Add(1)
}

func End() {
	counter.Add(-1)
}

//lint:ignore U1000 used by package utilities/goroutines
func wait() <-chan struct{} {
	doneCh := make(chan struct{})
	go func() {
		for counter.Load() > 0 {
			time.Sleep(time.Second)
		}
		close(doneCh)
	}()
	return doneCh
}

//lint:ignore U1000 used by package utilities/goroutines
func count() int64 {
	return counter.Load()
}
