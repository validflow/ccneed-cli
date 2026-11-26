# 安装 CCNeed

[English](../en/INSTALL.md) · 中文

全局 CLI 执行命令；原生插件或扩展加载同一份 Skill 和采集校验，事件和权限界面按各宿主机制接入。

## 安装与升级

需要 Node.js 22.14+、npm 及所选宿主。审阅官方来源并获得安装授权后：

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

安装器安装精确版本的全局 CLI，协调所选宿主的原生插件注册。升级时列出所有需要更新的宿主；不会自动登录或创建云端任务。npm 全局命令目录需在 Agent 的 PATH 中；本机安装不会把命令装进远程沙箱。

| 宿主 | 安装标识 | 原生注册 | 额外权限配置 |
| --- | --- | --- | --- |
| Claude Code | `claude` | 调用 Claude marketplace/plugin 命令，核实用户级版本及启用状态 | 宿主现有权限 |
| Codex | `codex` | 调用 Codex marketplace/plugin 命令，核实已装版本及启用状态 | 宿主现有权限 |
| WorkBuddy 桌面版 | `workbuddy` | 按结果完成桌面端原生 marketplace/plugin 操作 | 宿主现有权限 |
| Kimi Code | `kimi` | 原生 `/plugins` 导入步骤 | 宿主现有权限 |
| OpenClaw | `openclaw` | 原生链接插件 | 宿主现有权限 |
| Pi | `pi` | 原生 Pi package 和 extension | 宿主现有权限 |
| ZCode | `zcode` | 桌面插件管理器导入本地 marketplace | 宿主现有权限 |
| Qoder | `qoder` | 原生本地插件 | 宿主现有权限 |
| Trae CLI 2.0 | `trae` | 独立原生插件 | 宿主现有权限 |
| DeepSeek Harness | `dsh` | 原生 Cordis bundle | 宿主现有权限 |

Claude Code、Codex、WorkBuddy 使用本地 marketplace `ccneed` 和插件 `ccneed@ccneed`；Kimi 使用 manifest ID `ccneed`。宿主从插件加载统一 Skill 和 hooks，不重复注册全局 Skill 或用户级 hooks。升级需重跑安装器并完成各宿主步骤，单独 `npm install -g` 不会刷新原生缓存。

检查 JSON 的 `plugins[]`。`installed` 表示原生管理器报告目标版本已启用，仍需信任 hooks 并重载会话。`action_required`、`failed` 表示接入未完成。WorkBuddy 桌面端目前没有已验证的公开 CLI 注册命令，因此会明确等待桌面操作，不能用 CodeBuddy CLI 注册冒充桌面端成功。

新增宿主返回 `action_required` 和原生管理命令／应用内导入步骤。安装器准备同版本完整插件，不修改这些宿主的 profile，也不冒报已启用。在目标宿主和 profile 中完成操作，核对版本并重载。多个宿主可用逗号选择，例如 `--agent codex,pi`。

重启宿主，审阅并信任 hooks，确认 Skill 能加载。安装时信任插件与每次采集前授权是两回事。启停、卸载使用宿主原生插件管理。卸载不会卸载全局 CLI，也不会自动清理额外的 Codex/WB 权限规则；可保留这些保守规则，确认移除接入后仅清理对应 CCNeed 项，不删除其他规则。

下载需要代理时设置 `HTTPS_PROXY` 或 `HTTP_PROXY`；下载器也识别 `NO_PROXY` 及小写形式。固定版本时替换 `@latest`；RC 使用 `@next`。

## Kimi Code

运行 `npx @validflow/ccneed-cli@latest install --agent kimi`，然后在 **Kimi Code CLI 2.5.3** 或兼容的更新原生版本中，按返回结果执行 `/plugins install "包路径"`。启用 `ccneed`、核对目标版本，再 `/reload` 或新开会话。原生管理器会复制插件，每次更新后都需重新导入。安装器不修改 Kimi profile、`KIMI_CODE_HOME` 或用户配置；这一步需要原生管理器操作，安装器返回 `action_required`。

包内 manifest 遵循[官方插件文档](https://www.kimi.com/code/docs/en/kimi-code-cli/customization/plugins)。已用原生 2.5.3 的隔离 profile 验证安装、启停和更新到 0.1.12：识别 1 个 Skill、3 个 hooks，无加载错误。已停维的 Python `kimi-cli` 1.x 是不同产品，不能加载这套插件；请使用[官方原生安装器](https://www.kimi.com/code/docs/en/kimi-code-cli/guides/getting-started)。

正式采集和 HITL 执行遵循 Kimi 原生权限：手动／按需模式按配置询问，自动模式和会话授权按用户意愿执行，CCNeed 不额外强制弹窗或禁止这些模式。Agent 在提交前检查方案。

Kimi 通过 `sessionStart.skill` 加载说明，SessionStart/PostToolUse 不注入返回的 Hook 文本。Agent 依据 CLI 完整结果和保存的 JSON 展示方案。`kb plan` 和处理 HITL 的 `tasks get/watch` 使用新的绝对 `--output` 文件，Hook 在结果被截成 2000 字符时从同一文件读取全文。不会覆盖已有文件；本地 Hook 不读取远程 Bash 的输出文件。

## 新增宿主的兼容要求

- **OpenClaw 和 Pi：**通过原生插件管理器加载 CCNeed，权限遵循当前设置。
- **ZCode／Qoder：**导入完整原生插件，加载各自显式声明的 hook 文件；使用原生 argv 执行，保留空格和中文路径。按结果完成 marketplace／本地插件操作；协议测试不能替代桌面界面验收。
- **Trae CLI 2.0：**已用原生 0.215.0 验证。执行 `traecli plugin install "包路径/plugins/trae" --name ccneed`，再用 `traecli plugin validate ccneed` 核对 **1 个 Skill、2 个 hooks**。全局 CLI 升级后执行 `traecli plugin upgrade ccneed` 并重载会话；原生管理器切换完整活动缓存，旧缓存由它管理。Trae CLI 使用受支持的 `.codex-plugin` 格式，因此独立插件根目录内的 Skill/hooks 在发行时从同一源码装配，不与 Codex 入口冲突；这不代表 Trae IDE 已适配。
- **DeepSeek Harness：**使用 0.2.2-rc.3 或兼容的 0.2.x，宿主自身需要 **Node.js 24.9+**。当前 npm `latest` 可能指向不兼容的旧版本。选择正在使用的 profile：`dsh plugin --profile <name> add "包路径"`，用 `dsh --profile <name> --dump-config` 和 Plugins 页核对。原生 bundle 注册同一份 Skill和结果上下文事件；移除使用 `dsh plugin --profile <name> remove @validflow/ccneed-cli`。

OpenClaw、Pi、DSH 使用原生扩展事件，不套用 shell hook 名称。文件路径需能被运行插件和 CLI 的进程读取；本地插件不能校验另一套远程文件系统。截断的结果不能视为有效的已展示方案。

协议参考：[OpenClaw 工具策略](https://docs.openclaw.ai/plugins/hooks/tool-policy)、[Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md)、[Pi extensions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md)、[ZCode 插件](https://zcode.z.ai/cn/docs/plugin)、[Qoder 插件](https://docs.qoder.com/cli/plugins)、[Trae CLI 2.0](https://docs.trae.cn/cli_get-started-with-trae-cli-2)、[DSH bundle](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/user/develop/basic/publish.md)。

## 采集确认

Agent 展示完整方案并等待用户确认，修改后重新校验和展示；确认后执行 `kb start`，遵循宿主现有权限和沙箱。插件不追加执行审批，不能保证宿主每次弹窗。CLI 校验展示内容与 token 一致，服务端负责签名及有效期校验（线上可用性需另行验证）。云端 HITL 按 `resume_schema` 提交用户回答。

升级时仅迁移内容完全匹配旧安装器模板的 Codex `ccneed.rules`，自定义规则保留；其他宿主或已修改的规则需由用户自行检查。重启并信任新插件 hooks。

## 已有配置与失败处理

保留其他设置与自定义 Skill，仅迁移可识别的旧安装器 Skill 链接/hooks。未知或修改文件需人工处理。npm 托管 Skill 被修改时，替换前会停止；先将定制内容保留到其他 Skill 名称下。

CLI 成功但原生注册失败会逐宿主报告，不算整体成功。解决原因后重试；npm 与外部原生管理器不是一个事务。同名异源 marketplace 不会静默改指向。普通 npm `postinstall` 不修改宿主配置。

安装错误会标明失败阶段及安全的错误码。原生插件失败时还会返回 `plugins[].diagnostic`；这时 CLI 可能已更新，使用前需逐宿主检查状态。

| 错误 | 处理方式 |
| --- | --- |
| `EPERM` / `EACCES` | 查看本地 npm 日志中的目标路径。当前终端或 Agent 会话需有权写入对应缓存／安装目录，在具备权限的会话中重试。 |
| `EBUSY` | 文件被占用。关闭占用该文件的进程后重试；若占用路径在 npm 缓存中，可改用新的可写缓存目录。 |
| `ENOENT` | 检查 Node.js、npm 和所选宿主已安装，并能从 PATH 找到。 |
| `MARKETPLACE_UNAVAILABLE` | 在本机运行宿主的 marketplace list 命令，修复它指出的来源；出错的可能是其他插件源。 |
| 其他失败 | 检查本地 npm 日志或对应宿主操作；网络错误还需核对 registry 和代理。为避免泄露凭证，不直接输出子进程原始日志。 |

**CCNeed 安装器启动前**发生的 npm/npx 错误，无法由安装器捕获。缓存出错时，可在原安装命令中使用 `npx --cache <新的可写缓存目录> ...` 重试。换缓存不会解决 npm 全局目录无写权限的问题；安装器不会自动提权、删除缓存或结束进程。

非默认 `CODEX_HOME` 或 `CLAUDE_CONFIG_DIR` 会返回需手动配置 profile，不会把插件与确认规则写到不同目录。统一安装使用默认 profile；自定义 profile 需自行配置并验证原生插件。

## 验证与登录

```bash
ccneed version --no-update-check
ccneed --help
ccneed auth login
# 将官网链接和确认码交给用户，由用户在网页上同意。
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
```

`auth: not_started` 表示还没登录。不替用户批准网站授权，不读浏览器 Cookie 或输出秘密。默认正式服务 `https://ccneed.ai`；仅获授权的开发者使用 `--api-url` 和隔离的 staging 账号，见[贡献说明](CONTRIBUTING.md)。

## 源码构建

需要 Git 与 Go 1.25+：

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
```

将 `go env GOBIN` 或 GOPATH 的 `bin` 加入 PATH。这只构建开发命令，不注册原生插件；记录 revision，并让插件来源保持一致。离线安装器帮助：`node scripts/npm/run.mjs install --help`。

参考：[Claude 插件](https://code.claude.com/docs/en/plugins-reference)、[Codex 插件 hooks](https://learn.chatgpt.com/docs/hooks#plugin-bundled-hooks)、[Codex 规则](https://learn.chatgpt.com/docs/agent-configuration/rules)、[WorkBuddy 权限模式](https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Permission-Modes)。协议核对使用 Claude Code 2.1.193、Codex 0.165.0、WorkBuddy 桌面版 5.3.8；宿主升级仍需验收。
