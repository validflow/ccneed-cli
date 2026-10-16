package cli

import (
	"encoding/json"
	"time"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
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
	var limit int
	var beforeSeq, afterSeq int64
	timeout := 60 * time.Second
	name := "get"
	if watch {
		name = "watch"
	}
	command := &cobra.Command{Use: name, Short: "Read a public turn snapshot", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			page := tasks.PageOptions{Limit: limit, BeforeSeq: beforeSeq, AfterSeq: afterSeq}
			if err := validateTaskReadOptions(cmd, threadID, turnID, page, timeout, watch); err != nil {
				return err
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
			result, err := service.GetPage(cmd.Context(), threadID, turnID, page)
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
	} else {
		command.Flags().IntVar(&limit, "limit", 20, "Progress milestones to return (1-100)")
		command.Flags().Int64Var(&beforeSeq, "before-seq", 0, "Older progress before this exclusive sequence")
		command.Flags().Int64Var(&afterSeq, "after-seq", 0, "New progress after this exclusive sequence")
	}
	return command
}

// validateTaskReadOptions rejects invalid pagination and watch arguments before connecting.
//
// Args:
//
//	cmd: Active command, used to distinguish an explicit zero cursor from omission.
//	threadID: Root conversation UUID.
//	turnID: Durable turn UUID.
//	page: Requested progress page for a one-shot get.
//	timeout: Local watch deadline.
//	watch: Whether this command observes for a bounded period.
//
// Returns:
//
//	An input error, or nil.
//
// Errors:
//
//	Returns invalid_input for malformed IDs, cursors, limits, or timeouts.
func validateTaskReadOptions(cmd *cobra.Command, threadID, turnID string, page tasks.PageOptions, timeout time.Duration, watch bool) error {
	if err := tasks.ValidateIDs(threadID, turnID); err != nil {
		return err
	}
	if watch {
		return tasks.ValidateTimeout(timeout)
	}
	if (cmd.Flags().Changed("before-seq") && page.BeforeSeq == 0) ||
		(cmd.Flags().Changed("after-seq") && page.AfterSeq == 0) ||
		(cmd.Flags().Changed("before-seq") && cmd.Flags().Changed("after-seq")) {
		return &api.Error{Code: "invalid_input"}
	}
	return tasks.ValidatePageOptions(page)
}
