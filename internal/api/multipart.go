package api

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

// UploadJSON 流式提交一个文件，保持 origin、凭证和禁止重定向约束。
//
// Args:
//
//	ctx: 本地上下文，取消不代表撤销已完成上传。
//	endpoint: /api/v1 下的固定上传路径。
//	filename: 消费方已校验的普通文件名。
//	contentType: 消费方根据支持格式选取的 MIME。
//	source: 消费方提供的有界普通文件流。
//	result: 有界 JSON 响应解码目标。
//
// Returns:
//
//	单次 HTTP 或响应错误，或 nil。
//
// Errors:
//
//	上传最长五分钟，不自动重试；网络错误不回显请求内容。
func (c *Client) UploadJSON(ctx context.Context, endpoint, filename, contentType string, source io.Reader, result any) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	request, err := c.jsonRequest(ctx, http.MethodPost, endpoint, nil, nil)
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	request.Body = reader
	request.ContentLength = -1
	request.Header.Set("Content-Type", form.FormDataContentType())
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := writeMultipart(form, filename, contentType, source)
		_ = writer.CloseWithError(err)
	}()
	defer func() { _ = reader.Close(); <-done }()
	transport := *c.http
	transport.Timeout = 5 * time.Minute
	response, err := transport.Do(request)
	if err != nil {
		return &Error{Code: "transport_error", cause: err}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &Error{Code: statusCode(response.StatusCode), Status: response.StatusCode}
	}
	return decodeResponse(response.Body, result)
}

// writeMultipart 只写一个标准 file part，结束或失败均释放上传协程。
//
// Args:
//
//	form: 当前请求的 multipart 写入器。
//	filename: 已校验的普通文件名。
//	contentType: 明确的文件 MIME。
//	source: 已限制长度的字节源。
//
// Returns:
//
//	multipart 写入结果。
//
// Errors:
//
//	读取、管道或 multipart 结束失败返回原始 I/O 错误给 HTTP 错误链。
func writeMultipart(form *multipart.Writer, filename, contentType string, source io.Reader) error {
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": filename}))
	headers.Set("Content-Type", contentType)
	part, err := form.CreatePart(headers)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, source); err != nil {
		return err
	}
	return form.Close()
}
