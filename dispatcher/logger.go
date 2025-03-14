package dispatcher

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
)

// LogxAdapter 适配 go-zero 的 logx 到 asynq 需要的日志接口
type LogxAdapter struct{}

// NewLogxAdapter 创建一个新的日志适配器
func NewLogxAdapter() *LogxAdapter {
	return &LogxAdapter{}
}

// Debug 实现 asynq 日志接口的 Debug 方法
func (l *LogxAdapter) Debug(args ...interface{}) {
	logx.Debug(fmt.Sprint(args...))
}

// Info 实现 asynq 日志接口的 Info 方法
func (l *LogxAdapter) Info(args ...interface{}) {
	logx.Info(fmt.Sprint(args...))
}

// Warn 实现 asynq 日志接口的 Warn 方法
func (l *LogxAdapter) Warn(args ...interface{}) {
	logx.Debug(fmt.Sprint(args...))
}

// Error 实现 asynq 日志接口的 Error 方法
func (l *LogxAdapter) Error(args ...interface{}) {
	logx.Error(fmt.Sprint(args...))
}

// Fatal 实现 asynq 日志接口的 Fatal 方法
func (l *LogxAdapter) Fatal(args ...interface{}) {
	logx.Error(fmt.Sprint(args...)) // 使用 Error 而不是 Fatal 以避免程序退出
}
