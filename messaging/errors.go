package messaging

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeNetworkError          string = "S:NetworkError"
	ErrCodeMissingMessageHandler string = "S:MissingMessageHandler"
)

var (
	ErrNetworkError          error = errors.New(ErrCodeNetworkError, "network error")
	ErrMissingMessageHandler error = errors.New(ErrCodeMissingMessageHandler, "missing message handler")
)
