package gerr

import (
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GError gRPC error
type GError struct {
	Code ErrCode
	Msg  string
}

// 实现 error 接口
func (e *GError) Error() string {
	return fmt.Sprintf("code: %v, Msg: %s", e.Code, e.Msg)
}

// 转换为 gRPC 状态错误
func (e *GError) GRPCStatus() error {
	return status.Errorf(codes.Code(e.Code), e.Msg)
}

// New 创建并直接返回 gRPC 状态错误
func New(code ErrCode, msg string) error {
	return status.Errorf(codes.Code(code), msg)
}

// NewGError 创建 GError 对象
func NewGError(code ErrCode, msg string) *GError {
	return &GError{Code: code, Msg: msg}
}

// Wrap 包装已有错误，直接返回 gRPC 状态错误
func Wrap(code ErrCode, err error) error {
	return status.Errorf(codes.Code(code), err.Error())
}

// WrapGError 包装已有错误，返回 GError 对象
func WrapGError(code ErrCode, err error) *GError {
	return &GError{Code: code, Msg: err.Error()}
}

// Is 判断错误是否是特定的错误码
func Is(err error, targetCode ErrCode) bool {
	var e *GError
	if errors.As(err, &e) {
		return e.Code == targetCode
	}
	st, ok := status.FromError(err)
	return ok && ErrCode(st.Code()) == targetCode
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
		return &GError{Code: ErrCode(codes.Unknown), Msg: err.Error()}
	}
	return &GError{Code: ErrCode(st.Code()), Msg: st.Message()}
}
