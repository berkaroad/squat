package goroutine

import (
	_ "unsafe"
)

//go:linkname Wait github.com/berkaroad/squat/internal/gcounter.wait
func Wait() <-chan struct{}

//go:linkname Count github.com/berkaroad/squat/internal/gcounter.count
func Count() int64
