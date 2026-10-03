package artifacts

import (
	"bytes"
	"context"
	"mime"
	"strings"
	"unicode/utf8"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/resources"
)

const maxContentBytes int64 = 5 * 1024 * 1024

// Text 是适合 JSON 展示的公开文本产出，不原样写终端控制字节。
type Text struct {
	ThreadID    string `json:"thread_id"`
	ArtifactID  string `json:"artifact_id"`
	ContentType string `json:"content_type"`
	Content     string `json:"text"`
}

// ValidateIDs 检查真实会话和产出身份，不接收远程文件路径或 URL。
//
// Args:
//
//	threadID: 根会话 UUID。
//	artifactID: 公开产出 UUID。
//
// Returns:
//
//	参数检查结果。
//
// Errors:
//
//	非规范 UUID 返回 invalid_input。
func ValidateIDs(threadID, artifactID string) error {
	if !resources.ValidUUID(threadID) || !resources.ValidUUID(artifactID) {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// Read 读取服务器所有权校验后的文本产出，二进制内容必须显式下载。
//
// Args:
//
//	ctx: 本地读取上下文。
//	threadID: 已有根会话 UUID。
//	artifactID: findings/reports 的稳定产出 UUID。
//
// Returns:
//
//	最多五 MiB 的 UTF-8 文本与 MIME，或错误。
//
// Errors:
//
//	鉴权、超限、非文本 MIME 或非 UTF-8 内容不原样写终端。
func (c *Client) Read(ctx context.Context, threadID, artifactID string) (Text, error) {
	if err := ValidateIDs(threadID, artifactID); err != nil {
		return Text{}, err
	}
	var buffer bytes.Buffer
	_, contentType, err := c.api.CopyContent(ctx, contentEndpoint(threadID, artifactID), &buffer, maxContentBytes)
	if err != nil {
		return Text{}, err
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !textMediaType(mediaType) || !utf8.Valid(buffer.Bytes()) {
		return Text{}, &api.Error{Code: "binary_artifact"}
	}
	return Text{ThreadID: threadID, ArtifactID: artifactID, ContentType: contentType, Content: buffer.String()}, nil
}

// textMediaType 接受服务器标注的文本类别，不根据二进制字节猜格式。
//
// Args:
//
//	mediaType: 已解析的 MIME 主类别。
//
// Returns:
//
//	文本或 JSON 类型时为真。
func textMediaType(mediaType string) bool {
	return strings.HasPrefix(mediaType, "text/") || mediaType == "application/json" || mediaType == "application/markdown"
}

// contentEndpoint 构建同一根会话下的既有产出下载端点。
//
// Args:
//
//	threadID: 已校验的根会话 UUID。
//	artifactID: 已校验的产出 UUID。
//
// Returns:
//
//	不含任意路径或远程 origin 的 API 路径。
func contentEndpoint(threadID, artifactID string) string {
	return "/api/v1/usight-agent/threads/" + threadID + "/workdir/artifacts/" + artifactID + "/content"
}
