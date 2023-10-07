package messaging

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeNetworkError          string = "S:NetworkError"
	ErrCodeMissingMessageHandler string = "S:MissingMessageHandler"
)

var (
	ErrNetworkError          error = errors.NewWithCode(ErrCodeNetworkError, "network error")
	ErrMissingMessageHandler error = errors.NewWithCode(ErrCodeMissingMessageHandler, "missing message handler")
)
