package eventstore

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeDuplicateCommandIDOrStreamVersion string = errors.SysErrCodePrefix + "es:DuplicateCommandID+StreamVersion"
	ErrCodeUnexpectedVersion                 string = errors.SysErrCodePrefix + "es:UnexpectedVersion"
	ErrCodeEventStreamConcurrencyConflict    string = errors.SysErrCodePrefix + "es:EventStreamConcurrencyConflict"
)

var (
	ErrDuplicateCommandIDOrStreamVersion error = errors.NewWithCode(ErrCodeDuplicateCommandIDOrStreamVersion, "duplicate command-id or stream-version")
	ErrUnexpectedVersion                 error = errors.NewWithCode(ErrCodeUnexpectedVersion, "unexpected stream-version")
	ErrEventStreamConcurrencyConflict    error = errors.NewWithCode(ErrCodeEventStreamConcurrencyConflict, "concurrency conflict with eventstream")
)
