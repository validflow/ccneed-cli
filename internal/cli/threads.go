package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/conversations"
)

// threadsCommand 提供真实会话列表和后续资料管理。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	模块化会话命令组。
func threadsCommand(connection *connectionConfig) *cobra.Command {
	command := commandGroup("threads", "List and manage website conversations")
	command.AddCommand(threadListCommand(connection))
	command.AddCommand(bindingReadCommand(connection), bindingAppendCommand(connection))
	return command
}

// threadListCommand 显示标题用于选择，之后必须用真实 thread_id 调用。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	会话列表命令。
func threadListCommand(connection *connectionConfig) *cobra.Command {
	view := "active"
	limit := 50
	command := &cobra.Command{Use: "list", Short: "List recent conversations and their real IDs", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := conversations.ValidateList(view, limit); err != nil {
			return err
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := conversations.New(client).List(cmd.Context(), view, limit)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().StringVar(&view, "view", view, "active or archived")
	command.Flags().IntVar(&limit, "limit", limit, "Local display limit (1–200); the service returns at most 200 recent entries")
	return command
}
