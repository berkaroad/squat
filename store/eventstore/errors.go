package eventstore

import "errors"

var (
	ErrDuplicateKey      error = errors.New("duplicate key error")
	ErrUnexpectedVersion error = errors.New("unexpected stream version")
)
