package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"net"
)

const (
	// CtxJWTUserId   用户id
	CtxJWTUserId = "uid"
	// CtxJWTUsername 用户名
	CtxJWTUsername = "username"
	// CtxIp          ip
	CtxIp = "ip"
	// CtxDomain      域名
	CtxDomain = "domain"
	// CtxRegion       区域
	CtxRegion = "region"
	// CtxDeviceID     设备id
	CtxDeviceID = "device_id"
	// CtxDeviceType  设备类型
	CtxDeviceType = "device_type"
	// CtxBrowserFingerprint 浏览器指纹
	CtxBrowserFingerprint = "browser_fingerprint"
	// CtxCurrencyCode 币种code
	CtxCurrencyCode = "currency_code"
	// CtxRequestClientInfo 请求客户端信息
	CtxRequestClientInfo = "request_client_info"
)

const (
	// RegionKey 地区
	RegionKey = "X-Region"
	// DeviceIDKey 设备id
	DeviceIDKey = "X-Device-ID"
	// DeviceTypeKey 设备类型
	DeviceTypeKey = "X-Device-Type"
)

// WithMetadata 上下文数据
func WithMetadata(ctx context.Context, key, val any) context.Context {
	return context.WithValue(ctx, key, val)
}

// GetMetadataFromCtx 获取上下文数据
func GetMetadataFromCtx(ctx context.Context, key any) any {
	return ctx.Value(key)
}

// GetMetadata 上下文取值
func GetMetadata[T any](ctx context.Context, key any) (T, bool) {
	if val, ok := ctx.Value(key).(T); ok {
		return val, true
	}
	var zero T
	return zero, false
}

// GetUidFromCtx 从上下文中获取uid
func GetUidFromCtx(ctx context.Context) int64 {
	val := ctx.Value(CtxJWTUserId)
	if val == nil {
		return 0
	}
	uidNum, ok := val.(json.Number)
	if !ok {
		return cast.ToInt64(uidNum)
	}
	uid, _ := uidNum.Int64()
	return uid
}

// GetUsernameFromCtx 从上下文中获取username
func GetUsernameFromCtx(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxJWTUsername))
}

// GetCurrencyCodeFromCtx 从上下文中获取currency_code
func GetCurrencyCodeFromCtx(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxCurrencyCode))
}

// GetIpFromCtx 从上下文中获取ip
func GetIpFromCtx(ctx context.Context) string {
	if val := ctx.Value(CtxIp); val != nil {
		switch v := val.(type) {
		case string:
			return v
		case net.IP:
			return v.String()
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
	if domain, ok := GetMetadata[string](ctx, CtxDomain); ok {
		return domain
	}
	return ""
}

// GetDeviceIDFromCtx 从上下文中获取设备id
func GetDeviceIDFromCtx(ctx context.Context) string {
	if deviceID, ok := GetMetadata[string](ctx, CtxDeviceID); ok {
		return deviceID
	}
	return ""
}

// GetDeviceTypeFromCtx 从上下文中获取设备类型
func GetDeviceTypeFromCtx(ctx context.Context) string {
	if deviceType, ok := GetMetadata[string](ctx, CtxDeviceType); ok {
		return deviceType
	}
	return ""
}

// GetBrowserFingerprintFromCtx 从上下文中获取浏览器指纹
func GetBrowserFingerprintFromCtx(ctx context.Context) string {
	if browserFingerprint, ok := GetMetadata[string](ctx, CtxBrowserFingerprint); ok {
		return browserFingerprint
	}
	return ""
}

// GetRegionFromCtx 从上下文中获取区域
func GetRegionFromCtx(ctx context.Context) string {
	if region, ok := GetMetadata[string](ctx, CtxRegion); ok {
		return region
	}
	return ""
}

type RequestClientInfo struct {
	IP       string // 客户端IP地址
	Platform string // 平台：Windows、Linux等
	OS       string // 操作系统
	Browser  string // 浏览器信息
	IsMobile bool   // 是否是手机端
}

// GetRequestClientInfoFromCtx 从上下文中获取RequestClientInfo
func GetRequestClientInfoFromCtx(ctx context.Context) *RequestClientInfo {
	val := ctx.Value(CtxRequestClientInfo)
	if val == nil {
		logx.Errorf("get request client info empty")
		return nil
	}

	info, ok := val.(RequestClientInfo)
	if !ok {
		logx.Errorf("invalid request client info: %+v", val)
		return nil
	}
	return &info
}
