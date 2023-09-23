package goroutine

import (
	"context"
	"sync/atomic"
	"time"
)

var counter atomic.Int64

func Go(ctx context.Context, action func(ctx context.Context)) {
	counter.Add(1)
	go func() {
		defer counter.Add(-1)
		action(ctx)
	}()
}

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

func count() int64 {
	return counter.Load()
}
