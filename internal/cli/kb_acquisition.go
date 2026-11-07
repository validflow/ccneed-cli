package cli

import (
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/localfile"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// kbPlanCommand 请求服务器准备方案，同时保留 stdout 供宿主展示最终方案。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	无采集副作用的准备命令。
func kbPlanCommand(connection *connectionConfig) *cobra.Command {
	var filename, output string
	command := &cobra.Command{Use: "plan", Short: "Validate a collection plan and fit its quote to the available credits", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			plan, err := readKBPlanFile(filename)
			if err != nil {
				return err
			}
			if output != "" {
				if _, err := localfile.ValidateOutput(output); err != nil {
					return err
				}
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := resources.New(client).PlanKnowledgeBase(cmd.Context(), plan)
			if err != nil {
				return err
			}
			if output != "" {
				if err := resources.SavePreparedPlan(result, output); err != nil {
					return err
				}
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}}
	command.Flags().StringVar(&filename, "plan-file", "", "Structured plan JSON file; this command does not create or crawl a knowledge base")
	command.Flags().StringVar(&output, "output", "", "Optional new file for the exact prepared response; the full response still appears on stdout")
	return command
}

// kbStartCommand 将完整准备结果交给服务端执行，权限确认由宿主执行策略触发。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	需要原生权限确认的付费采集命令。
func kbStartCommand(connection *connectionConfig) *cobra.Command {
	var filename, requestID string
	command := &cobra.Command{Use: "start", Short: "Start the exact displayed and user-approved prepared plan", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !resources.ValidUUID(requestID) {
				return &api.Error{Code: "invalid_input"}
			}
			plan, err := readKBPlanFile(filename)
			if err != nil {
				return err
			}
			if _, err := resources.ValidatePreparedPlan(plan); err != nil {
				return err
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := resources.New(client).StartKnowledgeBase(cmd.Context(), plan, requestID)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}}
	command.Flags().StringVar(&filename, "plan-file", "", "Unmodified complete server-prepared response, including its plan_token")
	command.Flags().StringVar(&requestID, "request-id", "", "Stable UUID for this approved execution; do not replace it after an unknown result")
	return command
}

// readKBPlanFile 读取有界的普通 JSON 对象，不隐式读取 stdin 或跟随符号链接。
//
// Args:
//
//	filename: 宿主明确选择的输入或准备结果文件。
//
// Returns:
//
//	完整 JSON 对象，或安全错误。
//
// Errors:
//
//	缺少文件、非普通文件、超过 512 KiB 或 JSON 无效时失败。
func readKBPlanFile(filename string) (map[string]json.RawMessage, error) {
	if filename == "" {
		return nil, &api.Error{Code: "invalid_input"}
	}
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, &api.Error{Code: "plan_read"}
	}
	if !info.Mode().IsRegular() || info.Size() > 512*1024 {
		return nil, &api.Error{Code: "invalid_input"}
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, &api.Error{Code: "plan_read"}
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, 512*1024+1))
	if err != nil {
		return nil, &api.Error{Code: "plan_read"}
	}
	var plan map[string]json.RawMessage
	if len(data) > 512*1024 || json.Unmarshal(data, &plan) != nil || plan == nil {
		return nil, &api.Error{Code: "invalid_input"}
	}
	return plan, nil
}
