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
	ErrInvalidSnapshot          error = errors.NewWithCode(ErrCodeGetSnapshotFail, "invalid snapshot")
	ErrInvalidEventStream       error = errors.NewWithCode(ErrCodeGetSnapshotFail, "invalid eventstream")
	ErrGetSnapshotFail          error = errors.NewWithCode(ErrCodeGetSnapshotFail, "get snapshot fail")
	ErrSaveSnapshotFail         error = errors.NewWithCode(ErrCodeSaveSnapshotFail, "save snapshot fail")
	ErrQueryEventStreamListFail error = errors.NewWithCode(ErrCodeQueryEventStreamListFail, "query eventstream list fail")
	ErrAppendEventStreamFail    error = errors.NewWithCode(ErrCodeAppendEventStreamFail, "append eventstream fail")
)
