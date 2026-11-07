package resources

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/localfile"
)

// SavePreparedPlan 将完整准备结果保存为新文件，不改变后续 stdout 展示的数据。
//
// Args:
//
//	plan: 服务器返回的完整公开方案。
//	output: 用户明确选择、尚不存在的文件路径。
//
// Returns:
//
//	写入错误，或 nil。
//
// Errors:
//
//	目标存在、编码或持久化失败时返回安全错误，不覆盖文件。
func SavePreparedPlan(plan map[string]json.RawMessage, output string) error {
	target, err := localfile.ValidateOutput(output)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(plan)
	if err != nil || len(encoded) > 512*1024 {
		return &api.Error{Code: "invalid_response"}
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".ccneed-kb-plan-*")
	if err != nil {
		return &api.Error{Code: "output_write"}
	}
	defer func() { _ = temporary.Close(); _ = os.Remove(temporary.Name()) }()
	if _, err := temporary.Write(append(encoded, '\n')); err != nil {
		return &api.Error{Code: "output_write"}
	}
	return localfile.Publish(temporary, target)
}
