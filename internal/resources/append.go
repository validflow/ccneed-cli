package resources

import (
	"context"
	"errors"
	"github.com/validflow/ccneed-cli/internal/api"
	"net/http"
	"strconv"
)

// ValidateAppend 要求显式 CAS 版本和至少一项增加，不接受移除或替换。
//
// Args:
//
//	threadID: 真实根会话 UUID。
//	selection: 本次需要增加的已有资源。
//	version: 先读取的非负十进制服务器版本。
//
// Returns:
//
//	参数检查结果。
//
// Errors:
//
//	缺少资料、版本或标识无效时返回 invalid_input。
func ValidateAppend(threadID string, selection Selection, version string) error {
	if !ValidUUID(threadID) || !validRevision(version) || (len(selection.KBIDs) == 0 && len(selection.FileIDs) == 0) {
		return &api.Error{Code: "invalid_input"}
	}
	return ValidateSelection(selection)
}

// GetBindings 获取当前绑定与窗口，每次读取服务器而非缓存版本。
//
// Args:
//
//	ctx: 本地请求上下文。
//	threadID: 已有根会话 UUID。
//
// Returns:
//
//	当前可用于追加确认的资源版本，或错误。
//
// Errors:
//
//	参数、归属或持久化响应无效时失败关闭。
func (c *Client) GetBindings(ctx context.Context, threadID string) (Bindings, error) {
	if !ValidUUID(threadID) {
		return Bindings{}, &api.Error{Code: "invalid_input"}
	}
	var result Bindings
	endpoint := "/api/v1/usight-agent/threads/" + threadID + "/resources"
	if err := c.api.DoJSON(ctx, http.MethodGet, endpoint, nil, nil, &result); err != nil {
		return Bindings{}, err
	}
	if result.ThreadID != threadID || !validBindings(result) {
		return Bindings{}, &api.Error{Code: "invalid_response"}
	}
	return result, nil
}

// Append 调用网站既有报告窗口追加接口，窗口关闭或版本冲突由服务器拒绝。
//
// Args:
//
//	ctx: 本地请求上下文。
//	threadID: 原始根会话 UUID。
//	selection: 本次增加的已有资源。
//	version: 用户确认的服务器版本。
//
// Returns:
//
//	新绑定及版本回执，或错误。
//
// Errors:
//
//	不自动改版本或重试；响应未知时先 GetBindings 核对。
func (c *Client) Append(ctx context.Context, threadID string, selection Selection, version string) (Appended, error) {
	if err := ValidateAppend(threadID, selection, version); err != nil {
		return Appended{}, err
	}
	if selection.KBIDs == nil {
		selection.KBIDs = []int64{}
	}
	if selection.FileIDs == nil {
		selection.FileIDs = []string{}
	}
	body := struct {
		KBIDs   []int64  `json:"add_kb_ids"`
		FileIDs []string `json:"add_library_file_ids"`
		Version string   `json:"expected_metadata_version"`
	}{selection.KBIDs, selection.FileIDs, version}
	var result Appended
	endpoint := "/api/v1/usight-agent/threads/" + threadID + "/resources/append"
	if err := c.api.DoJSON(ctx, http.MethodPost, endpoint, nil, body, &result); err != nil {
		return Appended{}, writeFailure(err)
	}
	result.ThreadID = threadID
	if !validBindings(result.Bindings) {
		return Appended{}, &api.Error{Code: "acceptance_unknown"}
	}
	return result, nil
}

// validRevision 限制版本到服务器的非负整数契约。
//
// Args:
//
//	version: 服务器版本字符串。
//
// Returns:
//
//	可以无损解析为 uint64 时为真。
func validRevision(version string) bool {
	_, err := strconv.ParseUint(version, 10, 64)
	return err == nil
}

// validBindings 校验资料身份，不将单次增加上限误用于累计绑定数量。
//
// Args:
//
//	value: 服务器当前资料状态。
//
// Returns:
//
//	版本、窗口及全部标识有效时为真。
func validBindings(value Bindings) bool {
	if !ValidUUID(value.ThreadID) || !validRevision(value.Version) {
		return false
	}
	if value.Window != "initial" && value.Window != "locked" && value.Window != "report_window" {
		return false
	}
	for _, id := range value.KBIDs {
		if id <= 0 {
			return false
		}
	}
	for _, id := range value.FileIDs {
		if !ValidUUID(id) {
			return false
		}
	}
	return true
}

// writeFailure 区分明确拒绝和未知写入，不做盲目重试。
//
// Args:
//
//	err: 单次网络或响应错误。
//
// Returns:
//
//	可公开的写入错误。
//
// Errors:
//
//	非明确 4xx 返回 acceptance_unknown。
func writeFailure(err error) error {
	var remote *api.Error
	if errors.As(err, &remote) && remote.Status >= 400 && remote.Status < 500 {
		return err
	}
	return &api.Error{Code: "acceptance_unknown"}
}
