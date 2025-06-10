package crypto

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"sync"
)

// Manager 加密管理器
type Manager struct {
	services map[string]*XCryptoService
	mu       sync.RWMutex
}

// NewManager 创建加密管理器
func NewManager() *Manager {
	return &Manager{
		services: make(map[string]*XCryptoService),
	}
}

// RegisterService 注册加密服务
func (m *Manager) RegisterService(name string, service *XCryptoService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[name] = service
}

// GetService 获取加密服务
func (m *Manager) GetService(name string) (*XCryptoService, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	service, exists := m.services[name]
	return service, exists
}

// GetDefaultService 获取默认加密服务
func (m *Manager) GetDefaultService() (*XCryptoService, bool) {
	return m.GetService("default")
}

// ListServices 列出所有服务
func (m *Manager) ListServices() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.services))
	for name := range m.services {
		names = append(names, name)
	}
	return names
}

// RemoveService 移除服务
func (m *Manager) RemoveService(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.services, name)
}

// RegisterAESGCM 注册AES-GCM服务
func (m *Manager) RegisterAESGCM(name, key string, debug bool) error {
	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		return err
	}

	service := NewCryptoService(encryptor, debug)
	m.RegisterService(name, service)
	return nil
}

// RegisterAESCBC 注册AES-CBC服务
func (m *Manager) RegisterAESCBC(name, key string, debug bool) error {
	encryptor, err := NewAESCBCEncryptor(key)
	if err != nil {
		return err
	}

	service := NewCryptoService(encryptor, debug)
	m.RegisterService(name, service)
	return nil
}

// 全局管理器实例
var (
	globalManager = NewManager()
	globalMu      sync.RWMutex
)

// GetGlobalManager 获取全局管理器
func GetGlobalManager() *Manager {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalManager
}

// RegisterGlobalAESGCM 注册全局AES-GCM服务
func RegisterGlobalAESGCM(key string, debug bool) error {
	return globalManager.RegisterAESGCM("default", key, debug)
}

// RegisterGlobalAESCBC 注册全局AES-CBC服务
func RegisterGlobalAESCBC(key string, debug bool) error {
	return globalManager.RegisterAESCBC("default", key, debug)
}

// GetGlobalService 获取全局加密服务
func GetGlobalService() (*XCryptoService, error) {
	service, exists := globalManager.GetDefaultService()
	if !exists {
		return nil, fmt.Errorf("no default crypto service registered")
	}
	return service, nil
}

// 便捷函数
func QuickEncrypt(data interface{}) (*EncryptedData, error) {
	service, err := GetGlobalService()
	if err != nil {
		return nil, err
	}
	return service.EncryptJSON(data)
}

func QuickDecrypt(encryptedData *EncryptedData, target interface{}) error {
	service, err := GetGlobalService()
	if err != nil {
		return err
	}
	return service.DecryptJSON(encryptedData, target)
}

func QuickEncryptString(plaintext string) (string, error) {
	service, err := GetGlobalService()
	if err != nil {
		return "", err
	}
	return service.EncryptString(plaintext)
}

func QuickDecryptString(ciphertext string) (string, error) {
	service, err := GetGlobalService()
	if err != nil {
		return "", err
	}
	return service.DecryptString(ciphertext)
}

// IsEncryptedFormat 检查是否为加密格式
func IsEncryptedFormat(data []byte) bool {
	var encryptedData EncryptedData
	if err := jsonx.Unmarshal(data, &encryptedData); err != nil {
		logx.Errorf("jsonx.Unmarshal failed: %v", err)
		return false
	}
	return encryptedData.Encrypted && encryptedData.Data != ""
}

// EncryptRequest 加密请求数据
func EncryptRequest(data interface{}) ([]byte, error) {
	service, err := GetGlobalService()
	if err != nil {
		return nil, err
	}

	encryptedData, err := service.EncryptJSON(data)
	if err != nil {
		return nil, err
	}

	return jsonx.Marshal(map[string]interface{}{
		"data": encryptedData.Data,
	})
}

// DecryptRequest 解密请求数据
func DecryptRequest(encryptedBytes []byte, target interface{}) error {
	var encryptedData EncryptedData

	if err := jsonx.Unmarshal(encryptedBytes, &encryptedData); err != nil {
		return err
	}

	service, err := GetGlobalService()
	if err != nil {
		return err
	}

	return service.DecryptJSON(&encryptedData, target)
}
