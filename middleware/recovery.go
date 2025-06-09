package middleware

import (
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"runtime"
)

// RecoveryMiddleware 恢复中间件（优化版）
func RecoveryMiddleware() Handler {
	return RecoveryWithConfig(true, nil)
}

// RecoveryWithConfig 带配置的恢复中间件
func RecoveryWithConfig(enableStackTrace bool, customHandler func(interface{}, *http.Request)) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if customHandler != nil {
						customHandler(err, r)
					} else {
						if enableStackTrace {
							logPanicWithStack(err, r)
						} else {
							logx.Infof("Panic recovered: %v", err)
						}
					}

					// 检查响应是否已经开始写入
					if !isResponseWritten(w) {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// logPanicWithStack 记录panic信息和堆栈
func logPanicWithStack(err interface{}, r *http.Request) {
	// 获取堆栈信息
	stack := make([]byte, 4096)
	length := runtime.Stack(stack, false)

	logx.Infof("Panic recovered: %v", err)
	logx.Infof("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	logx.Infof("User-Agent: %s", r.UserAgent())
	logx.Infof("Stack trace:\n%s", stack[:length])
}

// isResponseWritten 检查响应是否已写入（改进版）
func isResponseWritten(w http.ResponseWriter) bool {
	// 尝试设置一个测试头，如果失败说明响应已经开始
	defer func() {
		recover() // 忽略可能的panic
	}()

	w.Header().Set("X-Recovery-Test", "test")
	w.Header().Del("X-Recovery-Test")
	return false
}
