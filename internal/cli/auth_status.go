package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/account"
	"github.com/validflow/ccneed-cli/internal/credential"
)

// verifiedStatus 只在服务器完成账号只读鉴权后声明 authenticated。
type verifiedStatus struct {
	credential.Status
	Authenticated bool `json:"authenticated"`
}

// statusCommand 区分本地凭证元信息与显式服务器校验。
//
// Args:
//
//	connection: 用户选择的 origin 和凭证连接器。
//	vault: 系统凭证库。
//
// Returns:
//
//	默认不联网、--verify 才验证的状态命令。
func statusCommand(connection *connectionConfig, vault *credential.Vault) *cobra.Command {
	verify := false
	command := &cobra.Command{
		Use: "status", Short: "Inspect local credentials; use --verify to check server authorization", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, err := connection.vault(vault).Inspect(cmd.Context(), connection.origin)
			if err != nil {
				return err
			}
			if !verify {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			if _, err := account.New(client).Credits(cmd.Context()); err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(verifiedStatus{Status: status, Authenticated: true})
		},
	}
	command.Flags().BoolVar(&verify, "verify", false, "Verify this origin's credential with the server")
	return command
}
