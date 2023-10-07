package errors

func New(code string, text string) error {
	return &errorStringWithCode{text: text, code: code}
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
