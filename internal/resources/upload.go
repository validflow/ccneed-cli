package resources

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/validflow/ccneed-cli/internal/api"
)

const maxUploadBytes int64 = 100 * 1024 * 1024

var supportedTypes = map[string]string{
	".pdf": "application/pdf", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".webp": "image/webp", ".gif": "image/gif", ".txt": "text/plain", ".md": "text/markdown",
	".csv": "text/csv", ".json": "application/json", ".ipynb": "application/x-ipynb+json",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

// UploadSource 持有显式选择的普通文件，不存入会话或复制到本地缓存。
type UploadSource struct {
	Name        string
	Size        int64
	ContentType string
	file        *os.File
}

// OpenUpload 在获取凭证前检查普通文件、格式、大小和打开后的身份。
//
// Args:
//
//	filename: 用户明确指定的本地文件路径。
//
// Returns:
//
//	待上传字节源，或安全错误；调用方必须关闭。
//
// Errors:
//
//	非普通文件、链接、空文件、超出一百 MiB 或不支持格式返回 invalid_file。
func OpenUpload(filename string) (*UploadSource, error) {
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxUploadBytes {
		return nil, &api.Error{Code: "invalid_file"}
	}
	name := filepath.Base(filename)
	contentType, ok := supportedTypes[strings.ToLower(filepath.Ext(name))]
	if !ok || !safeUploadName(name) {
		return nil, &api.Error{Code: "invalid_file"}
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, &api.Error{Code: "invalid_file"}
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Size() != info.Size() {
		_ = file.Close()
		return nil, &api.Error{Code: "invalid_file"}
	}
	return &UploadSource{Name: name, Size: info.Size(), ContentType: contentType, file: file}, nil
}

// safeUploadName 限制文件名到服务端可处理的普通名称，不携带路径或控制字符。
//
// Args:
//
//	name: 从用户选择路径得到的 basename。
//
// Returns:
//
//	Unicode 名称有效且无特殊路径或控制字符时为真。
func safeUploadName(name string) bool {
	return name != "" && len(name) <= 255 && utf8.ValidString(name) && !strings.ContainsAny(name, "\\/:") &&
		strings.IndexFunc(name, func(value rune) bool { return value < 32 || value == 127 }) == -1
}

// Close 释放本地源文件，不删除用户文件。
//
// Returns:
//
//	文件关闭结果。
//
// Errors:
//
//	关闭失败返回安全 file_close 错误，不回显路径。
func (s *UploadSource) Close() error {
	if s.file.Close() != nil {
		return &api.Error{Code: "file_close"}
	}
	return nil
}

// Upload 独立上传用户文件，完成后返回可用于初始绑定或追加的资料库标识。
//
// Args:
//
//	ctx: 本地上传上下文。
//	source: 已打开校验的普通文件，调用方负责关闭。
//
// Returns:
//
//	公开资料库回执，或错误。
//
// Errors:
//
//	响应未知不自动重新上传；明确拒绝保留状态码。
func (c *Client) Upload(ctx context.Context, source *UploadSource) (UserFile, error) {
	if source == nil || source.file == nil {
		return UserFile{}, &api.Error{Code: "invalid_file"}
	}
	var result UserFile
	err := c.api.UploadJSON(ctx, "/api/v1/usight-agent/library/files", source.Name, source.ContentType, io.LimitReader(source.file, source.Size+1), &result)
	if err != nil {
		return UserFile{}, writeFailure(err)
	}
	if !validFile(result) || result.Size != source.Size || result.Name != source.Name {
		return UserFile{}, &api.Error{Code: "acceptance_unknown"}
	}
	return result, nil
}
