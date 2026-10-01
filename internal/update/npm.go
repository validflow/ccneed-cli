package update

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/validflow/ccneed-cli/internal/version"
)

var packagePattern = regexp.MustCompile(`^@[a-z0-9-]+/[a-z0-9-]+$`)

// NPMSource 只查询发行时注入的官方 npm 包稳定通道，不跟随 GitHub tag。
type NPMSource struct {
	name      string
	transport http.RoundTripper
}

// NewNPMSource 构造不携带产品凭证的固定 registry 来源。
//
// Args:
//
//	name: 构建时注入的官方包名，不能由服务端响应覆盖。
//	transport: 可注入网络边界；nil 使用默认传输。
//
// Returns:
//
//	最新稳定 npm 版本来源。
func NewNPMSource(name string, transport http.RoundTripper) *NPMSource {
	return &NPMSource{name: name, transport: transport}
}

// Latest 查询 npm latest，仅接受匹配包名且已公开的稳定版本。
//
// Args:
//
//	ctx: 检查的取消上下文。
//
// Returns:
//
//	npm 版本，或安全错误。
//
// Errors:
//
//	拒绝无效包名、重定向、超时、超限响应和非稳定版本。
func (s *NPMSource) Latest(ctx context.Context) (string, error) {
	if !packagePattern.MatchString(s.name) {
		return "", errors.New("invalid update package identity")
	}
	address := "https://registry.npmjs.org/" + url.PathEscape(s.name) + "/latest"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", errors.New("npm update request unavailable")
	}
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Transport: s.transport, Timeout: 2 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("npm update source unavailable")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("no published stable npm package available")
	}
	return decodePackage(response.Body, s.name)
}

// decodePackage 限制公开元数据大小并校验官方包身份。
//
// Args:
//
//	body: 未认证 npm 响应流。
//	name: 预期包名。
//
// Returns:
//
//	稳定版本，或契约错误。
//
// Errors:
//
//	无效 JSON、超出 64 KiB、包名不一致或预发布返回错误。
func decodePackage(body io.Reader, name string) (string, error) {
	data, err := io.ReadAll(io.LimitReader(body, 65537))
	if err != nil || len(data) > 65536 {
		return "", errors.New("invalid npm update response")
	}
	var manifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.Name != name || !version.IsStable(manifest.Version) {
		return "", errors.New("invalid stable npm package")
	}
	return manifest.Version, nil
}
