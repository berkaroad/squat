package commanding

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeEmptyCommandID                string = errors.SysErrCodePrefix + "EmptyCommandID"
	ErrCodeWaitFromCommandHandlerTimeout string = errors.SysErrCodePrefix + "WaitFromCommandHandlerTimeout"
	ErrCodeWaitFromEventHandlerTimeout   string = errors.SysErrCodePrefix + "WaitFromEventHandlerTimeout"
	ErrCodeWaitCommandResultCancelled    string = errors.SysErrCodePrefix + "WaitCommandResultCancelled"
	ErrCodeStoppedCommandBus             string = errors.SysErrCodePrefix + "StoppedCommandBus"
)

var (
	ErrEmptyCommandID                error = errors.NewWithCode(ErrCodeEmptyCommandID, "command-id is empty")
	ErrWaitFromCommandHandlerTimeout error = errors.NewWithCode(ErrCodeWaitFromCommandHandlerTimeout, "wait command result from command handler timeout")
	ErrWaitFromEventHandlerTimeout   error = errors.NewWithCode(ErrCodeWaitFromEventHandlerTimeout, "wait command result from event handler timeout")
	ErrWaitCommandResultCancelled    error = errors.NewWithCode(ErrCodeWaitCommandResultCancelled, "wait command result cancelled")
	ErrStoppedCommandBus             error = errors.NewWithCode(ErrCodeStoppedCommandBus, "stopped command bus")
)
