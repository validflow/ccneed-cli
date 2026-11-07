// Package localfile 校验本地文件目标并原子发布完整下载，不覆盖已有文件。
package localfile

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/validflow/ccneed-cli/internal/api"
)

// ValidateOutput 验证用户指定目标，拒绝静默覆盖和 Windows 备用数据流。
//
// Args:
//
//	output: 用户明确选择的本地文件路径。
//
// Returns:
//
//	规范绝对路径，或安全错误。
//
// Errors:
//
//	已存在目标返回 output_exists；非法文件名或父目录返回 invalid_output。
func ValidateOutput(output string) (string, error) {
	if output == "" || strings.ContainsAny(filepath.Base(output), "\\/:") || strings.IndexFunc(output, func(r rune) bool { return r < 32 || r == 127 }) != -1 {
		return "", &api.Error{Code: "invalid_output"}
	}
	target, err := filepath.Abs(output)
	if err != nil {
		return "", &api.Error{Code: "invalid_output"}
	}
	_, err = os.Lstat(target)
	if err == nil {
		return "", &api.Error{Code: "output_exists"}
	}
	if !os.IsNotExist(err) {
		return "", &api.Error{Code: "invalid_output"}
	}
	parent, err := os.Stat(filepath.Dir(target))
	if err != nil || !parent.IsDir() {
		return "", &api.Error{Code: "invalid_output"}
	}
	return target, nil
}

// Publish 完成持久化后以硬链接发布新路径，最终竞争中的既有文件同样不被覆盖。
//
// Args:
//
//	file: 仅由当前命令创建的临时文件。
//	target: 已校验的最终绝对路径。
//
// Returns:
//
//	发布结果。
//
// Errors:
//
//	不支持安全硬链接时返回 output_write，不退回会覆盖目标的 rename。
func Publish(file *os.File, target string) error {
	if file.Sync() != nil || file.Close() != nil {
		return &api.Error{Code: "output_write"}
	}
	if err := os.Link(file.Name(), target); err != nil {
		if os.IsExist(err) {
			return &api.Error{Code: "output_exists"}
		}
		return &api.Error{Code: "output_write"}
	}
	return nil
}
