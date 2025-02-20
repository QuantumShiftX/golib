package interceptor

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/QuantumShiftX/golib/metadata"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcMeta "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RequestInfoInterceptor RPC请求信息中间件，提取客户端详细信息
func RequestInfoInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {

	md, ok := grpcMeta.FromIncomingContext(ctx)
	if !ok {
		md = grpcMeta.MD{}
	}

	// 生成或获取追踪ID和请求ID
	traceID := getFirstMetadataValue(md, metadata.HeaderTraceID)
	if traceID == "" {
		traceID = uuid.New().String()
	}

	requestID := getFirstMetadataValue(md, metadata.HeaderRequestID)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// 构建完整的客户端信息
	clientInfo := metadata.RequestClientInfo{
		IP:          getClientIP(ctx, md),
		UserAgent:   getFirstMetadataValue(md, metadata.HeaderUserAgent),
		Platform:    getFirstMetadataValue(md, metadata.HeaderPlatform),
		OS:          getFirstMetadataValue(md, metadata.HeaderOS),
		Browser:     getFirstMetadataValue(md, metadata.HeaderBrowser),
		BrowserVer:  extractBrowserVersion(getFirstMetadataValue(md, metadata.HeaderBrowser)),
		IsMobile:    getFirstMetadataValue(md, metadata.HeaderMobile) == "true",
		DeviceID:    getFirstMetadataValue(md, metadata.HeaderDeviceID),
		DeviceType:  getFirstMetadataValue(md, metadata.HeaderDeviceType),
		ScreenSize:  getFirstMetadataValue(md, metadata.HeaderScreenSize),
		Language:    getFirstMetadataValue(md, metadata.HeaderLanguage),
		Timezone:    getFirstMetadataValue(md, metadata.HeaderTimezone),
		Referrer:    getFirstMetadataValue(md, metadata.HeaderReferrer),
		RequestTime: time.Now(),
	}

	// 将信息添加到上下文
	newCtx := context.Background()
	newCtx = metadata.WithMetadata(newCtx, metadata.CtxRequestClientInfo, clientInfo)
	newCtx = metadata.WithMetadata(newCtx, metadata.CtxTraceID, traceID)
	newCtx = metadata.WithMetadata(newCtx, metadata.CtxRequestID, requestID)
	newCtx = metadata.WithMetadata(newCtx, metadata.CtxIp, clientInfo.IP)
	newCtx = metadata.WithMetadata(newCtx, metadata.CtxDeviceID, clientInfo.DeviceID)
	newCtx = metadata.WithMetadata(newCtx, metadata.CtxDeviceType, clientInfo.DeviceType)

	// 添加区域信息
	region := getFirstMetadataValue(md, metadata.HeaderRegion)
	if region != "" {
		newCtx = metadata.WithMetadata(newCtx, metadata.CtxRegion, region)
	}

	// 添加浏览器指纹
	fingerprint := getFirstMetadataValue(md, metadata.HeaderBrowserFingerprint)
	if fingerprint != "" {
		newCtx = metadata.WithMetadata(newCtx, metadata.CtxBrowserFingerprint, fingerprint)
	}

	// 打印请求日志（根据方法决定日志级别）
	if shouldLogDetailed(info.FullMethod) {
		logx.WithContext(newCtx).Infof("[%s] RPC请求: 方法=%s, 客户端=%s, TraceID=%s",
			requestID, info.FullMethod, clientInfo.IP, traceID)
	} else {
		logx.WithContext(newCtx).Debugf("[%s] RPC请求: 方法=%s, TraceID=%s",
			requestID, info.FullMethod, traceID)
	}

	// 继续处理请求
	return handler(newCtx, req)
}

// RecoveryInterceptor 防止RPC服务因panic而崩溃的拦截器
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {

	defer func() {
		if r := recover(); r != nil {
			traceID := metadata.GetTraceIDFromCtx(ctx)
			stackTrace := string(debug.Stack())

			// 记录错误日志
			logx.WithContext(ctx).Errorf("[PANIC RECOVERED] TraceID=%s, Method=%s\nError: %v\nStack: %s",
				traceID, info.FullMethod, r, stackTrace)

			// 返回一个友好的错误信息给客户端
			err = status.Errorf(codes.Internal,
				"服务器内部错误，请稍后重试 (TraceID: %s)", traceID)
		}
	}()

	return handler(ctx, req)
}

// MetricsInterceptor 收集RPC请求指标的拦截器
func MetricsInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {

	startTime := time.Now()

	// 提取method名称，去掉包路径
	methodName := extractMethodName(info.FullMethod)

	// 处理请求并收集结果
	resp, err = handler(ctx, req)

	// 计算持续时间
	duration := time.Since(startTime).Milliseconds()

	// 记录指标
	recordMetrics(methodName, duration, err)

	return resp, err
}

// AuthInterceptor 简化版认证拦截器 - 只传递认证信息
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {

	md, ok := grpcMeta.FromIncomingContext(ctx)
	if !ok {
		md = grpcMeta.MD{}
	}

	// 获取认证Token并添加到上下文
	authToken := getAuthToken(md)
	newCtx := metadata.WithMetadata(ctx, metadata.CtxToken, authToken)

	return handler(newCtx, req)
}

// RateLimitInterceptor 限流拦截器
func RateLimitInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {

	clientIP := getClientIP(ctx, nil)
	userID := metadata.GetUidFromCtx(ctx)

	// 根据IP或用户ID进行限流检查
	identifier := clientIP
	if userID > 0 {
		identifier = fmt.Sprintf("user:%d", userID)
	}

	// TODO: 实现实际的限流逻辑，这里仅为示例
	if !checkRateLimit(identifier, info.FullMethod) {
		return nil, status.Error(codes.ResourceExhausted,
			"请求频率过高，请稍后再试")
	}

	return handler(ctx, req)
}

// LoggingInterceptor 详细日志拦截器
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {

	startTime := time.Now()
	traceID := metadata.GetTraceIDFromCtx(ctx)
	userID := metadata.GetUidFromCtx(ctx)

	// 记录请求开始
	if shouldLogDetailed(info.FullMethod) {
		logx.WithContext(ctx).Infof("[%s] 开始处理RPC请求: 方法=%s, 用户ID=%d, 请求=%+v",
			traceID, info.FullMethod, userID, req)
	}

	// 处理请求
	resp, err = handler(ctx, req)

	// 记录请求结果
	duration := time.Since(startTime)
	if err != nil {
		logx.WithContext(ctx).Errorf("[%s] RPC请求失败: 方法=%s, 用户ID=%d, 耗时=%v, 错误=%v",
			traceID, info.FullMethod, userID, duration, err)
	} else if shouldLogDetailed(info.FullMethod) {
		logx.WithContext(ctx).Infof("[%s] RPC请求完成: 方法=%s, 用户ID=%d, 耗时=%v",
			traceID, info.FullMethod, userID, duration)
	}

	return resp, err
}

// 辅助函数：从元数据中获取第一个值
func getFirstMetadataValue(md grpcMeta.MD, key string) string {
	if values := md.Get(key); len(values) > 0 {
		return values[0]
	}
	return ""
}

// 辅助函数：获取客户端IP
func getClientIP(ctx context.Context, md grpcMeta.MD) string {
	// 尝试从peer信息中获取
	if p, ok := peer.FromContext(ctx); ok {
		if addr := p.Addr.String(); addr != "" {
			// 去除端口号部分
			if idx := strings.LastIndex(addr, ":"); idx > 0 {
				return addr[:idx]
			}
			return addr
		}
	}

	// 尝试从元数据头中获取
	if md == nil {
		var ok bool
		md, ok = grpcMeta.FromIncomingContext(ctx)
		if !ok {
			return ""
		}
	}

	// 按优先级检查不同的IP头
	for _, header := range []string{
		metadata.HeaderRealIP,
		metadata.HeaderForwardedFor,
		metadata.HeaderOriginalForwardedFor,
		metadata.HeaderClientIP,
		metadata.HeaderCFConnectingIP, // Cloudflare
	} {
		if ips := md.Get(header); len(ips) > 0 && ips[0] != "" {
			// 如果是x-forwarded-for，可能包含多个IP，取第一个
			if header == metadata.HeaderForwardedFor {
				parts := strings.Split(ips[0], ",")
				return strings.TrimSpace(parts[0])
			}
			return ips[0]
		}
	}

	return ""
}

// 辅助函数：提取浏览器版本
func extractBrowserVersion(browser string) string {
	if browser == "" {
		return ""
	}

	parts := strings.Split(browser, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// 辅助函数：提取方法名
func extractMethodName(fullMethod string) string {
	if idx := strings.LastIndex(fullMethod, "/"); idx >= 0 {
		return fullMethod[idx+1:]
	}
	return fullMethod
}

// 辅助函数：检查是否需要详细日志
func shouldLogDetailed(fullMethod string) bool {
	if strings.Contains(fullMethod, "Health") ||
		strings.Contains(fullMethod, "Ping") ||
		strings.Contains(fullMethod, "Metrics") {
		return false
	}
	return true
}

// 辅助函数：从元数据中获取认证Token
func getAuthToken(md grpcMeta.MD) string {
	if values := md.Get(metadata.HeaderAuthorization); len(values) > 0 {
		auth := values[0]
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}

	if values := md.Get(metadata.HeaderToken); len(values) > 0 {
		return values[0]
	}

	return ""
}

// 辅助函数：检查限流
func checkRateLimit(identifier string, method string) bool {
	// TODO: 实现实际的限流逻辑，这里仅为示例
	return true
}

// 创建默认拦截器链
func CreateDefaultInterceptorChain() grpc.UnaryServerInterceptor {
	return ChainUnaryInterceptors(
		RecoveryInterceptor,    // 首先恢复panic
		RequestInfoInterceptor, // 提取请求信息
		AuthInterceptor,        // 认证信息传递
		RateLimitInterceptor,   // 限流
		MetricsInterceptor,     // 指标收集
		LoggingInterceptor,     // 详细日志记录（最后执行）
	)
}

// ChainUnaryInterceptors 链式组合多个拦截器
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return current(currentCtx, currentReq, info, next)
			}
		}

		chain := handler
		// 反向遍历以保持正确的执行顺序
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(ctx, req)
	}
}
