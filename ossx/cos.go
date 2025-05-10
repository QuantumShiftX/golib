package ossx

import (
	"context"
	"errors"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CosStorage 实现腾讯云COS存储
type CosStorage struct {
	client     *cos.Client
	bucket     string
	region     string
	requestURL string
}

type CosStorageConfig struct {
	// bucket地址 用于解析appid bucket_name
	BucketURL string `json:"bucket_url"`
	// 请求地址 用于请求图片
	RequestURL string `json:"request_url"`
	// 用户 secret_id
	SecretID string `json:"secret_id"`
	// 密钥
	SecretKey string `json:"secret_key"`
}

// NewCosStorage 创建腾讯云COS存储实例
func NewCosStorage(c CosStorageConfig) (*CosStorage, error) {
	// 参数校验
	if c.SecretID == "" {
		return nil, errors.New("cos secret_id is required")
	}
	if c.SecretKey == "" {
		return nil, errors.New("cos secret_key is required")
	}
	if c.BucketURL == "" {
		return nil, errors.New("cos bucket_url is required")
	}

	// 解析Bucket URL
	u, err := url.Parse(c.BucketURL)
	if err != nil {
		return nil, fmt.Errorf("cos parse bucket_url fail: %w", err)
	}

	// 从URL提取bucket和region信息
	parts := strings.Split(u.Host, ".")
	var bucket, region string
	if len(parts) >= 1 {
		bucket = parts[0]
	}
	if len(parts) >= 3 {
		region = parts[2]
	}

	// 创建COS客户端
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.SecretID,
			SecretKey: c.SecretKey,
		},
	})

	// 如果没有提供RequestURL，则使用BucketURL
	requestURL := c.RequestURL
	if requestURL == "" {
		requestURL = c.BucketURL
	}

	return &CosStorage{
		client:     client,
		bucket:     bucket,
		region:     region,
		requestURL: requestURL,
	}, nil
}

// Upload 实现Storage接口的上传方法
func (s *CosStorage) Upload(ctx context.Context, file io.Reader, path, contentType string) (string, error) {
	// 标准化路径，处理前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 创建上传选项
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: contentType,
		},
	}

	// 执行文件上传
	_, err := s.client.Object.Put(ctx, path, file, opt)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to cos: %w", err)
	}

	// 生成访问URL
	var fileURL string
	if strings.HasPrefix(s.requestURL, "http") {
		// 如果requestURL已经有协议前缀，直接拼接
		fileURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(s.requestURL, "/"), path)
	} else {
		// 否则添加https前缀
		fileURL = fmt.Sprintf("https://%s/%s", strings.TrimSuffix(s.requestURL, "/"), path)
	}

	return fileURL, nil
}

// Delete 实现Storage接口的删除方法
func (s *CosStorage) Delete(ctx context.Context, path string) error {
	// 标准化路径，处理前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 执行删除操作
	_, err := s.client.Object.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete file from cos: %w", err)
	}
	return nil
}

// GetObjectInfo 获取对象信息（扩展功能）
func (s *CosStorage) GetObjectInfo(ctx context.Context, path string) (*cos.Object, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 列出对象（只列出这一个）
	opt := &cos.BucketGetOptions{
		Prefix:  path,
		MaxKeys: 1,
	}

	result, _, err := s.client.Bucket.Get(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	// 检查是否找到对象
	if len(result.Contents) == 0 {
		return nil, fmt.Errorf("object not found: %s", path)
	}

	// 返回找到的对象信息
	return &result.Contents[0], nil
}

// CreateSignedURL 创建带签名的临时访问URL（扩展功能）
func (s *CosStorage) CreateSignedURL(ctx context.Context, path string, expiration int64) (string, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 创建带签名的临时URL
	signedURL, err := s.client.Object.GetPresignedURL(ctx, http.MethodGet, path, s.client.GetCredential().SecretID, s.client.GetCredential().SecretKey, time.Duration(expiration), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create signed URL: %w", err)
	}

	return signedURL.String(), nil
}

// UploadDirectory 上传整个目录（扩展功能）
func (s *CosStorage) UploadDirectory(ctx context.Context, localDir, remotePath string) error {
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// 构建远程路径
		remoteFilePath := filepath.Join(remotePath, relPath)
		remoteFilePath = filepath.ToSlash(remoteFilePath) // 确保使用正斜杠

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		// 检测内容类型
		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// 上传文件
		_, err = s.Upload(ctx, file, remoteFilePath, contentType)
		if err != nil {
			return fmt.Errorf("failed to upload file %s: %w", path, err)
		}

		return nil
	})
}

// ListObjects 列出指定前缀的对象（扩展功能）
func (s *CosStorage) ListObjects(ctx context.Context, prefix string, maxKeys int) ([]string, error) {
	// 标准化前缀
	prefix = strings.TrimPrefix(prefix, "/")

	// 设置列表选项
	opt := &cos.BucketGetOptions{
		Prefix:  prefix,
		MaxKeys: maxKeys,
	}

	// 获取对象列表
	result, _, err := s.client.Bucket.Get(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// 提取对象键
	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		keys = append(keys, obj.Key)
	}

	return keys, nil
}
