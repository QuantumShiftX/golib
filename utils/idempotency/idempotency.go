package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumShiftX/golib/stores/redisx"
	"github.com/QuantumShiftX/golib/stores/redisx/redislock"
	"github.com/coocood/freecache"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// 默认配置
	DefaultLocalCacheSize = 100 * 1024 * 1024 // 100MB
	DefaultExpiration     = 10 * time.Minute  // 默认过期时间
	DefaultLockTTL        = 10 * time.Second  // 锁过期时间
	MinCacheExpireSeconds = 60                // 最小缓存时间
	CacheTimeOffset       = 60                // 缓存时间偏移量
)

// IdemResult 幂等结果（扩展功能）
type IdemResult struct {
	Status    string      `json:"status"`           // processing, completed
	Result    interface{} `json:"result,omitempty"` // 业务执行结果
	Error     string      `json:"error,omitempty"`  // 错误信息
	Timestamp int64       `json:"timestamp"`        // 时间戳
}

// IdemService 提供幂等性检查服务
type IdemService struct {
	redisClient redis.UniversalClient
	localCache  *freecache.Cache // 本地缓存
	rs          *redsync.Redsync // 分布式锁
	keyPrefix   string
	expiration  time.Duration
	lockTTL     time.Duration
	hashPool    sync.Pool // SHA256计算复用池
}

// NewIdempotencyService 创建幂等性服务实例
func NewIdempotencyService(redisClient redis.UniversalClient, keyPrefix string, size int, expiration time.Duration) *IdemService {
	if redisClient == nil {
		panic("redis client is nil")
	}

	if size <= 0 {
		size = DefaultLocalCacheSize
	}

	if expiration <= 0 {
		expiration = DefaultExpiration
	}

	return &IdemService{
		redisClient: redisClient,
		localCache:  freecache.NewCache(size),
		rs:          redislock.New(redisx.Engine),
		keyPrefix:   keyPrefix,
		expiration:  expiration,
		lockTTL:     DefaultLockTTL,
		hashPool: sync.Pool{
			New: func() interface{} {
				return sha256.New()
			},
		},
	}
}

// CheckIdempotency 检查操作是否重复（核心方法）
// requestID: 请求唯一标识符（如订单号）
// data: 请求数据（用于生成唯一键）
// 返回值：
// - bool: true 表示请求是新的（未重复），false 表示请求重复
// - error: 操作过程中的错误
func (s *IdemService) CheckIdempotency(ctx context.Context, requestID string, data interface{}) (bool, error) {
	// 生成幂等键
	key, err := s.generateKey(requestID, data)
	if err != nil {
		return false, fmt.Errorf("generate idempotency key error: %w", err)
	}

	logx.WithContext(ctx).Infof("[CheckIdempotency] checking key: %s", key)

	// ===== 快速路径：无锁检查 =====

	// 1. 先查本地缓存
	if _, err = s.localCache.Get([]byte(key)); err == nil {
		logx.WithContext(ctx).Infof("[CheckIdempotency] local cache hit: %s", key)
		return false, nil
	}

	// 2. 查询Redis中是否存在该键
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check redis key error: %w", err)
	}
	if exists > 0 {
		// Redis存在，同步到本地缓存
		s.updateLocalCache(key, []byte("1"))
		logx.WithContext(ctx).Infof("[CheckIdempotency] redis hit: %s", key)
		return false, nil
	}

	// ===== 慢速路径：加锁处理 =====

	// 3. 尝试获取分布式锁（使用独立的锁键）
	lockKey := fmt.Sprintf("%s:lock", key)
	mutex := s.rs.NewMutex(
		lockKey,
		redsync.WithExpiry(s.lockTTL),
		redsync.WithTries(3), // 尝试3次
	)

	if err = mutex.Lock(); err != nil {
		logx.WithContext(ctx).Infof("[CheckIdempotency] lock failed, treating as duplicate: %+v", err)
		// 获取锁失败，说明有其他请求正在处理，视为重复请求
		return false, nil
	}
	defer func() {
		if _, err := mutex.Unlock(); err != nil {
			logx.WithContext(ctx).Errorf("[CheckIdempotency] unlock failed: %+v", err)
		}
	}()

	// 4. ⭐ 双重检查（关键步骤！）
	exists, err = s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("double check redis key error: %w", err)
	}
	if exists > 0 {
		// 其他请求已经设置了键，同步到本地缓存
		s.updateLocalCache(key, []byte("1"))
		logx.WithContext(ctx).Infof("[CheckIdempotency] double check hit: %s", key)
		return false, nil
	}

	// 5. 在Redis中设置幂等标记
	err = s.redisClient.Set(ctx, key, "1", s.expiration).Err()
	if err != nil {
		return false, fmt.Errorf("set redis key error: %w", err)
	}

	// 6. 设置成功，同时更新本地缓存
	s.updateLocalCache(key, []byte("1"))

	logx.WithContext(ctx).Infof("[CheckIdempotency] new request accepted: %s", key)
	return true, nil
}

// CheckIdempotencyWithResult 检查幂等性并支持存储结果（扩展功能）
func (s *IdemService) CheckIdempotencyWithResult(ctx context.Context, requestID string, data interface{}) (bool, *IdemResult, error) {
	key, err := s.generateKey(requestID, data)
	if err != nil {
		return false, nil, fmt.Errorf("generate key error: %w", err)
	}

	// 快速路径：检查是否已完成
	if result := s.getResult(ctx, key); result != nil {
		return false, result, nil
	}

	// 获取锁处理
	lockKey := fmt.Sprintf("%s:lock", key)
	mutex := s.rs.NewMutex(
		lockKey,
		redsync.WithExpiry(s.lockTTL),
		redsync.WithTries(3),
	)

	if err = mutex.Lock(); err != nil {
		// 可能正在处理中
		return false, &IdemResult{Status: "processing"}, nil
	}
	defer mutex.Unlock()

	// 双重检查
	if result := s.getResult(ctx, key); result != nil {
		return false, result, nil
	}

	// 设置处理中状态
	processingResult := &IdemResult{
		Status:    "processing",
		Timestamp: time.Now().Unix(),
	}

	if err = s.setResult(ctx, key, processingResult); err != nil {
		return false, nil, err
	}

	return true, nil, nil
}

// CompleteIdempotency 标记操作完成并存储结果
func (s *IdemService) CompleteIdempotency(ctx context.Context, requestID string, data interface{}, result interface{}, resultErr error) error {
	key, err := s.generateKey(requestID, data)
	if err != nil {
		return fmt.Errorf("generate key error: %w", err)
	}

	completedResult := &IdemResult{
		Status:    "completed",
		Result:    result,
		Timestamp: time.Now().Unix(),
	}

	if resultErr != nil {
		completedResult.Error = resultErr.Error()
	}

	return s.setResult(ctx, key, completedResult)
}

// DeleteIdempotencyKey 删除幂等性键
// 用于处理失败时清理已设置的键
func (s *IdemService) DeleteIdempotencyKey(ctx context.Context, requestID string, data interface{}) error {
	key, err := s.generateKey(requestID, data)
	if err != nil {
		return fmt.Errorf("generate key error when deleting: %w", err)
	}

	// 删除Redis中的键
	if err = s.redisClient.Del(ctx, key).Err(); err != nil {
		logx.WithContext(ctx).Errorf("delete redis key failed, key=%v, err=%v", key, err)
		return err
	}

	// 删除本地缓存
	s.localCache.Del([]byte(key))

	logx.WithContext(ctx).Infof("[DeleteIdempotencyKey] deleted key: %s", key)
	return nil
}

// GetCacheStats 获取缓存使用情况
func (s *IdemService) GetCacheStats() string {
	entriesCount := s.localCache.EntryCount()
	hitRate := s.localCache.HitRate()
	return fmt.Sprintf("缓存条目数: %d, 命中率: %.2f%%", entriesCount, hitRate*100)
}

// ===== 私有辅助方法 =====

// generateKey 生成幂等性键
func (s *IdemService) generateKey(requestID string, data interface{}) (string, error) {
	if requestID == "" {
		return "", errors.New("requestID cannot be empty")
	}

	// 使用对象池复用hasher
	hasher := s.hashPool.Get().(interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
		Reset()
	})
	defer func() {
		hasher.Reset()
		s.hashPool.Put(hasher)
	}()

	// 将数据序列化为 JSON
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal data error: %w", err)
	}

	// 组合请求ID和数据
	hasher.Write([]byte(requestID))
	hasher.Write([]byte(":"))
	hasher.Write(dataBytes)

	// 计算哈希
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	// 返回带前缀的完整键
	return fmt.Sprintf("%s:%s", s.keyPrefix, hashString), nil
}

// updateLocalCache 更新本地缓存
func (s *IdemService) updateLocalCache(key string, value []byte) {
	expireSeconds := int(s.expiration.Seconds()) - CacheTimeOffset
	if expireSeconds <= 0 {
		expireSeconds = MinCacheExpireSeconds
	}

	if err := s.localCache.Set([]byte(key), value, expireSeconds); err != nil {
		// 本地缓存设置失败不影响主流程，仅记录日志
		logx.Errorf("update local cache failed, key=%s, err=%v", key, err)
	}
}

// getResult 获取执行结果
func (s *IdemService) getResult(ctx context.Context, key string) *IdemResult {
	// 先查本地缓存
	if data, err := s.localCache.Get([]byte(key)); err == nil {
		var result IdemResult
		if err = json.Unmarshal(data, &result); err == nil {
			return &result
		}
	}

	// 查Redis
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	var result IdemResult
	if err = json.Unmarshal([]byte(data), &result); err != nil {
		return nil
	}

	// 同步到本地缓存
	s.updateLocalCache(key, []byte(data))
	return &result
}

// setResult 设置执行结果
func (s *IdemService) setResult(ctx context.Context, key string, result *IdemResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result error: %w", err)
	}

	if err = s.redisClient.Set(ctx, key, string(data), s.expiration).Err(); err != nil {
		return fmt.Errorf("set redis result error: %w", err)
	}

	s.updateLocalCache(key, data)
	return nil
}
