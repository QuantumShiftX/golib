package xerr

import (
	serr "errors"
	"fmt"
)

// XErr 统一错误类型
type XErr struct {
	Code ErrCode `json:"code"`
	Msg  string  `json:"msg"`
	err  error   // 原始错误，可以为nil
}

// 实现 error 接口
func (e *XErr) Error() string {
	if e.Msg == "" && e.err != nil {
		return e.err.Error()
	}
	return e.Msg
}

// 实现 XError 接口
func (e *XErr) ErrorCode() int {
	return int(e.Code)
}

// GetOriginalError 获取原始错误
func (e *XErr) GetOriginalError() error {
	return e.err
}

// Unwrap 实现错误解包
func (e *XErr) Unwrap() error {
	return e.err
}

// New 创建自定义错误
func New(code ErrCode, msg string, args ...interface{}) *XErr {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &XErr{
		Code: code,
		Msg:  msg,
	}
}

// Wrap 包装错误
func Wrap(code ErrCode, err error, msg string, args ...interface{}) *XErr {
	if err == nil {
		return nil
	}

	wrappedMsg := msg
	if len(args) > 0 {
		wrappedMsg = fmt.Sprintf(msg, args...)
	}

	// 检查是否已经是 XErr
	var xe *XErr
	if serr.As(err, &xe) {
		// 如果已经是XErr，保留原有的code，除非明确指定了新code
		if code == 0 {
			code = xe.Code
		}
		if wrappedMsg == "" {
			wrappedMsg = xe.Msg
		} else {
			wrappedMsg = fmt.Sprintf("%s: %s", wrappedMsg, xe.Msg)
		}
	} else if wrappedMsg == "" {
		wrappedMsg = err.Error()
	} else {
		wrappedMsg = fmt.Sprintf("%s: %s", wrappedMsg, err.Error())
	}

	return &XErr{
		Code: code,
		Msg:  wrappedMsg,
		err:  err,
	}
}

// NewParamErr 创建参数错误
func NewParamErr(msg string) *XErr {
	return New(ParamError, msg)
}

// IsXErr 检查错误是否为XErr类型
func IsXErr(err error) bool {
	if err == nil {
		return false
	}

	var xe *XErr
	return serr.As(err, &xe)
}

// GetErrorCode 从错误中获取错误码
func GetErrorCode(err error) int {
	if err == nil {
		return 0
	}

	var xe *XErr
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

	var xe *XErr
	if serr.As(err, &xe) {
		return xe.Error()
	}

	return err.Error()
}

// GetCodeAndMessage 从错误中提取错误代码和消息
func GetCodeAndMessage(err error) (int, string, bool) {
	if err == nil {
		return 0, "", false
	}

	var xe *XErr
	if serr.As(err, &xe) {
		return xe.ErrorCode(), xe.Error(), true
	}

	// 不是自定义错误
	return 0, err.Error(), false
}

// ToJSON 将错误转换为JSON格式的错误信息
func ToJSON(err error) map[string]interface{} {
	if err == nil {
		return map[string]interface{}{
			"code": 0,
			"msg":  "",
		}
	}

	code, msg, _ := GetCodeAndMessage(err)
	return map[string]interface{}{
		"code": code,
		"msg":  msg,
	}
}

// FromError 从标准错误转换为XErr
func FromError(err error) *XErr {
	if err == nil {
		return nil
	}

	var xe *XErr
	if serr.As(err, &xe) {
		return xe
	}

	return &XErr{
		Code: ServerInternalError, // 默认为服务器内部错误
		Msg:  err.Error(),
		err:  err,
	}
}

// HandleError 简化版错误处理函数
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	// 检查是否为自定义错误
	if IsXErr(err) {
		return err
	}

	// 包装成默认的服务器错误
	return Wrap(ServerInternalError, err, "系统错误")
}

// HandleParamError 处理参数错误
func HandleParamError(err error) error {
	if err == nil {
		return nil
	}

	if IsXErr(err) {
		return err
	}

	return Wrap(ParamError, err, "参数错误")
}

// HandleDBError 处理数据库错误
func HandleDBError(err error) error {
	if err == nil {
		return nil
	}

	if IsXErr(err) {
		return err
	}

	return Wrap(DbError, err, "数据库错误")
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
	err := ErrorServer

	// 获取错误信息
	code, msg, isCustom := GetCodeAndMessage(err)
	fmt.Printf("错误代码: %d, 错误消息: %s, 是否自定义: %v\n", code, msg, isCustom)

	// 转换为JSON
	jsonErr := ToJSON(err)
	fmt.Printf("JSON错误: %v\n", jsonErr)

	// 创建新的错误
	//paramErr := New(ParamError, "无效的用户ID: %d", 1001)

	// 包装标准库错误
	stdErr := serr.New("无法连接数据库")
	wrappedErr := Wrap(DbError, stdErr, "查询用户记录失败")

	// 获取包装错误的信息
	code, msg, _ = GetCodeAndMessage(wrappedErr)
	fmt.Printf("包装错误代码: %d, 错误消息: %s\n", code, msg)
}
