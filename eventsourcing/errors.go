package eventsourcing

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeInvalidSnapshot    string = errors.SysErrCodePrefix + "InvalidSnapshot"
	ErrCodeInvalidEventStream string = errors.SysErrCodePrefix + "InvalidEventStream"
)

var (
	ErrInvalidSnapshot    error = errors.NewWithCode(ErrCodeInvalidSnapshot, "invalid snapshot")
	ErrInvalidEventStream error = errors.NewWithCode(ErrCodeInvalidEventStream, "invalid eventstream")
)
