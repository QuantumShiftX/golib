package ossx

import (
	"context"
	"fmt"
	configx "github.com/QuantumShiftX/golib/ossx/config"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	Local      = "local"
	AmazonS3   = "s3"
	ALiYunOSS  = "oss"
	TencentCOS = "cos"
)

// Storage 存储接口
type Storage interface {
	Upload(ctx context.Context, file io.Reader, path, contentType string) (string, error)
	Delete(ctx context.Context, path string) error
	// 获取签名URL的方法
	CreateSignedURL(ctx context.Context, path string, expiration time.Duration) (string, error)
}

// UploadResult 文件上传结果
type UploadResult struct {
	// 完整URL (原始URL，可能无法直接访问)
	URL string `json:"url"`
	// 签名URL (可以直接访问的URL)
	SignedURL string `json:"signed_url"`
	// 相对路径
	RelativePath string `json:"relative_path"`
	// 文件名
	FileName string `json:"file_name"`
	// 文件类型
	FileType string `json:"file_type"`
	// 文件大小
	Size int64 `json:"size"`
	// 存储类型
	StorageType string `json:"storage_type"`
	// 签名URL过期时间（Unix时间戳）
	SignedURLExpire int64 `json:"signed_url_expire,omitempty"`
}

var (
	once     sync.Once
	Uploader *UploadManager
)

// UploadManager 上传管理器
type UploadManager struct {
	mu           sync.RWMutex
	configs      map[string]configx.StorageConfig
	storages     map[string]Storage
	uploadConfig *configx.UploadConfig
	errors       []error
}

// Must 初始化上传管理器
func Must(configs ...configx.StorageConfig) {
	var err error

	once.Do(func() {
		Uploader = &UploadManager{
			configs:      make(map[string]configx.StorageConfig),
			storages:     make(map[string]Storage),
			uploadConfig: configx.NewDefaultUploadConfig(),
			errors:       make([]error, 0),
		}
		for _, c := range configs {
			if err = Uploader.addStorage(c); err != nil {
				Uploader.errors = append(Uploader.errors, err)
			}

			// 如果配置中有上传配置，则使用配置中的
			if c.UploadConfig != nil {
				Uploader.uploadConfig = c.UploadConfig
			}
		}
	})

	if len(Uploader.errors) > 0 {
		panic(fmt.Sprintf("errors initializing storages: %v", Uploader.errors))
	}
}

// SetUploadConfig 设置上传配置
func (u *UploadManager) SetUploadConfig(config *configx.UploadConfig) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.uploadConfig = config
}

func (u *UploadManager) addStorage(cfg configx.StorageConfig) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if cfg.Type == "" || cfg.Bucket == "" {
		return fmt.Errorf("invalid storage configuration for type %s", cfg.Type)
	}

	u.configs[cfg.Type] = cfg
	var s Storage
	var err error

	switch cfg.Type {
	case AmazonS3:
		s, err = newS3Storage(cfg)
	case ALiYunOSS:
		s, err = newOSSStorage(cfg)
	case TencentCOS:
		s, err = newCOSStorage(cfg)
	case Local:
		s, err = newLocalStorage(cfg)
	default:
		return fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}

	if err != nil {
		return err
	}

	u.storages[cfg.Type] = s
	return nil
}

// newCOSStorage 创建腾讯云COS存储实例
func newCOSStorage(cfg configx.StorageConfig) (Storage, error) {
	cosConfig := CosStorageConfig{
		BucketURL:  fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region),
		RequestURL: cfg.CdnDomain,
		SecretID:   cfg.AccessKey,
		SecretKey:  cfg.SecretKey,
	}

	if cosConfig.RequestURL == "" {
		cosConfig.RequestURL = cosConfig.BucketURL
	}

	return NewCosStorage(cosConfig)
}

// Upload 上传文件 - 使用指定的存储类型
func (u *UploadManager) Upload(ctx context.Context, storageType string, file io.Reader, header *multipart.FileHeader, userId int64) (*UploadResult, error) {
	// 查找存储实例
	storage, ok := u.storages[storageType]
	if !ok {
		return nil, fmt.Errorf("storage type %s not initialized", storageType)
	}

	// 验证文件
	if err := u.uploadConfig.ValidateFile(header); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// 获取文件信息
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = u.detectContentType(file, header.Filename)
	}

	// 获取文件类型分类
	fileType := configx.DetectFileType(contentType)

	// 生成唯一文件名
	fileName := uuid.New().String() + filepath.Ext(header.Filename)

	// 生成文件路径
	path := u.uploadConfig.PathGenerator(userId, fileType, fileName)

	// 重置文件读取位置（如果文件是 io.Seeker）
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to reset file reader: %w", err)
		}
	}

	// 执行上传操作
	url, err := storage.Upload(ctx, file, path, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// 生成签名URL（24小时有效期）
	signedURL, err := storage.CreateSignedURL(ctx, path, 24*time.Hour)
	if err != nil {
		// 如果生成签名URL失败，仍然返回原始结果，但记录错误
		logx.WithContext(ctx).Errorf("failed to generate signed URL: %v", err)
		signedURL = url // 使用原始URL作为fallback
	}

	// 返回上传结果
	result := &UploadResult{
		URL:          url,
		SignedURL:    signedURL,
		RelativePath: "/" + strings.TrimPrefix(path, "/"),
		FileName:     fileName,
		FileType:     fileType,
		Size:         header.Size,
		StorageType:  storageType,
	}

	// 如果是签名URL，添加过期时间
	if signedURL != url {
		result.SignedURLExpire = time.Now().Add(24 * time.Hour).Unix()
	}

	return result, nil
}

// UploadWithUid 直接使用userId上传文件（简化版）
func (u *UploadManager) UploadWithUid(ctx context.Context, storageType string, file multipart.File, header *multipart.FileHeader, userId int64) (*UploadResult, error) {
	return u.Upload(ctx, storageType, file, header, userId)
}

// Delete 删除文件
func (u *UploadManager) Delete(ctx context.Context, storageType string, path string) error {
	storage, ok := u.storages[storageType]
	if !ok {
		return fmt.Errorf("storage type %s not initialized", storageType)
	}
	return storage.Delete(ctx, path)
}

// GetSignedURL 为已存在的文件生成签名URL
func (u *UploadManager) GetSignedURL(ctx context.Context, storageType string, path string, expiration time.Duration) (string, error) {
	storage, ok := u.storages[storageType]
	if !ok {
		return "", fmt.Errorf("storage type %s not initialized", storageType)
	}

	return storage.CreateSignedURL(ctx, path, expiration)
}

// detectContentType 检测文件内容类型
func (u *UploadManager) detectContentType(file io.Reader, filename string) string {
	// 尝试通过文件扩展名来确定 MIME 类型
	ext := strings.ToLower(filepath.Ext(filename))
	ct := mime.TypeByExtension(ext)
	if ct != "" {
		return ct
	}

	// 读取文件的前 512 字节来检测 MIME 类型
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "application/octet-stream"
	}

	// 如果成功读取了数据，检测 MIME 类型
	if n > 0 {
		return http.DetectContentType(buf[:n])
	}

	// 如果读取的字节数为 0，无法检测类型，返回默认值
	return "application/octet-stream"
}
