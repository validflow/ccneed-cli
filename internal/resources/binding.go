// Package resources 管理现有知识库、用户文件及追加绑定契约。
package resources

import (
	"github.com/validflow/ccneed-cli/internal/api"
	"regexp"
)

var uuidPattern = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

// Selection 是明确选择的已有资源标识，不包含本地文件或采集配置。
type Selection struct {
	KBIDs   []int64
	FileIDs []string
}

// ValidUUID 验证规范 UUID，禁止调用方改变目标 API 路径。
//
// Args:
//
//	value: 用户或服务器返回的公开标识。
//
// Returns:
//
//	小写规范 UUID 时为真。
func ValidUUID(value string) bool { return uuidPattern.MatchString(value) }

// ValidateSelection 在网络请求前检查已有资源的数量和标识。
//
// Args:
//
//	selection: 最多二十个知识库与一百个用户文件标识。
//
// Returns:
//
//	参数检查结果。
//
// Errors:
//
//	无效标识或超限返回 invalid_input，不接收解绑参数。
func ValidateSelection(selection Selection) error {
	if len(selection.KBIDs) > 20 || len(selection.FileIDs) > 100 {
		return &api.Error{Code: "invalid_input"}
	}
	for _, id := range selection.KBIDs {
		if id <= 0 {
			return &api.Error{Code: "invalid_input"}
		}
	}
	for _, id := range selection.FileIDs {
		if !ValidUUID(id) {
			return &api.Error{Code: "invalid_input"}
		}
	}
	return nil
}
