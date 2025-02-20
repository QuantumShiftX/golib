package metadata

// Header keys
const (
	// Standard headers (标准 HTTP 头,使用大写)
	HeaderAuthorization  = "Authorization"
	HeaderUserAgent      = "User-Agent"
	HeaderAcceptLanguage = "Accept-Language"
	HeaderReferrer       = "Referer"
	HeaderForwardedFor   = "X-Forwarded-For"

	// Custom headers (自定义头,使用 x- 前缀和小写)
	HeaderPlatform           = "x-platform"
	HeaderOS                 = "x-os"
	HeaderBrowser            = "x-browser"
	HeaderMobile             = "x-mobile"
	HeaderDeviceID           = "x-device-id"
	HeaderDeviceType         = "x-device-type"
	HeaderBrowserFingerprint = "x-browser-fingerprint"
	HeaderRegion             = "x-region"
	HeaderLanguage           = "x-language"
	HeaderTimezone           = "x-timezone"
	HeaderTraceID            = "x-trace-id"
	HeaderRequestID          = "x-request-id"
	HeaderScreenSize         = "x-screen-size"
	HeaderRealIP             = "x-real-ip"

	HeaderOriginalForwardedFor = "x-original-forwarded-for"
	HeaderClientIP             = "x-client-ip"
	HeaderCFConnectingIP       = "x-cf-connecting-ip"
	HeaderToken                = "x-token"
)

// Context keys
const (
	// User related
	CtxJWTUserId         = "uid"              // 用户id
	CtxJWTUsername       = "username"         // 用户名
	CtxUserRole          = "user_role"        // 用户角色
	CtxUserPermissions   = "user_permissions" // 用户权限
	CtxUserStatus        = "user_status"      // 用户状态
	CtxUserLastLoginTime = "last_login_time"  // 最后登录时间
	CtxUserAgentId       = "agent_id"         // 代理ID
	CtxUserParentAgentId = "parent_agent_id"  // 上级代理ID

	// Request related
	CtxIp                 = "ip"                  // ip
	CtxDomain             = "domain"              // 域名
	CtxRegion             = "region"              // 区域
	CtxDeviceID           = "device_id"           // 设备id
	CtxDeviceType         = "device_type"         // 设备类型
	CtxBrowserFingerprint = "browser_fingerprint" // 浏览器指纹
	CtxCurrencyCode       = "currency_code"       // 币种code
	CtxRequestClientInfo  = "request_client_info" // 请求客户端信息
	CtxLanguage           = "language"            // 语言
	CtxTimezone           = "timezone"            // 时区
	CtxSessionID          = "session_id"          // 会话ID
	CtxTraceID            = "trace_id"            // 追踪ID
	CtxRequestID          = "request_id"          // 请求ID
	CtxRequestTime        = "request_time"        // 请求时间

	// Auth related
	CtxIsAuthenticated = "is_authenticated" // 是否已认证
	CtxAuthType        = "auth_type"        // 认证类型 (JWT, OAuth, etc.)
	CtxToken           = "token"            // Token
	CtxTokenExpiry     = "token_expiry"     // Token过期时间
	CtxIssuer          = "issuer"           // Token颁发者
)

// Metrics keys
const (
	MetricKeyRequestDuration = "rpc_request_duration_ms" // 请求耗时
	MetricKeyRequestTotal    = "rpc_request_total"       // 请求总数
	MetricKeyRequestError    = "rpc_request_error_total" // 错误请求数
)
