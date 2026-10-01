package update

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/validflow/ccneed-cli/internal/version"
)

// GitHubSource 仅查询公开的 CLI 发行，不携带任何平台凭证。
type GitHubSource struct{ transport http.RoundTripper }

// NewGitHubSource 构造固定公开仓库的发行来源。
//
// Args:
//
//	transport: HTTP 传输或 nil；测试可替换网络边界。
//
// Returns:
//
//	不使用产品认证信息的来源。
func NewGitHubSource(transport http.RoundTripper) *GitHubSource {
	return &GitHubSource{transport: transport}
}

// Latest 请求公开仓库的最新稳定 tag；私有、无发行或离线都返回错误。
//
// Args:
//
//	ctx: 带有期限的检查上下文。
//
// Returns:
//
//	稳定 tag 或安全错误。
//
// Errors:
//
//	返回发行不可用或格式错误；不返回原始响应内容。
func (s *GitHubSource) Latest(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/validflow/ccneed-cli/releases/latest", nil)
	if err != nil {
		return "", errors.New("release request unavailable")
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "ccneed-cli-update-check")
	client := &http.Client{Transport: s.transport, Timeout: 2 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("release source unavailable")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("no public stable release available")
	}
	return decodeRelease(response.Body)
}

// decodeRelease 只提取固定来源中的公开 tag，并限制响应大小。
//
// Args:
//
//	body: 不包含平台认证信息的发行响应。
//
// Returns:
//
//	稳定发行 tag，或契约错误。
//
// Errors:
//
//	拒绝超出 64 KiB、无效 JSON、草稿和预发行。
func decodeRelease(body io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(body, 65537))
	if err != nil || len(data) > 65536 {
		return "", errors.New("invalid release response")
	}
	var release struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return "", errors.New("invalid release response")
	}
	if release.Draft || release.Prerelease || !version.IsStable(release.Tag) {
		return "", errors.New("no stable release available")
	}
	return release.Tag, nil
}
