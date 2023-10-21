package publishedstore

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeDuplicateAggregateID string = errors.SysErrCodePrefix + "ps:DuplicateAggregateID"
)

var (
	ErrDuplicateAggregateID error = errors.NewWithCode(ErrCodeDuplicateAggregateID, "duplicate aggregate-id")
)
