// Package tasks 连接现有云端回合接口；不运行 Agent 或重新提交观察中的任务。
package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/validflow/ccneed-cli/internal/api"
)

var identifier = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

// PageOptions selects a bounded, exclusive progress page within one turn.
type PageOptions struct {
	Limit       int
	BeforeSeq   int64
	AfterSeq    int64
	AfterSeqSet bool
}

// ValidatePageOptions rejects inconsistent cursors before a network request.
//
// Args:
//
//	options: Page limit and optional exclusive sequence cursor.
//
// Returns:
//
//	An invalid_input error, or nil.
//
// Errors:
//
//	Returns invalid_input for out-of-range limits or inconsistent cursors.
func ValidatePageOptions(options PageOptions) error {
	afterSelected := options.AfterSeqSet || options.AfterSeq > 0
	if options.Limit < 1 || options.Limit > 100 || options.BeforeSeq < 0 || options.AfterSeq < 0 ||
		(options.BeforeSeq != 0 && afterSelected) {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// Client 负责回合协议适配，不持有终端或本地文件句柄。
type Client struct{ api *api.Client }

// New 将任务能力绑定到已经认证、按 origin 隔离的传输。
//
// Args:
//
//	client: 当前环境的 API 传输。
//
// Returns:
//
//	任务服务。
func New(client *api.Client) *Client { return &Client{api: client} }

// ValidateIDs 在建立连接前拒绝非规范 UUID，防止路径被调用方改变。
//
// Args:
//
//	threadID: 根会话标识。
//	turnID: 持久化回合标识。
//
// Returns:
//
//	参数错误，或 nil。
//
// Errors:
//
//	返回 invalid_input，不回显用户输入。
func ValidateIDs(threadID, turnID string) error {
	if !identifier.MatchString(threadID) || !identifier.MatchString(turnID) {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// Get 获取指定回合的公开快照，不访问原始 hydrate 或 SSE。
//
// Args:
//
//	ctx: 本地请求上下文。
//	threadID: 所属根会话 UUID。
//	turnID: 指定 durable turn UUID。
//
// Returns:
//
//	公开回合快照，或错误。
//
// Errors:
//
//	参数、鉴权、传输或响应不符合指定回合契约时失败关闭。
func (c *Client) Get(ctx context.Context, threadID, turnID string) (Detail, error) {
	return c.GetPage(ctx, threadID, turnID, PageOptions{Limit: 20})
}

// GetPage reads a bounded progress page and the latest status of one owned turn.
//
// Args:
//
//	ctx: Local request context.
//	threadID: Root conversation UUID.
//	turnID: Durable turn UUID.
//	options: Bounded, exclusive progress page selection.
//
// Returns:
//
//	The public turn snapshot, or an error.
//
// Errors:
//
//	Rejects invalid arguments, access failures, and malformed responses.
func (c *Client) GetPage(ctx context.Context, threadID, turnID string, options PageOptions) (Detail, error) {
	if err := ValidateIDs(threadID, turnID); err != nil {
		return Detail{}, err
	}
	if err := ValidatePageOptions(options); err != nil {
		return Detail{}, err
	}
	query := url.Values{"limit": {strconv.Itoa(options.Limit)}}
	if options.BeforeSeq > 0 {
		query.Set("before_seq", strconv.FormatInt(options.BeforeSeq, 10))
	}
	if options.AfterSeqSet || options.AfterSeq > 0 {
		query.Set("after_seq", strconv.FormatInt(options.AfterSeq, 10))
	}
	var result Detail
	endpoint := "/api/v1/usight-agent/threads/" + threadID + "/turns/" + turnID
	if err := c.api.DoJSON(ctx, http.MethodGet, endpoint, query, nil, &result); err != nil {
		return Detail{}, err
	}
	if !validDetail(result, threadID, turnID) {
		return Detail{}, &api.Error{Code: "invalid_response"}
	}
	if result.Artifacts == nil {
		result.Artifacts = []Artifact{}
	}
	if result.Progress.Events == nil {
		result.Progress.Events = []ProgressEvent{}
	}
	result.WebURL = c.api.Origin() + "/insightflow/" + threadID
	return result, nil
}

// validDetail 校验稳定状态及身份，避免误将其他回合或坏响应当成完成。
//
// Args:
//
//	result: 待校验的公开快照。
//	threadID: 请求的根会话 UUID。
//	turnID: 请求的回合 UUID。
//
// Returns:
//
//	身份、状态、确认项与产出满足公开契约时为真。
func validDetail(result Detail, threadID, turnID string) bool {
	if result.ThreadID != threadID || result.TurnID != turnID || result.ConversationTurnIndex < 1 || len(result.Artifacts) > 200 {
		return false
	}
	if !validState(result) {
		return false
	}
	return validArtifacts(result.Artifacts) && validProgress(result.Progress)
}

// validProgress accepts only a bounded, ordered public milestone page.
//
// Args:
//
//	page: Decoded progress page from the service.
//
// Returns:
//
//	True when milestones and cursors are structurally valid.
func validProgress(page ProgressPage) bool {
	if len(page.Events) > 100 || (page.NextBeforeSeq != nil && *page.NextBeforeSeq < 1) ||
		(page.NextAfterSeq != nil && *page.NextAfterSeq < 1) {
		return false
	}
	previous := int64(0)
	for _, event := range page.Events {
		if event.Seq <= previous || !validProgressEvent(event) {
			return false
		}
		previous = event.Seq
	}
	return true
}

// validProgressEvent enforces the public milestone vocabulary and safe size bound.
//
// Args:
//
//	event: One decoded milestone.
//
// Returns:
//
//	True for a complete event with a known stage and state.
func validProgressEvent(event ProgressEvent) bool {
	if event.Seq < 1 || event.Summary == "" || len(event.Summary) > 200 {
		return false
	}
	switch event.Stage {
	case "knowledge_base", "data_collection", "analysis", "report", "research", "work":
	default:
		return false
	}
	switch event.State {
	case "started", "completed", "failed", "waiting_user":
		return true
	default:
		return false
	}
}

// validArtifacts 验证产出标识、类型和大小，不接受任意目录对象。
//
// Args:
//
//	artifacts: 公开 manifest 项。
//
// Returns:
//
//	每项满足 findings/reports 契约时为真。
func validArtifacts(artifacts []Artifact) bool {
	for _, artifact := range artifacts {
		if !identifier.MatchString(artifact.ID) || artifact.Name == "" {
			return false
		}
		if artifact.Type != "finding" && artifact.Type != "report" {
			return false
		}
		if artifact.Size != nil && *artifact.Size < 0 {
			return false
		}
	}
	return true
}

// validState 校验云端状态、公开状态和待确认数据的对应关系。
//
// Args:
//
//	result: 已解码的公开回合数据。
//
// Returns:
//
//	状态与数据相容时为真。
func validState(result Detail) bool {
	if !validCloudStatus(result.Status, result.CloudStatus) {
		return false
	}
	if result.Status == "waiting_user" && !validAction(result.PendingAction) {
		return false
	}
	return (result.Status == "completed" || result.Answer == nil) && (result.Status == "waiting_user" || result.PendingAction == nil)
}

// validCloudStatus 将稳定公开状态与已有云端状态一一对应。
//
// Args:
//
//	status: 公开状态。
//	cloud: 持久化云端状态。
//
// Returns:
//
//	两种状态互相兼容时为真。
func validCloudStatus(status, cloud string) bool {
	if status == "interrupted" {
		return cloud == "interrupted" || cloud == "paused"
	}
	expected, ok := map[string]string{
		"pending": "pending", "running": "running", "completed": "completed",
		"failed": "failed", "cancelled": "cancelled", "waiting_external": "waiting",
		"waiting_user": "paused",
	}[status]
	return ok && cloud == expected
}

// validAction 只接受可回传的动作标识和 JSON 对象 schema。
//
// Args:
//
//	action: 当前用户待确认项。
//
// Returns:
//
//	动作与回答 schema 结构有效时为真。
func validAction(action *Action) bool {
	if action == nil || action.ID == "" || len(action.ID) > 200 || action.Type == "" {
		return false
	}
	var schema map[string]json.RawMessage
	return json.Unmarshal(action.Schema, &schema) == nil && schema != nil
}
