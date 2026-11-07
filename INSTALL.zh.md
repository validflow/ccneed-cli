# CCNeed CLI 与 Skill 安装指南

[English](INSTALL.md) · 中文

默认全局安装 CLI，Skill 接入所选 Agent 的**当前用户目录**。不同项目复用同一套版本，不改用户项目。安装与升级共用一个入口；登录仍由用户在官网明确授权。

> 发行状态：本分支保留 0.1.6 版本号，不重新发布该版本。新增的 `kb plan`、`kb start`、`kb export` 和采集 plugin，需要发布包含本次改动的新 CLI 版本，并部署配套 Club 接口。`@latest` 安装的是已发布稳定包，不包含尚未发行的分支改动；CLI、Skill、hooks 必须来自同一版本，合并代码不会自动发布 npm。

## 正式版：一次安装，也用于升级

确认官方来源、发行状态及用户同意后，选择宿主：

```bash
# 同一命令用于首次安装和后续升级
npx @validflow/ccneed-cli@latest install --agent codex

# 使用 WorkBuddy；或指定 codex,workbuddy 一起接入
npx @validflow/ccneed-cli@latest install --agent workbuddy
```

需要 Node 22.14.0+ 和 npm。CLI 安装在 npm 全局命令目录；该目录必须在宿主实际执行环境的 PATH 中。本机安装不会自动提供给远程沙箱。上述新版本发布后，安装器还会仅向明确选择的宿主合并采集 hooks 与 ask 规则，保留其他设置；不切换权限模式、不读取账号、不创建任务。

如果访问 GitHub 需要代理，请在安装所在终端设置 `HTTPS_PROXY`（或 `HTTP_PROXY`）。二进制下载会遵循这些变量及 `NO_PROXY`，也支持小写形式；包内不预设个人代理地址或端口。

`@latest` 指向 npm 当前稳定版；若需复现特定版本，可将它替换为明确的版本号。安装器会全局安装同一精确版本的 CLI 与 Skill。

宿主入口按明确的 `--agent` 选择建立：

| 宿主 | 用户级 Skill 入口 |
| --- | --- |
| Codex | `~/.agents/skills/ccneed` |
| WorkBuddy 桌面端 | `~/.workbuddy/skills/ccneed` |
| Claude Code | `~/.claude/skills/ccneed` |

这些入口是**目录链接，不是复制件**：Windows 用 junction，其他平台用 symlink，全部指向全局 npm 包的 `skills/ccneed`。升级包后，已经接入的宿主一起读到新 Skill；即使升级时只指定一个宿主，其他已有链接也会跟随同一包更新。

不能创建链接就明确失败，不复制一份旧 Skill 冒充同步成功。不在 npm 的普通 postinstall 中隐式写入所有宿主；首次宿主接入由统一入口和用户明确选择完成。

完整包包含 Skill、按需读取的参考说明与采集 hooks。发行和安装验证 Skill 与 CLI 二进制；目录存在不代表宿主已加载。旧版 `.codebuddy/skills/ccneed` 链接保留，WorkBuddy 桌面端改用实际的 `.workbuddy` 目录。

## 采集确认接入

本节适用于上述配套新版本。先核对安装版本与命令帮助；已发布的 0.1.6 包不包含这些采集 hooks。

| 宿主 | 新增配置 | 使用前提 |
| --- | --- | --- |
| Claude Code | `~/.claude/settings.json` 的展示/执行 hooks 与 ask 规则 | 交互式 default/acceptEdits，关闭自动批准 |
| Codex | `~/.codex/hooks.json` 与 `~/.codex/rules/ccneed.rules` | `on-request`、`approvals_reviewer = "user"`，启用 hooks/rules |
| WorkBuddy 桌面端 | `~/.workbuddy/settings.json` 的 hooks、ask 规则与 `sandbox.commandSafety.ask` | 默认权限，付费采集用原生 Bash，每次仅允许本次 |

重新打开所选宿主，审阅并信任新 hooks。验收时，准备方案必须展示研究需求、各平台关键词/配额与积分余额；执行同一方案必须出现原生权限窗口。可在验收时拒绝，确认没有创建或启动采集。文件和单测通过不能代替这一步。

WorkBuddy 采集要求其原生 Bash 工具（Windows 使用 Git Bash），保留用户原有沙箱与安全规则。当前 PowerShell 沙箱包装无法可靠匹配采集前缀，付费执行会明确拒绝并提示改用同一宿主的 Bash。每次选择**仅允许本次**；选择本会话放行代表主动扩大该前缀的授权，需要新建宿主会话恢复逐次确认。

独立调用 `ccneed kb start ...`，不包进其他脚本，不用 shell 重定向隐藏计划。Codex 当前 hook 不提供命令覆盖的工作目录，因此方案/答案文件使用绝对路径；它的 PreToolUse 不支持 `ask`，原生 `.rules` 负责请求确认。已知的 bypass、never、auto-review 模式会拒绝执行。主动关闭 hook/rules 或覆盖审批设置后，不属于已支持的确认配置。

宿主协议依据 [Claude hooks](https://code.claude.com/docs/en/hooks)、[Codex hooks](https://learn.chatgpt.com/docs/hooks) 与 [rules](https://learn.chatgpt.com/docs/agent-configuration/rules)，并检查了本机 WorkBuddy 5.3.8 桌面运行时代码。[WorkBuddy 权限说明](https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Permission-Modes)区分默认权限与完全访问；逐宿主的真实权限窗口仍需按部署版本记录验收结果。

## 更新规则与冲突

- 更新提醒只提醒，不自动升级。用户同意后再次运行统一安装入口。
- 自定义同名目录/文件/其他链接不会被覆盖；原官方 Skill 被改过，统一升级在 npm 替换包之前停止。先保留定制内容，再人工解决冲突，不自动删除或覆盖。
- 官方文件由 npm 管理，定制请另存为不同技能名称。不要绕过统一入口用裸 npm 重装来保留修改；npm 自己的替换操作可能覆盖包内改动。
- 安装器拒绝旧入口静默降级。网络、校验或链接失败不返回成功；修复原因后重试同一入口。链接阶段失败仅撤回本次新增的链接，已有入口与用户文件保留；npm 包本身不承诺跨进程事务回滚。
- 更换 npm 前缀/Node 安装环境可能改变全局包路径；旧入口冲突时停止并提示，不猜测并改写其他位置。
- 更新后重启或重新加载宿主。磁盘文件同步不代表一个已打开会话中的旧 Skill 立即刷新。独立的界面上传件或手工复制件不在自动同步范围。

## 验证，然后由用户登录

```bash
ccneed version --no-update-check
ccneed --help
```

核对 CLI 版本、全局命令 PATH 与所选宿主 Skill 加载状态。安装器的 `restart_required` 表示需重新加载；`auth: not_started` 表示没有登录。不能把目录存在当成加载成功。

用户要求登录后，CLI 默认连接正式线上服务 `https://ccneed.ai`；普通用户不需要设置 `--api-url`：

```bash
ccneed auth login
# 将官网链接和公开确认码交给用户，由用户登录并同意。
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
```

不代点同意、不输出秘密、不读取浏览器 Cookie。`--api-url` 仅供**获授权的开发者**连接 staging 联调：每条命令都显式使用同一个 staging 地址和测试账号；远程 HTTP 还需 `--allow-insecure-http`。staging 凭证不能回退正式环境，详见 [贡献指南](CONTRIBUTING.zh.md)。CLI 参数本身不授予 staging 权限，服务端仍须验证账号是否获准访问。

## 从源码构建（开发者）

用户同意使用源码后，在新的干净目录：

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
```

需要 Git、Go 1.25+。将 `go env GOBIN` 返回目录加入 PATH；为空时使用 GOPATH 下的 bin。记录 checkout SHA，使用该 checkout 的完整 `skills/ccneed`。这条开发路径不是 npm 联动升级路径；源码 CLI 和独立技能要人工保持同一 revision，不能声称可自动同步。

仓库内可离线查看安装器帮助：`node scripts/npm/run.mjs install --help`。源码采用 MIT 许可；不要把源码构建当作正式 npm 发行产物，也不要把目录链接存在误认为宿主已成功加载 Skill。

## 宿主官方说明

[Codex 技能与链接](https://developers.openai.com/codex/skills) · [WorkBuddy 用户级配置](https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Setting) · [CodeBuddy 目录说明](https://www.codebuddy.cn/docs/cli/codebuddy-dir) · [Claude Code 技能](https://code.claude.com/docs/en/skills)

客户端独立导入的 Skill 不属于本机 npm 联动更新；仍需验证所用客户端的加载与原生权限确认。
