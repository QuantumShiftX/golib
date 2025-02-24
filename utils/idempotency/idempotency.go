package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/QuantumShiftX/golib/stores/redisx"
	"github.com/QuantumShiftX/golib/stores/redisx/redislock"
	"github.com/coocood/freecache"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

// IdempotencyService 提供幂等性检查服务
type IdempotencyService struct {
	redisClient redis.UniversalClient
	localCache  *freecache.Cache // 本地缓存
	rs          *redsync.Redsync // 分布式锁
	keyPrefix   string
	expiration  time.Duration
}

// NewIdempotencyService 创建幂等性服务实例
func NewIdempotencyService(redisClient redis.UniversalClient, keyPrefix string, size int, expiration time.Duration) *IdempotencyService {
	return &IdempotencyService{
		redisClient: redisClient,
		localCache:  freecache.NewCache(size), // 创建 size  大小的本地缓存
		rs:          redislock.New(redisx.Engine),
		keyPrefix:   keyPrefix,
		expiration:  expiration,
	}
}

// CheckIdempotency 检查操作是否重复
// requestID: 请求唯一标识符（如订单号）
// data: 请求数据（用于生成唯一键）
// 返回值：
// - bool: true 表示请求是新的（未重复），false 表示请求重复
// - error: 操作过程中的错误
func (s *IdempotencyService) CheckIdempotency(ctx context.Context, requestID string, data interface{}) (bool, error) {
	// 生成幂等键
	key, err := s.generateKey(requestID, data)
	if err != nil {
		return false, fmt.Errorf("generate idempotency key error: %w", err)
	}
	// 1. 先查本地缓存
	if _, err := s.localCache.Get([]byte(key)); err == nil {
		// 本地缓存命中，说明是重复请求
		logx.WithContext(ctx).Infof("[CheckIdempotency] local cache hit: %s", key)
		return false, nil
	}
	// 2. 查询Redis中是否存在该键
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check redis key error: %w", err)
	}
	if exists > 0 {
		logx.WithContext(ctx).Infof("[CheckIdempotency] redis hit: %s", key)
		return false, nil
	}
	// 3. 尝试获取分布式锁
	mutex := s.rs.NewMutex(
		key,
		redsync.WithExpiry(30*time.Second),
		redsync.WithTries(1),
	)
	if err = mutex.Lock(); err != nil {
		logx.WithContext(ctx).Infof("[CheckIdempotency] lock error: %+v", err)
		// 获取锁失败，说明是重复请求
		return false, nil
	}
	defer mutex.Unlock()
	// 获取锁成功，说明是新请求
	// 注意：这里不主动释放锁，让它自然过期
	// 这样可以在锁的有效期内防止重复处理
	// 4. 在Redis中设置幂等标记
	err = s.redisClient.Set(ctx, key, "1", s.expiration).Err()
	if err != nil {
		return false, fmt.Errorf("set redis key error: %w", err)
	}
	// 5. 设置成功，同时更新本地缓存
	expireSeconds := int(s.expiration.Seconds() - 60) // 设置略短的本地缓存时间
	err = s.localCache.Set([]byte(key), []byte("1"), expireSeconds)
	if err != nil {
		// 本地缓存设置失败不影响主流程
		logx.WithContext(ctx).Errorf("设置本地缓存失败 key = %v,err = ：%v", []byte(key), err)
	}
	return true, nil
}

// DeleteIdempotencyKey 删除幂等性键
// 用于处理失败时清理已设置的键
func (s *IdempotencyService) DeleteIdempotencyKey(ctx context.Context, requestID string, data interface{}) {
	key, err := s.generateKey(requestID, data)
	if err != nil {
		logx.WithContext(ctx).Errorf("generate key error when deleting: %v", err)
		return
	}
	// 删除Redis中的键
	err = s.redisClient.Del(ctx, key).Err()
	if err != nil {
		logx.WithContext(ctx).Errorf("删除Redis键失败 key = %v, err = %v", key, err)
	}
	// 删除本地缓存
	s.localCache.Del([]byte(key))
	return
}

// GetCacheStats 获取缓存使用情况
func (s *IdempotencyService) GetCacheStats() string {
	entriesCount := s.localCache.EntryCount()
	return fmt.Sprintf("缓存条目数: %d", entriesCount)
}

// generateKey 生成幂等性键
// 使用请求ID和请求数据的组合生成唯一的键
func (s *IdempotencyService) generateKey(requestID string, data interface{}) (string, error) {
	// 将数据序列化为 JSON
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal data error: %w", err)
	}

	// 组合请求ID和数据
	combined := fmt.Sprintf("%s:%s", requestID, string(dataBytes))

	// 计算 SHA256 哈希
	hash := sha256.Sum256([]byte(combined))
	hashString := hex.EncodeToString(hash[:])

	// 返回带前缀的完整键
	return fmt.Sprintf("%s:%s", s.keyPrefix, hashString), nil
}
