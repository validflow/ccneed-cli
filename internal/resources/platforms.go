package resources

import (
	"context"
	"net/http"
	"regexp"

	"github.com/validflow/ccneed-cli/internal/api"
)

var platformID = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// PlatformListing 包含当前服务允许采集的正式平台 ID，不维护本地平台枚举。
type PlatformListing struct {
	Platforms []string `json:"platforms"`
}

// AvailablePlatforms 查询当前环境的可采集平台，不创建知识库或启动采集。
//
// Args:
//
//	ctx: 本次只读查询的上下文。
//
// Returns:
//
//	服务端返回的平台列表，或安全错误；显式空列表表示当前没有可用平台。
//
// Errors:
//
//	认证、传输、缺少列表或平台 ID 格式无效时失败，不回退到内置平台。
func (c *Client) AvailablePlatforms(ctx context.Context) (PlatformListing, error) {
	var result PlatformListing
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/insightflow/available-platforms", nil, nil, &result); err != nil {
		return PlatformListing{}, err
	}
	if result.Platforms == nil {
		return PlatformListing{}, &api.Error{Code: "invalid_response"}
	}
	for _, id := range result.Platforms {
		if !platformID.MatchString(id) {
			return PlatformListing{}, &api.Error{Code: "invalid_response"}
		}
	}
	return result, nil
}
