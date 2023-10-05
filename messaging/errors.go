package messaging

import "errors"

var (
	ErrNetworkError          error = errors.New("network error")
	ErrMissingMessageHandler error = errors.New("missing message handler")
)
