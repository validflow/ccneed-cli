package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/artifacts"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// artifactsCommand 提供网站可见的公开工作空间产出。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	产出命令组。
func artifactsCommand(connection *connectionConfig) *cobra.Command {
	command := commandGroup("artifacts", "Read public findings and reports")
	command.AddCommand(artifactListCommand(connection))
	command.AddCommand(artifactContentCommand(connection, false), artifactContentCommand(connection, true))
	return command
}

// artifactListCommand 获取有上限的公开 manifest，不显示内部执行 UUID。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	产出列表命令。
func artifactListCommand(connection *connectionConfig) *cobra.Command {
	var threadID string
	command := &cobra.Command{Use: "list", Short: "List current public findings and reports", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if !resources.ValidUUID(threadID) {
			return &api.Error{Code: "invalid_input"}
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := artifacts.New(client).List(cmd.Context(), threadID)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().StringVar(&threadID, "thread-id", "", "Existing root conversation UUID")
	return command
}
