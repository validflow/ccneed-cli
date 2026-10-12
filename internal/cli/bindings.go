package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// bindingReadCommand 读取当前绑定和版本，不使用原始 hydrate 接口。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	资料状态命令。
func bindingReadCommand(connection *connectionConfig) *cobra.Command {
	var threadID string
	command := &cobra.Command{Use: "resources", Short: "Read current bindings, server window and CAS version", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if !resources.ValidUUID(threadID) {
			return &api.Error{Code: "invalid_input"}
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := resources.New(client).GetBindings(cmd.Context(), threadID)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().StringVar(&threadID, "thread-id", "", "Existing root conversation UUID")
	return command
}

// bindingAppendCommand 要求调用方确认实际版本，只增加已有资源。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	受服务器窗口和 CAS 约束的追加命令。
func bindingAppendCommand(connection *connectionConfig) *cobra.Command {
	var threadID, version string
	var selection resources.Selection
	command := &cobra.Command{Use: "append", Short: "Append existing resources in an open report window", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := resources.ValidateAppend(threadID, selection, version); err != nil {
			return err
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := resources.New(client).Append(cmd.Context(), threadID, selection, version)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().StringVar(&threadID, "thread-id", "", "Existing root conversation UUID")
	command.Flags().StringVar(&version, "expected-version", "", "Exact metadata_version read from threads resources")
	command.Flags().Int64SliceVar(&selection.KBIDs, "kb-id", nil, "Existing knowledge-base IDs to add")
	command.Flags().StringSliceVar(&selection.FileIDs, "file-id", nil, "Existing file UUIDs to add")
	return command
}
