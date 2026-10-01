package cli

import (
	"context"

	"github.com/validflow/ccneed-cli/internal/api"
)

// connectionConfig 保存显式选择的 origin；凭证解析器必须只返回该 origin 的凭证。
type connectionConfig struct {
	origin  string
	connect func(context.Context, string) (*api.Client, error)
}

// client 先校验 origin，再要求认证层创建独立绑定的传输。
//
// Args:
//
//	ctx: 命令上下文。
//
// Returns:
//
//	当前 origin 的 API client，或安全错误。
//
// Errors:
//
//	origin 无效或未登录时失败关闭，不自动回退生产。
func (c *connectionConfig) client(ctx context.Context) (*api.Client, error) {
	origin, err := api.ParseOrigin(c.origin)
	if err != nil {
		return nil, &api.Error{Code: "invalid_origin"}
	}
	if c.connect == nil {
		return nil, &api.Error{Code: "not_authenticated"}
	}
	return c.connect(ctx, origin.String())
}
