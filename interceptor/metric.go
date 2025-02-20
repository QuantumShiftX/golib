package interceptor

import (
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/status"
)

var (
	// 请求总数计数器
	requestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "rpc",
			Subsystem: "requests",
			Name:      "total",
			Help:      "RPC请求总数",
		},
		[]string{"method"},
	)

	// 请求持续时间直方图
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "rpc",
			Subsystem: "requests",
			Name:      "duration_ms",
			Help:      "RPC请求处理时间（毫秒）",
			Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"method"},
	)

	// 错误请求计数器
	requestError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "rpc",
			Subsystem: "requests",
			Name:      "error_total",
			Help:      "RPC请求错误总数",
		},
		[]string{"method", "code"},
	)
)

// InitMetrics 初始化所有指标
func InitMetrics() {
	// 注册所有指标
	prometheus.MustRegister(requestTotal)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(requestError)
}

// 指标收集函数
func recordMetrics(method string, duration int64, err error) {
	// 增加请求计数
	requestTotal.WithLabelValues(method).Inc()

	// 记录请求持续时间
	requestDuration.WithLabelValues(method).Observe(float64(duration))

	// 如果有错误，记录错误计数
	if err != nil {
		code := status.Code(err).String()
		requestError.WithLabelValues(method, code).Inc()
	}
}
