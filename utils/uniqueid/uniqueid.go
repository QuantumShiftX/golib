package uniqueid

import (
	"errors"
	"fmt"
	"github.com/sony/sonyflake"
	"log"
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
	flakeOnce  sync.Once // 用于确保Flake只初始化一次
	randSource rand.Source
	randMutex  sync.Mutex // 保护随机数生成的锁
	// 用于保证ID生成的互斥锁
	idMutex sync.Mutex
	// 初始化错误
	initError error
)

// 安全初始化函数
func initFlake() {
	flakeOnce.Do(func() {
		// 创建Sonyflake实例
		flakeSettings := sonyflake.Settings{
			MachineID: getMachineID,
		}
		flake = sonyflake.NewSonyflake(flakeSettings)

		// 检查flake是否成功初始化
		if flake == nil {
			initError = errors.New("failed to initialize sonyflake")
			log.Printf("ERROR: %v", initError)
		}

		// 使用当前时间戳初始化随机源
		randSource = rand.NewSource(time.Now().UnixNano())
	})
}

// 确保初始化函数被调用
func ensureInit() error {
	initFlake()
	return initError
}

// GenId 生成一个唯一的雪花ID
func GenId() (int64, error) {
	// 确保已初始化
	if err := ensureInit(); err != nil {
		return 0, err
	}

	// 双重检查flake是否为nil
	if flake == nil {
		return 0, errors.New("sonyflake instance is nil")
	}

	id, err := flake.NextID()
	if err != nil {
		return 0, fmt.Errorf("failed to generate ID: %v", err)
	}
	return int64(id), nil
}

// Deprecated: GenUserID 可能会有重复的ID
func GenUserID() (id uint64, err error) {
	// 确保已初始化
	if err := ensureInit(); err != nil {
		return 0, err
	}

	// 双重检查flake是否为nil
	if flake == nil {
		return 0, errors.New("sonyflake instance is nil")
	}

	// 生成一个雪花ID
	if id, err = flake.NextID(); err != nil {
		return 0, fmt.Errorf("failed to generate snowflake ID: %v", err)
	}

	// 获取当前时间戳的纳秒部分，用于增加随机性
	timestamp := time.Now().UnixNano()

	// 线程安全地使用随机数生成器
	randMutex.Lock()
	randGen := rand.New(randSource)
	random := randGen.Intn(1000) // 生成0-999之间的随机数
	randMutex.Unlock()

	// 结合雪花ID、时间戳和随机数生成ID
	combined := fmt.Sprintf("%d%d%d", id, timestamp%1000000000, random)

	// 确保combined字符串长度足够
	if len(combined) < 10 {
		return 0, fmt.Errorf("generated ID string too short: %s", combined)
	}

	// 取组合字符串的最后10位数字作为最终的ID
	finalID, err := strconv.ParseUint(combined[len(combined)-10:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to generate 10-digit ID: %v", err)
	}

	return finalID, nil
}

// GenInviteCode 根据用户ID生成6位邀请码
func GenInviteCode(userID uint64) (string, error) {
	if userID == 0 {
		return "", errors.New("userID cannot be zero")
	}

	// 线程安全地使用随机数生成器
	randMutex.Lock()
	randSource := rand.NewSource(time.Now().UnixNano() ^ int64(userID))
	randGen := rand.New(randSource)
	randMutex.Unlock()

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
			// 确保索引在有效范围内
			if index < 0 || index >= len(inviteCodeChars) {
				index = randGen.Intn(len(inviteCodeChars))
			}
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

func GenDefaultSnowID() (int64, error) {
	snowID, err := GenSnowIDWithLength(0, 0)
	return int64(snowID), err
}

// GenSnowIDWithLength 生成指定位数的ID
// minDigits: 最小位数（6-12）
// maxDigits: 最大位数（6-12）
func GenSnowIDWithLength(minDigits, maxDigits int) (id uint64, err error) {
	// 确保已初始化
	if err := ensureInit(); err != nil {
		return 0, err
	}

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

	// 双重检查flake是否为nil
	if flake == nil {
		return 0, errors.New("sonyflake instance is nil")
	}

	// 生成一个雪花ID
	if id, err = flake.NextID(); err != nil {
		return 0, fmt.Errorf("failed to generate snowflake ID: %v", err)
	}

	// 获取当前时间戳的纳秒部分，用于增加随机性
	timestamp := time.Now().UnixNano()

	// 线程安全地使用随机数生成器
	randMutex.Lock()
	randGen := rand.New(randSource)
	random := randGen.Intn(10000) // 生成0-9999之间的随机数
	randMutex.Unlock()

	// 结合雪花ID、时间戳和随机数生成ID
	combined := fmt.Sprintf("%d%d%d", id, timestamp%1000000000, random)

	// 动态决定取多少位
	digits := minDigits
	if minDigits != maxDigits {
		randMutex.Lock()
		digits = minDigits + randGen.Intn(maxDigits-minDigits+1)
		randMutex.Unlock()
	}

	// 如果combined的长度小于所需位数，在前面补0
	for len(combined) < digits {
		combined = "0" + combined
	}

	// 根据选择的位数生成最终ID
	finalStr := combined
	if len(combined) > digits {
		randMutex.Lock()
		startPos := randGen.Intn(len(combined) - digits + 1)
		randMutex.Unlock()
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

// GetMachineID 导出获取机器ID的方法，便于外部使用
func GetMachineID() (uint16, error) {
	return getMachineID()
}

// 获取机器 ID，先后尝试K8s、Docker和MAC地址
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

// isRunningInK8s 判断是否在Kubernetes环境中运行
func isRunningInK8s() (bool, error) {
	// 检查K8s环境特有的路径和环境变量
	if _, err := os.Stat("/var/run/secrets/kubernetes.io"); err == nil {
		return true, nil
	}

	// 检查常见的K8s环境变量
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true, nil
	}

	return false, nil
}

// 获取机器ID从Kubernetes环境
func getMachineIDFromK8s() (uint16, error) {
	var identifiers []string

	// 尝试获取Pod名称 - 通常包含部署名称和唯一标识
	podName := os.Getenv("HOSTNAME")
	if podName != "" {
		identifiers = append(identifiers, podName)
	}

	// 尝试获取Pod IP - 每个Pod通常有唯一的IP
	podIP := os.Getenv("POD_IP")
	if podIP == "" {
		// 尝试从网络接口获取IP
		podIP = getFirstNonLoopbackIP()
	}
	if podIP != "" {
		identifiers = append(identifiers, podIP)
	}

	// 尝试获取Pod UID
	if uid := getPodUID(); uid != "" {
		identifiers = append(identifiers, uid)
	}

	// 尝试获取节点名称 - 区分不同的工作节点
	nodeName := os.Getenv("NODE_NAME")
	if nodeName != "" {
		identifiers = append(identifiers, nodeName)
	}

	// 尝试获取节点IP
	nodeIP := os.Getenv("NODE_IP")
	if nodeIP != "" {
		identifiers = append(identifiers, nodeIP)
	}

	// 如果收集到了标识信息，使用它们计算一个机器ID
	if len(identifiers) > 0 {
		// 连接所有标识符并计算哈希
		idStr := strings.Join(identifiers, "-")
		hashValue := sum([]byte(idStr))

		// 记录用于生成机器ID的信息，便于排障
		log.Printf("K8s machine ID generated from: %s (hash: %d)", idStr, hashValue)

		return uint16(hashValue % 1024), nil
	}

	// 如果无法获取K8s特定信息，尝试使用进程ID和创建时间
	pid := os.Getpid()
	startTime := getProcessStartTime(pid)
	if startTime > 0 {
		idStr := fmt.Sprintf("pid-%d-time-%d", pid, startTime)
		hashValue := sum([]byte(idStr))
		log.Printf("K8s machine ID generated from process info: %s (hash: %d)", idStr, hashValue)
		return uint16(hashValue % 1024), nil
	}

	// 如果无法获取K8s特定信息，返回错误
	return 0, errors.New("unable to determine machine ID from Kubernetes environment")
}

// getFirstNonLoopbackIP 获取第一个非回环IP地址
func getFirstNonLoopbackIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if !v.IP.IsLoopback() && v.IP.To4() != nil {
					return v.IP.String()
				}
			}
		}
	}

	return ""
}

// getPodUID 尝试获取Pod的UID
func getPodUID() string {
	// 尝试从downward API获取
	if uid := os.Getenv("POD_UID"); uid != "" {
		return uid
	}

	// 尝试从主机名获取，某些K8s实现将UID作为主机名的一部分
	hostname := os.Getenv("HOSTNAME")
	if hostname != "" {
		parts := strings.Split(hostname, "-")
		if len(parts) > 2 {
			// 通常Pod名称格式为: deployment-name-random-uuid
			// 取最后几个部分可能是随机生成的标识符
			return strings.Join(parts[len(parts)-2:], "-")
		}
	}

	return ""
}

// getProcessStartTime 获取进程的启动时间（UNIX时间戳），失败返回0
func getProcessStartTime(pid int) int64 {
	// 尝试从/proc获取进程信息
	procStatPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(procStatPath)
	if err != nil {
		return 0
	}

	// 解析进程状态信息
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return 0
	}

	// 第22个字段是进程的启动时间
	startTime, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return 0
	}

	return startTime
}

// 判断是否在 Docker 容器中运行
func isRunningInDocker() (bool, error) {
	// 检查容器的特征文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true, nil
	}

	// 检查 cgroup 信息
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		// 在某些环境中，这些文件可能无法读取
		// 但这可能并不是错误，只是不在Docker环境
		return false, nil
	}
	return strings.Contains(string(data), "docker"), nil
}

// 获取容器 ID（适用于 Docker 环境）
func getContainerID() (string, error) {
	// 获取容器 ID
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/self/cgroup: %v", err)
	}

	// 从文件内容中提取容器 ID
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

// 获取机器 ID 基于 MAC 地址（不在 Docker 环境下）
func getMachineIDFromMac() (uint16, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		// 如果无法获取接口，返回一个基于主机名的备用ID
		hostname, hErr := os.Hostname()
		if hErr != nil {
			// 最后的备用方案：使用时间戳
			return uint16(time.Now().UnixNano() % 1024), nil
		}
		return uint16(sum([]byte(hostname)) % 1024), nil
	}

	// 查找第一个有效的网卡并获取其 MAC 地址
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.HardwareAddr == nil || len(iface.HardwareAddr) == 0 {
			continue
		}
		return uint16(sum(iface.HardwareAddr) % 1024), nil
	}

	// 如果没有找到有效接口，使用主机名作为备用
	hostname, err := os.Hostname()
	if err != nil {
		// 最后的备用方案：使用时间戳
		return uint16(time.Now().UnixNano() % 1024), nil
	}
	return uint16(sum([]byte(hostname)) % 1024), nil
}

// 计算字节数组的和作为机器 ID
func sum(data []byte) int {
	if len(data) == 0 {
		return int(time.Now().UnixNano() % 1024)
	}

	total := 0
	for _, b := range data {
		total += int(b)
	}
	return total
}
