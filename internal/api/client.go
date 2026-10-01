package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const requestTimeout = 30 * time.Second
const maxJSONBytes = 4 * 1024 * 1024

// Client 将每个请求和凭证绑定到一个明确选择的服务 origin。
type Client struct {
	origin *url.URL
	token  string
	http   *http.Client
}

// New 创建不跟随重定向、不自动重试的 API 客户端。
//
// Args:
//
//	origin: 已被用户或配置明确选择的服务 origin。
//	token: 该 origin 的凭证；空值用于尚未认证的请求。
//	transport: 可注入 HTTP 传输；nil 使用标准传输。
//
// Returns:
//
//	origin 绑定的客户端，或配置错误。
//
// Errors:
//
//	返回不合法的 origin 或凭证格式错误，不回显输入值。
func New(origin, token string, transport http.RoundTripper) (*Client, error) {
	parsed, err := ParseOrigin(origin)
	if err != nil {
		return nil, err
	}
	if strings.IndexFunc(token, func(char rune) bool { return char < 33 || char > 126 }) != -1 {
		return nil, &Error{Code: "invalid_credential"}
	}
	return &Client{origin: parsed, token: token, http: &http.Client{
		Transport:     transport,
		Timeout:       requestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

// DoJSON 执行单次有限期请求；取消仅取消本地请求，不代表取消云端任务。
//
// Args:
//
//	ctx: 调用方上下文。
//	method: 明确的 HTTP 方法。
//	path: /api/v1 下的 API 路径，不包含 origin 或 query。
//	query: 查询参数，不得包含凭证。
//	body: 可 JSON 编码的请求；nil 不发送请求体。
//	result: JSON 接收目标；nil 表示无返回体。
//
// Returns:
//
//	单次请求错误，或 nil。
//
// Errors:
//
//	返回安全的网络、状态码或响应契约错误，不回显原始服务响应。
func (c *Client) DoJSON(ctx context.Context, method, path string, query url.Values, body, result any) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	request, err := c.jsonRequest(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return &Error{Code: "transport_error", cause: err}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &Error{Code: statusCode(response.StatusCode), Status: response.StatusCode}
	}
	if result == nil {
		return nil
	}
	return decodeResponse(response.Body, result)
}

// jsonRequest 只在已绑定 origin 的 /api/v1 路径构造请求。
//
// Args:
//
//	ctx: 已有限期的上下文。
//	method: HTTP 方法。
//	endpoint: 未包含查询的绝对 API 路径。
//	query: 查询参数。
//	body: JSON 请求对象或 nil。
//
// Returns:
//
//	携带当前 origin 凭证的请求，或构造错误。
//
// Errors:
//
//	返回路径、JSON 编码或 HTTP 方法错误；不回显请求内容。
func (c *Client) jsonRequest(ctx context.Context, method, endpoint string, query url.Values, body any) (*http.Request, error) {
	if !validEndpoint(endpoint) {
		return nil, &Error{Code: "invalid_endpoint"}
	}
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			return nil, &Error{Code: "request_encoding", cause: err}
		}
	}
	target := *c.origin
	target.Path, target.RawQuery = endpoint, query.Encode()
	request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(encoded))
	if err != nil {
		return nil, &Error{Code: "invalid_request", cause: err}
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	return request, nil
}

// validEndpoint 拒绝 origin 替换、路径规范化及嵌入查询。
//
// Args:
//
//	endpoint: 请求路径。
//
// Returns:
//
//	规范化的 /api/v1 子路径时为 true。
func validEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "/api/v1/") && path.Clean(endpoint) == endpoint &&
		!strings.ContainsAny(endpoint, "?#\\")
}

// decodeResponse 限制内存占用，并拒绝无效或多份 JSON 响应。
//
// Args:
//
//	body: 响应字节流。
//	result: 解码目标。
//
// Returns:
//
//	解码错误，或 nil。
//
// Errors:
//
//	返回读取失败、超过 4 MiB 或 JSON 契约错误，不包含响应内容。
func decodeResponse(body io.Reader, result any) error {
	data, err := io.ReadAll(io.LimitReader(body, maxJSONBytes+1))
	if err != nil {
		return &Error{Code: "response_read", cause: err}
	}
	if len(data) > maxJSONBytes {
		return &Error{Code: "response_too_large"}
	}
	if err := json.Unmarshal(data, result); err != nil {
		return &Error{Code: "invalid_response", cause: errors.New("response JSON does not match expected contract")}
	}
	return nil
}
