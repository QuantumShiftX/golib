package config

import (
	"fmt"
	"os"
	"strings"
)

// CryptoConfig 加密配置
type CryptoConfig struct {
	Enable      bool     `json:"enable,optional" yaml:"enable"`
	Key         string   `json:"key,optional" yaml:"key"`
	EnableURI   []string `json:"enable_uri,optional" yaml:"enable_uri"`
	FailOnError bool     `json:"fail_on_error,optional" yaml:"fail_on_error"`
	Algorithm   string   `json:"algorithm,optional" yaml:"algorithm"`
	Debug       bool     `json:"debug,optional" yaml:"debug"`
}

// DefaultCryptoConfig 默认加密配置
func DefaultCryptoConfig() *CryptoConfig {
	return &CryptoConfig{
		Enable:      false,
		Key:         "",
		EnableURI:   []string{},
		FailOnError: false,
		Algorithm:   "AES-GCM",
		Debug:       false,
	}
}

// LoadFromEnv 从环境变量加载
func (c *CryptoConfig) LoadFromEnv() {
	if enable := os.Getenv("CRYPTO_ENABLE"); enable == "true" {
		c.Enable = true
	}

	if key := os.Getenv("CRYPTO_KEY"); key != "" {
		c.Key = key
	}

	if uris := os.Getenv("CRYPTO_ENABLE_URI"); uris != "" {
		c.EnableURI = strings.Split(uris, ",")
	}

	if failOnError := os.Getenv("CRYPTO_FAIL_ON_ERROR"); failOnError == "true" {
		c.FailOnError = true
	}

	if algorithm := os.Getenv("CRYPTO_ALGORITHM"); algorithm != "" {
		c.Algorithm = algorithm
	}

	if debug := os.Getenv("CRYPTO_DEBUG"); debug == "true" {
		c.Debug = true
	}
}

// Validate 验证加密配置
func (c *CryptoConfig) Validate() error {
	if !c.Enable {
		return nil
	}

	if len(c.Key) != 32 {
		return fmt.Errorf("crypto key must be exactly 32 bytes, got %d", len(c.Key))
	}

	validAlgorithms := []string{"AES-GCM", "AES-CBC"}
	for _, alg := range validAlgorithms {
		if c.Algorithm == alg {
			return nil
		}
	}

	return fmt.Errorf("unsupported algorithm: %s", c.Algorithm)
}

// ShouldEncrypt 检查路径是否需要加密
func (c *CryptoConfig) ShouldEncrypt(path string) bool {
	if !c.Enable {
		return false
	}

	// 如果EnableURI为空，加密所有路径
	if len(c.EnableURI) == 0 {
		return true
	}

	for _, uri := range c.EnableURI {
		if matchPath(path, uri) {
			return true
		}
	}

	return false
}

// matchPath 路径匹配
func matchPath(path, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 支持前缀通配符
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	// 支持后缀通配符
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(path, suffix)
	}

	// 精确匹配
	return strings.HasPrefix(path, pattern)
}
