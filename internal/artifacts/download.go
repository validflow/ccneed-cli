package artifacts

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/validflow/ccneed-cli/internal/api"
)

// Downloaded 表示完整校验并写入用户显式目标的本地文件。
type Downloaded struct {
	ThreadID    string `json:"thread_id"`
	ArtifactID  string `json:"artifact_id"`
	Path        string `json:"saved_path"`
	Size        int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"content_type"`
}

// ValidateOutput 验证用户指定目标，拒绝静默覆盖和 Windows 备用数据流。
//
// Args:
//
//	output: 用户明确选择的本地文件路径。
//
// Returns:
//
//	规范绝对路径，或安全错误。
//
// Errors:
//
//	已存在目标返回 output_exists；非法文件名或父目录返回 invalid_output。
func ValidateOutput(output string) (string, error) {
	if output == "" || strings.ContainsAny(filepath.Base(output), "\\/:") || strings.IndexFunc(output, func(r rune) bool { return r < 32 || r == 127 }) != -1 {
		return "", &api.Error{Code: "invalid_output"}
	}
	target, err := filepath.Abs(output)
	if err != nil {
		return "", &api.Error{Code: "invalid_output"}
	}
	_, err = os.Lstat(target)
	if err == nil {
		return "", &api.Error{Code: "output_exists"}
	}
	if !os.IsNotExist(err) {
		return "", &api.Error{Code: "invalid_output"}
	}
	parent, err := os.Stat(filepath.Dir(target))
	if err != nil || !parent.IsDir() {
		return "", &api.Error{Code: "invalid_output"}
	}
	return target, nil
}

// Download 流式写入同目录临时文件，完整成功后以不覆盖的硬链接发布目标。
//
// Args:
//
//	ctx: 本地下载上下文。
//	threadID: 原始根会话 UUID。
//	artifactID: 公开产出 UUID。
//	output: 用户显式选择的目标，不采用服务端文件名。
//
// Returns:
//
//	保存路径、字节数、校验和及 MIME，或错误。
//
// Errors:
//
//	下载失败清理自建临时文件；最终目标已存在时绝不覆盖或删除。
func (c *Client) Download(ctx context.Context, threadID, artifactID, output string) (Downloaded, error) {
	if err := ValidateIDs(threadID, artifactID); err != nil {
		return Downloaded{}, err
	}
	target, err := ValidateOutput(output)
	if err != nil {
		return Downloaded{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".ccneed-download-*")
	if err != nil {
		return Downloaded{}, &api.Error{Code: "output_write"}
	}
	name := temporary.Name()
	defer func() { _ = temporary.Close(); _ = os.Remove(name) }()
	hash := sha256.New()
	size, contentType, err := c.api.CopyContent(ctx, contentEndpoint(threadID, artifactID), io.MultiWriter(temporary, hash), maxContentBytes)
	if err != nil {
		return Downloaded{}, err
	}
	if err := finishDownload(temporary, target); err != nil {
		return Downloaded{}, err
	}
	return Downloaded{ThreadID: threadID, ArtifactID: artifactID, Path: target, Size: size, SHA256: fmt.Sprintf("%x", hash.Sum(nil)), ContentType: contentType}, nil
}

// finishDownload 完成持久化后发布新路径，最终竞争中的既有文件同样不被覆盖。
//
// Args:
//
//	file: 仅由当前命令创建的临时文件。
//	target: 用户明确选择的最终文件路径。
//
// Returns:
//
//	发布结果。
//
// Errors:
//
//	不支持安全硬链接时返回 output_write，不退回会覆盖目标的 rename。
func finishDownload(file *os.File, target string) error {
	if file.Sync() != nil || file.Close() != nil {
		return &api.Error{Code: "output_write"}
	}
	if err := os.Link(file.Name(), target); err != nil {
		if os.IsExist(err) {
			return &api.Error{Code: "output_exists"}
		}
		return &api.Error{Code: "output_write"}
	}
	return nil
}
