package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TokensZhuanfa/dujiao-shop/internal/config"
	"github.com/google/uuid"
)

// 默认值（config 未显式配置时使用）
const (
	defaultCredentialsRoot    = "./credentials"
	defaultCredentialsMaxSize = int64(1 << 30) // 1 GB
)

// CredentialFileService 卡密文件私有存储服务。
//
// 与 UploadService 的区别：
//   - 写入到 ./credentials/<YYYY>/<MM>/<uuid><ext>，**不**挂在 r.Static("/uploads")
//     的根目录，外部不可直接通过 URL 访问。下载必须经鉴权后的 handler。
//   - 不限制文件类型 / 扩展名 / 尺寸维度（任何二进制都允许）。
//   - 单文件上限默认 1 GB，可经 config.Credentials.MaxSize 调整。
type CredentialFileService struct {
	root    string
	maxSize int64
}

// NewCredentialFileService 创建卡密文件存储服务实例。
func NewCredentialFileService(cfg *config.Config) *CredentialFileService {
	root := strings.TrimSpace(cfg.Credentials.Root)
	if root == "" {
		root = defaultCredentialsRoot
	}
	maxSize := cfg.Credentials.MaxSize
	if maxSize <= 0 {
		maxSize = defaultCredentialsMaxSize
	}
	return &CredentialFileService{root: root, maxSize: maxSize}
}

// CredentialFileResult 上传结果。
type CredentialFileResult struct {
	RelPath          string // 相对 root 的路径（例如 "2026/05/uuid.bin"），存进数据库
	OriginalFilename string // 原始文件名
	Size             int64
	ContentType      string
}

// MaxSize 返回当前生效的单文件大小上限（字节）。
func (s *CredentialFileService) MaxSize() int64 { return s.maxSize }

// SaveCredentialFile 保存上传的文件到私有目录，返回元数据。
func (s *CredentialFileService) SaveCredentialFile(file *multipart.FileHeader) (*CredentialFileResult, error) {
	if file == nil {
		return nil, errors.New("文件为空")
	}
	if file.Size <= 0 {
		return nil, errors.New("文件大小为 0")
	}
	if file.Size > s.maxSize {
		return nil, fmt.Errorf("文件大小超过限制（最大 %d MB）", s.maxSize/1024/1024)
	}

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// 嗅探 Content-Type（仅用于元数据展示，不做白名单校验）
	buf := make([]byte, 512)
	n, _ := src.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	ext := strings.ToLower(filepath.Ext(file.Filename))
	filename := uuid.New().String() + ext
	relPath := filepath.ToSlash(filepath.Join(year, month, filename))

	absDir, err := s.absJoin(filepath.Join(year, month))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, err
	}
	absFile := filepath.Join(absDir, filename)

	dst, err := os.OpenFile(absFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(absFile)
		return nil, err
	}

	return &CredentialFileResult{
		RelPath:          relPath,
		OriginalFilename: file.Filename,
		Size:             file.Size,
		ContentType:      contentType,
	}, nil
}

// OpenForRead 打开私有目录下的文件用于读取。
// relPath 必须是相对路径，不允许包含 ".." 或绕过 root。
func (s *CredentialFileService) OpenForRead(relPath string) (*os.File, os.FileInfo, error) {
	abs, err := s.absJoin(relPath)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, nil, err
	}
	if info.IsDir() {
		return nil, nil, errors.New("目标是目录而非文件")
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, nil, err
	}
	return f, info, nil
}

// Delete 删除私有目录下的文件（路径穿越校验）。文件不存在视为成功。
func (s *CredentialFileService) Delete(relPath string) error {
	if strings.TrimSpace(relPath) == "" {
		return nil
	}
	abs, err := s.absJoin(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// absJoin 把相对路径安全地拼到 root 下。任何越界访问都会返回错误。
func (s *CredentialFileService) absJoin(relPath string) (string, error) {
	cleaned := filepath.Clean("/" + filepath.FromSlash(relPath))
	if cleaned == "/" {
		return "", errors.New("非法路径")
	}
	rootAbs, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(rootAbs, cleaned)
	// 防 symlink + ".." 越界
	if !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) && abs != rootAbs {
		return "", errors.New("路径越界")
	}
	return abs, nil
}
