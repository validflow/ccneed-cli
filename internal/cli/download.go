package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/artifacts"
)

// artifactContentCommand 分离文本读取与用户显式指定路径的下载。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//	download: 为真时写入用户指定路径，否则返回 JSON 文本。
//
// Returns:
//
//	单个公开产出内容命令。
func artifactContentCommand(connection *connectionConfig, download bool) *cobra.Command {
	var threadID, artifactID, output string
	name := "read"
	if download {
		name = "download"
	}
	command := &cobra.Command{Use: name, Short: "Read public text or save a report to an explicit local path", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := artifacts.ValidateIDs(threadID, artifactID); err != nil {
			return err
		}
		if download {
			if _, err := artifacts.ValidateOutput(output); err != nil {
				return err
			}
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		service := artifacts.New(client)
		if download {
			result, err := service.Download(cmd.Context(), threadID, artifactID, output)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		result, err := service.Read(cmd.Context(), threadID, artifactID)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().StringVar(&threadID, "thread-id", "", "Root conversation UUID")
	command.Flags().StringVar(&artifactID, "artifact-id", "", "Stable public artifact UUID")
	if download {
		command.Flags().StringVar(&output, "output", "", "Explicit local destination; existing files are never overwritten")
	}
	return command
}
