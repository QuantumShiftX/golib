package etcdc

import (
	"errors"
	"github.com/jinzhu/copier"
	configurator "github.com/zeromicro/go-zero/core/configcenter"
	"github.com/zeromicro/go-zero/core/configcenter/subscriber"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Etcd[T any] struct {
	configurator configurator.Configurator[T]
	config       Config
	stopRefresh  chan struct{}
	mu           sync.RWMutex
	refreshed    atomic.Bool // 用于记录是否进行过刷新
	lastRefresh  time.Time   // 记录上次刷新时间
}

// NewEtcd 实例化etcd
func NewEtcd[T any](c Config) *Etcd[T] {
	logx.Infof("Initializing Etcd client with config: Host=%s, Key=%s, User=%s, EnableAutoRefresh=%v",
		c.Host, c.Key, maskUsername(c.User), c.TokenRefresh.EnableAutoRefresh)

	var cc subscriber.EtcdConf
	_ = copier.Copy(&cc, &c)
	cc.Hosts = strings.Split(c.Host, ",")

	// 设置默认token刷新配置
	if c.TokenRefresh.RefreshInterval == 0 {
		c.TokenRefresh.RefreshInterval = 10 * time.Minute
		logx.Infof("Using default token refresh interval: %v", c.TokenRefresh.RefreshInterval)
	}
	if c.TokenRefresh.RefreshBefore == 0 {
		c.TokenRefresh.RefreshBefore = 2 * time.Minute
		logx.Infof("Using default refresh before interval: %v", c.TokenRefresh.RefreshBefore)
	}

	// 在这里预先注册etcd的用户凭证，确保discov包能使用它们
	if c.User != "" && c.Pass != "" {
		logx.Infof("Registering etcd account credentials for hosts: %v", cc.Hosts)
		discov.RegisterAccount(cc.Hosts, c.User, c.Pass)
	} else {
		logx.Info("No etcd credentials provided, authentication will not be used")
	}

	// TLS配置
	if c.CertFile != "" || c.CertKeyFile != "" || c.CACertFile != "" {
		logx.Infof("Configuring TLS for etcd with CertFile=%s, CertKeyFile=%s, CACertFile=%s",
			c.CertFile, c.CertKeyFile, c.CACertFile)
		err := discov.RegisterTLS(cc.Hosts, c.CertFile, c.CertKeyFile, c.CACertFile, c.InsecureSkipVerify)
		if err != nil {
			logx.Errorf("Failed to register TLS for etcd: %v", err)
		}
	}

	// 创建EtcdSubscriber
	logx.Info("Creating EtcdSubscriber...")
	etcdSub := subscriber.MustNewEtcdSubscriber(cc)
	logx.Info("EtcdSubscriber created successfully")

	etcd := &Etcd[T]{
		configurator: configurator.MustNewConfigCenter[T](configurator.Config{
			Type: "json",
		}, etcdSub),
		config:      c,
		stopRefresh: make(chan struct{}),
		lastRefresh: time.Now(),
	}
	logx.Info("Etcd instance created successfully")

	// 如果启用了自动刷新token，则启动token刷新协程
	if c.TokenRefresh.EnableAutoRefresh && c.User != "" && c.Pass != "" {
		logx.Info("Auto token refresh is enabled, starting token refresher")
		go etcd.startTokenRefresher()
	} else {
		logx.Info("Auto token refresh is disabled or credentials not provided")
	}

	return etcd
	//return &Etcd[T]{
	//	configurator: configurator.MustNewConfigCenter[T](configurator.Config{
	//		Type: "json",
	//	}, subscriber.MustNewEtcdSubscriber(cc)),
	//}
}

// maskUsername 掩盖用户名，只显示前两个和后两个字符
func maskUsername(username string) string {
	if len(username) <= 4 {
		return "****"
	}
	return username[:2] + "****" + username[len(username)-2:]
}

// GetConfig 获取配置，增加错误恢复逻辑
func (ctr *Etcd[T]) GetConfig() (T, error) {
	var result T
	var err error

	logx.Infof("Getting config for key: %s", ctr.config.Key)
	err = ctr.withRecovery(func() error {
		result, err = ctr.configurator.GetConfig()
		return err
	})

	if err != nil {
		logx.Errorf("Failed to get config: %v", err)
		return result, err
	}

	logx.Infof("Successfully retrieved config for key: %s", ctr.config.Key)
	return result, err
}

func (ctr *Etcd[T]) Listener(listener func(ec *Etcd[T])) {
	logx.Info("Adding listener for config changes")
	listener(ctr)
	ctr.configurator.AddListener(func() {
		logx.Info("Config change detected, notifying listener")
		listener(ctr)
	})
}

// refreshToken 刷新认证token
func (ctr *Etcd[T]) refreshToken() error {
	// 保存当前时间作为刷新时间
	now := time.Now()

	// 使用锁保护并发访问
	ctr.mu.Lock()
	defer ctr.mu.Unlock()

	// 检查是否离上次刷新太近（至少间隔1秒）
	if now.Sub(ctr.lastRefresh) < time.Second {
		logx.Info("Token refresh requested too frequently, skipping")
		return nil
	}

	logx.Info("Refreshing etcd token credentials...")
	if ctr.config.User == "" || ctr.config.Pass == "" {
		logx.Error("Cannot refresh token: missing username or password")
		return errors.New("missing username or password")
	}

	// 使用discov.RegisterAccount重新注册账户凭证
	// 这会使go-zero内部在下一次连接时使用新的凭证
	hosts := strings.Split(ctr.config.Host, ",")

	// 注册前记录日志
	logx.Infof("Re-registering account for hosts: %v with user: %s",
		hosts, maskUsername(ctr.config.User))

	discov.RegisterAccount(hosts, ctr.config.User, ctr.config.Pass)

	// 更新刷新标志和时间
	ctr.refreshed.Store(true)
	ctr.lastRefresh = now

	logx.Info("Successfully refreshed etcd token credentials")
	return nil
}

// startTokenRefresher 启动token刷新协程
func (ctr *Etcd[T]) startTokenRefresher() {
	ticker := time.NewTicker(ctr.config.TokenRefresh.RefreshInterval)
	defer ticker.Stop()

	logx.Infof("Token refresher started, will refresh every %v", ctr.config.TokenRefresh.RefreshInterval)

	// 立即刷新一次token
	if err := ctr.refreshToken(); err != nil {
		logx.Errorf("Initial token refresh failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			logx.Info("Token refresh timer triggered")
			if err := ctr.refreshToken(); err != nil {
				logx.Errorf("Failed to refresh etcd token: %v", err)
			} else {
				logx.Info("Scheduled token refresh completed successfully")
			}
		case <-ctr.stopRefresh:
			logx.Info("Token refresher stopped due to close signal")
			return
		}
	}
}

// 实现错误检测和自动重连
func (ctr *Etcd[T]) withRecovery(fn func() error) error {
	err := fn()
	if err != nil && isAuthError(err) {
		logx.Errorf("Detected auth error: %v, attempting to refresh token", err)

		// 尝试刷新token
		if refreshErr := ctr.refreshToken(); refreshErr != nil {
			logx.Errorf("Failed to refresh token after auth error: %v", refreshErr)
			return err // 返回原始错误，因为刷新失败
		}

		logx.Info("Token refreshed successfully after auth error, retrying operation")
		// 重试原始操作
		retryErr := fn()
		if retryErr != nil {
			logx.Errorf("Operation still failed after token refresh: %v", retryErr)
			return retryErr
		}

		logx.Info("Operation succeeded after token refresh")
		return nil // 成功
	}
	return err
}

// isAuthError 检查是否是认证错误
func isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	isAuth := strings.Contains(errStr, "invalid auth token") ||
		strings.Contains(errStr, "etcdserver: user name is empty") ||
		strings.Contains(errStr, "etcdserver: authentication failed")

	if isAuth {
		logx.Infof("Detected authentication error: %v", err)
	}

	return isAuth
}

// HasRefreshed 返回是否进行过token刷新
func (ctr *Etcd[T]) HasRefreshed() bool {
	return ctr.refreshed.Load()
}

// LastRefreshTime 返回上次刷新token的时间
func (ctr *Etcd[T]) LastRefreshTime() time.Time {
	ctr.mu.RLock()
	defer ctr.mu.RUnlock()
	return ctr.lastRefresh
}

// Close 关闭etcd客户端和token刷新协程
func (ctr *Etcd[T]) Close() error {
	logx.Info("Closing Etcd client and stopping token refresher")
	// 停止token刷新协程
	close(ctr.stopRefresh)
	logx.Info("Etcd client closed successfully")
	return nil
}
