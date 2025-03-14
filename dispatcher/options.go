package dispatcher

import (
	"crypto/tls"
	"github.com/hibiken/asynq"
	"time"
)

// RedisConfig 包含Redis连接相关配置
type RedisConfig struct {
	Addr         string        `json:"addr"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	DB           int           `json:"db"`
	DialTimeout  time.Duration `json:"dialTimeout"`
	ReadTimeout  time.Duration `json:"readTimeout"`
	WriteTimeout time.Duration `json:"writeTimeout"`
	PoolSize     int           `json:"poolSize"`
	TLSConfig    *tls.Config   `json:"-"` // 不序列化TLS配置
}

// QueuePriority 定义任务队列优先级
type QueuePriority struct {
	Low    int `json:"low"`
	Normal int `json:"normal"`
	High   int `json:"high"`
}

// ServerConfig 包含服务器相关配置
type ServerConfig struct {
	Concurrency     int           `json:"concurrency"`
	ShutdownTimeout time.Duration `json:"shutdownTimeout"`
	QueuePriorities QueuePriority `json:"queuePriorities"`
}

// MonitoringConfig 包含监控服务配置
type MonitoringConfig struct {
	Enabled bool   `json:"enabled"`
	Address string `json:"address"`
	Path    string `json:"path"`
}

// Options 包含所有组件配置
type Options struct {
	Redis      RedisConfig      `json:"redis"`
	Server     ServerConfig     `json:"server"`
	Monitoring MonitoringConfig `json:"monitoring"`
}

// DefaultOptions 返回默认配置
func DefaultOptions() *Options {
	return &Options{
		Redis: RedisConfig{
			Addr:         "localhost:6379",
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
		},
		Server: ServerConfig{
			Concurrency:     10,
			ShutdownTimeout: 30 * time.Second,
			QueuePriorities: QueuePriority{
				Low:    1,
				Normal: 3,
				High:   6,
			},
		},
		Monitoring: MonitoringConfig{
			Enabled: true,
			Address: ":9547",
			Path:    "/monitoring",
		},
	}
}

// ToRedisClientOpt 转换为asynq.RedisClientOpt
func (o *Options) ToRedisClientOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:         o.Redis.Addr,
		Username:     o.Redis.Username,
		Password:     o.Redis.Password,
		DB:           o.Redis.DB,
		DialTimeout:  o.Redis.DialTimeout,
		ReadTimeout:  o.Redis.ReadTimeout,
		WriteTimeout: o.Redis.WriteTimeout,
		PoolSize:     o.Redis.PoolSize,
		TLSConfig:    o.Redis.TLSConfig,
	}
}
