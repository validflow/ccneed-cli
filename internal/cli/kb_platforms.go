package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// kbPlatformsCommand 提供制定采集方案前的只读平台发现入口。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	查询当前服务可采集平台 ID 的命令。
func kbPlatformsCommand(connection *connectionConfig) *cobra.Command {
	return &cobra.Command{Use: "platforms", Short: "List available collection platform IDs before preparing a plan", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := resources.New(client).AvailablePlatforms(cmd.Context())
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}}
}
