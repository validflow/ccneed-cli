package cli

import (
	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/credential"
)

// authCommand 将本地凭证状态输出与凭证读取、HTTP 业务请求分开。
//
// Args:
//
//	connection: 用户显式选择的服务 origin。
//	vault: 系统凭证存储；未配置时状态检查失败关闭。
//
// Returns:
//
//	分离登录、撤销、本地状态和显式服务器校验的认证命令组。
func authCommand(connection *connectionConfig, vault *credential.Vault) *cobra.Command {
	command := commandGroup("auth", "Manage CCNeed authorization")
	command.AddCommand(loginCommand(connection, vault), logoutCommand(connection, vault))
	command.AddCommand(statusCommand(connection, vault))
	return command
}
