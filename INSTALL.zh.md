# 安装 CCNeed

[English](INSTALL.md) · 中文

全局 CLI 执行命令；Claude Code、Codex、WorkBuddy 桌面版通过各自原生插件加载统一 Skill 与采集 hooks，由宿主管理插件缓存和生命周期。

> 发行准备：本分支版本为 0.1.9，Agent 方案上下文和导出指引需确认该版本发布成功后使用。0.1.7 引入的采集与导出命令仍需配套 Club 接口。

## 安装与升级

需要 Node.js 22.14+、npm 及所选宿主。审阅官方来源并获得安装授权后：

```bash
npx @validflow/ccneed-cli@latest install --agent claude
# 也可以：--agent codex、--agent workbuddy、--agent claude,codex,workbuddy
```

安装器安装精确版本的全局 CLI，协调所选宿主的原生插件注册。升级时列出所有需要更新的宿主；不会自动登录或创建云端任务。npm 全局命令目录需在 Agent 的 PATH 中；本机安装不会把命令装进远程沙箱。

| 宿主 | 原生注册 | 额外权限配置 |
| --- | --- | --- |
| Claude Code | 调用 Claude marketplace/plugin 命令，核实用户级版本及启用状态 | 插件 PreToolUse 请求原生 `ask` |
| Codex | 调用 Codex marketplace/plugin 命令，核实已装版本及启用状态 | `~/.codex/rules/ccneed.rules` 为付费前缀设置 prompt |
| WorkBuddy 桌面版 | 按结果完成桌面端原生 marketplace/plugin 操作 | `~/.workbuddy/settings.json` 的兼容 ask 与 `sandbox.commandSafety.ask` |

全局包提供名为 `ccneed` 的本地 marketplace，插件为 `ccneed@ccneed`。宿主从插件加载 Skill 与 hooks，不重复注册独立全局 Skill 或用户级 hooks。原生缓存可以有副本；升级需重跑统一入口，不能认为单独 `npm install -g` 会刷新所有宿主。

检查 JSON 的 `plugins[]`。`installed` 表示原生管理器报告目标版本已启用，仍需信任 hooks 并重载会话。`action_required`、`failed` 表示接入未完成。WorkBuddy 桌面端目前没有已验证的公开 CLI 注册命令，因此会明确等待桌面操作，不能用 CodeBuddy CLI 注册冒充桌面端成功。

重启宿主，审阅并信任 hooks，确认 Skill 能加载。安装时信任插件与每次采集前授权是两回事。启停、卸载使用宿主原生插件管理。卸载不会卸载全局 CLI，也不会自动清理额外的 Codex/WB 权限规则；可保留这些保守规则，确认移除接入后仅清理对应 CCNeed 项，不删除其他规则。

下载需要代理时设置 `HTTPS_PROXY` 或 `HTTP_PROXY`；下载器也识别 `NO_PROXY` 及小写形式。固定版本时替换 `@latest`；RC 使用 `@next`。

## 采集确认

1. Agent 准备方案，CCNeed 校验并比较预计积分与余额。
2. PostToolUse 把方案上下文交给 Agent，由 Agent 用自然语言展示研究需求、各平台实体词和其他关键词、整个平台配额、预计积分与余额，再等待对话确认；不展示内部 T1/T2/T3 分配。
3. PreToolUse 核对同一最终方案，只显示短提示，不重复整份方案；正式执行仍需宿主原生权限确认。修改或过期后重新校验和展示，对话 HITL 批准同样受保护。
4. 采集完成后，Agent 通常自动导出 CSV 并交付本地文件；用户指定格式、目录或明确只要云端分析时按其要求处理。

| 宿主 | 支持的采集模式 |
| --- | --- |
| Claude Code | 交互式 default/acceptEdits、人工审批 |
| Codex | `on-request`、`approvals_reviewer = "user"`、启用 hooks/rules |
| WorkBuddy 桌面版 | 默认权限、原生 Bash、每次仅允许本次 |

Codex PreToolUse 不支持 `ask`，由原生命令规则确认。WorkBuddy 当前 PowerShell 沙箱包装无法可靠匹配付费前缀，因此会拦截付费 PowerShell；改用 Bash，Windows 可使用 Git Bash。允许整个会话会停止该前缀的逐次询问，需新会话恢复。已知 bypass/never/自动审批模式会被拒绝。

正式采集使用独立 `ccneed kb start ...`，不套脚本、复合命令或重定向。Codex 方案/回答文件需用绝对路径，因为 hooks 收不到命令覆盖后的工作目录。停用 hooks/rules 或人工审批后，不再满足本接入的逐次确认要求。

验收时准备方案并检查展示；拒绝原生采集确认，验证没有启动，再重新发起并批准。对话 HITL 同样验收。单元测试和文件存在不能代替桌面 UI 验收。

## 已有配置与失败处理

保留其他设置与自定义 Skill，仅迁移可识别的旧安装器 Skill 链接/hooks。未知或修改文件需人工处理。npm 托管 Skill 被修改时，替换前会停止；先将定制内容保留到其他 Skill 名称下。

CLI 成功但原生注册失败会逐宿主报告，不算整体成功。解决原因后重试；npm 与外部原生管理器不是一个事务。同名异源 marketplace 不会静默改指向。普通 npm `postinstall` 不修改宿主配置。

非默认 `CODEX_HOME` 或 `CLAUDE_CONFIG_DIR` 会返回需手动配置 profile，不会把插件与确认规则写到不同目录。统一安装使用默认 profile；自定义 profile 需将原生插件和权限规则一起配置并验证。

## 验证与登录

```bash
ccneed version --no-update-check
ccneed --help
ccneed auth login
# 将官网链接和确认码交给用户，由用户在网页上同意。
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
```

`auth: not_started` 表示还没登录。不替用户批准网站授权，不读浏览器 Cookie 或输出秘密。默认正式服务 `https://ccneed.ai`；仅获授权的开发者使用 `--api-url` 和隔离的 staging 账号，见[贡献说明](CONTRIBUTING.zh.md)。

## 源码构建

需要 Git 与 Go 1.25+：

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
```

将 `go env GOBIN` 或 GOPATH 的 `bin` 加入 PATH。这只构建开发命令，不注册原生插件；记录 revision，并让插件来源保持一致。离线安装器帮助：`node scripts/npm/run.mjs install --help`。

参考：[Claude 插件](https://code.claude.com/docs/en/plugins-reference)、[Codex 插件 hooks](https://learn.chatgpt.com/docs/hooks#plugin-bundled-hooks)、[Codex 规则](https://learn.chatgpt.com/docs/agent-configuration/rules)、[WorkBuddy 权限模式](https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Permission-Modes)。协议核对使用 Claude Code 2.1.193、Codex 0.165.0、WorkBuddy 桌面版 5.3.8；宿主升级仍需验收。
