package resources

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/validflow/ccneed-cli/internal/api"
)

// Client 按现有 API 管理账号资源，不复制知识库创建编排。
type Client struct{ api *api.Client }

// New 注入当前 origin 的认证传输。
//
// Args:
//
//	client: 已绑定 origin 的 API 连接。
//
// Returns:
//
//	资源能力客户端。
func New(client *api.Client) *Client { return &Client{api: client} }

// KnowledgeBases 返回当前账号的知识库状态和是否可绑定。
//
// Args:
//
//	ctx: 本地请求上下文。
//
// Returns:
//
//	知识库摘要列表，或错误。
//
// Errors:
//
//	鉴权、传输或摘要身份无效时失败关闭。
func (c *Client) KnowledgeBases(ctx context.Context) (KnowledgeListing, error) {
	rows := []KnowledgeBase{}
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/usight-agent/kbs", nil, nil, &rows); err != nil {
		return KnowledgeListing{}, err
	}
	for _, row := range rows {
		if row.ID <= 0 || row.Name == "" || row.PostCount < 0 || row.PublishedPostCount < 0 || row.CollectedCount < 0 {
			return KnowledgeListing{}, &api.Error{Code: "invalid_response"}
		}
	}
	if rows == nil {
		rows = []KnowledgeBase{}
	}
	return KnowledgeListing{KnowledgeBases: rows}, nil
}

// KnowledgeBaseStatus 根据真实数字标识从现有账号列表选取状态。
//
// Args:
//
//	ctx: 本地请求上下文。
//	id: 已有知识库正整数标识。
//
// Returns:
//
//	当前知识库摘要，或错误。
//
// Errors:
//
//	无权访问或不存在均返回 not_found，不用名称猜测标识。
func (c *Client) KnowledgeBaseStatus(ctx context.Context, id int64) (KnowledgeBase, error) {
	if id <= 0 {
		return KnowledgeBase{}, &api.Error{Code: "invalid_input"}
	}
	listing, err := c.KnowledgeBases(ctx)
	if err != nil {
		return KnowledgeBase{}, err
	}
	for _, row := range listing.KnowledgeBases {
		if row.ID == id {
			return row, nil
		}
	}
	return KnowledgeBase{}, &api.Error{Code: "not_found"}
}

// ValidateFileLimit 校验现有文件列表服务端数量上限。
//
// Args:
//
//	limit: 一到五百条。
//
// Returns:
//
//	参数检查结果。
//
// Errors:
//
//	超界返回 invalid_input。
func ValidateFileLimit(limit int) error {
	if limit < 1 || limit > 500 {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// Files 读取当前账号的独立上传文件，不隐式绑定到会话。
//
// Args:
//
//	ctx: 本地请求上下文。
//	limit: 服务端返回上限。
//
// Returns:
//
//	可绑定文件标识与可能截断提示，或错误。
//
// Errors:
//
//	参数、鉴权或响应身份无效时失败关闭。
func (c *Client) Files(ctx context.Context, limit int) (FileListing, error) {
	if err := ValidateFileLimit(limit); err != nil {
		return FileListing{}, err
	}
	rows := []UserFile{}
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/usight-agent/library/files", url.Values{"limit": {strconv.Itoa(limit)}}, nil, &rows); err != nil {
		return FileListing{}, err
	}
	if len(rows) > limit {
		return FileListing{}, &api.Error{Code: "invalid_response"}
	}
	for _, row := range rows {
		if !validFile(row) {
			return FileListing{}, &api.Error{Code: "invalid_response"}
		}
	}
	if rows == nil {
		rows = []UserFile{}
	}
	return FileListing{Files: rows, Limit: limit, HasMorePossible: len(rows) == limit}, nil
}

// validFile 检查服务返回的公开文件标识和元数据。
//
// Args:
//
//	file: 用户资料库摘要。
//
// Returns:
//
//	标识、名字、大小和状态有效时为真。
func validFile(file UserFile) bool {
	return ValidUUID(file.ID) && file.Name != "" && file.Size >= 0 && file.Status != ""
}
