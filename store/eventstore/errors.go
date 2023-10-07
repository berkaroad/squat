package eventstore

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeDuplicateKey      string = "S:DuplicateKey"
	ErrCodeUnexpectedVersion string = "S:UnexpectedVersion"
)

var (
	ErrDuplicateKey      error = errors.New(ErrCodeDuplicateKey, "duplicate key error")
	ErrUnexpectedVersion error = errors.New(ErrCodeUnexpectedVersion, "unexpected stream version")
)
