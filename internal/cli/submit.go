package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/requests"
	"github.com/validflow/ccneed-cli/internal/tasks"
)

// submitCommand 提交已有会话的一轮输入，立即返回回合身份而非阻塞云端执行。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	异步提交命令。
func submitCommand(connection *connectionConfig) *cobra.Command {
	var threadID, requestID string
	command := &cobra.Command{Use: "run MESSAGE", Short: "Submit one cloud turn in an existing conversation", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := tasks.ValidateSubmission(threadID, requestID, args[0]); err != nil {
				return err
			}
			directory, err := requests.DefaultDirectory()
			if err != nil {
				return err
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := tasks.New(client).Submit(cmd.Context(), directory, threadID, requestID, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}}
	command.Flags().StringVar(&threadID, "thread-id", "", "Existing root conversation UUID")
	command.Flags().StringVar(&requestID, "request-id", "", "Original request UUID when explicitly retrying; otherwise generated and recorded before submission")
	return command
}
