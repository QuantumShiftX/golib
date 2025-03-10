package idgen

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sony/sonyflake"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// 邀请码字符集，去掉了容易混淆的字符
	inviteCodeChars = "1234567890ABCDEFGHIJKLMNPQRSTUVWXYZ"
	// 邀请码长度
	inviteCodeLength = 8
	// 最小允许的ID位数
	minAllowedDigits = 8
	// 最大允许的ID位数
	maxAllowedDigits = 16
	// 默认生成ID的位数
	defaultDigits = 10
	// 节点ID位数
	nodeIDBits = 10
	// 序列号位数
	sequenceBits = 12
	// 时间戳左移位数
	timestampLeftShift = nodeIDBits + sequenceBits
	// 节点ID左移位数
	nodeIDLeftShift = sequenceBits
	// 序列号掩码
	sequenceMask = int64(-1) ^ (int64(-1) << sequenceBits)
	// 节点ID掩码
	nodeIDMask = int64(-1) ^ (int64(-1) << nodeIDBits)
	// 时钟回拨等待时间（毫秒）
	clockBackwardWaitMs = 5
	// Redis键前缀
	redisKeyPrefix = "idgen:uniqueid:"
	// 节点ID过期时间（天），设置为更长的时间以减少重复风险
	nodeIDExpiryDays = 30
	// 持久化节点ID的本地文件
	nodeIDPersistFile = "/tmp/idgen_node_id.dat"
	// 段预加载阈值 - 当剩余ID数量低于总容量的10%时，预加载下一个段
	segmentPreloadThreshold = 0.1
	// 默认步长
	segmentSize = 1000
)

// ID段结构，用于本地缓存一段Redis分配的ID
type IDSegment struct {
	current int64 // 当前位置
	max     int64 // 最大位置
	base    int64 // 基础值
}

var (
	flake      *sonyflake.Sonyflake
	flakeOnce  sync.Once // 用于确保Flake只初始化一次
	randSource = rand.NewSource(time.Now().UnixNano())
	randMutex  sync.Mutex // 保护随机数生成的锁
	// 初始化错误
	initError error
	// 上次生成的时间戳（毫秒）
	lastTimestamp int64
	// 序列号计数器 - 原子操作确保线程安全
	sequence int64
	// 节点ID
	nodeID int64
	// 时钟回拨检测锁
	clockBackwardLock sync.Mutex
	// 数字ID生成锁 - 为每个位数创建独立的锁
	digitLocks = make(map[int]*sync.Mutex)
	// 各位数ID的计数器映射 - 为每个位数创建独立的计数器
	digitCounters = make(map[int]*uint64)
	// 数字计数器锁
	counterMutex sync.RWMutex
	// 用于确保计数器初始化的锁
	counterInitLock sync.Mutex
	// 系统启动时间，用于生成唯一标识
	startTime = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano() / 1000000
	// 上下文
	ctx = context.Background()
	// 服务器本地ID段缓存
	serverSegments = make(map[string]*IDSegment)
	// 确保节点ID刷新的互斥锁
	refreshNodeIDLock sync.Mutex
	// 新增全局变量用于存储待用的段
	pendingSegments      map[string]*IDSegment
	segmentLoadingLock   sync.Mutex
	segmentLoadingStatus = make(map[string]bool)
	// 全局上下文和取消函数
	globalCtx       context.Context
	globalCtxCancel context.CancelFunc
)

type IDGenX struct {
	rdb redis.UniversalClient //redis
	ctx context.Context       // 上下文，用于控制后台任务
}

func NewIDGenX(rdb redis.UniversalClient) *IDGenX {

	// 创建一个可取消的上下文
	if globalCtx == nil {
		globalCtx, globalCtxCancel = context.WithCancel(context.Background())
	}

	return &IDGenX{rdb: rdb, ctx: globalCtx}
}

// 添加一个关闭方法，用于优雅关闭
func (x *IDGenX) Shutdown() {
	if globalCtxCancel != nil {
		globalCtxCancel()
	}
}

// 初始化各位数的计数器和锁
func initDigitCounters() {
	counterInitLock.Lock()
	defer counterInitLock.Unlock()

	for i := minAllowedDigits; i <= maxAllowedDigits; i++ {
		if _, exists := digitCounters[i]; !exists {
			var counter uint64 = 0
			digitCounters[i] = &counter
			digitLocks[i] = &sync.Mutex{}
		}
	}
}

// 安全初始化函数
func (x *IDGenX) initFlake() {
	flakeOnce.Do(func() {
		// 初始化计数器和锁
		initDigitCounters()

		// 创建Sonyflake实例
		flakeSettings := sonyflake.Settings{
			MachineID: getMachineID,
			// 设置起始时间为2023年1月1日
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		flake = sonyflake.NewSonyflake(flakeSettings)

		// 检查flake是否成功初始化
		if flake == nil {
			initError = errors.New("failed to initialize sonyflake")
			log.Printf("ERROR: %v", initError)
		}

		// 初始化上次时间戳
		lastTimestamp = time.Now().UnixNano() / 1000000 // 转换为毫秒

		// 获取并设置节点ID
		machineID, err := getMachineID()
		if err != nil {
			log.Printf("Warning: Failed to get machine ID: %v, using random value", err)
			// 使用随机值作为备用
			randMutex.Lock()
			machineID = uint16(rand.New(randSource).Intn(1024))
			randMutex.Unlock()
		}

		// 首先尝试从本地文件加载之前保存的节点ID
		savedNodeID, err := loadNodeIDFromFile()
		if err == nil && savedNodeID > 0 {
			// 验证此节点ID是否仍然有效
			if x.rdb != nil {
				valid, err := x.isNodeIDValid(savedNodeID)
				if err == nil && valid {
					nodeID = savedNodeID
					log.Printf("Restored nodeID from local file: %d", nodeID)

					// 更新Redis中的节点ID过期时间
					go x.refreshNodeIDExpiry(savedNodeID)
					return
				}
			} else {
				// 如果Redis不可用，直接使用保存的节点ID
				nodeID = savedNodeID
				log.Printf("Redis unavailable, using saved nodeID: %d", nodeID)
				return
			}
		}

		// 使用Redis分配唯一节点ID（如果Redis可用）
		if x.rdb != nil {
			allocatedNodeID, err := x.allocateNodeIDFromRedis(int64(machineID))
			if err == nil {
				nodeID = allocatedNodeID
				log.Printf("Allocated nodeID from Redis: %d", nodeID)

				// 保存分配的节点ID到本地文件
				saveNodeIDToFile(nodeID)

				// 启动一个goroutine定期刷新节点ID的过期时间
				go x.startNodeIDRefreshTask()
			} else {
				nodeID = int64(machineID) & nodeIDMask
				log.Printf("Failed to allocate nodeID from Redis: %v, using local nodeID: %d", err, nodeID)
			}
		} else {
			nodeID = int64(machineID) & nodeIDMask
			log.Printf("Redis unavailable, using local nodeID: %d", nodeID)
		}
	})
}

// 从本地文件加载节点ID
func loadNodeIDFromFile() (int64, error) {
	data, err := os.ReadFile(nodeIDPersistFile)
	if err != nil {
		return 0, err
	}

	nodeIDStr := strings.TrimSpace(string(data))
	return strconv.ParseInt(nodeIDStr, 10, 64)
}

// 保存节点ID到本地文件
func saveNodeIDToFile(id int64) error {
	return os.WriteFile(nodeIDPersistFile, []byte(fmt.Sprintf("%d", id)), 0644)
}

// 检查节点ID在Redis中是否仍然有效
func (x *IDGenX) isNodeIDValid(id int64) (bool, error) {
	if x.rdb == nil {
		return false, errors.New("Redis client not initialized")
	}

	nodeKey := fmt.Sprintf("%snode:%d", redisKeyPrefix, id)
	exists, err := x.rdb.Exists(ctx, nodeKey).Result()
	if err != nil {
		return false, err
	}

	return exists > 0, nil
}

// 刷新节点ID的过期时间
func (x *IDGenX) refreshNodeIDExpiry(id int64) {
	if x.rdb == nil {
		return
	}

	// 为Redis操作创建一个带超时的上下文
	redisCtx, cancel := context.WithTimeout(x.ctx, 5*time.Second)
	defer cancel()

	nodeKey := fmt.Sprintf("%snode:%d", redisKeyPrefix, id)

	// 使用新的过期时间
	expiry := time.Duration(nodeIDExpiryDays) * 24 * time.Hour

	// 尝试更新过期时间，使用带超时的上下文
	_, err := x.rdb.Expire(redisCtx, nodeKey, expiry).Result()
	if err != nil {
		log.Printf("Warning: Failed to refresh nodeID expiry: %v", err)

		// 检查错误是否是因为上下文取消导致的
		if redisCtx.Err() != nil {
			log.Printf("Redis operation was cancelled due to context timeout/cancellation")
			return
		}

		// 如果键不存在，重新设置
		success, err := x.rdb.SetNX(redisCtx, nodeKey, 1, expiry).Result()
		if err != nil || !success {
			log.Printf("Error: Failed to reclaim nodeID: %v", err)
		}
	}
}

// 启动节点ID刷新任务
func (x *IDGenX) startNodeIDRefreshTask() {

	// 创建一个子上下文，继承全局上下文的取消信号
	refreshCtx, cancel := context.WithCancel(x.ctx)
	defer cancel()
	// 定义一个tickerDuration为1天
	tickerDuration := 24 * time.Hour

	// 创建一个ticker
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	// 记录日志
	log.Printf("Starting node ID refresh task, will refresh every %v", tickerDuration)

	// 立即执行一次刷新操作
	x.refreshNodeIDExpiry(nodeID)

	// 循环等待下一次刷新或上下文取消
	for {
		select {
		case <-ticker.C:
			// 时间到，执行刷新操作
			refreshNodeIDLock.Lock()
			x.refreshNodeIDExpiry(nodeID)
			refreshNodeIDLock.Unlock()
			log.Printf("Node ID %d refreshed successfully", nodeID)
		case <-refreshCtx.Done():
			// 上下文被取消，退出循环
			log.Printf("Node ID refresh task is shutting down")
			return
		}
	}
}

// 从Redis分配唯一的节点ID - 改进版本，使用更长的过期时间
func (x *IDGenX) allocateNodeIDFromRedis(preferredID int64) (int64, error) {
	if x.rdb == nil {
		return 0, errors.New("Redis client not initialized")
	}

	// 设置更长的过期时间以减少节点ID重用风险
	expiry := time.Duration(nodeIDExpiryDays) * 24 * time.Hour

	// 尝试设置首选节点ID
	nodeKey := fmt.Sprintf("%snode:%d", redisKeyPrefix, preferredID)
	success, err := x.rdb.SetNX(ctx, nodeKey, 1, expiry).Result()
	if err == nil && success {
		// 成功设置首选ID
		return preferredID, nil
	}

	// 如果首选ID已被占用，尝试寻找可用ID
	for i := 0; i < 1024; i++ {
		nodeKey := fmt.Sprintf("%snode:%d", redisKeyPrefix, i)
		success, err := x.rdb.SetNX(ctx, nodeKey, 1, expiry).Result()
		if err == nil && success {
			return int64(i), nil
		}
	}

	return 0, errors.New("no available node IDs in Redis")
}

// 确保初始化函数被调用
func (x *IDGenX) ensureInit() error {
	x.initFlake()
	return initError
}

// 获取Redis分布式锁
func (x *IDGenX) getRedisLock(key string, expiry time.Duration) (bool, string, error) {
	if x.rdb == nil {
		return false, "", errors.New("Redis client not initialized")
	}

	// 生成唯一值用于锁识别
	value := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())

	// 尝试设置锁
	success, err := x.rdb.SetNX(ctx, key, value, expiry).Result()
	if err != nil {
		return false, "", err
	}

	return success, value, nil
}

// 释放Redis分布式锁
func (x *IDGenX) releaseRedisLock(key, value string) (bool, error) {
	if x.rdb == nil {
		return false, errors.New("Redis client not initialized")
	}

	// 使用Lua脚本确保只释放自己的锁
	script := `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end`

	result, err := x.rdb.Eval(ctx, script, []string{key}, value).Result()
	if err != nil {
		return false, err
	}

	return result.(int64) == 1, nil
}

// 使用Redis增量计数器生成序列号
func (x *IDGenX) getRedisSequence(prefix string) (int64, error) {
	if x.rdb == nil {
		return 0, errors.New("Redis client not initialized")
	}

	key := fmt.Sprintf("%s%s:seq", redisKeyPrefix, prefix)
	return x.rdb.Incr(ctx, key).Result()
}

// 基于Redis段分片算法生成唯一ID
// 将ID空间分成多个段，每个段由Redis原子计数分配
func (x *IDGenX) getUniqueIDFromRedisSegment(idType string, digits int) (int64, error) {
	if x.rdb == nil {
		return 0, errors.New("Redis client not initialized")
	}

	// 计算最小值和最大值
	minValue := int64(1)
	for i := 1; i < digits; i++ {
		minValue *= 10
	}
	maxValue := minValue*10 - 1

	// 段计数器的键
	segmentKey := fmt.Sprintf("%s%s:%d", redisKeyPrefix, idType, digits)

	// 获取当前服务器的本地段
	counterMutex.Lock()
	segment, ok := serverSegments[segmentKey]

	// 如果本地段不存在或已用完，检查是否有待用段
	if !ok || segment.current >= segment.max {
		// 首先检查是否有预加载的段
		if pendingSegment, hasPending := pendingSegments[segmentKey]; hasPending {
			// 使用预加载的段
			serverSegments[segmentKey] = pendingSegment
			segment = pendingSegment
			// 从待用段中移除
			delete(pendingSegments, segmentKey)
			log.Printf("Using preloaded segment for %s", segmentKey)
		} else {
			// 如果没有预加载的段，从Redis获取新段
			nextVal, err := x.rdb.Incr(ctx, segmentKey).Result()
			if err != nil {
				counterMutex.Unlock()
				return 0, fmt.Errorf("failed to get new segment from Redis: %v", err)
			}

			// 计算新段的范围
			segmentStart := (nextVal-1)*segmentSize + 1

			// 保存新段到本地
			serverSegments[segmentKey] = &IDSegment{
				current: 0,
				max:     segmentSize,
				base:    segmentStart,
			}
			segment = serverSegments[segmentKey]
		}
	} else {
		// 检查是否需要预加载下一个段
		// 当当前段使用超过80%且没有正在加载新段时，触发预加载
		preloadThreshold := int64(float64(segment.max) * (1 - segmentPreloadThreshold))
		if segment.current >= preloadThreshold {
			segmentLoadingLock.Lock()
			loading, exists := segmentLoadingStatus[segmentKey]
			segmentLoadingLock.Unlock()

			if !exists || !loading {
				// 异步预加载下一个段
				go x.preloadNextSegment(idType, digits)
			}
		}
	}

	// 从本地段分配ID
	localOffset := segment.current
	segment.current++
	counterMutex.Unlock()

	// 计算最终ID
	baseID := segment.base + localOffset

	// 计算实际ID值，确保在指定位数范围内
	finalID := (baseID % (maxValue - minValue + 1)) + minValue

	// 确保位数正确
	finalIDStr := fmt.Sprintf("%d", finalID)
	if len(finalIDStr) > digits {
		finalIDStr = finalIDStr[len(finalIDStr)-digits:]
		finalID, _ = strconv.ParseInt(finalIDStr, 10, 64)
	} else if len(finalIDStr) < digits {
		// 如果位数不足，在左侧补充
		padStr := strings.Repeat("0", digits-len(finalIDStr))
		finalIDStr = padStr + finalIDStr
		finalID, _ = strconv.ParseInt(finalIDStr, 10, 64)
	}

	return finalID, nil
}

// 预加载下一个段的函数
func (x *IDGenX) preloadNextSegment(idType string, digits int) {
	segmentKey := fmt.Sprintf("%s%s:%d", redisKeyPrefix, idType, digits)

	// 使用锁保护加载状态
	segmentLoadingLock.Lock()
	// 检查是否已在加载中
	if loading, exists := segmentLoadingStatus[segmentKey]; exists && loading {
		segmentLoadingLock.Unlock()
		return
	}

	// 标记为正在加载
	segmentLoadingStatus[segmentKey] = true
	segmentLoadingLock.Unlock()

	// 在函数返回时清除加载标志
	defer func() {
		segmentLoadingLock.Lock()
		segmentLoadingStatus[segmentKey] = false
		segmentLoadingLock.Unlock()
	}()

	// 创建一个带超时的上下文
	ctx, cancel := context.WithTimeout(x.ctx, 5*time.Second)
	defer cancel()

	// 从Redis获取新段
	nextVal, err := x.rdb.Incr(ctx, segmentKey).Result()
	if err != nil {
		log.Printf("Warning: Failed to preload next segment from Redis: %v", err)
		return
	}

	// 计算新段的范围
	segmentStart := (nextVal-1)*segmentSize + 1

	// 创建新的段
	newSegment := &IDSegment{
		current: 0,
		max:     segmentSize,
		base:    segmentStart,
	}

	// 将新段添加到待用段映射中
	counterMutex.Lock()
	// 检查当前是否有激活的段
	if segment, ok := serverSegments[segmentKey]; ok {
		// 如果当前段已经存在但差不多用完了，用新段替换它
		if segment.current >= segment.max {
			serverSegments[segmentKey] = newSegment
		} else {
			// 否则，将新段存储在另一个映射中，以便后续使用
			if pendingSegments == nil {
				pendingSegments = make(map[string]*IDSegment)
			}
			pendingSegments[segmentKey] = newSegment
		}
	} else {
		// 如果当前没有激活的段，直接使用新段
		serverSegments[segmentKey] = newSegment
	}
	counterMutex.Unlock()

	log.Printf("Successfully preloaded next segment for %s, starting at %d", segmentKey, segmentStart)
}

// GenId 生成一个唯一的雪花ID (原始长整型)
func (x *IDGenX) GenId() (int64, error) {
	// 确保已初始化
	if err := x.ensureInit(); err != nil {
		return 0, err
	}

	// 双重检查flake是否为nil
	if flake == nil {
		return 0, errors.New("sonyflake instance is nil")
	}

	// 尝试使用Redis生成全局唯一ID
	if x.rdb != nil {
		// 从Redis获取序列号
		seq, err := x.getRedisSequence("snowflake")
		if err == nil {
			// 生成雪花ID
			id, err := flake.NextID()
			if err != nil {
				return 0, fmt.Errorf("failed to generate ID: %v", err)
			}

			// 组合Redis序列号和雪花ID
			finalID := int64(id) ^ (seq << 10)
			return finalID, nil
		}
		// 如果Redis操作失败，回退到本地方式
		log.Printf("Warning: Failed to get sequence from Redis: %v, falling back to local generation", err)
	}

	id, err := flake.NextID()
	if err != nil {
		return 0, fmt.Errorf("failed to generate ID: %v", err)
	}
	return int64(id), nil
}

// 等待下一毫秒
func tilNextMillis(lastTimestamp int64) int64 {
	timestamp := timeGen()
	for timestamp <= lastTimestamp {
		time.Sleep(time.Microsecond)
		timestamp = timeGen()
	}
	return timestamp
}

// 获取当前时间戳（毫秒）
func timeGen() int64 {
	return time.Now().UnixNano() / 1000000
}

// 处理时钟回拨
func handleClockBackward(lastTs int64, currentTs int64) int64 {
	clockBackwardLock.Lock()
	defer clockBackwardLock.Unlock()

	if currentTs < lastTs {
		log.Printf("Clock moved backwards. Waiting until %d.", lastTs)
		// 等待一段时间
		time.Sleep(time.Duration(lastTs-currentTs+clockBackwardWaitMs) * time.Millisecond)
		return timeGen()
	}

	return currentTs
}

// 生成雪花算法ID
func generateSnowflakeID() (int64, error) {
	var timestamp int64
	var seq int64

	// 获取当前时间戳（毫秒）
	timestamp = timeGen()

	// 处理时钟回拨
	timestamp = handleClockBackward(atomic.LoadInt64(&lastTimestamp), timestamp)

	// 如果是同一毫秒内
	if timestamp == atomic.LoadInt64(&lastTimestamp) {
		// 序列号递增
		seq = (atomic.AddInt64(&sequence, 1) & sequenceMask)

		// 如果序列号用完，等待下一毫秒
		if seq == 0 {
			timestamp = tilNextMillis(atomic.LoadInt64(&lastTimestamp))
		}
	} else {
		// 不同毫秒，序列号重置
		atomic.StoreInt64(&sequence, 0)
		seq = 0
	}

	// 保存最后的时间戳
	atomic.StoreInt64(&lastTimestamp, timestamp)

	// 组合成最终的ID
	// ID结构: 时间戳部分 + 节点ID部分 + 序列号部分
	id := ((timestamp - startTime) << timestampLeftShift) | (nodeID << nodeIDLeftShift) | seq

	return id, nil
}

// GenUserID 生成一个10位的唯一用户ID
func (x *IDGenX) GenUserID() (id int64, err error) {
	return x.GenIDWithDigits(10)
}

// GenIDWithDigits 根据指定位数生成唯一ID
func (x *IDGenX) GenIDWithDigits(digits int) (int64, error) {
	// 参数验证
	if digits < minAllowedDigits || digits > maxAllowedDigits {
		digits = defaultDigits // 默认使用10位
	}

	// 确保已初始化
	if err := x.ensureInit(); err != nil {
		return 0, err
	}

	// 如果Redis可用，优先使用Redis段分配算法
	if x.rdb != nil {
		// 尝试从Redis段获取唯一ID
		id, err := x.getUniqueIDFromRedisSegment(fmt.Sprintf("digit:%d", digits), digits)
		if err == nil {
			return id, nil
		}

		// 如果Redis操作失败，记录日志并回退到本地生成
		log.Printf("Warning: Redis segment allocation failed: %v, falling back to local ID generation", err)
	}

	// 获取该位数对应的锁和计数器
	counterMutex.RLock()
	lock, lockExists := digitLocks[digits]
	counter, counterExists := digitCounters[digits]
	counterMutex.RUnlock()

	if !lockExists || !counterExists {
		counterMutex.Lock()
		var newCounter uint64 = 0
		digitCounters[digits] = &newCounter
		digitLocks[digits] = &sync.Mutex{}
		counter = &newCounter
		lock = digitLocks[digits]
		counterMutex.Unlock()
	}

	// 加锁确保该位数ID生成的原子性
	lock.Lock()
	defer lock.Unlock()

	// 生成雪花ID
	snowflakeID, err := generateSnowflakeID()
	if err != nil {
		return 0, fmt.Errorf("failed to generate snowflake ID: %v", err)
	}

	// 增加计数器值并获取
	counterValue := atomic.AddUint64(counter, 1)

	// 创建一个唯一的组合值
	// 1. 雪花ID - 保证全局唯一性
	// 2. 计数器值 - 确保同一节点连续生成的ID也不重复
	// 3. 加入时间精度到纳秒 - 进一步减少冲突可能

	// 获取当前纳秒时间戳的后几位
	nanoTimePart := (time.Now().UnixNano() % 10000)

	// 组合各部分生成一个种子
	seed := snowflakeID ^ int64(counterValue) ^ int64(nanoTimePart)

	// 使用种子创建一个随机源
	randMutex.Lock()
	r := rand.New(rand.NewSource(seed))
	randMutex.Unlock()

	// 生成数字ID
	var finalID uint64

	// 计算最小值和最大值
	minValue := uint64(1)
	for i := 1; i < digits; i++ {
		minValue *= 10
	}
	maxValue := minValue*10 - 1

	// 使用雪花ID的低位部分作为基础
	baseValue := uint64(snowflakeID & 0x7FFFFFFFFFFFFFFF) // 去掉符号位

	// 根据需要的位数生成最终ID
	finalID = (baseValue % (maxValue - minValue)) + minValue

	// 确保位数正确
	finalIDStr := fmt.Sprintf("%d", finalID)
	if len(finalIDStr) > digits {
		// 如果位数超出，截取右侧digits位
		finalIDStr = finalIDStr[len(finalIDStr)-digits:]
		finalID, _ = strconv.ParseUint(finalIDStr, 10, 64)
	} else if len(finalIDStr) < digits {
		// 如果位数不足，在左侧补充随机数字
		padLen := digits - len(finalIDStr)
		padStr := ""
		for i := 0; i < padLen; i++ {
			padStr += fmt.Sprintf("%d", r.Intn(10))
		}
		if padStr == strings.Repeat("0", padLen) {
			// 避免全0前缀
			padStr = "1" + padStr[1:]
		}
		finalIDStr = padStr + finalIDStr
		finalID, _ = strconv.ParseUint(finalIDStr, 10, 64)
	}

	// 确保ID不小于指定位数的最小值
	if finalID < minValue {
		finalID += minValue
	}

	return int64(finalID), nil
}

// GenSnowIDWithLength 生成指定位数范围内的ID
func (x *IDGenX) GenSnowIDWithLength(minDigits, maxDigits int) (int64, error) {
	// 如果参数无效，使用默认值
	if minDigits < minAllowedDigits || minDigits > maxAllowedDigits {
		minDigits = minAllowedDigits
	}
	if maxDigits < minDigits || maxDigits > maxAllowedDigits {
		maxDigits = maxAllowedDigits
	}

	// 随机选择位数(如果minDigits不等于maxDigits)
	digits := minDigits
	if minDigits != maxDigits {
		randMutex.Lock()
		r := rand.New(randSource)
		digits = minDigits + r.Intn(maxDigits-minDigits+1)
		randMutex.Unlock()
	}

	// 使用固定位数生成ID
	return x.GenIDWithDigits(digits)
}

// GenDefaultSnowID 生成默认位数(10位)的ID
func (x *IDGenX) GenDefaultSnowID() (int64, error) {
	snowID, err := x.GenIDWithDigits(10)
	return int64(snowID), err
}

// GenInviteCode 根据用户ID生成6位邀请码
func (x *IDGenX) GenInviteCode(userID uint64) (string, error) {
	if userID == 0 {
		return "", errors.New("userID cannot be zero")
	}

	// 使用一个专用的随机源
	randMutex.Lock()
	localSource := rand.NewSource(time.Now().UnixNano() ^ int64(userID))
	r := rand.New(localSource)
	randMutex.Unlock()

	// 将64位的userID拆分为多个较小的部分
	part1 := uint32(userID & 0xFFFF)         // 低16位
	part2 := uint32((userID >> 16) & 0xFFFF) // 次低16位
	part3 := uint32((userID >> 32) & 0xFFFF) // 次高16位
	part4 := uint32((userID >> 48) & 0xFFFF) // 高16位

	// 混合各部分生成混合值
	mixValue := (part1 ^ part2 ^ part3 ^ part4) | uint32(r.Intn(10000))

	// 生成邀请码
	var code strings.Builder
	code.Grow(inviteCodeLength)

	// 使用Redis生成唯一邀请码（如果可用）
	if x.rdb != nil {
		// 使用Redis分配一个唯一的邀请码序号
		inviteCodeSeq, err := x.rdb.Incr(ctx, redisKeyPrefix+"invitecode:seq").Result()
		if err == nil {
			// 使用确定的算法将序号转换为邀请码
			// 将序号混合用户ID特征，确保同一用户生成的邀请码不会太相似
			mixedSeq := inviteCodeSeq ^ int64(part1) ^ int64(part3)

			// 生成邀请码
			for i := 0; i < inviteCodeLength; i++ {
				// 确定性地选择字符
				charIndex := int((mixedSeq + int64(i)*79) % int64(len(inviteCodeChars)))
				code.WriteByte(inviteCodeChars[charIndex])
				// 更新混合值
				mixedSeq = (mixedSeq * 31) + int64(part2)
			}

			return code.String(), nil
		}

		// 如果Redis操作失败，回退到本地生成
		log.Printf("Warning: Failed to get invite code sequence from Redis: %v, falling back to local generation", err)
	}

	// 本地生成邀请码（Redis不可用时的回退方案）
	for i := 0; i < inviteCodeLength; i++ {
		if i < 3 {
			// 前3位使用ID特征
			index := int((mixValue + uint32(i)*7919) % uint32(len(inviteCodeChars)))
			if index < 0 || index >= len(inviteCodeChars) {
				index = r.Intn(len(inviteCodeChars))
			}
			code.WriteByte(inviteCodeChars[index])
			// 更新混合值
			mixValue = (mixValue * 31) + uint32(part1+part3)
		} else {
			// 后3位增加更多随机性
			randVal := r.Intn(len(inviteCodeChars))
			code.WriteByte(inviteCodeChars[randVal])
		}
	}

	return code.String(), nil
}

// VerifyInviteCode 验证邀请码格式
func (x *IDGenX) VerifyInviteCode(code string) bool {
	if code == "" {
		return false
	}

	if len(code) != inviteCodeLength {
		return false
	}

	code = strings.ToUpper(code)
	for _, c := range code {
		if !strings.ContainsRune(inviteCodeChars, c) {
			return false
		}
	}
	return true
}

// GetMachineID 导出获取机器ID的方法，便于外部使用
func (x *IDGenX) GetMachineID() (uint16, error) {
	return getMachineID()
}

// getMachineID 获取机器ID，先后尝试K8s、Docker和MAC地址
func getMachineID() (uint16, error) {
	// 首先检查是否在Kubernetes环境中
	isK8s, _ := isRunningInK8s()
	if isK8s {
		k8sID, err := getMachineIDFromK8s()
		if err == nil {
			return k8sID, nil
		}
		log.Printf("Warning: Failed to get machine ID from K8s: %v, falling back", err)
	}

	// 然后检查是否在Docker环境中
	isDocker, _ := isRunningInDocker()
	if isDocker {
		containerID, err := getContainerID()
		if err == nil && containerID != "" {
			return uint16(sum([]byte(containerID)) % 1024), nil
		}
		log.Printf("Warning: Failed to get container ID: %v, falling back", err)
	}

	// 最后尝试使用MAC地址
	return getMachineIDFromMac()
}

// 以下是原代码中未定义的辅助函数
func isRunningInK8s() (bool, error) {
	_, err := os.Stat("/var/run/secrets/kubernetes.io")
	return err == nil, nil
}

func getMachineIDFromK8s() (uint16, error) {
	// 尝试从K8s环境变量中获取更多信息
	var metadataParts []string

	// Pod名称
	podName := os.Getenv("HOSTNAME")
	if podName != "" {
		metadataParts = append(metadataParts, podName)
	}

	// 命名空间
	namespace := ""
	nsFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	if nsData, err := os.ReadFile(nsFile); err == nil {
		namespace = strings.TrimSpace(string(nsData))
		metadataParts = append(metadataParts, namespace)
	}

	// 服务账号
	saToken := ""
	saFile := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	if saData, err := os.ReadFile(saFile); err == nil {
		// 只取token的前10个字符作为标识，避免使用整个token
		saToken = strings.TrimSpace(string(saData))
		if len(saToken) > 10 {
			saToken = saToken[:10]
		}
		metadataParts = append(metadataParts, saToken)
	}

	// Pod UID
	podUID := ""
	podUIDFile := "/etc/podinfo/uid"
	if uidData, err := os.ReadFile(podUIDFile); err == nil {
		podUID = strings.TrimSpace(string(uidData))
		metadataParts = append(metadataParts, podUID)
	}

	// 如果能获取到Pod的IP地址
	podIP := os.Getenv("POD_IP")
	if podIP != "" {
		metadataParts = append(metadataParts, podIP)
	}

	// 如果没有获取到任何K8s特定信息，回退到简单的主机名方法
	if len(metadataParts) == 0 {
		hostname, err := os.Hostname()
		if err != nil {
			return 0, err
		}
		metadataParts = append(metadataParts, hostname)
	}

	// 组合所有收集到的信息，生成更加唯一的标识
	combinedMetadata := strings.Join(metadataParts, "-")

	// 计算哈希值并取模，确保在节点ID范围内(0-1023)
	return uint16(fnv32(combinedMetadata) % 1024), nil
}

// FNV-1a哈希算法，比简单求和更均匀
func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func isRunningInDocker() (bool, error) {
	_, err := os.Stat("/.dockerenv")
	if err == nil {
		return true, nil
	}

	// 另一种检查Docker的方法
	if _, err := os.Stat("/proc/1/cgroup"); err == nil {
		data, err := os.ReadFile("/proc/1/cgroup")
		if err == nil && strings.Contains(string(data), "docker") {
			return true, nil
		}
	}

	return false, nil
}

func getContainerID() (string, error) {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, "docker") {
			parts := strings.Split(line, "/")
			if len(parts) > 2 {
				return parts[len(parts)-1], nil
			}
		}
	}

	return "", errors.New("container ID not found")
}

func getMachineIDFromMac() (uint16, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}

	var hardwareAddr []byte
	for _, iface := range interfaces {
		if len(iface.HardwareAddr) > 0 && (iface.Flags&net.FlagLoopback) == 0 {
			hardwareAddr = iface.HardwareAddr
			break
		}
	}

	if len(hardwareAddr) == 0 {
		return 0, errors.New("no valid MAC address found")
	}

	return uint16(sum(hardwareAddr) % 1024), nil
}

func sum(data []byte) int {
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return sum
}
