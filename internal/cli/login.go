package cli

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/auth"
	"github.com/validflow/ccneed-cli/internal/credential"
)

// loginCommand 提供分步登录，默认立即返回官网链接；完成步骤才有界等待。
//
// Args:
//
//	connection: 用户选择的服务。
//	vault: 同一系统凭证存储。
//
// Returns:
//
//	不接收终端秘密参数的登录命令。
func loginCommand(connection *connectionConfig, vault *credential.Vault) *cobra.Command {
	complete := false
	timeout := time.Minute
	command := &cobra.Command{Use: "login", Short: "Start website authorization immediately, or complete it after user approval", Args: cobra.NoArgs}
	command.Flags().BoolVar(&complete, "complete", false, "Complete the pending authorization for this origin after website approval")
	command.Flags().DurationVar(&timeout, "timeout", time.Minute, "Bounded wait for --complete (1s to 10m); timeout does not grant authorization")
	command.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := api.New(connection.origin, "", nil)
		if err != nil {
			return err
		}
		service := auth.New(client, vault, nil)
		var outcome auth.Outcome
		if complete {
			outcome, err = service.Complete(cmd.Context(), timeout)
		} else {
			if cmd.Flags().Changed("timeout") {
				return &api.Error{Code: "invalid_input"}
			}
			outcome, err = service.Begin(cmd.Context())
		}
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(outcome)
	}
	return command
}

// logoutCommand 默认先撤销当前服务凭证，再清理本地；离线清理必须明确选择。
//
// Args:
//
//	connection: 当前服务的已认证连接。
//	vault: 按 origin 隔离的存储。
//
// Returns:
//
//	当前设备注销命令，不影响网站或其他设备。
func logoutCommand(connection *connectionConfig, vault *credential.Vault) *cobra.Command {
	localOnly := false
	command := &cobra.Command{Use: "logout", Short: "Revoke this client credential, then remove it locally", Args: cobra.NoArgs}
	command.Flags().BoolVar(&localOnly, "local-only", false, "Remove only local credentials; does not revoke server authorization")
	command.RunE = func(cmd *cobra.Command, _ []string) error {
		origin, err := api.ParseOrigin(connection.origin)
		if err != nil {
			return err
		}
		if !localOnly {
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			var response struct {
				Revoked bool `json:"revoked"`
			}
			if err := client.DoJSON(cmd.Context(), http.MethodPost, "/api/v1/client/auth/revoke", nil, nil, &response); err != nil {
				return err
			}
			if !response.Revoked {
				return &api.Error{Code: "invalid_response"}
			}
		}
		if err := vault.Forget(cmd.Context(), origin.String()); err != nil {
			return err
		}
		if err := vault.ForgetPending(cmd.Context(), origin.String()); err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			Origin  string `json:"origin"`
			Removed bool   `json:"removed"`
			Revoked bool   `json:"revoked"`
		}{Origin: origin.String(), Removed: true, Revoked: !localOnly})
	}
	return command
}
