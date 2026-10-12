package cli

import "github.com/spf13/cobra"

// commandGroup 让不存在的子命令返回失败，同时保留无参数帮助入口。
//
// Args:
//
//	name: 稳定命令组名称。
//	short: 命令组用途。
//
// Returns:
//
//	拒绝多余参数并显式输出帮助的命令组。
func commandGroup(name, short string) *cobra.Command {
	return &cobra.Command{Use: name, Short: short, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
}
