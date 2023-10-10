package goroutine

import (
	_ "unsafe"
)

//go:linkname Wait github.com/berkaroad/squat/internal/counter.wait
func Wait() <-chan struct{}

//go:linkname Count github.com/berkaroad/squat/internal/counter.count
func Count() int64
