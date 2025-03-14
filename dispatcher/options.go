package dispatcher

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/hibiken/asynq"
	"time"
)

// RedisConfig 包含Redis连接相关配置
type RedisConfig struct {
	Addr         string `json:"addr,optional"`
	Username     string `json:"username,optional"`
	Password     string `json:"password,optional"`
	DB           int    `json:"db,optional"`
	DialTimeout  int    `json:"dialTimeout,optional"`
	ReadTimeout  int    `json:"readTimeout,optional"`
	WriteTimeout int    `json:"writeTimeout,optional"`
	PoolSize     int    `json:"poolSize,optional"`
}

// QueuePriority 定义任务队列优先级
type QueuePriority struct {
	Low    int `json:"low"`
	Normal int `json:"normal"`
	High   int `json:"high"`
}

// ServerConfig 包含服务器相关配置
type ServerConfig struct {
	Concurrency     int           `json:"concurrency,optional"`
	ShutdownTimeout int           `json:"shutdownTimeout,optional"`
	QueuePriorities QueuePriority `json:"queuePriorities,optional"`
}

// MonitoringConfig 包含监控服务配置
type MonitoringConfig struct {
	Enabled bool   `json:"enabled,optional"`
	Address string `json:"address,optional"`
	Path    string `json:"path,optional"`
}

// Options 包含所有组件配置
type Options struct {
	Redis      RedisConfig      `json:"redis,optional"`
	Server     ServerConfig     `json:"server,optional"`
	Monitoring MonitoringConfig `json:"monitoring,optional"`
}

// NewOptions 从配置创建选项
func NewOptions(c config.Config) (*Options, error) {
	opts := DefaultOptions()
	err := c.Value("Scheduler").Scan(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load scheduler config: %w", err)
	}
	return opts, nil
}

// DefaultOptions 返回默认配置
func DefaultOptions() *Options {
	return &Options{
		Redis: RedisConfig{
			Addr:         "localhost:6379",
			DialTimeout:  5,
			ReadTimeout:  3,
			WriteTimeout: 3,
			PoolSize:     10,
		},
		Server: ServerConfig{
			Concurrency:     10,
			ShutdownTimeout: 30,
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
		DialTimeout:  time.Duration(o.Redis.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(o.Redis.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(o.Redis.WriteTimeout) * time.Second,
		PoolSize:     o.Redis.PoolSize,
	}
}

// ToDispatcherOptions 将配置转换为dispatcher.Options
func (c *Options) ToDispatcherOptions() (*Options, error) {
	// 创建默认配置
	opts := DefaultOptions()

	// 设置Redis配置
	if c.Redis.Addr != "" {
		opts.Redis.Addr = c.Redis.Addr
	}
	if c.Redis.Username != "" {
		opts.Redis.Username = c.Redis.Username
	}
	if c.Redis.Password != "" {
		opts.Redis.Password = c.Redis.Password
	}
	if c.Redis.DB != 0 {
		opts.Redis.DB = c.Redis.DB
	}
	if c.Redis.PoolSize != 0 {
		opts.Redis.PoolSize = c.Redis.PoolSize
	}

	// 设置超时时间
	if c.Redis.DialTimeout != 0 {
		opts.Redis.DialTimeout = c.Redis.DialTimeout
	}
	if c.Redis.ReadTimeout != 0 {
		opts.Redis.ReadTimeout = c.Redis.ReadTimeout
	}
	if c.Redis.WriteTimeout != 0 {
		opts.Redis.WriteTimeout = c.Redis.WriteTimeout
	}

	// 设置服务器配置
	if c.Server.Concurrency != 0 {
		opts.Server.Concurrency = c.Server.Concurrency
	}
	if c.Server.ShutdownTimeout != 0 {
		opts.Server.ShutdownTimeout = c.Server.ShutdownTimeout
	}

	// 设置队列优先级
	if c.Server.QueuePriorities.Low != 0 {
		opts.Server.QueuePriorities.Low = c.Server.QueuePriorities.Low
	}
	if c.Server.QueuePriorities.Normal != 0 {
		opts.Server.QueuePriorities.Normal = c.Server.QueuePriorities.Normal
	}
	if c.Server.QueuePriorities.High != 0 {
		opts.Server.QueuePriorities.High = c.Server.QueuePriorities.High
	}

	// 设置监控配置
	opts.Monitoring.Enabled = c.Monitoring.Enabled
	if c.Monitoring.Address != "" {
		opts.Monitoring.Address = c.Monitoring.Address
	}
	if c.Monitoring.Path != "" {
		opts.Monitoring.Path = c.Monitoring.Path
	}

	return opts, nil
}
