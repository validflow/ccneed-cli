package cli

import (
	"context"
	"net/url"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/credential"
)

// connectionConfig 保存显式选择的 origin；凭证解析器必须只返回该 origin 的凭证。
type connectionConfig struct {
	origin            string
	allowInsecureHTTP bool
	connect           func(context.Context, string) (*api.Client, error)
}

// selectedOrigin applies this invocation's explicit transport policy.
//
// Returns:
//
//	Canonical service origin, or a safe invalid-origin error.
//
// Errors:
//
//	Rejects malformed origins and remote HTTP without explicit opt-in.
func (c *connectionConfig) selectedOrigin() (*url.URL, error) {
	parse := api.ParseOrigin
	if c.allowInsecureHTTP {
		parse = api.ParseOriginWithInsecureHTTP
	}
	origin, err := parse(c.origin)
	if err != nil {
		return nil, &api.Error{Code: "invalid_origin"}
	}
	return origin, nil
}

// vault scopes the injected credential store to this invocation's transport policy.
//
// Args:
//
//	base: Default secure credential store.
//
// Returns:
//
//	The original store, or an explicit HTTP staging view.
func (c *connectionConfig) vault(base *credential.Vault) *credential.Vault {
	if c.allowInsecureHTTP {
		return base.WithInsecureHTTP()
	}
	return base
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
//	origin 无效、未登录或认证层返回不同地址的连接时失败关闭，不自动回退生产。
func (c *connectionConfig) client(ctx context.Context) (*api.Client, error) {
	origin, err := c.selectedOrigin()
	if err != nil {
		return nil, err
	}
	if c.connect == nil {
		return nil, &api.Error{Code: "not_authenticated"}
	}
	client, err := c.connect(ctx, origin.String())
	if err != nil {
		return nil, err
	}
	if client.Origin() == "" {
		return nil, &api.Error{Code: "not_authenticated"}
	}
	if client.Origin() != origin.String() {
		return nil, &api.Error{Code: "origin_mismatch"}
	}
	return client, nil
}
