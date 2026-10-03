package cli

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/tasks"
)

// Run 执行命令并将公开错误输出为 JSON，进程入口据此决定退出码。
//
// Args:
//
//	ctx: 本地进程上下文；取消不停止云端任务。
//	args: 不包含可执行文件名的命令参数。
//	options: 命令输出与构建依赖。
//
// Returns:
//
//	成功为 0，命令错误为非零。
func Run(ctx context.Context, args []string, options Options) int {
	command := New(options)
	command.SetArgs(args)
	if err := command.ExecuteContext(ctx); err != nil {
		response := struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Status  int    `json:"http_status,omitempty"`
			} `json:"error"`
			Recovery *tasks.Recovery `json:"recovery,omitempty"`
		}{}
		response.Error.Code = "command_failed"
		response.Error.Message = "Command failed. Run ccneed --help for available commands."
		var apiError *api.Error
		if errors.As(err, &apiError) {
			response.Error.Code = apiError.Code
			response.Error.Status = apiError.Status
			response.Error.Message = "Service request failed. Check authentication, service availability and command inputs."
		}
		var submission *tasks.SubmissionError
		if errors.As(err, &submission) {
			response.Recovery = &submission.Recovery
		}
		if response.Error.Code == "acceptance_unknown" {
			response.Error.Message = "Acceptance is unknown. Do not submit a new request; reuse the original identifiers to recover."
		}
		_ = json.NewEncoder(command.OutOrStdout()).Encode(response)
		return 1
	}
	return 0
}
