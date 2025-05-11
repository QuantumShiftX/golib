package ossx

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	configx "github.com/QuantumShiftX/golib/ossx/config"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// ossStorage 实现阿里云OSS的存储接口（使用SDK v2）
type ossStorage struct {
	client     *oss.Client
	bucketName string
	endpoint   string
	cdnDomain  string
}

// newOSSStorage 创建新的阿里云OSS存储实例（使用SDK v2）
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

	// 标准化 endpoint 格式
	endpoint := sc.Region
	if !strings.Contains(endpoint, ".aliyuncs.com") {
		// 如果只提供了区域ID，需要构建完整的endpoint
		if strings.HasPrefix(endpoint, "oss-") {
			endpoint = endpoint + ".aliyuncs.com"
		} else {
			endpoint = "oss-" + endpoint + ".aliyuncs.com"
		}
	}

	// 创建凭证提供者
	cred := credentials.NewStaticCredentialsProvider(sc.AccessKey, sc.SecretKey)

	// 创建配置
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(cred).
		WithRegion(sc.Region)

	// 创建客户端
	client := oss.NewClient(cfg)

	return &ossStorage{
		client:     client,
		bucketName: sc.Bucket,
		endpoint:   endpoint,
		cdnDomain:  sc.CdnDomain,
	}, nil
}

// Upload 实现Storage接口的上传方法（使用SDK v2）
func (s *ossStorage) Upload(ctx context.Context, file io.Reader, path, contentType string) (string, error) {
	// 标准化路径，处理前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 创建上传请求
	request := &oss.PutObjectRequest{
		Bucket:      oss.Ptr(s.bucketName),
		Key:         oss.Ptr(path),
		Body:        file,
		ContentType: oss.Ptr(contentType),
		//Acl:         oss.ObjectACLPublicRead,
	}

	// 执行文件上传
	_, err := s.client.PutObject(ctx, request)
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

// Delete 实现Storage接口的删除方法（使用SDK v2）
func (s *ossStorage) Delete(ctx context.Context, path string) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 删除指定的文件
	_, err := s.client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(s.bucketName),
		Key:    oss.Ptr(path),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from OSS: %w", err)
	}
	return nil
}

// CreateSignedURL 创建带签名的临时访问URL（使用SDK v2）
func (s *ossStorage) CreateSignedURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 计算过期时间（绝对时间）
	expirationTime := time.Now().Add(expiration)

	// 生成带签名的URL
	result, err := s.client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(s.bucketName),
		Key:    oss.Ptr(path),
	}, func(opts *oss.PresignOptions) {
		opts.Expiration = expirationTime
	})

	if err != nil {
		return "", fmt.Errorf("failed to create signed URL: %w", err)
	}

	return result.URL, nil
}

// GetObjectMeta 获取对象元数据（使用SDK v2）
func (s *ossStorage) GetObjectMeta(ctx context.Context, path string) (*oss.HeadObjectResult, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 获取对象元数据
	result, err := s.client.HeadObject(ctx, &oss.HeadObjectRequest{
		Bucket: oss.Ptr(s.bucketName),
		Key:    oss.Ptr(path),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	return result, nil
}

// ListObjects 列出指定前缀的对象（使用SDK v2）
func (s *ossStorage) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]string, error) {
	// 标准化前缀
	prefix = strings.TrimPrefix(prefix, "/")

	// 列出对象
	result, err := s.client.ListObjectsV2(ctx, &oss.ListObjectsV2Request{
		Bucket:  oss.Ptr(s.bucketName),
		Prefix:  oss.Ptr(prefix),
		MaxKeys: maxKeys,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// 提取对象键
	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

// CopyObject 在OSS内部复制对象（使用SDK v2）
func (s *ossStorage) CopyObject(ctx context.Context, srcPath, destPath string) error {
	// 标准化路径
	srcPath = strings.TrimPrefix(srcPath, "/")
	destPath = strings.TrimPrefix(destPath, "/")

	// 执行复制操作
	_, err := s.client.CopyObject(ctx, &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(s.bucketName),
		Key:          oss.Ptr(destPath),
		SourceKey:    oss.Ptr(srcPath),
		SourceBucket: oss.Ptr(s.bucketName),
	})

	if err != nil {
		return fmt.Errorf("failed to copy object within OSS: %w", err)
	}

	return nil
}

// UploadLargeFile 分片上传大文件（使用SDK v2）
func (s *ossStorage) UploadLargeFile(ctx context.Context, filePath, objectKey string, partSize int64) (string, error) {
	// 标准化路径
	objectKey = strings.TrimPrefix(objectKey, "/")

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 初始化分片上传
	initResult, err := s.client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(s.bucketName),
		Key:    oss.Ptr(objectKey),
	})

	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// 计算分片数量
	var partNumber int32 = 0
	// 收集上传的分片信息
	parts := make([]oss.UploadPart, 0)

	// 分片上传
	for offset := int64(0); offset < fileSize; offset += partSize {
		partNumber++

		// 计算当前分片大小
		currentPartSize := partSize
		if offset+partSize > fileSize {
			currentPartSize = fileSize - offset
		}

		// 设置文件读取位置
		_, err = file.Seek(offset, 0)
		if err != nil {
			// 如果失败，取消分片上传
			s.client.AbortMultipartUpload(ctx, &oss.AbortMultipartUploadRequest{
				Bucket:   oss.Ptr(s.bucketName),
				Key:      oss.Ptr(objectKey),
				UploadId: initResult.UploadId,
			})
			return "", fmt.Errorf("failed to seek file: %w", err)
		}

		// 上传分片
		partResult, err := s.client.UploadPart(ctx, &oss.UploadPartRequest{
			Bucket:     oss.Ptr(s.bucketName),
			Key:        oss.Ptr(objectKey),
			UploadId:   initResult.UploadId,
			PartNumber: partNumber,
			Body:       io.LimitReader(file, currentPartSize),
		})

		if err != nil {
			// 如果失败，取消分片上传
			s.client.AbortMultipartUpload(ctx, &oss.AbortMultipartUploadRequest{
				Bucket:   oss.Ptr(s.bucketName),
				Key:      oss.Ptr(objectKey),
				UploadId: initResult.UploadId,
			})
			return "", fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}
		// 创建分片信息字典
		partInfo := oss.UploadPart{
			PartNumber: partNumber,
			ETag:       partResult.ETag,
		}
		parts = append(parts, partInfo)
	}

	// 完成分片上传
	_, err = s.client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(s.bucketName),
		Key:      oss.Ptr(objectKey),
		UploadId: initResult.UploadId,
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: parts,
		},
	})

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

// ProcessImage 使用OSS图片处理服务处理图片（使用SDK v2）
func (s *ossStorage) ProcessImage(ctx context.Context, objectKey string, process string) (string, error) {
	// 标准化路径
	objectKey = strings.TrimPrefix(objectKey, "/")

	// 构建图片处理参数
	processKey := fmt.Sprintf("%s.processed", objectKey)

	// 使用标准库的base64编码
	processedKeyBase64 := base64.URLEncoding.EncodeToString([]byte(processKey))

	// 处理图片并保存结果
	_, err := s.client.ProcessObject(ctx, &oss.ProcessObjectRequest{
		Bucket:  oss.Ptr(s.bucketName),
		Key:     oss.Ptr(objectKey),
		Process: oss.Ptr(fmt.Sprintf("%s|sys/saveas,o_%s", process, processedKeyBase64)),
	})

	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// 生成处理后图片的URL
	var fileURL string
	if s.cdnDomain != "" {
		if strings.HasPrefix(s.cdnDomain, "http") {
			fileURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), processKey)
		} else {
			fileURL = fmt.Sprintf("https://%s/%s", strings.TrimSuffix(s.cdnDomain, "/"), processKey)
		}
	} else {
		fileURL = fmt.Sprintf("https://%s.%s/%s", s.bucketName, s.endpoint, processKey)
	}

	return fileURL, nil
}

// BatchDeleteObjects 批量删除对象（使用SDK v2）
func (s *ossStorage) BatchDeleteObjects(ctx context.Context, keys []string) error {
	// 标准化所有路径
	var objectKeys []oss.DeleteObject
	for _, key := range keys {
		key = strings.TrimPrefix(key, "/")
		objectKeys = append(objectKeys, oss.DeleteObject{Key: oss.Ptr(key)})
	}

	// 执行批量删除
	_, err := s.client.DeleteMultipleObjects(ctx, &oss.DeleteMultipleObjectsRequest{
		Bucket:  oss.Ptr(s.bucketName),
		Objects: objectKeys,
		Quiet:   true,
	})

	if err != nil {
		return fmt.Errorf("failed to batch delete objects: %w", err)
	}

	return nil
}
