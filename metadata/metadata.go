package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"net"
	"time"
)

// WithMetadata 向上下文添加数据
func WithMetadata(ctx context.Context, key, val any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, key, val)
}

// WithMultiMetadata 向上下文批量添加键值对
func WithMultiMetadata(ctx context.Context, keyVals map[string]interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	for k, v := range keyVals {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

// WithUserInfo 向上下文添加用户信息
func WithUserInfo(ctx context.Context, userId int64, username string) context.Context {
	ctx = WithMetadata(ctx, CtxJWTUserId, userId)
	return WithMetadata(ctx, CtxJWTUsername, username)
}

// WithRequestInfo 向上下文添加请求信息
func WithRequestInfo(ctx context.Context, ip, deviceID, deviceType string) context.Context {
	ctx = WithMetadata(ctx, CtxIp, ip)
	ctx = WithMetadata(ctx, CtxDeviceID, deviceID)
	ctx = WithMetadata(ctx, CtxDeviceType, deviceType)
	ctx = WithMetadata(ctx, CtxRequestTime, time.Now())
	return ctx
}

// GetMetadataFromCtx 从上下文获取数据
func GetMetadataFromCtx(ctx context.Context, key any) any {
	if ctx == nil {
		return nil
	}
	return ctx.Value(key)
}

// GetMetadata 类型安全的上下文取值
func GetMetadata[T any](ctx context.Context, key any) (T, bool) {
	if ctx == nil {
		var zero T
		return zero, false
	}

	if val, ok := ctx.Value(key).(T); ok {
		return val, true
	}
	var zero T
	return zero, false
}

// GetMetadataOrDefault 获取值或返回默认值
func GetMetadataOrDefault[T any](ctx context.Context, key any, defaultVal T) T {
	if val, ok := GetMetadata[T](ctx, key); ok {
		return val
	}
	return defaultVal
}

// GetUidFromCtx 从上下文中获取uid (增强版)
func GetUidFromCtx(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}

	val := ctx.Value(CtxJWTUserId)
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		uid, err := v.Int64()
		if err != nil {
			logx.Errorf("Failed to convert uid to int64: %v", err)
			return 0
		}
		return uid
	case string:
		return cast.ToInt64(v)
	default:
		return cast.ToInt64(val)
	}
}

// GetUsernameFromCtx 从上下文中获取username
func GetUsernameFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return cast.ToString(ctx.Value(CtxJWTUsername))
}

// GetCurrencyCodeFromCtx 从上下文中获取currency_code
func GetCurrencyCodeFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return cast.ToString(ctx.Value(CtxCurrencyCode))
}

// GetIpFromCtx 从上下文中获取ip (增强版)
func GetIpFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if val := ctx.Value(CtxIp); val != nil {
		switch v := val.(type) {
		case string:
			return v
		case net.IP:
			return v.String()
		case *net.IP:
			if v != nil {
				return v.String()
			}
		default:
			if s, ok := val.(fmt.Stringer); ok {
				return s.String()
			}
		}
	}
	return ""
}

// GetDomainFromCtx 从上下文中获取域名
func GetDomainFromCtx(ctx context.Context) string {
	return GetMetadataOrDefault(ctx, CtxDomain, "")
}

// GetDeviceIDFromCtx 从上下文中获取设备id
func GetDeviceIDFromCtx(ctx context.Context) string {
	return GetMetadataOrDefault(ctx, CtxDeviceID, "")
}

// GetDeviceTypeFromCtx 从上下文中获取设备类型
func GetDeviceTypeFromCtx(ctx context.Context) string {
	return GetMetadataOrDefault(ctx, CtxDeviceType, "")
}

// GetBrowserFingerprintFromCtx 从上下文中获取浏览器指纹
func GetBrowserFingerprintFromCtx(ctx context.Context) string {
	return GetMetadataOrDefault(ctx, CtxBrowserFingerprint, "")
}

// GetRegionFromCtx 从上下文中获取区域
func GetRegionFromCtx(ctx context.Context) string {
	return GetMetadataOrDefault(ctx, CtxRegion, "")
}

// GetUserRoleFromCtx 从上下文中获取用户角色
func GetUserRoleFromCtx(ctx context.Context) string {
	return GetMetadataOrDefault(ctx, CtxUserRole, "")
}

// GetUserPermissionsFromCtx 从上下文中获取用户权限
func GetUserPermissionsFromCtx(ctx context.Context) []string {
	perms, ok := GetMetadata[[]string](ctx, CtxUserPermissions)
	if !ok {
		return []string{}
	}
	return perms
}

// GetUserAgentIdFromCtx 从上下文中获取代理ID
func GetUserAgentIdFromCtx(ctx context.Context) int64 {
	return GetMetadataOrDefault(ctx, CtxUserAgentId, int64(0))
}

// GetParentAgentIdFromCtx 从上下文中获取上级代理ID
func GetParentAgentIdFromCtx(ctx context.Context) int64 {
	return GetMetadataOrDefault(ctx, CtxUserParentAgentId, int64(0))
}

// GetTraceIDFromCtx 从上下文中获取追踪ID
func GetTraceIDFromCtx(ctx context.Context) string {
	return GetMetadataOrDefault(ctx, CtxTraceID, "")
}

// GetRequestTimeFromCtx 从上下文中获取请求时间
func GetRequestTimeFromCtx(ctx context.Context) time.Time {
	t, ok := GetMetadata[time.Time](ctx, CtxRequestTime)
	if !ok {
		return time.Time{}
	}
	return t
}

// IsAuthenticated 检查用户是否已认证
func IsAuthenticated(ctx context.Context) bool {
	return GetMetadataOrDefault(ctx, CtxIsAuthenticated, false)
}

// RequestClientInfo 客户端信息结构体
type RequestClientInfo struct {
	IP          string // 客户端IP地址
	Platform    string // 平台：Windows、Linux等
	OS          string // 操作系统
	Browser     string // 浏览器信息
	BrowserVer  string // 浏览器版本
	IsMobile    bool   // 是否是手机端
	UserAgent   string // 完整的User-Agent
	DeviceID    string // 设备ID
	DeviceType  string // 设备类型
	AppVersion  string // 应用版本 (如果是App)
	ScreenSize  string // 屏幕尺寸
	Language    string // 语言
	Timezone    string // 时区
	Referrer    string // 引荐来源
	RequestTime int64  // 请求时间 毫秒
}

// GetRequestClientInfoFromCtx 从上下文中获取RequestClientInfo
func GetRequestClientInfoFromCtx(ctx context.Context) *RequestClientInfo {
	if ctx == nil {
		return nil
	}

	val := ctx.Value(CtxRequestClientInfo)
	if val == nil {
		logx.Debugf("get request client info empty")
		return nil
	}

	switch info := val.(type) {
	case RequestClientInfo:
		return &info
	case *RequestClientInfo:
		return info
	default:
		logx.Errorf("invalid request client info type: %T, value: %+v", val, val)
		return nil
	}
}

// UpdateRequestClientInfo 更新上下文中的客户端信息
func UpdateRequestClientInfo(ctx context.Context, updater func(*RequestClientInfo)) context.Context {
	if ctx == nil {
		return nil
	}

	info := GetRequestClientInfoFromCtx(ctx)
	if info == nil {
		info = &RequestClientInfo{}
	}

	updater(info)
	return WithMetadata(ctx, CtxRequestClientInfo, *info)
}

// CreateClientInfoFromHeaders 从HTTP头创建客户端信息
func CreateClientInfoFromHeaders(headers map[string][]string) *RequestClientInfo {
	info := &RequestClientInfo{
		IP:          getFirstHeaderValue(headers, "X-Real-IP", "X-Forwarded-For"),
		UserAgent:   getFirstHeaderValue(headers, HeaderUserAgent),
		DeviceID:    getFirstHeaderValue(headers, HeaderDeviceID),
		DeviceType:  getFirstHeaderValue(headers, HeaderDeviceType),
		IsMobile:    getFirstHeaderValue(headers, HeaderMobile) == "true",
		Browser:     getFirstHeaderValue(headers, HeaderBrowser),
		Language:    getFirstHeaderValue(headers, HeaderLanguage),
		Timezone:    getFirstHeaderValue(headers, HeaderTimezone),
		RequestTime: time.Now().UnixMilli(),
	}
	return info
}

// 辅助函数: 获取第一个非空头部值
func getFirstHeaderValue(headers map[string][]string, keys ...string) string {
	for _, key := range keys {
		if values, exists := headers[key]; exists && len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	return ""
}

// ExportMetadataToMap 将上下文中的元数据导出到map
func ExportMetadataToMap(ctx context.Context, keys []string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range keys {
		if val := ctx.Value(key); val != nil {
			result[key] = val
		}
	}
	return result
}

// WithTracing 添加追踪信息到上下文
func WithTracing(ctx context.Context, traceID, requestID string) context.Context {
	ctx = WithMetadata(ctx, CtxTraceID, traceID)
	return WithMetadata(ctx, CtxRequestID, requestID)
}

// HasAnyRole 检查用户是否拥有指定角色之一
func HasAnyRole(ctx context.Context, roles ...string) bool {
	userRole := GetUserRoleFromCtx(ctx)
	if userRole == "" {
		return false
	}

	for _, role := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// HasPermission 检查用户是否拥有指定权限
func HasPermission(ctx context.Context, permission string) bool {
	permissions := GetUserPermissionsFromCtx(ctx)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
