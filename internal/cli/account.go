package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/account"
	"github.com/validflow/ccneed-cli/internal/api"
)

// accountCommand 提供当前账号的只读概览、余额和近期记录。
//
// Args:
//
//	connection: 显式 origin 和认证解析器。
//
// Returns:
//
//	不包含支付写操作的账号命令组。
func accountCommand(connection *connectionConfig) *cobra.Command {
	command := &cobra.Command{Use: "account", Short: "Read account credits, subscriptions and recent credit history"}
	command.AddCommand(&cobra.Command{
		Use: "credits", Short: "Read available credits", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := account.New(client).Credits(cmd.Context())
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "overview", Short: "Read credits and effective subscriptions", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := account.New(client).Overview(cmd.Context())
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	})
	command.AddCommand(accountHistoryCommand(connection))
	return command
}

// accountHistoryCommand 暴露近期积分窗口，并在连接服务前校验本地数量限制。
//
// Args:
//
//	connection: 当前 origin 的认证解析器。
//
// Returns:
//
//	不包含分页或支付写操作的记录命令。
func accountHistoryCommand(connection *connectionConfig) *cobra.Command {
	limit := 20
	command := &cobra.Command{
		Use: "history", Short: "Read recent credit history (source window: 200 entries)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if limit < 1 || limit > 200 {
				return &api.Error{Code: "invalid_input"}
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := account.New(client).History(cmd.Context(), limit)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	command.Flags().IntVar(&limit, "limit", limit, "Number of recent entries to return (1–200; not server pagination)")
	return command
}
