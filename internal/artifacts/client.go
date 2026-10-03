// Package artifacts 只访问公开 findings/reports，不接收对象存储地址或原始事件路径。
package artifacts

import (
	"context"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/resources"
	"github.com/validflow/ccneed-cli/internal/tasks"
	"net/http"
)

// Client 连接当前 origin 的公开产出接口。
type Client struct{ api *api.Client }

// Listing 是当前工作空间的公开产出，不表示历史字节快照。
type Listing struct {
	ThreadID  string           `json:"thread_id"`
	Artifacts []tasks.Artifact `json:"artifacts"`
	HasMore   bool             `json:"has_more"`
}

// New 注入 origin 绑定的认证连接。
//
// Args:
//
//	client: 当前环境 API 传输。
//
// Returns:
//
//	产出能力客户端。
func New(client *api.Client) *Client { return &Client{api: client} }

// List 获取至多两百个公开产出，保留服务器超限提示。
//
// Args:
//
//	ctx: 本地请求上下文。
//	threadID: 真实根会话 UUID。
//
// Returns:
//
//	公开产出 manifest，或错误。
//
// Errors:
//
//	无效标识、归属或产出契约失败时返回安全错误。
func (c *Client) List(ctx context.Context, threadID string) (Listing, error) {
	if !resources.ValidUUID(threadID) {
		return Listing{}, &api.Error{Code: "invalid_input"}
	}
	var result Listing
	endpoint := "/api/v1/usight-agent/threads/" + threadID + "/artifacts"
	if err := c.api.DoJSON(ctx, http.MethodGet, endpoint, nil, nil, &result); err != nil {
		return Listing{}, err
	}
	if result.ThreadID != threadID || len(result.Artifacts) > 200 {
		return Listing{}, &api.Error{Code: "invalid_response"}
	}
	for _, item := range result.Artifacts {
		if !validArtifact(item) {
			return Listing{}, &api.Error{Code: "invalid_response"}
		}
	}
	if result.Artifacts == nil {
		result.Artifacts = []tasks.Artifact{}
	}
	return result, nil
}

// validArtifact 检查稳定产出标识与允许类别，不信任服务器提供的下载地址。
//
// Args:
//
//	item: 公开 manifest 项。
//
// Returns:
//
//	findings/reports 元数据有效时为真。
func validArtifact(item tasks.Artifact) bool {
	if !resources.ValidUUID(item.ID) || item.Name == "" {
		return false
	}
	if item.Type != "finding" && item.Type != "report" {
		return false
	}
	return item.Size == nil || *item.Size >= 0
}
