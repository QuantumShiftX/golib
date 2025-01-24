package xerr

import (
	"fmt"
	"github.com/zeromicro/x/errors"
)

// New 创建自定义错误
func New(code ErrCode, msg string, args ...interface{}) error {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return errors.New(int(code), msg)
}

// NewParamErr ...
func NewParamErr(msg string) error {
	return New(ParamError, msg)
}
