package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
)

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
	Hash   []byte
	Salt   []byte
	Config *PasswordConf
}

// HashPassword 对密码进行哈希
func HashPassword(password string, config *PasswordConf) (*PasswordHash, error) {
	// 生成随机盐值
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		config.Time,
		config.Memory,
		config.Threads,
		config.KeyLen,
	)

	return &PasswordHash{
		Hash:   hash,
		Salt:   salt,
		Config: config,
	}, nil
}

// VerifyPassword 验证密码
func VerifyPassword(password string, ph *PasswordHash) bool {
	hash := argon2.IDKey(
		[]byte(password),
		ph.Salt,
		ph.Config.Time,
		ph.Config.Memory,
		ph.Config.Threads,
		ph.Config.KeyLen,
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
func FromString(s string, config *PasswordConf) (*PasswordHash, error) {
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
		Hash:   hash,
		Salt:   salt,
		Config: config,
	}, nil
}
