package cli

import (
	"encoding/json"
	"time"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/tasks"
)

// tasksCommand 提供指定云端回合的公开读取和有界观察，不重新发起任务。
//
// Args:
//
//	connection: 当前 origin 的认证连接配置。
//
// Returns:
//
//	模块化任务命令组。
func tasksCommand(connection *connectionConfig) *cobra.Command {
	command := commandGroup("tasks", "Read and observe cloud conversation turns")
	command.AddCommand(taskReadCommand(connection, false), taskReadCommand(connection, true))
	command.AddCommand(taskReplyCommand(connection))
	command.AddCommand(taskListCommand(connection))
	return command
}

// taskReadCommand 在连接服务前校验身份参数和本地等待期限。
//
// Args:
//
//	connection: 当前 origin 的认证连接配置。
//	watch: 为真时有界观察，为假时只读取一次。
//
// Returns:
//
//	单个查询或观察命令。
func taskReadCommand(connection *connectionConfig, watch bool) *cobra.Command {
	var threadID, turnID string
	timeout := 60 * time.Second
	name := "get"
	if watch {
		name = "watch"
	}
	command := &cobra.Command{Use: name, Short: "Read a public turn snapshot", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := tasks.ValidateIDs(threadID, turnID); err != nil {
				return err
			}
			if watch {
				if err := tasks.ValidateTimeout(timeout); err != nil {
					return err
				}
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			service := tasks.New(client)
			if watch {
				result, err := service.Watch(cmd.Context(), threadID, turnID, timeout)
				if err != nil {
					return err
				}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			result, err := service.Get(cmd.Context(), threadID, turnID)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	command.Flags().StringVar(&threadID, "thread-id", "", "Root conversation UUID")
	command.Flags().StringVar(&turnID, "turn-id", "", "Durable conversation turn UUID")
	if watch {
		command.Flags().DurationVar(&timeout, "timeout", timeout, "Local observation deadline (1s–10m); does not cancel the cloud task")
	}
	return command
}
