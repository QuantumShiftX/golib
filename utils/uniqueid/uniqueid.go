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
	inviteCodeChars = "1234567890ABCDEFGHIJKLMNPQRSTUVWXYZ"
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

// GenInviteCode 根据用户ID生成6位邀请码
func GenInviteCode(userID uint64) (string, error) {
	// 创建一个安全的随机数生成器
	randSource := rand.NewSource(time.Now().UnixNano() ^ int64(userID))
	randGen := rand.New(randSource)

	// 使用userId的各个位进行混合，防止溢出
	// 将64位的userID拆分为多个较小的部分
	part1 := uint32(userID & 0xFFFF)         // 低16位
	part2 := uint32((userID >> 16) & 0xFFFF) // 次低16位
	part3 := uint32((userID >> 32) & 0xFFFF) // 次高16位
	part4 := uint32((userID >> 48) & 0xFFFF) // 高16位

	// 混合各部分生成混合值
	mixValue := (part1 ^ part2 ^ part3 ^ part4) | uint32(randGen.Intn(10000))

	// 生成邀请码
	var code strings.Builder
	code.Grow(inviteCodeLength)

	// 生成6位邀请码
	for i := 0; i < inviteCodeLength; i++ {
		// 每次使用不同的混合算法，增加随机性
		if i < 3 {
			// 前3位使用ID特征
			index := int((mixValue + uint32(i)*7919) % uint32(len(inviteCodeChars)))
			code.WriteByte(inviteCodeChars[index])
			mixValue = (mixValue * 31) + uint32(part1+part3)
		} else {
			// 后3位增加更多随机性
			randVal := randGen.Intn(len(inviteCodeChars))
			code.WriteByte(inviteCodeChars[randVal])
		}
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
