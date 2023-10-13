package eventsourcing

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeAggregateNoChange        string = "S:AggregateNoChange"
	ErrCodeEmptyAggregateID         string = "S:EmptyAggregateID"
	ErrCodeInvalidSnapshot          string = "S:InvalidSnapshot"
	ErrCodeInvalidEventStream       string = "S:InvalidEventStream"
	ErrCodeGetSnapshotFail          string = "S:GetSnapshotFail"
	ErrCodeQueryEventStreamListFail string = "S:QueryEventStreamListFail"
	ErrCodeAppendEventStreamFail    string = "S:AppendEventStreamFail"
)

var (
	ErrAggregateNoChange        error = errors.NewWithCode(ErrCodeAggregateNoChange, "aggregate no change")
	ErrEmptyAggregateID         error = errors.NewWithCode(ErrCodeEmptyAggregateID, "aggregate id is empty")
	ErrInvalidSnapshot          error = errors.NewWithCode(ErrCodeInvalidSnapshot, "invalid snapshot")
	ErrInvalidEventStream       error = errors.NewWithCode(ErrCodeInvalidEventStream, "invalid eventstream")
	ErrGetSnapshotFail          error = errors.NewWithCode(ErrCodeGetSnapshotFail, "get snapshot fail")
	ErrQueryEventStreamListFail error = errors.NewWithCode(ErrCodeQueryEventStreamListFail, "query eventstream list fail")
	ErrAppendEventStreamFail    error = errors.NewWithCode(ErrCodeAppendEventStreamFail, "append eventstream fail")
)
