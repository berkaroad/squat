package messaging

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeMissingMessageHandler string = "S:MissingMessageHandler"
)

var (
	ErrMissingMessageHandler error = errors.NewWithCode(ErrCodeMissingMessageHandler, "missing message handler")
)
