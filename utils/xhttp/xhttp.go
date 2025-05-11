package xhttp

import (
	"context"
	"github.com/QuantumShiftX/golib/gerr"
	"github.com/QuantumShiftX/golib/xerr"
	"github.com/zeromicro/go-zero/core/trace"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
	"google.golang.org/grpc/status"
	"net/http"
)

const (
	// BusinessCodeOK represents the business code for success.
	BusinessCodeOK = 0
	// BusinessMsgOk represents the business message for success.
	BusinessMsgOk = "ok"
)

// BaseResponse is the base response struct.
type BaseResponse[T any] struct {
	// Code represents the business code, not the http status code.
	Code int `json:"code" xml:"code"`
	// Msg represents the business message, if Code = BusinessCodeOK,
	// and Msg is empty, then the Msg will be set to BusinessMsgOk.
	Message string `json:"message" xml:"message"`
	// Data represents the business data.
	Data T `json:"data,omitempty" xml:"data,omitempty"`
	// Trace id for trace
	TraceID string `json:"trace_id,omitempty" xml:"trace_id,omitempty"`
}

// JsonBaseResponseCtx writes v into w with appropriate http status code.
func JsonBaseResponseCtx(ctx context.Context, w http.ResponseWriter, v any) {
	var (
		traceId = trace.TraceIDFromContext(ctx)
		spanID  = trace.SpanIDFromContext(ctx)
	)

	if len(spanID) > 0 {
		w.Header().Set("X-Span-Id", spanID)
		w.Header().Set("X-Trace-Id", traceId)
	}

	// 获取 HTTP 状态码
	httpStatus := getHttpStatusFromError(v)

	// 写入响应（修复：正确传入状态码参数）
	httpx.WriteJsonCtx(ctx, w, httpStatus, wrapBaseResponse(v, ctx))
}

// getHttpStatusFromError 根据错误类型返回对应的 HTTP 状态码
func getHttpStatusFromError(v any) int {
	var code int

	switch data := v.(type) {
	case *xerr.XErr:
		code = int(data.Code)
	case *gerr.GError:
		code = int(data.Code)
	case *errors.CodeMsg:
		code = data.Code
	case errors.CodeMsg:
		code = data.Code
	case *status.Status:
		code = int(data.Code())
	case error:
		// 普通 error 默认返回 500
		return http.StatusInternalServerError
	default:
		// 成功情况
		return http.StatusOK
	}

	// 将业务错误码映射到 HTTP 状态码
	switch code {
	case 401:
		return http.StatusUnauthorized
	case 403:
		return http.StatusForbidden
	case 404:
		return http.StatusNotFound
	case 500:
		return http.StatusInternalServerError
	default:
		// 其他情况保持 200
		return http.StatusOK
	}
}

func wrapBaseResponse(v any, ctx context.Context) BaseResponse[any] {
	var resp BaseResponse[any]

	// 设置 trace id
	if ctx != nil {
		resp.TraceID = trace.TraceIDFromContext(ctx)
	}

	switch data := v.(type) {
	case *xerr.XErr:
		resp.Code = int(data.Code)
		resp.Message = data.Msg
	case *gerr.GError:
		resp.Code = int(data.Code)
		resp.Message = data.Msg
	case *errors.CodeMsg:
		resp.Code = data.Code
		resp.Message = data.Msg
	case errors.CodeMsg:
		resp.Code = data.Code
		resp.Message = data.Msg
	case *status.Status:
		resp.Code = int(data.Code())
		resp.Message = data.Message()
	case error:
		resp.Code = 500
		resp.Message = data.Error()
	default:
		resp.Code = BusinessCodeOK
		resp.Message = BusinessMsgOk
		resp.Data = v
	}
	return resp
}
