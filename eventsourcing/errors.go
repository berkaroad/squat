package eventsourcing

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeInvalidSnapshot          string = "S:InvalidSnapshot"
	ErrCodeInvalidEventStream       string = "S:InvalidEventStream"
	ErrCodeGetSnapshotFail          string = "S:GetSnapshotFail"
	ErrCodeSaveSnapshotFail         string = "S:SaveSnapshotFail"
	ErrCodeQueryEventStreamListFail string = "S:QueryEventStreamListFail"
	ErrCodeAppendEventStreamFail    string = "S:AppendEventStreamFail"
)

var (
	ErrInvalidSnapshot          error = errors.New(ErrCodeGetSnapshotFail, "invalid snapshot")
	ErrInvalidEventStream       error = errors.New(ErrCodeGetSnapshotFail, "invalid eventstream")
	ErrGetSnapshotFail          error = errors.New(ErrCodeGetSnapshotFail, "get snapshot fail")
	ErrSaveSnapshotFail         error = errors.New(ErrCodeSaveSnapshotFail, "save snapshot fail")
	ErrQueryEventStreamListFail error = errors.New(ErrCodeQueryEventStreamListFail, "query eventstream list fail")
	ErrAppendEventStreamFail    error = errors.New(ErrCodeAppendEventStreamFail, "append eventstream fail")
)
