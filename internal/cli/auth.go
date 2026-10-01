package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/credential"
)

// authCommand 将本地凭证状态输出与凭证读取、HTTP 业务请求分开。
//
// Args:
//
//	connection: 用户显式选择的服务 origin。
//	vault: 系统凭证存储；未配置时状态检查失败关闭。
//
// Returns:
//
//	不声称完成官网授权的认证命令组。
func authCommand(connection *connectionConfig, vault *credential.Vault) *cobra.Command {
	command := &cobra.Command{Use: "auth", Short: "Manage CCNeed authorization"}
	command.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Inspect local credentials; does not validate server authorization",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, err := vault.Inspect(cmd.Context(), connection.origin)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		},
	})
	return command
}
