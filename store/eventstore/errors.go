package eventstore

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeDuplicateKey                   string = "S:DuplicateKey"
	ErrCodeUnexpectedVersion              string = "S:UnexpectedVersion"
	ErrCodeEventStreamConcurrencyConflict string = "S:EventStreamConcurrencyConflict"
)

var (
	ErrDuplicateKey                   error = errors.NewWithCode(ErrCodeDuplicateKey, "duplicate key error")
	ErrUnexpectedVersion              error = errors.NewWithCode(ErrCodeUnexpectedVersion, "unexpected stream version")
	ErrEventStreamConcurrencyConflict error = errors.NewWithCode(ErrCodeEventStreamConcurrencyConflict, "concurrency conflict with eventstream")
)
