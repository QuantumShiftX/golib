package config

import (
	"fmt"
	"os"
)

// GlobalConfig 全局配置
type GlobalConfig struct {
	Debug      bool              `json:"debug,optional" yaml:"debug"`
	Crypto     *CryptoConfig     `json:"crypto,optional,omitempty" yaml:"crypto,omitempty"`
	Middleware *MiddlewareConfig `json:"middleware,optional,omitempty" yaml:"middleware,omitempty"`
}

// DefaultGlobalConfig 默认全局配置
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Debug:      false,
		Crypto:     DefaultCryptoConfig(),
		Middleware: DefaultMiddlewareConfig(),
	}
}

// LoadFromEnv 从环境变量加载配置
func (c *GlobalConfig) LoadFromEnv() {
	if debug := os.Getenv("GLOBAL_DEBUG"); debug == "true" {
		c.Debug = true
	}

	if c.Crypto != nil {
		c.Crypto.LoadFromEnv()
	}

	if c.Middleware != nil {
		c.Middleware.LoadFromEnv()
	}
}

// Validate 验证配置
func (c *GlobalConfig) Validate() error {
	if c.Crypto != nil {
		if err := c.Crypto.Validate(); err != nil {
			return fmt.Errorf("crypto config validation failed: %w", err)
		}
	}

	if c.Middleware != nil {
		if err := c.Middleware.Validate(); err != nil {
			return fmt.Errorf("middleware config validation failed: %w", err)
		}
	}

	return nil
}

// XConfigBuilder 配置构建器
type XConfigBuilder struct {
	config *GlobalConfig
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder() *XConfigBuilder {
	return &XConfigBuilder{
		config: DefaultGlobalConfig(),
	}
}

// WithDebug 设置调试模式
func (b *XConfigBuilder) WithDebug(debug bool) *XConfigBuilder {
	b.config.Debug = debug
	return b
}

// WithCrypto 设置加密配置
func (b *XConfigBuilder) WithCrypto(crypto *CryptoConfig) *XConfigBuilder {
	b.config.Crypto = crypto
	return b
}

// WithMiddleware 设置中间件配置
func (b *XConfigBuilder) WithMiddleware(middleware *MiddlewareConfig) *XConfigBuilder {
	b.config.Middleware = middleware
	return b
}

// EnableCrypto 启用加密
func (b *XConfigBuilder) EnableCrypto(debug bool, key string, uris ...string) *XConfigBuilder {
	if b.config.Crypto == nil {
		b.config.Crypto = DefaultCryptoConfig()
	}
	b.config.Crypto.Enable = true
	b.config.Crypto.Key = key
	b.config.Crypto.EnableURI = uris
	b.config.Crypto.Debug = debug
	return b
}

// EnableCORS 启用CORS
func (b *XConfigBuilder) EnableCORS(origins ...string) *XConfigBuilder {
	if b.config.Middleware == nil {
		b.config.Middleware = DefaultMiddlewareConfig()
	}
	b.config.Middleware.EnableCORS = true
	if len(origins) > 0 {
		b.config.Middleware.CORS.AllowOrigins = origins
	}
	return b
}

// Build 构建配置
func (b *XConfigBuilder) Build() (*GlobalConfig, error) {
	if err := b.config.Validate(); err != nil {
		return nil, err
	}
	return b.config, nil
}
