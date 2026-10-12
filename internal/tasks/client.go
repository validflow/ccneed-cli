// Package tasks 连接现有云端回合接口；不运行 Agent 或重新提交观察中的任务。
package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/validflow/ccneed-cli/internal/api"
)

var identifier = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

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
	if err := ValidateIDs(threadID, turnID); err != nil {
		return Detail{}, err
	}
	var result Detail
	endpoint := "/api/v1/usight-agent/threads/" + threadID + "/turns/" + turnID
	if err := c.api.DoJSON(ctx, http.MethodGet, endpoint, nil, nil, &result); err != nil {
		return Detail{}, err
	}
	if !validDetail(result, threadID, turnID) {
		return Detail{}, &api.Error{Code: "invalid_response"}
	}
	if result.Artifacts == nil {
		result.Artifacts = []Artifact{}
	}
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
	return validArtifacts(result.Artifacts)
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
