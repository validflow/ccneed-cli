package api

import (
	"context"
	"io"
	"net/http"
	"time"
)

// CopyContent 在受信 origin 内流式读取有界产出，不跟随下载地址或重定向。
//
// Args:
//
//	ctx: 本地请求上下文。
//	endpoint: /api/v1 下使用真实产出标识的路径。
//	output: 消费方拥有的目标字节流。
//	limit: 最大允许字节数，必须为正。
//
// Returns:
//
//	实际字节数、响应 MIME 和错误。
//
// Errors:
//
//	单次读取最长两分钟；状态、传输或超限均不回显原始响应。
func (c *Client) CopyContent(ctx context.Context, endpoint string, output io.Writer, limit int64) (int64, string, error) {
	if limit < 1 {
		return 0, "", &Error{Code: "invalid_input"}
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	request, err := c.jsonRequest(ctx, http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("Accept", "*/*")
	transport := *c.http
	transport.Timeout = 2 * time.Minute
	response, err := transport.Do(request)
	if err != nil {
		return 0, "", &Error{Code: "transport_error", cause: err}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, "", &Error{Code: statusCode(response.StatusCode), Status: response.StatusCode}
	}
	if response.ContentLength > limit {
		return 0, "", &Error{Code: "response_too_large"}
	}
	count, err := io.Copy(output, io.LimitReader(response.Body, limit+1))
	if err != nil {
		return count, "", &Error{Code: "response_read", cause: err}
	}
	if count > limit {
		return count, "", &Error{Code: "response_too_large"}
	}
	return count, response.Header.Get("Content-Type"), nil
}
