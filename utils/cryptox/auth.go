package cryptox

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
	"sync"
)

var (
	passwordConfigOnce sync.Once
	passwordConf       *PasswordConf
)

// InitPasswordConfig 初始化密码配置
func InitPasswordConfig(config *PasswordConf) {
	passwordConfigOnce.Do(func() {
		if config == nil {
			passwordConf = DefaultConfig()
		} else {
			passwordConf = config
		}
	})
}

// PasswordConf 密码哈希配置
type PasswordConf struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

// DefaultConfig 返回推荐的配置参数
func DefaultConfig() *PasswordConf {
	return &PasswordConf{
		Time:    1,
		Memory:  64 * 1024,
		Threads: 4,
		KeyLen:  32,
	}
}

// PasswordHash 密码哈希结构
type PasswordHash struct {
	Hash []byte
	Salt []byte
}

// GetPasswordConfig 获取密码配置
func GetPasswordConfig() *PasswordConf {
	if passwordConf == nil {
		InitPasswordConfig(nil)
	}
	return passwordConf
}

// HashPassword 使用单例配置进行哈希
func HashPassword(password string) (*PasswordHash, error) {
	// 生成随机盐值
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	config := GetPasswordConfig()
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		config.Time,
		config.Memory,
		config.Threads,
		config.KeyLen,
	)

	return &PasswordHash{
		Hash: hash,
		Salt: salt,
	}, nil
}

// VerifyPassword 验证密码
func VerifyPassword(password string, ph *PasswordHash) bool {
	config := GetPasswordConfig()
	hash := argon2.IDKey(
		[]byte(password),
		ph.Salt,
		config.Time,
		config.Memory,
		config.Threads,
		config.KeyLen,
	)
	return base64.StdEncoding.EncodeToString(hash) == base64.StdEncoding.EncodeToString(ph.Hash)
}

// ToString 将哈希信息转为字符串存储
func (ph *PasswordHash) ToString() string {
	hash := base64.StdEncoding.EncodeToString(ph.Hash)
	salt := base64.StdEncoding.EncodeToString(ph.Salt)
	return hash + ":" + salt
}

// FromString 从字符串还原哈希信息
func FromString(s string) (*PasswordHash, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid hash string format")
	}

	hash, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	salt, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	return &PasswordHash{
		Hash: hash,
		Salt: salt,
	}, nil
}
