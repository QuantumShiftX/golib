package etcdc

import "time"

type Config struct {
	Host               string
	Key                string
	ID                 int64  `json:",optional"`
	User               string `json:",optional"`
	Pass               string `json:",optional"`
	CertFile           string `json:",optional"`
	CertKeyFile        string `json:",optional=CertFile"`
	CACertFile         string `json:",optional=CertFile"`
	InsecureSkipVerify bool   `json:",optional"`
	// TokenRefresh token刷新相关配置
	TokenRefresh TokenRefreshConfig `json:",optional"`
}

// TokenRefreshConfig 定义token刷新相关配置
type TokenRefreshConfig struct {
	// 是否启用自动刷新token
	EnableAutoRefresh bool `json:",optional"`
	// token刷新间隔，默认为10分钟
	RefreshInterval time.Duration `json:",optional"`
	// token刷新前的提前量，默认为2分钟
	RefreshBefore time.Duration `json:",optional"`
}
