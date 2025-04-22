package metadata

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContextOptions 用于自定义创建上下文的选项
type ContextOptions struct {
	Timeout    time.Duration
	TraceID    string
	WithTrace  bool
	WithCancel bool
}

// DefaultOptions 返回默认选项配置
func DefaultOptions() *ContextOptions {
	return &ContextOptions{
		Timeout:    10 * time.Second,
		WithTrace:  true,
		WithCancel: true,
	}
}

// New 创建一个新的上下文，可选超时和追踪功能
func New(parentCtx context.Context, opts *ContextOptions) (context.Context, context.CancelFunc) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var (
		ctx                       = parentCtx
		cancel context.CancelFunc = func() {} // 默认空函数
	)

	// 添加超时
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	} else if opts.WithCancel {
		ctx, cancel = context.WithCancel(ctx)
	}

	// 添加追踪ID
	if opts.WithTrace {
		traceID := opts.TraceID
		if traceID == "" {
			traceID = uuid.NewString()
		}
		ctx = logx.ContextWithFields(ctx, logx.Field(CtxTraceID, traceID))
	}

	return ctx, cancel
}

// Background 从空上下文创建一个新的上下文
func Background(opts *ContextOptions) (context.Context, context.CancelFunc) {
	return New(context.Background(), opts)
}

// FromRequest 从请求上下文创建一个新的分离上下文
// 适用于后台任务，不受原始请求上下文取消影响
func FromRequest(reqCtx context.Context, opts *ContextOptions) (context.Context, context.CancelFunc) {
	// 从请求上下文复制值但基于新的Background
	return New(context.Background(), opts)
}
