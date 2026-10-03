package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/requests"
	"github.com/validflow/ccneed-cli/internal/tasks"
)

// requestsCommand 将本地恢复查询独立于云端任务状态和执行控制。
//
// Args:
//
//	connection: 用户显式选择的 origin；读取不获取凭证。
//
// Returns:
//
//	本地恢复命令组。
func requestsCommand(connection *connectionConfig) *cobra.Command {
	command := commandGroup("requests", "Find local original submission IDs; check cloud acceptance with tasks list")
	command.AddCommand(requestListCommand(connection))
	return command
}

// requestListCommand 返回当前 origin/thread 的原始标识，不包含提示词摘要。
//
// Args:
//
//	connection: 当前服务 origin。
//
// Returns:
//
//	本地提交引用查询命令。
func requestListCommand(connection *connectionConfig) *cobra.Command {
	var threadID string
	command := &cobra.Command{Use: "list", Short: "Recover original request IDs after terminal interruption", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := tasks.ValidateIDs(threadID, threadID); err != nil {
				return err
			}
			origin, err := api.ParseOrigin(connection.origin)
			if err != nil {
				return err
			}
			directory, err := requests.DefaultDirectory()
			if err != nil {
				return err
			}
			records, err := requests.List(directory, origin.String(), threadID)
			if err != nil {
				return err
			}
			result := struct {
				Requests        []requests.Reference `json:"requests"`
				CloudAcceptance string               `json:"cloud_acceptance"`
			}{records, "unknown"}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	command.Flags().StringVar(&threadID, "thread-id", "", "Original conversation UUID")
	return command
}
