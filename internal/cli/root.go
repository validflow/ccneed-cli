// Package cli 负责命令参数与公开输出，不执行云端 Agent runtime。
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/credential"
	"github.com/validflow/ccneed-cli/internal/version"
)

// Options 是命令层的可注入依赖，不包含隐式全局状态。
type Options struct {
	Version     version.Info
	Stdout      io.Writer
	Stderr      io.Writer
	CheckUpdate func(context.Context, string) string
	Connect     func(context.Context, string) (*api.Client, error)
	Credentials *credential.Vault
}

// New 创建独立命令树；机器结果写 stdout，诊断写 stderr。
//
// Args:
//
//	options: 构建信息、输出通道和可选更新检测器。
//
// Returns:
//
//	可在测试或进程入口执行的命令树。
func New(options Options) *cobra.Command {
	command := &cobra.Command{Use: "ccneed", Short: "Connect your agent to CCNeed", SilenceErrors: true, SilenceUsage: true}
	command.SetOut(options.Stdout)
	command.SetErr(options.Stderr)
	noUpdateCheck := false
	command.PersistentFlags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable update checks and reminders for this invocation")
	connection := &connectionConfig{origin: "https://ccneed.ai", connect: options.Connect}
	if connection.connect == nil && options.Credentials != nil {
		connection.connect = func(ctx context.Context, origin string) (*api.Client, error) {
			return connection.vault(options.Credentials).Connect(ctx, origin)
		}
	}
	command.PersistentFlags().StringVar(&connection.origin, "api-url", connection.origin, "Explicit service origin for development; credentials stay origin-bound")
	command.PersistentFlags().BoolVar(&connection.allowInsecureHTTP, "allow-insecure-http", false, "Allow plaintext HTTP for an explicitly selected staging IP; credentials are exposed in transit")
	command.AddCommand(accountCommand(connection))
	command.AddCommand(tasksCommand(connection))
	command.AddCommand(submitCommand(connection))
	command.AddCommand(threadsCommand(connection))
	command.AddCommand(requestsCommand(connection))
	command.AddCommand(kbCommand(connection), filesCommand(connection), artifactsCommand(connection))
	command.AddCommand(authCommand(connection, options.Credentials))
	command.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print CLI build identity as JSON",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return json.NewEncoder(command.OutOrStdout()).Encode(options.Version)
		},
	})
	command.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
		if cmd.HasSubCommands() || noUpdateCheck || options.CheckUpdate == nil {
			return
		}
		latest := options.CheckUpdate(cmd.Context(), options.Version.Version)
		if version.NewerStable(latest, options.Version.Version) {
			_, _ = fmt.Fprintf(command.ErrOrStderr(), "CCNeed CLI %s is available. See https://github.com/validflow/ccneed-cli for the official update instructions.\n", latest)
		}
	}
	return command
}
