package config

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
)

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	EnableCORS     bool                   `json:"enable_cors" yaml:"enable_cors"`
	EnableLogging  bool                   `json:"enable_logging" yaml:"enable_logging"`
	EnableRecovery bool                   `json:"enable_recovery" yaml:"enable_recovery"`
	CORS           *CORSConfig            `json:"cors,omitempty" yaml:"cors,omitempty"`
	Logging        *LoggingConfig         `json:"logging,omitempty" yaml:"logging,omitempty"`
	Custom         map[string]interface{} `json:"custom,omitempty" yaml:"custom,omitempty"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	// 基本配置
	AllowOrigins     []string `json:"allow_origins" yaml:"allow_origins"`         // 允许的来源
	AllowMethods     []string `json:"allow_methods" yaml:"allow_methods"`         // 允许的方法
	AllowHeaders     []string `json:"allow_headers" yaml:"allow_headers"`         // 允许的请求头
	ExposeHeaders    []string `json:"expose_headers" yaml:"expose_headers"`       // 暴露的响应头
	AllowCredentials bool     `json:"allow_credentials" yaml:"allow_credentials"` // 是否允许凭证
	MaxAge           int      `json:"max_age" yaml:"max_age"`                     // 预检请求缓存时间（秒）

	// 高级配置
	AllowWildcard   bool `json:"allow_wildcard" yaml:"allow_wildcard"`     // 是否允许通配符
	AllowWebSockets bool `json:"allow_websockets" yaml:"allow_websockets"` // 是否允许WebSocket
	Debug           bool `json:"debug" yaml:"debug"`                       // 是否开启调试模式
	OptionsResponse int  `json:"options_response" yaml:"options_response"` // OPTIONS请求的响应状态码
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level         string `json:"level" yaml:"level"`
	Format        string `json:"format" yaml:"format"`
	OutputPath    string `json:"output_path" yaml:"output_path"`
	EnableTrace   bool   `json:"enable_trace" yaml:"enable_trace"`
	EnableMetrics bool   `json:"enable_metrics" yaml:"enable_metrics"`
}

// DefaultMiddlewareConfig 默认中间件配置
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		EnableCORS:     true,
		EnableLogging:  true,
		EnableRecovery: true,
		CORS: &CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
				http.MethodOptions,
				http.MethodPatch,
			},
			AllowHeaders: []string{"*"},
			ExposeHeaders: []string{
				"Content-Length",
				"Content-Type",
				"X-Config-Complete",
				"X-Bind-App",
				"X-Bind-Contacts",
			},
			AllowCredentials: false,
			MaxAge:           86400,
			AllowWildcard:    true,
			AllowWebSockets:  true,
			Debug:            false,
			OptionsResponse:  http.StatusNoContent,
		},
		Logging: &LoggingConfig{
			Level:         "info",
			Format:        "json",
			OutputPath:    "stdout",
			EnableTrace:   false,
			EnableMetrics: false,
		},
		Custom: make(map[string]interface{}),
	}
}

// LoadFromEnv 从环境变量加载
func (m *MiddlewareConfig) LoadFromEnv() {
	if enableCORS := os.Getenv("MIDDLEWARE_CORS"); enableCORS == "false" {
		m.EnableCORS = false
	}

	if enableLogging := os.Getenv("MIDDLEWARE_LOGGING"); enableLogging == "false" {
		m.EnableLogging = false
	}

	if enableRecovery := os.Getenv("MIDDLEWARE_RECOVERY"); enableRecovery == "false" {
		m.EnableRecovery = false
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		m.Logging.Level = logLevel
	}

	if maxAge := os.Getenv("CORS_MAX_AGE"); maxAge != "" {
		if age, err := strconv.Atoi(maxAge); err == nil {
			m.CORS.MaxAge = age
		}
	}
}

// Validate 验证中间件配置
func (m *MiddlewareConfig) Validate() error {
	if m.Logging != nil {
		validLevels := []string{"debug", "info", "warn", "error"}
		found := false
		for _, level := range validLevels {
			if m.Logging.Level == level {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid logging level: %s", m.Logging.Level)
		}
	}

	return nil
}
