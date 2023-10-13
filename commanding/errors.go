package commanding

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeWaitFromCommandHandlerTimeout string = "S:WaitFromCommandHandlerTimeout"
	ErrCodeWaitFromEventHandlerTimeout   string = "S:WaitFromEventHandlerTimeout"
)

var (
	ErrWaitFromCommandHandlerTimeout error = errors.NewWithCode(ErrCodeWaitFromCommandHandlerTimeout, "wait command result from command handler timeout")
	ErrWaitFromEventHandlerTimeout   error = errors.NewWithCode(ErrCodeWaitFromEventHandlerTimeout, "wait command result from event handler timeout")
)
