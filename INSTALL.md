# CCNeed CLI 与 Skill 安装指南

默认全局安装 CLI，Skill 接入所选 Agent 的**当前用户目录**。不同项目复用同一套版本，不改用户项目。安装与升级共用一个入口；登录仍由用户在官网明确授权。

> 首个候选版正在发布。先在 npm 确认 `@validflow/ccneed-cli@next` 已存在，再试装；以下 `@latest` 命令只在正式版发布后使用。配套云端接口尚未完成正式部署验收，部分命令可能暂不可用。

## 正式版：一次安装，也用于升级

确认官方来源、发行状态及用户同意后，选择宿主：

```bash
# 正式版发布后执行；同一命令用于后续升级
npx @validflow/ccneed-cli@latest install --agent codex

# 使用 WorkBuddy；或指定 codex,workbuddy 一起接入
npx @validflow/ccneed-cli@latest install --agent workbuddy
```

需要 Node 22.14.0+ 和 npm。CLI 安装在 npm 全局命令目录；该目录必须在宿主实际执行环境的 PATH 中。本机安装不会自动提供给远程沙箱，不默认申请管理员权限。安装器不读取账号、创建任务或修改宿主安全配置。

候选版在 npm 发布成功后，显式选择 `next` 入口，例如 `npx @validflow/ccneed-cli@next install --agent codex`；也可以把 `@next` 改为精确版本 `@0.1.0-rc.1`。安装器会全局安装同一精确版本的 CLI 与 Skill，不会回退到 `latest`。没有正式版前，`@latest` 不作为可用入口。

宿主入口按明确的 `--agent` 选择建立：

| 宿主 | 用户级 Skill 入口 |
| --- | --- |
| Codex | `~/.agents/skills/ccneed` |
| WorkBuddy 本地代码环境 | `~/.codebuddy/skills/ccneed` |
| Claude Code | `~/.claude/skills/ccneed` |

这些入口是**目录链接，不是复制件**：Windows 用 junction，其他平台用 symlink，全部指向全局 npm 包的 `skills/ccneed`。升级包后，已经接入的宿主一起读到新 Skill；即使升级时只指定一个宿主，其他已有链接也会跟随同一包更新。

不能创建链接就明确失败，不复制一份旧 Skill 冒充同步成功。不在 npm 的普通 postinstall 中隐式写入所有宿主；首次宿主接入由统一入口和用户明确选择完成。

完整 Skill 包含 `SKILL.md` 与 `references/auth.md`、`account.md`、`tasks.md`、`resources.md`。发行、安装前后及 npm 启动入口验证完整摘要；CLI 二进制另验证摘要与实际版本。仅有 tag、程序或入口文件不算发行完整。

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

配套服务已部署、用户要求登录后，默认连接 `https://ccneed.ai`：

```bash
ccneed auth login
# 将官网链接和公开确认码交给用户，由用户登录并同意。
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
```

不代点同意、不输出秘密、不读取浏览器 Cookie。开发联调在每条命令上显式使用同一个 `--api-url`；远程 HTTP 还需 `--allow-insecure-http` 和 staging 测试账号。staging 凭证不能回退正式环境，详见 [贡献指南](CONTRIBUTING.md)。

## 当前开发安装

用户同意使用源码后，在新的干净目录：

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
```

需要 Git、Go 1.25+。将 `go env GOBIN` 返回目录加入 PATH；为空时使用 GOPATH 下的 bin。记录 checkout SHA，使用该 checkout 的完整 `skills/ccneed`。这条开发路径不是 npm 联动升级路径；源码 CLI 和独立技能要人工保持同一 revision，不能声称可自动同步。

仓库内可离线查看安装器帮助：`node scripts/npm/run.mjs install --help`。源码采用 MIT 许可；公开 npm RC 与真实宿主加载仍需发行后验收，不要把源码构建当作正式发行产物。

## 宿主官方说明

[Codex 技能与链接](https://developers.openai.com/codex/skills) · [WorkBuddy 用户级配置](https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Setting) · [CodeBuddy 目录说明](https://www.codebuddy.cn/docs/cli/codebuddy-dir) · [Claude Code 技能](https://code.claude.com/docs/en/skills)

WorkBuddy 的非本地代码场景可走客户端技能导入，但它是独立安装方式，不声称能跟随本机 npm 更新。当前实际 WorkBuddy 客户端加载仍待验收。
