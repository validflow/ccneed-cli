package resources

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/validflow/ccneed-cli/internal/api"
)

// KBStartResult 标识已接收的知识库采集，不声称采集已经完成。
type KBStartResult struct {
	KBID      int64  `json:"kb_id"`
	TaskRunID int64  `json:"task_run_id"`
	Status    string `json:"status"`
}

// PlanKnowledgeBase 将宿主生成的方案交给服务器校验、报价和适配余额，不启动采集。
//
// Args:
//
//	ctx: 本次准备请求的上下文。
//	plan: 宿主生成的结构化研究需求、平台和关键词。
//
// Returns:
//
//	服务器的完整公开方案，或安全错误。
//
// Errors:
//
//	参数、认证或响应无效时失败，不创建知识库或扣款。
func (c *Client) PlanKnowledgeBase(ctx context.Context, plan map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	if plan == nil {
		return nil, &api.Error{Code: "invalid_input"}
	}
	var result map[string]json.RawMessage
	if err := c.api.DoJSON(ctx, http.MethodPost, "/api/v1/usight-agent/kbs/plan", nil, plan, &result); err != nil {
		return nil, err
	}
	var status string
	if json.Unmarshal(result["status"], &status) != nil || (status != "ready" && status != "insufficient_credits" && status != "invalid_plan") {
		return nil, &api.Error{Code: "invalid_response"}
	}
	return result, nil
}

// ValidatePreparedPlan 检查待展示文件与 token 内签名方案一致，服务器仍负责验签和过期校验。
//
// Args:
//
//	plan: 原样保存的服务器准备结果，包含 plan_token。
//
// Returns:
//
//	准备结果对应的 token，或安全错误。
//
// Errors:
//
//	非就绪方案、缺少 token 或展示字段与 token 不一致时拒绝执行。
func ValidatePreparedPlan(plan map[string]json.RawMessage) (string, error) {
	var status, token string
	if json.Unmarshal(plan["status"], &status) != nil || status != "ready" {
		return "", &api.Error{Code: "invalid_input"}
	}
	if json.Unmarshal(plan["plan_token"], &token) != nil {
		return "", &api.Error{Code: "invalid_input"}
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return "", &api.Error{Code: "invalid_input"}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !preparedPlanMatches(plan, payload) {
		return "", &api.Error{Code: "plan_mismatch"}
	}
	return token, nil
}

// preparedPlanMatches 比较 JSON 语义，不把 token 内容当成本地可授予的权限。
//
// Args:
//
//	plan: 服务器准备结果。
//	payload: JWT 中间段解码后的公开数据。
//
// Returns:
//
//	删除 plan_token 后的展示数据与签名 plan 完全一致时为真。
func preparedPlanMatches(plan map[string]json.RawMessage, payload []byte) bool {
	var claims struct {
		Plan json.RawMessage `json:"plan"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return false
	}
	visible := make(map[string]json.RawMessage, len(plan))
	for key, value := range plan {
		if key != "plan_token" {
			visible[key] = value
		}
	}
	encoded, err := json.Marshal(visible)
	if err != nil {
		return false
	}
	left, leftErr := comparableJSON(encoded)
	right, rightErr := comparableJSON(claims.Plan)
	return leftErr == nil && rightErr == nil && reflect.DeepEqual(left, right)
}

// comparableJSON 保留数字文本，避免大整数经过浮点转换后掩盖方案差异。
//
// Args:
//
//	data: 完整 JSON 值。
//
// Returns:
//
//	可比较的数据，或 JSON 错误。
//
// Errors:
//
//	无效 JSON 返回解析错误，不回显内容。
func comparableJSON(data []byte) (any, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err := decoder.Decode(&value)
	return value, err
}

// StartKnowledgeBase 仅提交用户确认过的签名方案和幂等请求标识，不自动重试。
//
// Args:
//
//	ctx: 本地请求上下文，取消不撤回已接收的云端采集。
//	plan: 完整服务器准备结果。
//	requestID: 本次确认使用的规范 UUID；响应未知时重用而非生成新标识。
//
// Returns:
//
//	已接收的知识库及采集标识，或错误。
//
// Errors:
//
//	方案或标识无效时不发请求；无法确认写入是否接收时返回 acceptance_unknown。
func (c *Client) StartKnowledgeBase(ctx context.Context, plan map[string]json.RawMessage, requestID string) (KBStartResult, error) {
	if !ValidUUID(requestID) {
		return KBStartResult{}, &api.Error{Code: "invalid_input"}
	}
	token, err := ValidatePreparedPlan(plan)
	if err != nil {
		return KBStartResult{}, err
	}
	request := struct {
		PlanToken string `json:"plan_token"`
		RequestID string `json:"request_id"`
	}{token, requestID}
	var result KBStartResult
	if err := c.api.DoJSON(ctx, http.MethodPost, "/api/v1/usight-agent/kbs/start", nil, request, &result); err != nil {
		return KBStartResult{}, writeFailure(err)
	}
	if result.KBID <= 0 || result.TaskRunID <= 0 || result.Status == "" {
		return KBStartResult{}, &api.Error{Code: "acceptance_unknown"}
	}
	return result, nil
}
