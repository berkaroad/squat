package goroutine

import (
	_ "unsafe"
)

//go:linkname Wait github.com/berkaroad/squat/internal/goroutine.wait
func Wait() <-chan struct{}

//go:linkname Count github.com/berkaroad/squat/internal/goroutine.count
func Count() int64
