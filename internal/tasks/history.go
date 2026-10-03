package tasks

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/validflow/ccneed-cli/internal/api"
)

// HistoryOptions 明确控制只读历史页和原始提交恢复查询。
type HistoryOptions struct {
	Limit     int
	Before    int
	RequestID string
}

// Summary 用原始回合标识公开状态，不含执行循环或 checkpoint。
type Summary struct {
	TurnID      string     `json:"turn_id"`
	Index       int        `json:"conversation_turn_idx"`
	RequestID   string     `json:"client_request_id"`
	CloudStatus string     `json:"cloud_status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

// History 返回从新到旧的回合摘要及排他顺序游标。
type History struct {
	ThreadID   string    `json:"thread_id"`
	Turns      []Summary `json:"turns"`
	NextBefore *int      `json:"next_before_turn_index"`
}

// ValidateHistory 在获取凭证前校验分页、会话及可选原始提交标识。
//
// Args:
//
//	threadID: 根会话 UUID。
//	options: 每页数量、排他游标和可选提交 UUID。
//
// Returns:
//
//	参数校验结果。
//
// Errors:
//
//	不合法身份、数量或游标返回 invalid_input。
func ValidateHistory(threadID string, options HistoryOptions) error {
	if err := ValidateIDs(threadID, threadID); err != nil {
		return err
	}
	if options.Limit < 1 || options.Limit > 100 || options.Before < 0 {
		return &api.Error{Code: "invalid_input"}
	}
	if options.RequestID != "" {
		return ValidateIDs(threadID, options.RequestID)
	}
	return nil
}

// List 读取原始会话持久化历史，不重新提交或根据标题猜测任务。
//
// Args:
//
//	ctx: 当前本地调用上下文。
//	threadID: 已选择的根会话。
//	options: 有限页参数及可选原请求过滤。
//
// Returns:
//
//	白名单历史页或安全错误。
//
// Errors:
//
//	保留 HTTP 安全错误；身份或分页响应不一致返回 invalid_response。
func (c *Client) List(ctx context.Context, threadID string, options HistoryOptions) (History, error) {
	if err := ValidateHistory(threadID, options); err != nil {
		return History{}, err
	}
	query := url.Values{"limit": {strconv.Itoa(options.Limit)}}
	if options.Before > 0 {
		query.Set("before_turn_index", strconv.Itoa(options.Before))
	}
	if options.RequestID != "" {
		query.Set("client_request_id", options.RequestID)
	}
	var result History
	if err := c.api.DoJSON(ctx, http.MethodGet, "/api/v1/usight-agent/threads/"+threadID+"/turns", query, nil, &result); err != nil {
		return History{}, err
	}
	if !validHistory(result, threadID, options) {
		return History{}, &api.Error{Code: "invalid_response"}
	}
	if result.Turns == nil {
		result.Turns = []Summary{}
	}
	return result, nil
}

// validHistory 验证所有回合按同一身份和严格递减游标返回。
//
// Args:
//
//	result: 解码后的白名单历史页。
//	threadID: 原始会话 UUID。
//	options: 发出请求的边界条件。
//
// Returns:
//
//	当前页身份、顺序、过滤及下一页游标均有效时为真。
func validHistory(result History, threadID string, options HistoryOptions) bool {
	if result.ThreadID != threadID || len(result.Turns) > options.Limit {
		return false
	}
	previous := options.Before
	for _, item := range result.Turns {
		if !validSummary(item, options.RequestID) || (previous > 0 && item.Index >= previous) {
			return false
		}
		previous = item.Index
	}
	if result.NextBefore == nil {
		return true
	}
	return len(result.Turns) == options.Limit && *result.NextBefore == previous
}

// validSummary 只接受可定位的原始回合和已知云端状态。
//
// Args:
//
//	item: 一个持久化回合摘要。
//	requestID: 可选原请求过滤条件。
//
// Returns:
//
//	字段满足公开协议且没有越过请求过滤时为真。
func validSummary(item Summary, requestID string) bool {
	if !identifier.MatchString(item.TurnID) || item.Index < 1 || item.RequestID == "" {
		return false
	}
	if requestID != "" && item.RequestID != requestID {
		return false
	}
	switch item.CloudStatus {
	case "pending", "running", "paused", "waiting", "interrupted", "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}
