package idempotency

import (
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Factory 幂等性服务工厂
type Factory struct {
	redisClient redis.UniversalClient
	services    map[string]*IdemService
	cfg         Config
	mu          sync.RWMutex
}

var (
	factory *Factory
	once    sync.Once
)

// Config 配置
type Config struct {
	// 本地缓存大小(字节)
	LocalCacheSize int `json:"local_cache_size"`
	// 各业务配置
	BusinessConfigs map[string]BusinessConfig `json:"business_configs"`
}

// BusinessConfig 业务配置
type BusinessConfig struct {
	KeyPrefix  string        `json:"key_prefix"`
	Expiration time.Duration `json:"expiration"`
}

// 默认配置
var defaultConfig = Config{
	LocalCacheSize: 100 * 1024 * 1024, // 100MB
	BusinessConfigs: map[string]BusinessConfig{
		"default": {
			KeyPrefix:  "default",
			Expiration: 5 * time.Minute,
		},
	},
}

// Must 初始化幂等性服务工厂(必须调用)
func Must(cfg Config, rdb redis.UniversalClient) {
	once.Do(func() {
		if factory == nil {
			factory = NewFactory(cfg, rdb)
		}
	})
}

// NewFactory 创建幂等性服务工厂
func NewFactory(cfg Config, rdb redis.UniversalClient) *Factory {
	// 使用默认配置补充
	if cfg.LocalCacheSize == 0 {
		cfg.LocalCacheSize = defaultConfig.LocalCacheSize
	}
	if cfg.BusinessConfigs == nil {
		cfg.BusinessConfigs = defaultConfig.BusinessConfigs
	}

	return &Factory{
		redisClient: rdb,
		services:    make(map[string]*IdemService),
		cfg:         cfg,
	}
}

// GetService 获取指定业务的幂等性服务
func (f *Factory) GetService(businessType string) *IdemService {
	f.mu.Lock()
	defer f.mu.Unlock()

	if service, exists := f.services[businessType]; exists {
		return service
	}

	// 获取业务配置
	config := f.getBusinessConfig(businessType)
	service := NewIdempotencyService(
		f.redisClient,
		config.KeyPrefix,
		f.cfg.LocalCacheSize,
		config.Expiration,
	)

	f.services[businessType] = service
	return service
}

func (f *Factory) getBusinessConfig(businessType string) BusinessConfig {
	if config, exists := f.cfg.BusinessConfigs[businessType]; exists {
		return config
	}
	return defaultConfig.BusinessConfigs["default"]
}

// Service 获取指定业务的幂等性服务(全局方法)
func Service(businessType string) *IdemService {
	if factory == nil {
		panic("idempotency factory not initialized, call Must() first")
	}
	return factory.GetService(businessType)
}
