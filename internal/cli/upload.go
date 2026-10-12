package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// fileUploadCommand 先验证本地文件，再独立上传，不自动绑定或启动采集。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	流式上传命令。
func fileUploadCommand(connection *connectionConfig) *cobra.Command {
	return &cobra.Command{Use: "upload FILE", Short: "Upload one local file independently (maximum 100 MiB)", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		source, err := resources.OpenUpload(args[0])
		if err != nil {
			return err
		}
		defer func() { _ = source.Close() }()
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := resources.New(client).Upload(cmd.Context(), source)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
}
