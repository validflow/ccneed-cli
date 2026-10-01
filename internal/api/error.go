package api

import "fmt"

// Error 是可公开的传输错误；原因保留用于 errors.Is，不用于终端原样输出。
type Error struct {
	Code   string `json:"code"`
	Status int    `json:"http_status,omitempty"`
	cause  error
}

// Error 返回不包含地址、凭证或服务响应内容的短说明。
//
// Returns:
//
//	安全的错误摘要。
func (e *Error) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("API request failed (%s, HTTP %d)", e.Code, e.Status)
	}
	return "API request failed (" + e.Code + ")"
}

// Unwrap 保留超时和取消原因，方便上层判断恢复操作。
//
// Returns:
//
//	内部错误；没有原因时为 nil。
//
// Errors:
//
//	仅暴露错误链，不执行副作用。
func (e *Error) Unwrap() error { return e.cause }

// statusCode 将 HTTP 状态映射为稳定的机器错误码。
//
// Args:
//
//	status: HTTP 响应状态。
//
// Returns:
//
//	不依赖服务文案的错误码。
func statusCode(status int) string {
	switch status {
	case 401:
		return "unauthorized"
	case 403:
		return "forbidden"
	case 404:
		return "not_found"
	case 409:
		return "conflict"
	case 422:
		return "invalid_input"
	case 429:
		return "rate_limited"
	}
	if status >= 300 && status < 400 {
		return "redirect_refused"
	}
	if status >= 500 {
		return "server_error"
	}
	return "request_failed"
}
