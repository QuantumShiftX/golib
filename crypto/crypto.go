package crypto

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/jsonx"
	"sync"
	"time"
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	Algorithm() string
}

// EncryptedData 加密数据结构
type EncryptedData struct {
	Encrypted bool   `json:"encrypted"`
	Data      string `json:"data"`
	Algorithm string `json:"algorithm,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// XCryptoService 加密服务
type XCryptoService struct {
	encryptor Encryptor
	debug     bool
	mu        sync.RWMutex // 保护并发访问
}

// NewCryptoService 创建加密服务
func NewCryptoService(encryptor Encryptor, debug bool) *XCryptoService {
	return &XCryptoService{
		encryptor: encryptor,
		debug:     debug,
	}
}

// EncryptJSON 加密JSON数据
func (s *XCryptoService) EncryptJSON(data interface{}) (*EncryptedData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jsonBytes, err := jsonx.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json marshal failed: %w", err)
	}

	encrypted, err := s.encryptor.Encrypt(string(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	return &EncryptedData{
		Encrypted: true,
		Data:      encrypted,
		Algorithm: s.encryptor.Algorithm(),
		Timestamp: getCurrentTimestamp(),
	}, nil
}

// DecryptJSON 解密JSON数据
func (s *XCryptoService) DecryptJSON(encryptedData *EncryptedData, target interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !encryptedData.Encrypted {
		return fmt.Errorf("data is not encrypted")
	}

	decrypted, err := s.encryptor.Decrypt(encryptedData.Data)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	if err = jsonx.Unmarshal([]byte(decrypted), target); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	return nil
}

// EncryptString 加密字符串
func (s *XCryptoService) EncryptString(plaintext string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.encryptor.Encrypt(plaintext)
}

// DecryptString 解密字符串
func (s *XCryptoService) DecryptString(ciphertext string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.encryptor.Decrypt(ciphertext)
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
