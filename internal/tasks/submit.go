package tasks

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/requests"
)

// Recovery 保留原始会话和幂等标识，避免响应丢失后生成新任务。
type Recovery struct {
	ThreadID  string `json:"thread_id"`
	RequestID string `json:"client_request_id"`
}

// SubmissionError 携带可公开恢复信息，不包含用户输入或服务内部响应。
type SubmissionError struct {
	Recovery Recovery
	cause    error
}

// Error 返回底层安全摘要，不暴露恢复记录路径或输入。
//
// Returns:
//
//	已清洗的错误摘要。
func (e *SubmissionError) Error() string { return e.cause.Error() }

// Unwrap 保留机器错误码和网络取消原因。
//
// Returns:
//
//	原始安全错误。
//
// Errors:
//
//	只提供错误链，不执行重试。
func (e *SubmissionError) Unwrap() error { return e.cause }

// Acknowledgement 表示请求已被云端接收，不表示该轮已完成。
type Acknowledgement struct {
	ThreadID              string `json:"thread_id"`
	TurnID                string `json:"turn_id"`
	RequestID             string `json:"client_request_id"`
	ConversationTurnIndex int    `json:"conversation_turn_idx"`
	Status                string `json:"status"`
	CloudStatus           string `json:"cloud_status"`
	Existing              bool   `json:"existing"`
}

// submitResponse 只解码现有接口中可公开的回合接收字段。
type submitResponse struct {
	ThreadID              string `json:"thread_id"`
	TurnID                string `json:"turn_id"`
	ConversationTurnIndex int    `json:"conversation_turn_idx"`
	Status                string `json:"status"`
	Existing              bool   `json:"existing"`
}

// ValidateSubmission 在获取凭证和连接之前拒绝无效输入。
//
// Args:
//
//	threadID: 真实会话 UUID。
//	requestID: 空值表示生成标识，否则须是 UUID。
//	message: 非空且最多两万 Unicode 字符的用户输入。
//
// Returns:
//
//	参数校验结果。
//
// Errors:
//
//	返回 invalid_input，不回显输入。
func ValidateSubmission(threadID, requestID, message string) error {
	if err := ValidateIDs(threadID, threadID); err != nil {
		return err
	}
	if requestID != "" && !identifier.MatchString(requestID) {
		return &api.Error{Code: "invalid_input"}
	}
	return ValidateMessage(message)
}

// ValidateMessage checks the first or subsequent user message before any network write.
//
// Args:
//
//	message: User input to submit to a cloud turn.
//
// Returns:
//
// An invalid_input error for empty, malformed, or oversized input.
func ValidateMessage(message string) error {
	if !utf8.ValidString(message) || strings.TrimSpace(message) == "" || utf8.RuneCountInString(message) > 20000 {
		return &api.Error{Code: "invalid_input"}
	}
	return nil
}

// Submit 先记录不可变幂等标识，再调用现有回合接口一次，不等待 Agent loop。
//
// Args:
//
//	ctx: 本地请求上下文，取消不停止已接收任务。
//	directory: 非秘密恢复记录目录。
//	threadID: 已有根会话 UUID。
//	requestID: 可选的原请求 UUID，重试必须沿用。
//	message: 本轮完整用户输入。
//
// Returns:
//
//	公开接收回执，或带相同请求标识的恢复错误。
//
// Errors:
//
//	响应丢失或无法确认接收时返回 acceptance_unknown，不自动再提交。
func (c *Client) Submit(ctx context.Context, directory, threadID, requestID, message string) (Acknowledgement, error) {
	if err := ValidateSubmission(threadID, requestID, message); err != nil {
		return Acknowledgement{}, err
	}
	if requestID == "" {
		var err error
		requestID, err = requests.NewID()
		if err != nil {
			return Acknowledgement{}, err
		}
	}
	record, err := requests.Prepare(directory, c.api.Origin(), threadID, requestID, message)
	if err != nil {
		return Acknowledgement{}, err
	}
	recovery := Recovery{ThreadID: record.ThreadID, RequestID: record.RequestID}
	var result submitResponse
	body := struct {
		Message   string `json:"message"`
		RequestID string `json:"client_request_id"`
	}{message, requestID}
	endpoint := "/api/v1/usight-agent/threads/" + threadID + "/turns"
	err = c.api.DoJSON(ctx, http.MethodPost, endpoint, nil, body, &result)
	if err != nil {
		return Acknowledgement{}, submissionFailure(recovery, err)
	}
	if result.ThreadID != threadID || !identifier.MatchString(result.TurnID) || result.ConversationTurnIndex < 1 || !validAcknowledgementStatus(result.Status) {
		return Acknowledgement{}, submissionFailure(recovery, &api.Error{Code: "invalid_response"})
	}
	return Acknowledgement{ThreadID: threadID, TurnID: result.TurnID, RequestID: requestID, ConversationTurnIndex: result.ConversationTurnIndex, Status: "accepted", CloudStatus: result.Status, Existing: result.Existing}, nil
}

// validAcknowledgementStatus 限制接收回执的云端状态集合。
//
// Args:
//
//	status: 现有接口持久化状态。
//
// Returns:
//
//	状态已定义时为真。
func validAcknowledgementStatus(status string) bool {
	switch status {
	case "pending", "running", "paused", "interrupted", "waiting", "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}

// submissionFailure 区分明确拒绝和未知接收，保留原幂等标识。
//
// Args:
//
//	recovery: 已落盘的原始请求身份。
//	err: 单次 HTTP 或响应校验错误。
//
// Returns:
//
//	可公开的恢复错误。
//
// Errors:
//
//	明确 4xx 保留拒绝码，其余情况标为 acceptance_unknown。
func submissionFailure(recovery Recovery, err error) error {
	var remote *api.Error
	if !errors.As(err, &remote) || remote.Status < 400 || remote.Status >= 500 {
		err = &api.Error{Code: "acceptance_unknown"}
	}
	return &SubmissionError{Recovery: recovery, cause: err}
}
