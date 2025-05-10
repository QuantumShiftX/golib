package config

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

// StorageConfig 存储配置
type StorageConfig struct {
	// 存储类型: local, s3, oss, cos 等
	Type string `json:"type,omitempty"`
	// 本地存储路径或云存储 bucket
	Bucket string `json:"bucket,omitempty"`
	// 访问密钥
	AccessKey string `json:"access_key,omitempty"`
	// 密钥
	SecretKey string `json:"secret_key,omitempty"`
	// 区域或Endpoint
	Region string `json:"region,omitempty"`
	// CDN域名（可选）
	CdnDomain string `json:"cdn_domain,omitempty"`
	// 基础存储路径
	BasePath string `json:"base_path,omitempty"`
	// 上传配置
	UploadConfig *UploadConfig `json:"upload_config,omitempty"`
}

// UploadConfig 上传限制配置
type UploadConfig struct {
	// 最大文件大小（字节）
	MaxSize int64 `json:"max_size"`
	// 允许的文件类型（MIME类型）
	AllowedTypes map[string]bool `json:"allowed_types"`
	// 基础上传路径
	BasePath string `json:"base_path"`
	// 路径生成器函数
	PathGenerator func(userId int64, fileType string, fileName string) string `json:"-"`
}

// NewDefaultUploadConfig 创建默认的上传配置
func NewDefaultUploadConfig() *UploadConfig {
	return &UploadConfig{
		AllowedTypes: map[string]bool{
			"image/jpeg":         true,
			"image/png":          true,
			"image/gif":          true,
			"image/webp":         true,
			"video/mp4":          true,
			"application/pdf":    true,
			"application/msword": true,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		},
		MaxSize:  100 * 1024 * 1024, // 100MB
		BasePath: "uploads",
		PathGenerator: func(userId int64, fileType string, fileName string) string {
			// 默认路径格式: uploads/202401/images/123/filename.jpg
			return fmt.Sprintf("%s/%s/%s/%d/%s",
				"uploads",
				time.Now().Format("200601"),
				fileType,
				userId,
				fileName,
			)
		},
	}
}

// ValidateFile 验证文件是否符合上传配置
func (c *UploadConfig) ValidateFile(header *multipart.FileHeader) error {
	// 检查文件大小
	if header.Size > c.MaxSize {
		return fmt.Errorf("文件大小超过限制，最大允许 %d 字节", c.MaxSize)
	}

	// 检查文件类型
	contentType := header.Header.Get("Content-Type")
	if !c.AllowedTypes[contentType] {
		return fmt.Errorf("不支持的文件类型: %s", contentType)
	}

	return nil
}

// DetectFileType 根据文件MIME类型检测文件分类
func DetectFileType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "images"
	case strings.HasPrefix(contentType, "video/"):
		return "videos"
	case strings.HasPrefix(contentType, "audio/"):
		return "audios"
	case strings.HasPrefix(contentType, "application/pdf"):
		return "documents"
	case strings.Contains(contentType, "word") || strings.Contains(contentType, "excel") || strings.Contains(contentType, "powerpoint"):
		return "documents"
	case strings.Contains(contentType, "text/"):
		return "texts"
	case strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "application/xml"):
		return "data"
	case strings.Contains(contentType, "application/zip") ||
		strings.Contains(contentType, "application/x-rar") ||
		strings.Contains(contentType, "application/x-gzip"):
		return "archives"
	default:
		return "others"
	}
}

// GetExtensionFromFilename 从文件名获取扩展名
func GetExtensionFromFilename(filename string) string {
	return filepath.Ext(filename)
}

// GenerateTimestampFilename 生成带时间戳的文件名
func GenerateTimestampFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%d%s", timestamp, ext)
}
