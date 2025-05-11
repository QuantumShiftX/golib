package ossx

import (
	"context"
	"fmt"
	configx "github.com/QuantumShiftX/golib/ossx/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// s3Storage 实现AWS S3的存储接口
type s3Storage struct {
	client    *s3.Client
	uploader  *manager.Uploader
	bucket    string
	region    string
	cdnDomain string // CDN域名（可选）
}

// newS3Storage 创建新的S3存储实例
func newS3Storage(sc configx.StorageConfig) (Storage, error) {
	if sc.AccessKey == "" || sc.SecretKey == "" {
		return nil, fmt.Errorf("s3 credentials are required")
	}
	if sc.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if sc.Region == "" {
		return nil, fmt.Errorf("s3 region is required")
	}

	// 创建AWS配置
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(sc.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			sc.AccessKey,
			sc.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	// 创建S3客户端
	client := s3.NewFromConfig(cfg)

	// 创建上传管理器
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024 // 5MB分片
		u.Concurrency = 3            // 3个并发上传
	})

	return &s3Storage{
		client:    client,
		uploader:  uploader,
		bucket:    sc.Bucket,
		region:    sc.Region,
		cdnDomain: sc.CdnDomain,
	}, nil
}

// Upload 实现Storage接口的上传方法
func (s *s3Storage) Upload(ctx context.Context, file io.Reader, path, contentType string) (string, error) {
	// 标准化路径，处理前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 使用上传管理器上传文件
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(path),
		Body:        file,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead, // 默认为公共可读
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
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
		// 使用S3默认域名生成URL
		fileURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, path)
	}

	return fileURL, nil
}

// Delete 实现Storage接口的删除方法
func (s *s3Storage) Delete(ctx context.Context, path string) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 删除指定的文件
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// CreateSignedURL 创建预签名URL（扩展功能）
func (s *s3Storage) CreateSignedURL(ctx context.Context, path string, expiration time.Duration) (string, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 创建预签名客户端
	presignClient := s3.NewPresignClient(s.client)

	// 创建GetObject请求的预签名URL
	presignResult, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", fmt.Errorf("failed to create presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

// GetObjectInfo 获取对象信息（扩展功能）
func (s *s3Storage) GetObjectInfo(ctx context.Context, path string) (*s3.HeadObjectOutput, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 获取对象元数据
	headOutput, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return headOutput, nil
}

// ListObjects 列出指定前缀的对象（扩展功能）
func (s *s3Storage) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]string, error) {
	// 标准化前缀
	prefix = strings.TrimPrefix(prefix, "/")

	// 列出对象
	listOutput, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// 提取对象键
	keys := make([]string, 0, len(listOutput.Contents))
	for _, obj := range listOutput.Contents {
		keys = append(keys, *obj.Key)
	}

	return keys, nil
}

// CopyObject 在S3内部复制对象（扩展功能）
func (s *s3Storage) CopyObject(ctx context.Context, srcPath, destPath string) error {
	// 标准化路径
	srcPath = strings.TrimPrefix(srcPath, "/")
	destPath = strings.TrimPrefix(destPath, "/")

	// 构建源对象的完整路径
	srcObject := fmt.Sprintf("%s/%s", s.bucket, srcPath)

	// 执行复制操作
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(srcObject),
		Key:        aws.String(destPath),
		ACL:        types.ObjectCannedACLPublicRead, // 默认为公共可读
	})

	if err != nil {
		return fmt.Errorf("failed to copy object within S3: %w", err)
	}

	return nil
}

// UploadDirectory 上传整个目录（扩展功能）
func (s *s3Storage) UploadDirectory(ctx context.Context, localDir, remotePath string) error {
	// 确保远程路径以斜杠结尾
	if !strings.HasSuffix(remotePath, "/") {
		remotePath = remotePath + "/"
	}

	// 遍历本地目录
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

// DeleteObjects 批量删除对象（扩展功能）
func (s *s3Storage) DeleteObjects(ctx context.Context, paths []string) error {
	// 如果路径列表为空，直接返回
	if len(paths) == 0 {
		return nil
	}

	// 构建要删除的对象列表
	objectIds := make([]types.ObjectIdentifier, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimPrefix(path, "/")
		objectIds = append(objectIds, types.ObjectIdentifier{
			Key: aws.String(path),
		})
	}

	// 执行批量删除
	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{
			Objects: objectIds,
			Quiet:   aws.Bool(true),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete objects from S3: %w", err)
	}

	return nil
}

// SetObjectACL 设置对象访问权限（扩展功能）
func (s *s3Storage) SetObjectACL(ctx context.Context, path string, acl types.ObjectCannedACL) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 设置对象ACL
	_, err := s.client.PutObjectAcl(ctx, &s3.PutObjectAclInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		ACL:    acl,
	})

	if err != nil {
		return fmt.Errorf("failed to set object ACL: %w", err)
	}

	return nil
}
