package middleware

import (
	"fmt"
	"github.com/QuantumShiftX/golib/config"
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware 日志中间件
func LoggingMiddleware(cfg *config.LoggingConfig) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 创建响应记录器
			recorder := NewResponseRecorder(w)

			// 记录请求开始
			if cfg != nil && cfg.EnableTrace {
				logRequest(r, cfg)
			}

			// 执行下一个处理器
			next.ServeHTTP(recorder, r)

			// 写入真实响应
			writeRecordedResponse(recorder, w)

			// 记录请求完成
			duration := time.Since(start)
			logResponse(r, recorder, duration, cfg)

			// 记录指标
			if cfg != nil && cfg.EnableMetrics {
				recordMetrics(r, recorder, duration)
			}
		})
	}
}

// logRequest 记录请求信息
func logRequest(r *http.Request, cfg *config.LoggingConfig) {
	if cfg.Format == "json" {
		log.Printf(`{"level":"info","msg":"request started","method":"%s","path":"%s","remote_addr":"%s"}`,
			r.Method, r.URL.Path, r.RemoteAddr)
	} else {
		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	}
}

// logResponse 记录响应信息
func logResponse(r *http.Request, recorder *ResponseRecorder, duration time.Duration, cfg *config.LoggingConfig) {
	if cfg == nil {
		log.Printf("Completed %s %s - %d in %v", r.Method, r.URL.Path, recorder.Status(), duration)
		return
	}

	if cfg.Format == "json" {
		log.Printf(`{"level":"info","msg":"request completed","method":"%s","path":"%s","status":%d,"duration":"%v","size":%d}`,
			r.Method, r.URL.Path, recorder.Status(), duration, recorder.Size())
	} else {
		log.Printf("Completed %s %s - %d in %v (%d bytes)",
			r.Method, r.URL.Path, recorder.Status(), duration, recorder.Size())
	}
}

// writeRecordedResponse 写入记录的响应
func writeRecordedResponse(recorder *ResponseRecorder, w http.ResponseWriter) {
	// 复制头部
	for k, v := range recorder.Header() {
		w.Header()[k] = v
	}

	// 写入状态码和响应体
	w.WriteHeader(recorder.Status())
	w.Write(recorder.Body().Bytes())
}

// recordMetrics 记录指标
func recordMetrics(r *http.Request, recorder *ResponseRecorder, duration time.Duration) {
	// 这里可以集成到指标系统，如Prometheus
	fmt.Printf("[Metrics] %s %s - %d - %v - %d bytes\n",
		r.Method, r.URL.Path, recorder.Status(), duration, recorder.Size())
}
