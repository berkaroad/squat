package messaging

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeMissingMessageHandler string = errors.SysErrCodePrefix + "MissingMessageHandler"
)

var (
	ErrMissingMessageHandler error = errors.NewWithCode(ErrCodeMissingMessageHandler, "missing message handler")
)
