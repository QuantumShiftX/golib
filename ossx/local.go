package ossx

import (
	"context"
	"fmt"
	configx "github.com/QuantumShiftX/golib/ossx/config"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// localStorage 实现本地文件系统存储
type localStorage struct {
	basePath  string
	cdnDomain string // CDN域名（可选）
}

// newLocalStorage 创建本地存储实例
func newLocalStorage(config configx.StorageConfig) (Storage, error) {
	if config.Bucket == "" {
		return nil, fmt.Errorf("local storage path is required")
	}

	// 确保存储目录存在
	if err := os.MkdirAll(config.Bucket, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &localStorage{
		basePath:  config.Bucket,
		cdnDomain: config.CdnDomain,
	}, nil
}

// Upload 实现Storage接口的上传方法
func (l *localStorage) Upload(ctx context.Context, file io.Reader, path, contentType string) (string, error) {
	// 标准化路径，处理前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 创建完整的路径，包括基础路径
	fullPath := filepath.Join(l.basePath, path)
	dir := filepath.Dir(fullPath)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// 写入文件内容
	if _, err := io.Copy(f, file); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 生成文件URL
	var url string
	if l.cdnDomain != "" {
		// 使用CDN域名
		if strings.HasPrefix(l.cdnDomain, "http") {
			url = fmt.Sprintf("%s/%s", strings.TrimSuffix(l.cdnDomain, "/"), path)
		} else {
			url = fmt.Sprintf("https://%s/%s", strings.TrimSuffix(l.cdnDomain, "/"), path)
		}
	} else {
		// 使用本地文件路径
		url = fmt.Sprintf("file://%s", fullPath)
	}

	return url, nil
}

// Delete 实现Storage接口的删除方法
func (l *localStorage) Delete(ctx context.Context, path string) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 检查文件是否存在
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，视为删除成功
			return nil
		}
		return fmt.Errorf("failed to check file existence: %w", err)
	}

	// 删除文件
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFileInfo 获取文件信息（扩展功能）
func (l *localStorage) GetFileInfo(path string) (os.FileInfo, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 获取文件信息
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return info, nil
}

// ListFiles 列出指定目录下的文件（扩展功能）
func (l *localStorage) ListFiles(path string) ([]string, error) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 确保目录存在
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// 列出文件
	files, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// 构建相对路径列表
	relativePaths := make([]string, 0, len(files))
	for _, file := range files {
		if !file.IsDir() {
			relativePath := filepath.Join(path, file.Name())
			relativePaths = append(relativePaths, relativePath)
		}
	}

	return relativePaths, nil
}

// CopyFile 复制文件（扩展功能）
func (l *localStorage) CopyFile(srcPath, destPath string) error {
	// 标准化路径
	srcPath = strings.TrimPrefix(srcPath, "/")
	destPath = strings.TrimPrefix(destPath, "/")

	// 构建完整路径
	srcFullPath := filepath.Join(l.basePath, srcPath)
	destFullPath := filepath.Join(l.basePath, destPath)

	// 确保目标目录存在
	destDir := filepath.Dir(destFullPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 打开源文件
	src, err := os.Open(srcFullPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(destFullPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

// MoveFile 移动文件（扩展功能）
func (l *localStorage) MoveFile(srcPath, destPath string) error {
	// 标准化路径
	srcPath = strings.TrimPrefix(srcPath, "/")
	destPath = strings.TrimPrefix(destPath, "/")

	// 构建完整路径
	srcFullPath := filepath.Join(l.basePath, srcPath)
	destFullPath := filepath.Join(l.basePath, destPath)

	// 确保目标目录存在
	destDir := filepath.Dir(destFullPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 移动文件
	if err := os.Rename(srcFullPath, destFullPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// WalkDirectory 遍历目录（扩展功能）
func (l *localStorage) WalkDirectory(path string, fn func(path string, info os.FileInfo) error) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 遍历目录
	return filepath.Walk(fullPath, func(walkPath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(l.basePath, walkPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// 跳过basePath本身
		if relPath == "." {
			return nil
		}

		// 调用用户函数
		return fn(relPath, info)
	})
}

// CreateDirectory 创建目录（扩展功能）
func (l *localStorage) CreateDirectory(path string) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 创建目录
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// DeleteDirectory 删除目录及其内容（扩展功能）
func (l *localStorage) DeleteDirectory(path string) error {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 检查路径是否存在
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 目录不存在，视为删除成功
			return nil
		}
		return fmt.Errorf("failed to check directory: %w", err)
	}

	// 确保是目录
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	// 删除目录及其内容
	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}

	return nil
}

// GetFileURL 获取文件URL（扩展功能）
func (l *localStorage) GetFileURL(path string) string {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 生成URL
	if l.cdnDomain != "" {
		// 使用CDN域名
		if strings.HasPrefix(l.cdnDomain, "http") {
			return fmt.Sprintf("%s/%s", strings.TrimSuffix(l.cdnDomain, "/"), path)
		}
		return fmt.Sprintf("https://%s/%s", strings.TrimSuffix(l.cdnDomain, "/"), path)
	}

	// 使用本地文件路径
	return fmt.Sprintf("file://%s", fullPath)
}

// ServeFile 提供HTTP文件服务（扩展功能）
func (l *localStorage) ServeFile(w http.ResponseWriter, r *http.Request, path string) {
	// 标准化路径
	path = strings.TrimPrefix(path, "/")

	// 构建完整路径
	fullPath := filepath.Join(l.basePath, path)

	// 使用http.ServeFile提供文件服务
	http.ServeFile(w, r, fullPath)
}
