package uniqueid

import (
	"errors"
	"fmt"
	"github.com/sony/sonyflake"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// 邀请码字符集，去掉了容易混淆的字符
	inviteCodeChars = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	// 邀请码长度
	inviteCodeLength = 6
)

var (
	flake      *sonyflake.Sonyflake
	randSource rand.Source
	// 用于保证ID生成的互斥锁
	idMutex sync.Mutex
)

func init() {
	flake = sonyflake.NewSonyflake(sonyflake.Settings{
		MachineID: getMachineID,
	})
	// 使用当前时间戳初始化随机源
	randSource = rand.NewSource(time.Now().UnixNano())
}

// GenId 生成一个唯一的雪花ID
func GenId() (int64, error) {
	id, err := flake.NextID()
	if err != nil {
		return 0, err
	}
	return int64(id), nil
}

func GenUserID() (id uint64, err error) {
	// 生成一个雪花ID
	if id, err = flake.NextID(); err != nil {
		return 0, fmt.Errorf("failed to generate snowflake ID: %v", err)
	}

	// 获取当前时间戳的纳秒部分，用于增加随机性
	timestamp := time.Now().UnixNano()

	// 创建一个新的随机数生成器，避免使用 rand.Seed
	randGen := rand.New(randSource)
	random := randGen.Intn(1000) // 生成0-999之间的随机数

	// 结合雪花ID、时间戳和随机数生成ID
	combined := fmt.Sprintf("%d%d%d", id, timestamp%1000000000, random)

	// 取组合字符串的最后10位数字作为最终的ID
	finalID, err := strconv.ParseUint(combined[len(combined)-10:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to generate 10-digit ID: %v", err)
	}

	return finalID, nil
}

// GenInviteCode 根据用户ID生成邀请码
func GenInviteCode(userID uint64) (string, error) {
	// 生成一个雪花ID
	sfid, err := flake.NextID()
	if err != nil {
		return "", fmt.Errorf("generate snowflake id failed: %v", err)
	}

	// 将雪花ID和用户ID混合
	// 使用位运算确保数值唯一性
	mixID := (int64(sfid) << 20) | (int64(userID) & 0xFFFFF) // 取用户ID后20位

	// 创建随机数生成器
	randGen := rand.New(randSource)

	// 生成邀请码
	var code strings.Builder
	code.Grow(inviteCodeLength)

	// 前6位使用混合ID映射
	for i := 0; i < 6; i++ {
		index := int(mixID % int64(len(inviteCodeChars)))
		code.WriteByte(inviteCodeChars[index])
		mixID /= int64(len(inviteCodeChars))
	}

	// 后2位使用随机数，增加分散度
	for i := 0; i < 2; i++ {
		index := randGen.Intn(len(inviteCodeChars))
		code.WriteByte(inviteCodeChars[index])
	}

	return code.String(), nil
}

// VerifyInviteCode 验证邀请码格式
func VerifyInviteCode(code string) bool {
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
func GenDefaultSnowID() (int64, error) {
	snowID, err := GenSnowIDWithLength(0, 0)
	return int64(snowID), err
}

// GenSnowIDWithLength 生成指定位数的ID
// minDigits: 最小位数（6-12）
// maxDigits: 最大位数（6-12）
func GenSnowIDWithLength(minDigits, maxDigits int) (id uint64, err error) {
	// 如果参数无效，使用默认值
	if minDigits < 6 || minDigits > 12 {
		minDigits = 6
	}
	if maxDigits < minDigits || maxDigits > 12 {
		maxDigits = 12
	}
	if maxDigits < minDigits {
		maxDigits = minDigits
	}

	idMutex.Lock()
	defer idMutex.Unlock()

	// 生成一个雪花ID
	if id, err = flake.NextID(); err != nil {
		return 0, fmt.Errorf("failed to generate snowflake ID: %v", err)
	}

	// 获取当前时间戳的纳秒部分，用于增加随机性
	timestamp := time.Now().UnixNano()

	// 创建一个新的随机数生成器
	randGen := rand.New(randSource)
	random := randGen.Intn(10000) // 生成0-9999之间的随机数

	// 结合雪花ID、时间戳和随机数生成ID
	combined := fmt.Sprintf("%d%d%d", id, timestamp%1000000000, random)

	// 动态决定取多少位
	digits := minDigits
	if minDigits != maxDigits {
		digits = minDigits + randGen.Intn(maxDigits-minDigits+1)
	}

	// 如果combined的长度小于所需位数，在前面补0
	for len(combined) < digits {
		combined = "0" + combined
	}

	// 根据选择的位数生成最终ID
	finalStr := combined
	if len(combined) > digits {
		startPos := randGen.Intn(len(combined) - digits + 1)
		finalStr = combined[startPos : startPos+digits]
	}

	// 转换为uint64
	finalID, err := strconv.ParseUint(finalStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to generate %d-digit ID: %v", digits, err)
	}

	// 确保ID至少有minDigits位
	minValue := uint64(1)
	for i := 0; i < minDigits-1; i++ {
		minValue *= 10
	}

	// 如果ID小于最小值，增加最小值
	if finalID < minValue {
		finalID += minValue
	}

	return finalID, nil
}

// 获取机器 ID 基于 Docker 环境
func getMachineID() (uint16, error) {
	// 判断是否在 Docker 环境中运行
	if isRunningInDocker() {
		// 尝试通过容器 ID 生成机器 ID
		containerID, err := getContainerID()
		if err != nil {
			return 0, fmt.Errorf("failed to get container ID: %v", err)
		}
		return uint16(sum([]byte(containerID)) % 1024), nil
	}

	// 如果不在 Docker 环境中，继续使用 MAC 地址方式
	return getMachineIDFromMac()
}

// 判断是否在 Docker 容器中运行
func isRunningInDocker() bool {
	// 检查容器的特征文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// 检查 cgroup 信息
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "docker")
}

// 获取容器 ID（适用于 Docker 环境）
func getContainerID() (string, error) {
	// 获取容器 ID
	// 一般情况下，可以通过读取 `/proc/self/cgroup` 获取容器 ID
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/self/cgroup: %v", err)
	}

	// 从文件内容中提取容器 ID
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "docker") {
			parts := strings.Split(line, "/")
			if len(parts) > 2 {
				return parts[len(parts)-1], nil
			}
		}
	}
	return "", errors.New("container ID not found")
}

// 获取机器 ID 基于 MAC 地址（不在 Docker 环境下）
func getMachineIDFromMac() (uint16, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	// 查找第一个有效的网卡并获取其 MAC 地址
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.HardwareAddr == nil || len(iface.HardwareAddr) == 0 {
			continue
		}
		return uint16(sum(iface.HardwareAddr) % 1024), nil
	}

	return 0, errors.New("no valid network interface with MAC address found")
}

// 计算字节数组的和作为机器 ID
func sum(data []byte) int {
	total := 0
	for _, b := range data {
		total += int(b)
	}
	return total
}
