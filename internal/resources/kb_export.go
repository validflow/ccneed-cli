package resources

import (
	"context"
	"os"
	"path/filepath"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/localfile"
)

const maxKBExportBytes = 1024 * 1024 * 1024

// KBExportResult 标识本地保存的数据集，不将导出描述为新的采集任务。
type KBExportResult struct {
	KBID       int64  `json:"kb_id"`
	KBName     string `json:"kb_name"`
	ExportedAt string `json:"exported_at"`
	Format     string `json:"format"`
	Path       string `json:"path"`
}

// ValidateKBExport 在连接服务器前检查 KB 标识、格式和显式输出路径。
//
// Args:
//
//	kbID: 知识库正整数标识。
//	format: csv 或 json。
//	output: 用户选择的目标路径；空字符串使用服务器安全文件名。
//
// Returns:
//
//	参数或输出路径错误，或 nil。
//
// Errors:
//
//	不允许的格式、标识或既有目标返回安全错误。
func ValidateKBExport(kbID int64, format, output string) error {
	if kbID <= 0 || (format != "csv" && format != "json") {
		return &api.Error{Code: "invalid_input"}
	}
	if output != "" {
		_, err := localfile.ValidateOutput(output)
		return err
	}
	return nil
}

// ExportKnowledgeBase 导出已有知识库数据，成功后以不可覆盖的路径发布完整文件。
//
// Args:
//
//	ctx: 本地下载上下文。
//	kbID: 用户拥有的知识库数字标识。
//	format: csv 或 json。
//	output: 可选本地目标；省略时采用服务器文件名。
//
// Returns:
//
//	KB 身份、时间、格式和保存路径，或错误。
//
// Errors:
//
//	网络、响应或落盘失败时清理临时文件，不覆盖已有文件。
func (c *Client) ExportKnowledgeBase(ctx context.Context, kbID int64, format, output string) (KBExportResult, error) {
	if err := ValidateKBExport(kbID, format, output); err != nil {
		return KBExportResult{}, err
	}
	directory := "."
	if output != "" {
		directory = filepath.Dir(output)
	}
	temporary, err := os.CreateTemp(directory, ".ccneed-kb-export-*")
	if err != nil {
		return KBExportResult{}, &api.Error{Code: "output_write"}
	}
	defer func() { _ = temporary.Close(); _ = os.Remove(temporary.Name()) }()
	metadata, err := c.api.CopyKBExport(ctx, kbID, format, temporary, maxKBExportBytes)
	if err != nil {
		return KBExportResult{}, err
	}
	if output == "" {
		output = metadata.Filename
	}
	target, err := localfile.ValidateOutput(output)
	if err != nil {
		return KBExportResult{}, err
	}
	if err := localfile.Publish(temporary, target); err != nil {
		return KBExportResult{}, err
	}
	return KBExportResult{KBID: kbID, KBName: metadata.KBName, ExportedAt: metadata.ExportedAt, Format: format, Path: target}, nil
}
