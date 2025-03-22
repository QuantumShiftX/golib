package xerr

import (
	serr "errors"
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

// 定义自定义错误接口
type XError interface {
	error
	ErrorCode() int
}

// 检查错误是否实现了XError接口
func IsXError(err error) bool {
	var xError XError
	ok := serr.As(err, &xError)
	return ok
}

// 从错误中获取错误码
func GetErrorCode(err error) int {
	var xe XError
	if serr.As(err, &xe) {
		return xe.ErrorCode()
	}
	return 0
}

// HandleError 简化版错误处理函数
// 只需传入一个错误参数
// 如果是自定义错误，直接返回
// 如果不是，则包装成默认的服务器错误
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	// 检查是否为自定义错误
	if IsXError(err) {
		// 已经是自定义错误，直接返回
		return err
	}

	// 不是自定义错误，包装成默认的服务器错误
	return New(ServerInternalError, "系统错误: "+err.Error())
}

// HandleParamError 处理参数错误
func HandleParamError(err error) error {
	if err == nil {
		return nil
	}

	if IsXError(err) {
		return err
	}

	return New(ParamError, "参数错误: "+err.Error())
}

// HandleDBError 处理数据库错误
func HandleDBError(err error) error {
	if err == nil {
		return nil
	}

	if IsXError(err) {
		return err
	}

	return New(DbError, "数据库错误: "+err.Error())
}
