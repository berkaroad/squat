package eventstore

import (
	"github.com/berkaroad/squat/errors"
)

const (
	ErrCodeDuplicateCommandID             string = errors.SysErrCodePrefix + "es:DuplicateCommandID"
	ErrCodeUnexpectedVersion              string = errors.SysErrCodePrefix + "es:UnexpectedVersion"
	ErrCodeEventStreamConcurrencyConflict string = errors.SysErrCodePrefix + "es:EventStreamConcurrencyConflict"
)

var (
	ErrUnexpectedVersion              error = errors.NewWithCode(ErrCodeUnexpectedVersion, "unexpected stream-version")
	ErrEventStreamConcurrencyConflict error = errors.NewWithCode(ErrCodeEventStreamConcurrencyConflict, "concurrency conflict with eventstream")
)

func IsErrDuplicateCommandID(err error) bool {
	return errors.GetErrorCode(err) == ErrCodeDuplicateCommandID
}

func NewErrDuplicateCommandID(commandID string) error {
	return errors.NewWithCodeAndStates(ErrCodeDuplicateCommandID, "duplicate command-id", map[string]string{"command-id": commandID})
}

func GetCommandIDFromErrDuplicateCommandID(err error) string {
	if !IsErrDuplicateCommandID(err) {
		return ""
	}
	states := errors.GetErrorStates(err)
	return states["command-id"]
}
