package snapshotstore

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeDuplicateAggregateID string = errors.SysErrCodePrefix + "ss:DuplicateAggregateID"
)

var (
	ErrDuplicateAggregateID error = errors.NewWithCode(ErrCodeDuplicateAggregateID, "duplicate aggregate-id")
)
