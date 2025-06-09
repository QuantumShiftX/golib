package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
)

// AESEncryptor AES加密器
type AESEncryptor struct {
	key       []byte
	algorithm string
	gcmCache  cipher.AEAD // 缓存GCM实例
	mu        sync.RWMutex
}

// initGCM 初始化GCM实例（性能优化）
func (e *AESEncryptor) initGCM() error {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.gcmCache = gcm
	e.mu.Unlock()

	return nil
}

// NewAESGCMEncryptor 创建AES-GCM加密器
func NewAESGCMEncryptor(key string) (*AESEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes long, got %d", len(key))
	}

	encryptor := &AESEncryptor{
		key:       []byte(key),
		algorithm: "AES-GCM",
	}

	// 预创建GCM实例
	if err := encryptor.initGCM(); err != nil {
		return nil, err
	}

	return encryptor, nil
}

// NewAESCBCEncryptor 创建AES-CBC加密器
func NewAESCBCEncryptor(key string) (*AESEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes long, got %d", len(key))
	}

	return &AESEncryptor{
		key:       []byte(key),
		algorithm: "AES-CBC",
	}, nil
}

// Encrypt 加密
func (e *AESEncryptor) Encrypt(plaintext string) (string, error) {
	switch e.algorithm {
	case "AES-GCM":
		return e.encryptGCM(plaintext)
	case "AES-CBC":
		return e.encryptCBC(plaintext)
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", e.algorithm)
	}
}

// Decrypt 解密
func (e *AESEncryptor) Decrypt(ciphertext string) (string, error) {
	switch e.algorithm {
	case "AES-GCM":
		return e.decryptGCM(ciphertext)
	case "AES-CBC":
		return e.decryptCBC(ciphertext)
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", e.algorithm)
	}
}

// Algorithm 返回算法名称
func (e *AESEncryptor) Algorithm() string {
	return e.algorithm
}

// encryptGCM AES-GCM加密
func (e *AESEncryptor) encryptGCM(plaintext string) (string, error) {
	e.mu.RLock()
	gcm := e.gcmCache
	e.mu.RUnlock()

	if gcm == nil {
		return "", fmt.Errorf("GCM not initialized")
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptGCM AES-GCM解密
func (e *AESEncryptor) decryptGCM(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	e.mu.RLock()
	gcm := e.gcmCache
	e.mu.RUnlock()

	if gcm == nil {
		return "", fmt.Errorf("GCM not initialized")
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// encryptCBC AES-CBC加密
func (e *AESEncryptor) encryptCBC(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	// 填充
	plainData := pkcs7Padding([]byte(plaintext), aes.BlockSize)

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(plainData))
	mode.CryptBlocks(ciphertext, plainData)

	// 将IV和密文组合
	result := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// decryptCBC AES-CBC解密
func (e *AESEncryptor) decryptCBC(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	if len(data) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := data[:aes.BlockSize]
	ciphertextBytes := data[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertextBytes))
	mode.CryptBlocks(plaintext, ciphertextBytes)

	// 去除填充
	plaintext = pkcs7UnPadding(plaintext)
	return string(plaintext), nil
}

// pkcs7Padding PKCS7填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// pkcs7UnPadding PKCS7去填充
func pkcs7UnPadding(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}
