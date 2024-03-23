package errors

import "encoding/json"

type MultiError struct {
	Errors []*Error `json:"errors"`
}

func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]*Error, 0),
	}
}

func (e *MultiError) Append(err ...*Error) {
	e.Errors = append(e.Errors, err...)
}

func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

func (e *MultiError) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}
