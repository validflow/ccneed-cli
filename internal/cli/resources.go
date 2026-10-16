package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/resources"
)

// kbCommand 分离只读知识库状态与需要用户在官网完成的创建流程。
//
// Args:
//
//	connection: 明确选择的服务 origin。
//
// Returns:
//
//	知识库命令组。
func kbCommand(connection *connectionConfig) *cobra.Command {
	command := commandGroup("kb", "Read knowledge bases; create through the website")
	command.AddCommand(kbListCommand(connection), kbCreateCommand(connection))
	command.AddCommand(kbGetCommand(connection))
	command.AddCommand(kbStatsCommand(connection))
	return command
}

// kbStatsCommand 返回知识库的有限聚合和同环境网页入口。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	知识库统计命令。
func kbStatsCommand(connection *connectionConfig) *cobra.Command {
	var id int64
	command := &cobra.Command{Use: "stats", Short: "Read published KB counts and distributions", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if id <= 0 {
			return &api.Error{Code: "invalid_input"}
		}
		origin, err := connection.selectedOrigin()
		if err != nil {
			return err
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		stats, err := resources.New(client).KnowledgeBaseStatistics(cmd.Context(), id)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			resources.KnowledgeBaseStats
			WebURL string `json:"web_url"`
		}{stats, origin.String() + "/insightflow/kb"})
	}}
	command.Flags().Int64Var(&id, "kb-id", 0, "Existing owned knowledge-base numeric ID")
	return command
}

// kbGetCommand 按明确知识库标识读取状态，不对同名资料做自动推断。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	单个知识库状态命令。
func kbGetCommand(connection *connectionConfig) *cobra.Command {
	var id int64
	command := &cobra.Command{Use: "get", Short: "Read one knowledge base by its numeric ID", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if id <= 0 {
			return &api.Error{Code: "invalid_input"}
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := resources.New(client).KnowledgeBaseStatus(cmd.Context(), id)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().Int64Var(&id, "kb-id", 0, "Existing owned knowledge-base numeric ID")
	return command
}

// kbListCommand 读取知识库当前可用状态，不触发采集或自动选择资料。
//
// Args:
//
//	connection: 当前认证连接。
//
// Returns:
//
//	知识库列表命令。
func kbListCommand(connection *connectionConfig) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List owned knowledge bases and collection status", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := resources.New(client).KnowledgeBases(cmd.Context())
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
}

// kbCreateCommand 只给出同一环境的官网入口，不复制意图、配额和确认流程。
//
// Args:
//
//	connection: 用户明确选择的服务 origin。
//
// Returns:
//
//	需要用户在网站完成的创建交接命令。
func kbCreateCommand(connection *connectionConfig) *cobra.Command {
	return &cobra.Command{Use: "create", Short: "Return the website knowledge-base creation entry", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		origin, err := connection.selectedOrigin()
		if err != nil {
			return err
		}
		result := struct {
			Status   string `json:"status"`
			URL      string `json:"url"`
			NextStep string `json:"next_step"`
		}{"website_required", origin.String() + "/insightflow/kb", "Create the knowledge base on this website; then use kb list to find its ID and selection status."}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
}

// filesCommand 分离独立文件资料库与对话绑定。
//
// Args:
//
//	connection: 当前 origin 认证连接。
//
// Returns:
//
//	文件命令组。
func filesCommand(connection *connectionConfig) *cobra.Command {
	command := commandGroup("files", "Manage independent uploaded user files")
	command.AddCommand(fileListCommand(connection))
	command.AddCommand(fileUploadCommand(connection))
	return command
}

// fileListCommand 显示可绑定的上传文件标识。
//
// Args:
//
//	connection: 当前 origin 认证连接。
//
// Returns:
//
//	资料库列表命令。
func fileListCommand(connection *connectionConfig) *cobra.Command {
	limit := 200
	command := &cobra.Command{Use: "list", Short: "List owned uploaded files", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := resources.ValidateFileLimit(limit); err != nil {
			return err
		}
		client, err := connection.client(cmd.Context())
		if err != nil {
			return err
		}
		result, err := resources.New(client).Files(cmd.Context(), limit)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	command.Flags().IntVar(&limit, "limit", limit, "Server result limit (1–500)")
	return command
}
