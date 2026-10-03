package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/conversations"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// threadsCommand 提供真实会话列表和初始资料绑定。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	模块化会话命令组。
func threadsCommand(connection *connectionConfig) *cobra.Command {
	command := commandGroup("threads", "Create and select real website conversations")
	command.AddCommand(threadCreateCommand(connection), threadListCommand(connection))
	command.AddCommand(bindingReadCommand(connection), bindingAppendCommand(connection))
	return command
}

// threadCreateCommand 将文件上传与会话初始绑定分开，不隐式创建知识库。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	会话创建命令。
func threadCreateCommand(connection *connectionConfig) *cobra.Command {
	var selection resources.Selection
	var title string
	command := &cobra.Command{Use: "create", Short: "Create a conversation with existing resources", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := conversations.ValidateCreate(selection, title); err != nil {
			return err
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := conversations.New(client).Create(cmd.Context(), selection, title)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().Int64SliceVar(&selection.KBIDs, "kb-id", nil, "Existing owned knowledge-base IDs (repeat or comma separate)")
	command.Flags().StringSliceVar(&selection.FileIDs, "file-id", nil, "Existing uploaded file UUIDs (repeat or comma separate)")
	command.Flags().StringVar(&title, "title", "", "Optional initial title; the service owns title generation")
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
