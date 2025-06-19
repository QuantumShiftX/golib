package middleware

import (
	"bytes"
	"github.com/QuantumShiftX/golib/config"
	"net/http"
)

// Handler 中间件处理器类型
type Handler func(http.Handler) http.Handler

// Chain 中间件链
type Chain struct {
	middlewares []Handler
}

// NewChain 创建中间件链
func NewChain(middlewares ...Handler) *Chain {
	return &Chain{
		middlewares: append([]Handler(nil), middlewares...),
	}
}

// Append 添加中间件
func (c *Chain) Append(middlewares ...Handler) *Chain {
	newMiddlewares := make([]Handler, len(c.middlewares)+len(middlewares))
	copy(newMiddlewares, c.middlewares)
	copy(newMiddlewares[len(c.middlewares):], middlewares)

	return &Chain{
		middlewares: newMiddlewares,
	}
}

// Extend 扩展中间件链
func (c *Chain) Extend(chain *Chain) *Chain {
	return c.Append(chain.middlewares...)
}

// Then 应用中间件链到最终处理器
func (c *Chain) Then(handler http.Handler) http.Handler {
	if handler == nil {
		handler = http.DefaultServeMux
	}

	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}

	return handler
}

// ThenFunc 应用中间件链到处理器函数
func (c *Chain) ThenFunc(handlerFunc http.HandlerFunc) http.Handler {
	return c.Then(handlerFunc)
}

// ResponseRecorder 响应记录器
type ResponseRecorder struct {
	http.ResponseWriter
	body        *bytes.Buffer
	status      int
	header      http.Header
	size        int
	wroteHeader bool
}

// NewResponseRecorder 创建响应记录器
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		header:         make(http.Header),
		status:         http.StatusOK,
	}
}

func (r *ResponseRecorder) Header() http.Header {
	return r.header
}

func (r *ResponseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	size, err := r.body.Write(data)
	r.size += size
	return size, err
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}
	r.status = statusCode
	r.wroteHeader = true
}

// Size 返回响应大小
func (r *ResponseRecorder) Size() int {
	return r.size
}

// Status 返回状态码
func (r *ResponseRecorder) Status() int {
	return r.status
}

// Body 返回响应体
func (r *ResponseRecorder) Body() *bytes.Buffer {
	return r.body
}

// IsWritten 检查是否已写入响应
func (r *ResponseRecorder) IsWritten() bool {
	return r.wroteHeader
}

// CreateMiddlewareChain 创建标准中间件链
func CreateMiddlewareChain(cfg *config.GlobalConfig) *Chain {
	chain := NewChain()

	//// 恢复中间件（最外层）
	//if cfg.Middleware != nil && cfg.Middleware.EnableRecovery {
	//	chain = chain.Append(RecoveryMiddleware())
	//}
	//
	//// 日志中间件
	//if cfg.Middleware != nil && cfg.Middleware.EnableLogging {
	//	chain = chain.Append(LoggingMiddleware(cfg.Middleware.Logging))
	//}
	//
	//// CORS中间件
	//if cfg.Middleware != nil && cfg.Middleware.EnableCORS {
	//	chain = chain.Append(CORSMiddleware(cfg.Middleware.CORS))
	//}

	// 加密中间件（最内层）
	if cfg.Crypto != nil && cfg.Crypto.Enable {
		chain = chain.Append(CryptoMiddleware(cfg.Crypto))
	}

	return chain
}
