package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/tasks"
)

// taskListCommand 按会话顺序列出历史回合或找回原始提交的确认。
//
// Args:
//
//	connection: 当前 origin 绑定的认证连接。
//
// Returns:
//
//	不执行云端写入的回合历史命令。
func taskListCommand(connection *connectionConfig) *cobra.Command {
	var threadID string
	options := tasks.HistoryOptions{Limit: 20}
	command := &cobra.Command{Use: "list", Short: "List durable conversation turns or recover the original submission", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := tasks.ValidateHistory(threadID, options); err != nil {
				return err
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := tasks.New(client).List(cmd.Context(), threadID, options)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	command.Flags().StringVar(&threadID, "thread-id", "", "Original conversation UUID")
	command.Flags().StringVar(&options.RequestID, "request-id", "", "Filter by the original submission UUID; never submits again")
	command.Flags().IntVar(&options.Limit, "limit", options.Limit, "Page size (1–100)")
	command.Flags().IntVar(&options.Before, "before-turn-index", 0, "Exclusive cursor from next_before_turn_index; omit for the newest page")
	return command
}
