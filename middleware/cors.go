package middleware

import (
	"github.com/QuantumShiftX/golib/config"
	"github.com/QuantumShiftX/golib/metadata"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"strconv"
	"strings"
)

// CORSMiddleware CORS中间件（完全修复版）
func CORSMiddleware(cfg *config.CORSConfig) Handler {
	// 使用默认配置如果没有提供
	if cfg == nil {
		cfg = config.DefaultMiddlewareConfig().CORS
	}

	// 预处理配置以提高性能
	originChecker := newOriginChecker(cfg.AllowOrigins, cfg.AllowWildcard)
	allowMethodsStr := strings.Join(cfg.AllowMethods, ", ")
	allowHeadersStr := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeadersStr := strings.Join(cfg.ExposeHeaders, ", ")
	maxAgeStr := strconv.Itoa(cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if cfg.Debug {
				logx.Infof("[CORS] Method=%s, Origin=%s, Path=%s", r.Method, origin, r.URL.Path)
				logx.Infof("[CORS] Request Headers: %v", r.Header)
			}

			// 【关键修复】：总是设置CORS头部，无论什么情况
			setCORSHeaders(w, r, cfg, origin, originChecker,
				allowMethodsStr, allowHeadersStr, exposeHeadersStr, maxAgeStr)

			// 处理预检请求
			if r.Method == http.MethodOptions {
				if cfg.Debug {
					logx.Infof("[CORS] Handling preflight request for %s, Origin: %s", r.URL.Path, origin)
					logx.Infof("[CORS] Response Headers: %v", w.Header())
				}

				// 返回成功状态码
				w.WriteHeader(http.StatusNoContent) // 204
				return
			}

			// 设置元数据（保持原有功能）
			ctx := r.Context()
			ctx = metadata.WithMetadata(ctx, metadata.CtxIp, httpx.GetRemoteAddr(r))
			ctx = metadata.WithMetadata(ctx, metadata.CtxDomain, r.Host)

			// 继续处理请求
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// originChecker 来源检查器
type originChecker struct {
	allowMap      map[string]bool
	hasWildcard   bool
	allowWildcard bool
	patterns      []string
}

// newOriginChecker 创建来源检查器
func newOriginChecker(allowOrigins []string, allowWildcard bool) *originChecker {
	checker := &originChecker{
		allowMap:      make(map[string]bool),
		allowWildcard: allowWildcard,
	}

	for _, origin := range allowOrigins {
		if origin == "*" {
			checker.hasWildcard = true
		} else if strings.Contains(origin, "*") && allowWildcard {
			checker.patterns = append(checker.patterns, origin)
		} else {
			checker.allowMap[origin] = true
		}
	}

	return checker
}

// isAllowed 检查来源是否被允许
func (c *originChecker) isAllowed(origin string) bool {
	if origin == "" {
		return c.hasWildcard
	}

	// 精确匹配
	if c.allowMap[origin] {
		return true
	}

	// 通配符匹配
	if c.hasWildcard {
		return true
	}

	// 模式匹配
	if c.allowWildcard {
		for _, pattern := range c.patterns {
			if matchOriginPattern(origin, pattern) {
				return true
			}
		}
	}

	return false
}

// setCORSHeaders 设置CORS头部（完全修复版）
func setCORSHeaders(w http.ResponseWriter, r *http.Request, cfg *config.CORSConfig,
	origin string, checker *originChecker,
	allowMethodsStr, allowHeadersStr, exposeHeadersStr, maxAgeStr string) {

	// 【关键修复】：总是设置CORS头部，确保浏览器能看到响应

	// 1. 设置 Access-Control-Allow-Origin - 这是最关键的头部
	if checker.isAllowed(origin) {
		if checker.hasWildcard && !cfg.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		} else {
			// 如果没有origin但允许通配符，设置通配符
			if checker.hasWildcard {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}
	} else {
		// 【修复重点】：即使origin不在允许列表中，也要设置CORS头部
		// 否则浏览器看不到任何CORS响应，会报"No Access-Control-Allow-Origin header"
		if cfg.Debug {
			logx.Errorf("[CORS] Origin not in allowed list: %s, but still setting CORS headers for debugging", origin)
		}

		// 在开发/调试模式下，仍然允许请求
		if cfg.Debug && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		} else if len(cfg.AllowOrigins) > 0 {
			// 生产模式下，至少设置第一个允许的origin
			w.Header().Set("Access-Control-Allow-Origin", cfg.AllowOrigins[0])
		}
	}

	// 2. 设置允许的方法（总是设置）
	if allowMethodsStr != "" {
		w.Header().Set("Access-Control-Allow-Methods", allowMethodsStr)
	} else {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	}

	// 3. 设置允许的请求头（总是设置）
	if len(cfg.AllowHeaders) > 0 {
		if len(cfg.AllowHeaders) == 1 && cfg.AllowHeaders[0] == "*" {
			if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "*")
			}
		} else {
			w.Header().Set("Access-Control-Allow-Headers", allowHeadersStr)
		}
	} else {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Origin, Accept")
	}

	// 4. 设置暴露的响应头
	if exposeHeadersStr != "" {
		w.Header().Set("Access-Control-Expose-Headers", exposeHeadersStr)
	}

	// 5. 设置凭证
	if cfg.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// 6. 设置预检缓存时间
	if cfg.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", maxAgeStr)
	} else {
		w.Header().Set("Access-Control-Max-Age", "86400") // 24小时
	}

	// 7. WebSocket支持
	if cfg.AllowWebSockets {
		if upgrade := r.Header.Get("Upgrade"); upgrade != "" {
			additionalHeaders := "Upgrade, Connection, Sec-WebSocket-Key, Sec-WebSocket-Version, Sec-WebSocket-Protocol"
			currentHeaders := w.Header().Get("Access-Control-Allow-Headers")
			if currentHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", currentHeaders+", "+additionalHeaders)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", additionalHeaders)
			}
		}
	}
}

// matchOriginPattern 匹配来源模式（支持通配符）
func matchOriginPattern(origin, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 支持子域名通配符，如 *.example.com
	if strings.HasPrefix(pattern, "*.") {
		domain := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(origin, "."+domain) || origin == domain
	}

	// 支持协议通配符，如 https://*.example.com
	if strings.Contains(pattern, "*.") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(origin, parts[0]) && strings.HasSuffix(origin, parts[1])
		}
	}

	return origin == pattern
}
