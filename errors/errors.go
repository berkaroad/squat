package errors

import (
	"fmt"
	"sync"
)

const (
	SysErrCodePrefix string = "squat:"
)

var mapping sync.Map

func NewWithCode(code string, text string) error {
	var err error
	if val, ok := mapping.Load(code); ok {
		err = val.(*errorStringWithCodeAndStates)
	} else {
		err = &errorStringWithCodeAndStates{text: text, code: code}
		mapping.Store(code, err)
	}
	return err
}

func NewWithCodeAndStates(code string, text string, states map[string]string) error {
	for k, v := range states {
		text += fmt.Sprintf(", %s=%s ", k, v)
	}
	err := &errorStringWithCodeAndStates{text: text, code: code, states: states}
	return err
}

func GetErrorCode(err error) string {
	if err == nil {
		return ""
	}

	for {
		if x, ok := err.(interface{ ErrorCode() string }); ok {
			return x.ErrorCode()
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return ""
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				code := GetErrorCode(err)
				if code != "" {
					return code
				}
			}
			return ""
		default:
			return ""
		}
	}
}

func GetErrorStates(err error) map[string]string {
	if err == nil {
		return nil
	}

	for {
		if x, ok := err.(interface {
			ErrorCode() string
			ErrorStates() map[string]string
		}); ok {
			return x.ErrorStates()
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return nil
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				code := GetErrorCode(err)
				if code != "" {
					return GetErrorStates(err)
				}
			}
			return nil
		default:
			return nil
		}
	}
}

type errorStringWithCodeAndStates struct {
	code   string
	text   string
	states map[string]string
}

func (e *errorStringWithCodeAndStates) Error() string {
	return e.text
}

func (e *errorStringWithCodeAndStates) ErrorCode() string {
	return e.code
}

func (e *errorStringWithCodeAndStates) ErrorStates() map[string]string {
	if len(e.states) == 0 {
		return nil
	}
	clone := make(map[string]string)
	for k, v := range e.states {
		clone[k] = v
	}
	return clone
}
