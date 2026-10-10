package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"time"
)

const reportPDFTimeout = 2 * time.Minute

// CopyReportPDF requests Club's owner-checked PDF renderer at the selected origin.
//
// Args:
//
//	ctx: Cancellation context for one export request.
//	threadID: Existing root conversation UUID.
//	artifactID: Existing report artifact UUID.
//	output: Destination writer owned by the caller.
//	limit: Maximum accepted PDF byte count.
//
// Returns:
//
//	Number of copied bytes or a safe request error.
//
// Errors:
//
//	Rejects non-PDF responses, oversized output, redirects, and network failures.
func (c *Client) CopyReportPDF(ctx context.Context, threadID, artifactID string, output io.Writer, limit int64) (int64, error) {
	if c == nil || c.origin == nil || output == nil || limit < 1 {
		return 0, &Error{Code: "invalid_input"}
	}
	ctx, cancel := context.WithTimeout(ctx, reportPDFTimeout)
	defer cancel()
	request, err := c.reportPDFRequest(ctx, threadID, artifactID)
	if err != nil {
		return 0, err
	}
	return c.copyPDFResponse(request, output, limit)
}

// reportPDFRequest binds report identifiers and the current credential to one origin.
//
// Args:
//
//	ctx: Bounded request context.
//	threadID: Existing root conversation UUID.
//	artifactID: Existing report artifact UUID.
//
// Returns:
//
//	Prepared same-origin request, or a safe encoding error.
//
// Errors:
//
//	Rejects requests that cannot be encoded or constructed.
func (c *Client) reportPDFRequest(ctx context.Context, threadID, artifactID string) (*http.Request, error) {
	requestBody, err := json.Marshal(struct {
		Source     string `json:"source"`
		ThreadID   string `json:"threadId"`
		ArtifactID string `json:"artifactId"`
	}{Source: "client", ThreadID: threadID, ArtifactID: artifactID})
	if err != nil {
		return nil, &Error{Code: "request_encoding", cause: err}
	}
	target := *c.origin
	target.Path = "/print/report/pdf/client"
	target.RawQuery = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(requestBody))
	if err != nil {
		return nil, &Error{Code: "invalid_request", cause: err}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/pdf")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	return request, nil
}

// copyPDFResponse streams one validated PDF response without following redirects.
//
// Args:
//
//	request: Prepared same-origin report export request.
//	output: Destination writer owned by the caller.
//	limit: Maximum accepted PDF byte count.
//
// Returns:
//
//	Number of copied bytes, or a safe request error.
//
// Errors:
//
//	Rejects non-PDF responses, oversized output, redirects, and network failures.
func (c *Client) copyPDFResponse(request *http.Request, output io.Writer, limit int64) (int64, error) {
	transport := *c.http
	transport.Timeout = reportPDFTimeout
	response, err := transport.Do(request)
	if err != nil {
		return 0, &Error{Code: "transport_error", cause: err}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, &Error{Code: statusCode(response.StatusCode), Status: response.StatusCode}
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/pdf" {
		return 0, &Error{Code: "invalid_response"}
	}
	if response.ContentLength > limit {
		return 0, &Error{Code: "response_too_large"}
	}
	count, err := io.Copy(output, io.LimitReader(response.Body, limit+1))
	if err != nil {
		return count, &Error{Code: "response_read", cause: err}
	}
	if count > limit {
		return count, &Error{Code: "response_too_large"}
	}
	return count, nil
}
