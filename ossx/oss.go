package ossx

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	configx "github.com/QuantumShiftX/golib/ossx/config"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// ossStorage 实现阿里云OSS的存储接口
type ossStorage struct {
	client     *oss.Client
	bucket     *oss.Bucket
	bucketName string
	endpoint   string
	cdnDomain  string // CDN域名（可选）
}

// newOSSStorage 创建新的阿里云OSS存储实例
func newOSSStorage(sc configx.StorageConfig) (Storage, error) {
	if sc.AccessKey == "" || sc.SecretKey == "" {
		return nil, fmt.Errorf("oss credentials are required")
	}
	if sc.Bucket == "" {
		return nil, fmt.Errorf("oss bucket is required")
	}
	if sc.Region == "" {
		return nil, fmt.Errorf("oss endpoint is required")
	}

	// 确保endpoint格式正确
	endpoint := sc.Region
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}

	// 提取endpoint的主机部分
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid oss endpoint: %w", err)
	}
	endpoint = endpointURL.Host

	// 创建OSS客户端实例
	client, err := oss.New(endpoint, sc.AccessKey, sc.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	// 获取存储空间
	bucket, err := client.Bucket(sc.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get OSS bucket: %w", err)
	}

	return &ossStorage{
		client:     client,
		bucket:     bucket,
		bucketName: sc.Bucket,
		endpoint:   endpoint,
		cdnDomain:  sc.CdnDomain,
	}, nil
}

// Upload 实现Storage接口的上传方法
func (s *ossStorage) Upload(ctx context.Context, file io.Reader, path, contentType string) (string, error) {
	// 标准化路径，处理前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 定义上传选项
	options := []oss.Option{
		oss.ContentType(contentType),
		oss.ObjectACL(oss.ACLPublicRead), // 默认为公共读取权限
	}

	// 执行文件上传
	err := s.bucket.PutObject(path, file, options...)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to OSS: %w", err)
	}

	// 生成文件URL
	var fileURL string
	if s.cdnDomain != "" {
		// 使用CDN域名生成URL
		if strings.HasPrefix(s.cdnDomain, "http") {
			fileURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), path)
		} else {
			fileURL = fmt.Sprintf("https://%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), path)
		}
	} else {
		// 使用OSS默认域名生成URL
		fileURL = fmt.Sprintf("https://%s.%s/%s", s.bucketName, s.endpoint, path)
	}

	return fileURL, nil
}

// Delete 实现Storage接口的删除方法
func (s *ossStorage) Delete(ctx context.Context, path string) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 删除指定的文件
	err := s.bucket.DeleteObject(path)
	if err != nil {
		return fmt.Errorf("failed to delete file from OSS: %w", err)
	}
	return nil
}

// CreateSignedURL 创建带签名的临时访问URL（扩展功能）
func (s *ossStorage) CreateSignedURL(path string, expiration time.Duration) (string, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 生成带签名的URL
	signedURL, err := s.bucket.SignURL(path, oss.HTTPGet, int64(expiration.Seconds()))
	if err != nil {
		return "", fmt.Errorf("failed to create signed URL: %w", err)
	}

	return signedURL, nil
}

// GetObjectMeta 获取对象元数据（扩展功能）
func (s *ossStorage) GetObjectMeta(path string) (http.Header, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 获取对象元数据 - OSS SDK返回的是http.Header
	meta, err := s.bucket.GetObjectMeta(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	return meta, nil
}

// ListObjects 列出指定前缀的对象（扩展功能）
func (s *ossStorage) ListObjects(prefix string, maxKeys int) ([]string, error) {
	// 标准化前缀
	prefix = strings.TrimPrefix(prefix, "/")

	// 列出对象
	result, err := s.bucket.ListObjects(oss.Prefix(prefix), oss.MaxKeys(maxKeys))
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// 提取对象键
	keys := make([]string, 0, len(result.Objects))
	for _, obj := range result.Objects {
		keys = append(keys, obj.Key)
	}

	return keys, nil
}

// CopyObject 在OSS内部复制对象（扩展功能）
func (s *ossStorage) CopyObject(srcPath, destPath string) error {
	// 标准化路径
	srcPath = strings.TrimPrefix(srcPath, "/")
	destPath = strings.TrimPrefix(destPath, "/")

	// 执行复制操作
	_, err := s.bucket.CopyObject(srcPath, destPath)
	if err != nil {
		return fmt.Errorf("failed to copy object within OSS: %w", err)
	}

	return nil
}

// MultipartUpload 分片上传大文件（扩展功能）
func (s *ossStorage) MultipartUpload(filePath, objectKey string, partSize int64) (string, error) {
	// 标准化路径
	objectKey = strings.TrimPrefix(objectKey, "/")

	// 初始化分片上传
	imur, err := s.bucket.InitiateMultipartUpload(objectKey)
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// 计算分片数量
	chunks := int(fileSize / partSize)
	if fileSize%partSize != 0 {
		chunks++
	}

	// 创建一个分片上传的UploadParts通道
	uploadPartsChan := make(chan oss.UploadPart, chunks)
	failPointChan := make(chan error, chunks)

	// 启动多个goroutine并发上传分片
	for i := 0; i < chunks; i++ {
		partNumber := i + 1
		start := int64(i) * partSize
		end := start + partSize
		if end > fileSize {
			end = fileSize
		}

		go func(partNumber int, start, end int64) {
			// 打开文件
			fd, err := os.Open(filePath)
			if err != nil {
				failPointChan <- fmt.Errorf("failed to open file: %w", err)
				return
			}
			defer fd.Close()

			// 移动到指定位置
			fd.Seek(start, os.SEEK_SET)

			// 上传分片
			part, err := s.bucket.UploadPart(imur, fd, end-start, partNumber)
			if err != nil {
				failPointChan <- fmt.Errorf("failed to upload part %d: %w", partNumber, err)
				return
			}

			uploadPartsChan <- part
		}(partNumber, start, end)
	}

	// 等待所有分片上传完成
	parts := make([]oss.UploadPart, chunks)
	for i := 0; i < chunks; i++ {
		select {
		case part := <-uploadPartsChan:
			parts[part.PartNumber-1] = part
		case err := <-failPointChan:
			// 取消分片上传
			s.bucket.AbortMultipartUpload(imur)
			return "", err
		}
	}

	// 完成分片上传
	_, err = s.bucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	// 生成文件URL
	var fileURL string
	if s.cdnDomain != "" {
		if strings.HasPrefix(s.cdnDomain, "http") {
			fileURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), objectKey)
		} else {
			fileURL = fmt.Sprintf("https://%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), objectKey)
		}
	} else {
		fileURL = fmt.Sprintf("https://%s.%s/%s", s.bucketName, s.endpoint, objectKey)
	}

	return fileURL, nil
}

// ProcessImage 使用OSS图片处理服务处理图片（扩展功能）
func (s *ossStorage) ProcessImage(objectKey string, process string) (string, error) {
	// 标准化路径
	objectKey = strings.TrimPrefix(objectKey, "/")

	// 处理图片并保存结果
	process = fmt.Sprintf("image/%s", process)

	// 创建图片处理的样式
	style := fmt.Sprintf("%s|sys/saveas,o_%s",
		process,
		base64.URLEncoding.EncodeToString([]byte(objectKey+".processed")))

	// 处理图片
	_, err := s.bucket.ProcessObject(objectKey, style)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// 根据OSS SDK实际返回结构构建URL
	processedObjectKey := objectKey + ".processed"
	var fileURL string
	if s.cdnDomain != "" {
		if strings.HasPrefix(s.cdnDomain, "http") {
			fileURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), processedObjectKey)
		} else {
			fileURL = fmt.Sprintf("https://%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), processedObjectKey)
		}
	} else {
		fileURL = fmt.Sprintf("https://%s.%s/%s", s.bucketName, s.endpoint, processedObjectKey)
	}

	return fileURL, nil
}
