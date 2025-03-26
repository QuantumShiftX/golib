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

// NewParamErr 创建参数错误
func NewParamErr(msg string) error {
	return New(ParamError, msg)
}

// 定义自定义错误接口
type XError interface {
	error
	ErrorCode() int
}

// IsXError 检查错误是否实现了XError接口
func IsXError(err error) bool {
	if err == nil {
		return false
	}

	var xError XError
	return serr.As(err, &xError)
}

// GetErrorCode 从错误中获取错误码
func GetErrorCode(err error) int {
	if err == nil {
		return 0
	}

	// 尝试解包为zeromicro错误
	var zerr *errors.CodeMsg
	if serr.As(err, &zerr) {
		return zerr.Code
	}

	// 再尝试使用XError接口
	var xe XError
	if serr.As(err, &xe) {
		return xe.ErrorCode()
	}

	return 0
}

// GetErrorMessage 从错误中获取错误消息
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	// 尝试解包为zeromicro错误
	var zerr *errors.CodeMsg
	if serr.As(err, &zerr) {
		return zerr.Msg
	}

	return err.Error()
}

// GetCodeAndMessage 从错误中提取错误代码和消息
// 返回错误代码、错误消息以及是否为自定义XError
func GetCodeAndMessage(err error) (int, string, bool) {
	if err == nil {
		return 0, "", false
	}

	// 尝试解包为zeromicro错误
	var zerr *errors.CodeMsg
	if serr.As(err, &zerr) {
		return zerr.Code, zerr.Msg, true
	}

	// 尝试使用我们的XError接口
	var xe XError
	if serr.As(err, &xe) {
		return xe.ErrorCode(), err.Error(), true
	}

	// 不是自定义错误
	return 0, err.Error(), false
}

// FormatError 将任何错误转换为ErrCodeMessage结构体
func FormatError(err error) ErrCodeMessage {
	if err == nil {
		return ErrCodeMessage{Code: 0, Msg: ""}
	}

	code, msg, _ := GetCodeAndMessage(err)
	return ErrCodeMessage{
		Code: ErrCode(code),
		Msg:  msg,
	}
}

// ExtractErrorDetails 返回错误代码和消息
func ExtractErrorDetails(err error) (ErrCode, string) {
	if err == nil {
		return 0, ""
	}

	code, msg, _ := GetCodeAndMessage(err)
	return ErrCode(code), msg
}

// WrapError 包装错误并保留原始错误信息
func WrapError(errType ErrCode, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	return New(errType, fmt.Sprintf("%s: %s", msg, err.Error()))
}

// ToJSON 将错误转换为JSON格式的错误信息
func ToJSON(err error) map[string]interface{} {
	if err == nil {
		return map[string]interface{}{
			"code": 0,
			"msg":  "",
		}
	}

	errInfo := FormatError(err)
	return map[string]interface{}{
		"code": errInfo.Code,
		"msg":  errInfo.Msg,
	}
}

// HandleError 简化版错误处理函数
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	// 检查是否为自定义错误
	if IsXError(err) || IsCodeMsg(err) {
		return err
	}

	// 包装成默认的服务器错误
	return WrapError(ServerInternalError, err, "系统错误")
}

// IsCodeMsg 检查错误是否为CodeMsg类型
func IsCodeMsg(err error) bool {
	if err == nil {
		return false
	}

	var codeMsg *errors.CodeMsg
	return serr.As(err, &codeMsg)
}

// HandleParamError 处理参数错误
func HandleParamError(err error) error {
	if err == nil {
		return nil
	}

	if IsXError(err) || IsCodeMsg(err) {
		return err
	}

	return WrapError(ParamError, err, "参数错误")
}

// HandleDBError 处理数据库错误
func HandleDBError(err error) error {
	if err == nil {
		return nil
	}

	if IsXError(err) || IsCodeMsg(err) {
		return err
	}

	return WrapError(DbError, err, "数据库错误")
}

// IsErrorCode 检查错误是否为指定的错误码
func IsErrorCode(err error, code ErrCode) bool {
	if err == nil {
		return false
	}

	errCode := GetErrorCode(err)
	return errCode == int(code)
}

// IsParamError 检查是否为参数错误
func IsParamError(err error) bool {
	return IsErrorCode(err, ParamError)
}

// IsUnauthorizedError 检查是否为无权限错误
func IsUnauthorizedError(err error) bool {
	return IsErrorCode(err, UnauthorizedError)
}

// IsDBError 检查是否为数据库错误
func IsDBError(err error) bool {
	return IsErrorCode(err, DbError)
}

// ExampleUsage 示例使用函数
func ExampleUsage() {
	// 使用预定义错误
	err := ErrUnauthorized

	// 提取错误信息
	errInfo := FormatError(err)
	fmt.Printf("错误代码: %d, 错误消息: %s\n", errInfo.Code, errInfo.Msg)

	// 转换为JSON
	jsonErr := ToJSON(err)
	fmt.Printf("JSON错误: %v\n", jsonErr)

	// 检查错误类型
	if IsUnauthorizedError(err) {
		fmt.Println("这是一个未授权错误")
	}
}
