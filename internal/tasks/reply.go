package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"unicode"

	"github.com/validflow/ccneed-cli/internal/api"
)

// ReplyResult 确认恢复请求已排队，不声称 action 或云端任务已完成。
type ReplyResult struct {
	ThreadID string `json:"thread_id"`
	TurnID   string `json:"turn_id"`
	ActionID string `json:"action_id"`
	Status   string `json:"status"`
}

// ParseAnswer 读取最多 64 KiB 的完整 JSON 对象，原始答案不进入错误说明。
//
// Args:
//
//	reader: 调用方提供的答案字节流。
//
// Returns:
//
//	可原样编码的 JSON 对象，或安全错误。
//
// Errors:
//
//	读取失败、超过大小限制、不是对象或有多份 JSON 时失败。
func ParseAnswer(reader io.Reader) (map[string]json.RawMessage, error) {
	data, err := io.ReadAll(io.LimitReader(reader, 65537))
	if err != nil {
		return nil, &api.Error{Code: "answer_read"}
	}
	var answer map[string]json.RawMessage
	if len(data) > 65536 || json.Unmarshal(data, &answer) != nil || answer == nil {
		return nil, &api.Error{Code: "invalid_input"}
	}
	return answer, nil
}

// ValidateAction 检查当前服务给出的动作标识，拒绝空值、控制字符和超长输入。
//
// Args:
//
//	actionID: 待确认动作标识，不接受自然语言替代。
//
// Returns:
//
//	参数错误，或 nil。
//
// Errors:
//
//	动作格式不可用时返回 invalid_input。
func ValidateAction(actionID string) error {
	if actionID == "" || len(actionID) > 200 {
		return &api.Error{Code: "invalid_input"}
	}
	for _, value := range actionID {
		if unicode.IsControl(value) || unicode.IsSpace(value) {
			return &api.Error{Code: "invalid_input"}
		}
	}
	return nil
}

// Reply 向既有 resume 提交指定回合的结构化确认，不自动重试写请求。
//
// Args:
//
//	ctx: 本地请求上下文；取消不能撤回云端已接收的回答。
//	threadID: 根会话 UUID。
//	turnID: 仍待确认的 durable turn UUID。
//	actionID: 最近一次公开快照的 action_id。
//	answer: 依据 resume_schema 填写、已经用户确认的 JSON 对象。
//
// Returns:
//
//	恢复排队确认，或安全错误。
//
// Errors:
//
//	过期或无权 action、schema 不匹配时返回服务错误；响应丢失时返回 acceptance_unknown。
func (c *Client) Reply(ctx context.Context, threadID, turnID, actionID string, answer map[string]json.RawMessage) (ReplyResult, error) {
	if err := ValidateIDs(threadID, turnID); err != nil {
		return ReplyResult{}, err
	}
	if err := ValidateAction(actionID); err != nil {
		return ReplyResult{}, err
	}
	if err := validateAnswerObject(answer); err != nil {
		return ReplyResult{}, err
	}
	request := struct {
		TurnID   string                     `json:"turn_id"`
		ActionID string                     `json:"action_id"`
		Payload  map[string]json.RawMessage `json:"payload"`
	}{turnID, actionID, answer}
	var response struct {
		ThreadID      string `json:"thread_id"`
		Status        string `json:"status"`
		NextTurnIndex int    `json:"next_turn_idx"`
	}
	err := c.api.DoJSON(ctx, http.MethodPost, "/api/v1/usight-agent/threads/"+threadID+"/resume", nil, request, &response)
	if err != nil {
		return ReplyResult{}, writeFailure(err)
	}
	if response.ThreadID != threadID || response.Status != "active" || response.NextTurnIndex < 1 {
		return ReplyResult{}, &api.Error{Code: "acceptance_unknown"}
	}
	return ReplyResult{ThreadID: threadID, TurnID: turnID, ActionID: actionID, Status: "queued"}, nil
}

// validateAnswerObject 在执行 HTTP 前验证直接调用服务的答案对象。
//
// Args:
//
//	answer: 待编码的结构化答案。
//
// Returns:
//
//	编码或参数错误，或 nil。
//
// Errors:
//
//	nil、无效 RawMessage 或超过 64 KiB 时返回 invalid_input。
func validateAnswerObject(answer map[string]json.RawMessage) error {
	if answer == nil {
		return &api.Error{Code: "invalid_input"}
	}
	encoded, err := json.Marshal(answer)
	if err != nil || len(encoded) > 65536 {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// writeFailure 区分明确拒绝与可能已经接收的写请求，绝不盲目重发。
//
// Args:
//
//	err: 单次写请求产生的安全 API 错误。
//
// Returns:
//
//	明确服务拒绝的原错误，或接受状态未知错误。
//
// Errors:
//
//	无法确认是否接收时返回 acceptance_unknown，要求先查询原回合。
func writeFailure(err error) error {
	var failure *api.Error
	if errors.As(err, &failure) && failure.Status >= 400 && failure.Status < 500 {
		return err
	}
	return &api.Error{Code: "acceptance_unknown"}
}
