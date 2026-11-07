package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// kbExportCommand 下载已有知识库的数据集，不要求用户走云端分析或网页确认。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	按数字 KB 标识导出 CSV 或 JSON 的命令。
func kbExportCommand(connection *connectionConfig) *cobra.Command {
	var kbID int64
	var format, output string
	command := &cobra.Command{Use: "export", Short: "Save existing knowledge-base data as CSV or JSON for local analysis", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := resources.ValidateKBExport(kbID, format, output); err != nil {
				return err
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := resources.New(client).ExportKnowledgeBase(cmd.Context(), kbID, format, output)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}}
	command.Flags().Int64Var(&kbID, "kb-id", 0, "Owned knowledge-base numeric ID")
	command.Flags().StringVar(&format, "format", "csv", "Data format: csv or json")
	command.Flags().StringVar(&output, "output", "", "Optional new local file; otherwise use the KB ID/name/UTC filename supplied by the server")
	return command
}
