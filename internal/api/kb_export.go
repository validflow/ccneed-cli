package api

import (
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// KBExportMetadata 是同源导出响应提供的知识库身份、UTC 时间及安全文件名。
type KBExportMetadata struct {
	Filename   string
	KBName     string
	ExportedAt string
}

// CopyKBExport 流式读取用户拥有的知识库数据，不跟随重定向或触发其他采集。
//
// Args:
//
//	ctx: 本地下载上下文。
//	kbID: 知识库正整数标识。
//	format: csv 或 json。
//	output: 调用方拥有的临时字节流。
//	limit: 最大接受字节数；超限报错，不截断为成功文件。
//
// Returns:
//
//	已验证的导出元数据，或安全错误。
//
// Errors:
//
//	身份、格式、响应头、网络或大小无效时失败。
func (c *Client) CopyKBExport(ctx context.Context, kbID int64, format string, output io.Writer, limit int64) (KBExportMetadata, error) {
	if kbID <= 0 || (format != "csv" && format != "json") || output == nil || limit <= 0 {
		return KBExportMetadata{}, &Error{Code: "invalid_input"}
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	endpoint := "/api/v1/usight-agent/kbs/" + strconv.FormatInt(kbID, 10) + "/export"
	request, err := c.jsonRequest(ctx, http.MethodGet, endpoint, url.Values{"format": {format}}, nil)
	if err != nil {
		return KBExportMetadata{}, err
	}
	request.Header.Set("Accept", "*/*")
	transport := *c.http
	transport.Timeout = 2 * time.Minute
	response, err := transport.Do(request)
	if err != nil {
		return KBExportMetadata{}, &Error{Code: "transport_error", cause: err}
	}
	defer func() { _ = response.Body.Close() }()
	return copyKBExportResponse(response, kbID, format, output, limit)
}

// copyKBExportResponse 在复制字节前校验格式和元数据，避免将错误响应发布为数据文件。
//
// Args:
//
//	response: 当前同源 HTTP 响应。
//	kbID: 请求的知识库标识。
//	format: 请求的输出格式。
//	output: 调用方临时字节流。
//	limit: 最大字节数。
//
// Returns:
//
//	导出元数据，或错误。
//
// Errors:
//
//	HTTP 拒绝、元数据无效、读取失败或超限时失败。
func copyKBExportResponse(response *http.Response, kbID int64, format string, output io.Writer, limit int64) (KBExportMetadata, error) {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return KBExportMetadata{}, &Error{Code: statusCode(response.StatusCode), Status: response.StatusCode}
	}
	metadata, err := parseKBExportHeaders(response.Header, kbID, format)
	if err != nil {
		return KBExportMetadata{}, err
	}
	if response.ContentLength > limit {
		return KBExportMetadata{}, &Error{Code: "response_too_large"}
	}
	count, err := io.Copy(output, io.LimitReader(response.Body, limit+1))
	if err != nil {
		return KBExportMetadata{}, &Error{Code: "response_read", cause: err}
	}
	if count > limit {
		return KBExportMetadata{}, &Error{Code: "response_too_large"}
	}
	return metadata, nil
}

// parseKBExportHeaders 验证知识库、文件类型和 UTC 时间，不信任服务端路径。
//
// Args:
//
//	headers: 导出的 HTTP 响应头。
//	kbID: 请求的知识库正整数标识。
//	format: csv 或 json。
//
// Returns:
//
//	安全元数据，或契约错误。
//
// Errors:
//
//	MIME、编码、文件名或时间与请求不一致时返回 invalid_response。
func parseKBExportHeaders(headers http.Header, kbID int64, format string) (KBExportMetadata, error) {
	mediaType, _, mediaErr := mime.ParseMediaType(headers.Get("Content-Type"))
	expected := map[string]string{"csv": "text/csv", "json": "application/json"}[format]
	if mediaErr != nil || mediaType != expected {
		return KBExportMetadata{}, &Error{Code: "invalid_response"}
	}
	disposition, parameters, err := mime.ParseMediaType(headers.Get("Content-Disposition"))
	if err != nil || disposition != "attachment" {
		return KBExportMetadata{}, &Error{Code: "invalid_response"}
	}
	name, err := url.PathUnescape(headers.Get("X-CCNeed-KB-Name"))
	if err != nil || name == "" {
		return KBExportMetadata{}, &Error{Code: "invalid_response"}
	}
	exportedAt := headers.Get("X-CCNeed-Exported-At")
	at, err := time.Parse(time.RFC3339, exportedAt)
	if err != nil || !validKBExportFilename(parameters["filename"], kbID, format, at) {
		return KBExportMetadata{}, &Error{Code: "invalid_response"}
	}
	return KBExportMetadata{Filename: parameters["filename"], KBName: name, ExportedAt: exportedAt}, nil
}

// validKBExportFilename 限定文件名为所请求 KB 的普通文件，禁止路径与 Windows 特殊字符。
//
// Args:
//
//	name: Content-Disposition 内解析的文件名。
//	kbID: 请求的知识库标识。
//	format: 输出扩展名。
//	at: 同一次导出的时间。
//
// Returns:
//
//	文件名身份、时间、扩展名和路径均合法时为真。
func validKBExportFilename(name string, kbID int64, format string, at time.Time) bool {
	prefix := "ccneed-kb-" + strconv.FormatInt(kbID, 10) + "-"
	suffix := "-" + at.UTC().Format("20060102T150405Z") + "." + format
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) || len(name) <= len(prefix)+len(suffix) {
		return false
	}
	return filepath.Base(name) == name && len(name) <= 240 && !strings.ContainsAny(name, "<>:\"/\\|?*") &&
		strings.IndexFunc(name, func(value rune) bool { return value < 32 || value == 127 }) == -1
}
