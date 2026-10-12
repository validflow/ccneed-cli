package cli

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/tasks"
)

// taskReplyCommand 使用一份通用答案文件，不按每个 HITL tool 创建命令格式。
//
// Args:
//
//	connection: 当前 origin 的认证连接配置。
//
// Returns:
//
//	单个结构化确认命令。
func taskReplyCommand(connection *connectionConfig) *cobra.Command {
	var threadID, turnID, actionID, answerFile string
	command := &cobra.Command{Use: "reply", Short: "Submit a user-confirmed JSON answer to the current action", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := tasks.ValidateIDs(threadID, turnID); err != nil {
				return err
			}
			if err := tasks.ValidateAction(actionID); err != nil {
				return err
			}
			answer, err := readAnswerFile(answerFile)
			if err != nil {
				return err
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := tasks.New(client).Reply(cmd.Context(), threadID, turnID, actionID, answer)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	command.Flags().StringVar(&threadID, "thread-id", "", "Root conversation UUID")
	command.Flags().StringVar(&turnID, "turn-id", "", "Durable conversation turn UUID")
	command.Flags().StringVar(&actionID, "action-id", "", "Pending action_id from the latest task snapshot")
	command.Flags().StringVar(&answerFile, "answer-file", "", "Regular JSON file matching resume_schema (maximum 64 KiB)")
	return command
}

// readAnswerFile 在连接服务前读取有界普通文件，不从参数自然语言猜测答案。
//
// Args:
//
//	filename: 用户指定的普通 JSON 文件；不隐式读取 stdin 或跟随符号链接。
//
// Returns:
//
//	完整结构化答案，或安全错误。
//
// Errors:
//
//	空路径、非常规文件、读取失败或 JSON 不合法时失败。
func readAnswerFile(filename string) (map[string]json.RawMessage, error) {
	if filename == "" {
		return nil, &api.Error{Code: "invalid_input"}
	}
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, &api.Error{Code: "answer_read"}
	}
	if !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil, &api.Error{Code: "invalid_input"}
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, &api.Error{Code: "answer_read"}
	}
	defer func() { _ = file.Close() }()
	return tasks.ParseAnswer(file)
}
