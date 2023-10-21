package domain

import "github.com/berkaroad/squat/errors"

const (
	ErrCodeEmptyEventID        string = errors.SysErrCodePrefix + "EmptyEventID"
	ErrCodeEmptyAggregateID    string = errors.SysErrCodePrefix + "EmptyAggregateID"
	ErrCodeAggregateNoChange   string = errors.SysErrCodePrefix + "AggregateNoChange"
	ErrCodeAggregateHasChanged string = errors.SysErrCodePrefix + "AggregateHasChanged"
	ErrCodeGetAggregateFail    string = errors.SysErrCodePrefix + "GetAggregateFail"
	ErrCodeSaveAggregateFail   string = errors.SysErrCodePrefix + "SaveAggregateFail"
)

var (
	ErrEmptyEventID        error = errors.NewWithCode(ErrCodeEmptyEventID, "event-id is empty")
	ErrEmptyAggregateID    error = errors.NewWithCode(ErrCodeEmptyAggregateID, "aggregate-id is empty")
	ErrAggregateNoChange   error = errors.NewWithCode(ErrCodeAggregateNoChange, "aggregate no change")
	ErrAggregateHasChanged error = errors.NewWithCode(ErrCodeAggregateHasChanged, "aggregate has changed")
	ErrGetAggregateFail    error = errors.NewWithCode(ErrCodeGetAggregateFail, "get aggregate fail")
	ErrSaveAggregateFail   error = errors.NewWithCode(ErrCodeSaveAggregateFail, "save aggregate fail")
)
