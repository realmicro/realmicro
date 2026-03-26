package errors

import "fmt"

type Error struct {
	ErrCode int64  `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

func (err *Error) Error() string {
	return fmt.Sprintf("%d:%s", err.ErrCode, err.ErrMsg)
}
