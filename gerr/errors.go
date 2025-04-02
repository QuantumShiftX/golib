package gerr

import (
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GError gRPC error
type GError struct {
	Code codes.Code
	Msg  string
}

// 实现 error 接口
func (e *GError) Error() string {
	return fmt.Sprintf("code: %v, Msg: %s", e.Code, e.Msg)
}

// 转换为 gRPC 状态错误
func (e *GError) GRPCStatus() error {
	return status.Errorf(e.Code, e.Msg)
}

// New 创建 GError
func New(code codes.Code, msg string) *GError {
	return &GError{Code: code, Msg: msg}
}

// Wrap 包装已有错误
func Wrap(code codes.Code, err error) *GError {
	return &GError{Code: code, Msg: err.Error()}
}

// Is 判断错误是否是特定的 GError
func Is(err error, target *GError) bool {
	var e *GError
	if errors.As(err, &e) {
		return e.Code == target.Code
	}
	st, ok := status.FromError(err)
	return ok && st.Code() == target.Code
}

// FromError 解析 gRPC 错误，转换为 GError
func FromError(err error) *GError {
	if err == nil {
		return nil
	}
	var e *GError
	if errors.As(err, &e) {
		return e
	}
	st, ok := status.FromError(err)
	if !ok {
		return &GError{Code: codes.Unknown, Msg: err.Error()}
	}
	return &GError{Code: st.Code(), Msg: st.Message()}
}
