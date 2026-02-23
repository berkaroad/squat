package eventing

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeStoppedEventBus string = errors.SysErrCodePrefix + "StoppedEventBus"
)

var (
	ErrStoppedEventBus error = errors.NewWithCode(ErrCodeStoppedEventBus, "stopped event bus")
)
