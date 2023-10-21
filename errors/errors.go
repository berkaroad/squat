package errors

import "sync"

const (
	SysErrCodePrefix string = "squat:"
)

var mapping sync.Map

func NewWithCode(code string, text string) error {
	var err error
	if val, ok := mapping.Load(code); ok {
		err = val.(*errorStringWithCode)
	} else {
		err = &errorStringWithCode{text: text, code: code}
		mapping.Store(code, err)
	}
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

type errorStringWithCode struct {
	code string
	text string
}

func (e *errorStringWithCode) Error() string {
	return e.text
}

func (e *errorStringWithCode) ErrorCode() string {
	return e.code
}
