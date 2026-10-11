# 贡献指南

## 开发流程

只维护一个 `main`，不增加 `develop`。初始基础提交经仓库所有者授权直接落在 main；之后使用短生命周期分支，提交 PR，检查与 review 通过后 **Squash and merge**。

```bash
git switch main
git pull --ff-only
git switch -c feat/short-description
make check
```

每次 Squash 合并后，从最新 main 开新分支；不要继续复用已经合并的功能分支。

## 分支、PR 与提交命名

命名采用文档约定，不设置 GitHub 分支名或 PR 标题限制，也不增加命名 CI 门禁。没有按推荐格式命名，仍可创建分支和提交 PR；维护者在合并前整理最终 PR 标题。命名宽松不改变测试、review 和 Squash 合并的要求。

| 对象 | 推荐格式 | 示例 |
| --- | --- | --- |
| 分支 | `type/short-description` | `feat/browser-login`、`fix/report-download` |
| PR 标题 | `type(scope): description` | `feat(auth): add browser login` |
| 分支内提交 | 推荐与 PR 标题相同的格式，允许临时迭代提交 | `fix(files): prevent download overwrite` |

分支建议使用简短、可读的描述；也接受 Agent 自动生成的分支名，不要求改名。标题中的 `type` 推荐使用 `feat`、`fix`、`docs`、`ci`、`test`、`refactor`、`perf`、`build` 或 `chore`；`scope` 可省略。说明改了什么，并在 PR 描述中说明为什么及如何验证。不兼容的变更用 `!` 标记，例如 `feat(auth)!: replace credential format`，并在描述中说明迁移方式。

内部需求的分支可以带 issue 标识，例如 `feat/CCN-1142-cli-foundation`。内部 PR 的最终标题保留 Conventional Commit 前缀，只附带对应的一个 Multica identifier，例如 `feat(cli): add account commands (CCN-1142)`；外部贡献者不需要 Multica identifier，有相关 GitHub issue 时在 PR 描述中关联即可。需求标识不放在产品代码或注释里。

Squash 提交默认使用 PR 标题。维护者在合并前将标题整理为 `type(scope): description`，确认它准确反映最终改动，并核对合并对话框中的提交标题。无需逐一整理分支里的临时提交，也不从分支名推断改动类型或版本号。版本由维护者决定；发行启用后，仅版本变更合入 main 才自动打 tag 和发布，普通功能合并不发布。

CLI 与宿主 Skill 使用同一套命名约定：新增 Skill 能力用 `feat(skill)`，修复 Agent 调用行为用 `fix(skill)`；仅排版或错字修改用 `docs(skill)`。不要因为 Skill 是 Markdown 文件，就将所有行为变更归类为文档修改。

## 模块按职责增加

`cmd/ccneed` 只放进程入口；命令解析放 `internal/cli`。API 客户端、凭证存储、输出与下载按真实需求逐步增加独立模块，不提前创建空目录或通用框架。

开发工具位于 `tools/`；不能成为 CLI 的运行时依赖。宿主 Skill 放在 `skills/ccneed/SKILL.md`，只维护一份说明。统一 npm 安装入口将所选宿主的用户级目录链接到同一全局包，更新不产生技能复制件；发行校验和包含完整 Skill 摘要。安装器只预检/操作明确选择的宿主，不覆盖已有自定义技能，旧托管技能被修改时先停止升级。真实宿主加载仍需验收，不能把文件链接存在当成加载成功。

## 先本地测试，再 staging 联调

```mermaid
flowchart TD
    A[Club 与 CLI 各自功能分支] --> B[本地确定性测试]
    B --> C[Club 分支部署到 staging]
    C --> D[CLI 连接该部署联调]
    D --> E[记录两仓 SHA 与验证结果]
    E --> F[后端按现有发布流程上线 sg-prod]
    F --> G[受控生产冒烟]
    G --> H[发布 CLI 二进制 / npm / 宿主 Skill]
```

- `staging` 和 `sg-prod` 是两套环境。测试数据库、存储、队列、任务执行和额度账本不得使用生产资源。
- 开发者通过 `--api-url` 显式选择服务 origin；默认只接受 HTTPS 或本机 HTTP，不接受路径、查询、重定向或地址内凭证。若分支预览暂时只有 HTTP，必须在**每条**联调命令上同时传 `--allow-insecure-http`，且远程地址必须是预览的 IP；它会让登录证明和凭证经明文网络传输，只能使用 staging 测试账号，不能对生产地址使用。正式发行默认 sg-prod 的 HTTPS，普通用户无须选择环境。该参数不是任意地址转发器；登录与服务端隔离仍需 staging 验证。
- API、网页登录、回调和会话/报告链接必须跟随同一环境；本地凭证按可信服务地址隔离，服务端也要拒绝跨环境凭证。
- 每轮集成记录 Club **实际部署**的 SHA、CLI SHA 和地址。只改 CLI 时复用同一健康 staging 部署；后端更新或预览过期再部署。现有 staging 预览默认有效期为 4 小时。
- 本地 API 替身用于契约和错误路径测试，不能替代真实 staging 登录、任务、HITL、文件及报告下载闭环。
- 不改变 Club 原有 develop → release/hotfix → main → Tag 部署流程。CLI 普通功能合并 main 不发布 npm；发行启用后，新版本准备 PR 合并才自动发布 CLI，不触发生产服务部署。

### 一次真实联调怎么做

1. 本地 Go 门禁先通过；修改 npm、发行或 CI 时另跑 `npm test`。
2. 将待测试的 Club 功能分支部署到 staging，核对实际部署 SHA 和测试资源归属。部署只是运行实例，不意味着数据库已隔离；必须确认没有连接 sg-prod。
3. 本地构建 CLI，不必先发布 npm。通过 `--api-url` 传入该部署的服务 origin，不传具体接口路径；仅当预览是 HTTP 时额外传 `--allow-insecure-http`。已有业务接口直接复用，不新增测试专用接口。
4. 首个闭环是 staging 浏览器授权后查询账号积分，再逐步测试文件、知识库、任务、HITL 和报告。授权、回调及业务网页必须跟随同一环境；认证层返回其他地址的连接时，CLI 在发请求前拒绝。
5. 保留 Club 实际部署 SHA、CLI 构建 SHA、目标 URL、测试 fixture 标识、结果及清理说明，不记录 token 或原始认证响应。只改 CLI 且部署健康兼容时复用；后端更新、过期或不兼容时才重新部署。

以上是联调流程，不是已上线声明。源码已实现官网设备授权与撤销、账号、资料、异步任务、通用 HITL、历史恢复和报告读取/下载，并有本地 HTTP 与存储测试；配套 Club 接口尚未完成部署及隔离验收。三平台真实凭证库和宿主安装也需要验收。staging 通过后仍须后端先上线、受控生产冒烟通过、最后发行 CLI。

## 质量门禁

`make check` 与 `bash scripts/check.sh` 是同一入口，检查时不自动改源码：

1. gofmt、依赖文件一致性。
2. Google 风格注释结构及禁止 issue 编号的代码注释。
3. go vet、固定版本 golangci-lint。
4. 单元/集成测试与构建；Linux CI 额外执行 race 检测。

GitHub Actions 在 Linux、macOS、Windows 执行测试与构建，Linux 额外执行完整 Go 门禁、race 检测、审查/发行/安装脚本测试与固定版本 actionlint。Go 本地门禁不增加 Node 运行依赖；修改 npm、发行或 CI 时另跑 `npm test`（需要 Node 22.14.0+）。

main 已启用分支保护：禁止删除和强推，必须通过 PR 合并，管理员也受约束；仓库只允许 Squash and merge。目前未设置必需 CI 状态。等真实 runner 与检查完成验证后，再要求 `quality` 和 `AI Code Review` 检查通过。CX Review 与 usight_agent 一样只自动运行同仓库 PR；fork PR 会跳过，不能把跳过当作完成审查。**仅提交 workflow 不等于已启用 CI 必需检查。**

第一阶段不设置任意覆盖率百分比。新增行为必须有能抓住回归的测试，覆盖率用于 review，不写凑数测试。先确认测试失败，再实现并确认通过。

## PR 与发布

使用 PR 模板说明范围、测试证据和风险。CX Review 与 usight_agent 一样在 `cc-review` runner 直接调用 Codex CLI 和仓库内 `review-pipeline` Skill，分别审查正确性、模块边界、安全、需求一致性，并二次核实 Critical / Important 候选。工作流把报告写为 PR 评论；维护者须在合并前处理确认的问题。审查失败时工作流发布失败通知，不得把未完成审查称为通过。

普通测试只在无生产凭证的 GitHub-hosted runner 执行。模型审查签出 PR merge ref 供只读检查，不运行 PR 构建或测试；PR 内容中的指令不具权威性。仅同仓库 PR 自动进入持有 Codex 登录状态的 runner；fork PR 不运行该工作流，未来开放外部贡献前须另行确定审查流程。

首次接入需先人工完成同等四类审查，将工作流合入 main，再授予本仓使用组织级 `cc-review` runner 的权限。runner 已登录 Codex 即可，不强制新增 token；实测通过后再设置必需检查。操作步骤见 [CX Review 运维说明](.agents/skills/review-pipeline/references/operations.md)。未配置或未实测前，不能宣称自动 review 已启用。

发布前确认 sg-prod 服务兼容，再验证生产登录与受控任务闭环。发行工作流已定义，但 `CCNEED_RELEASE_ENABLED` 默认未开启；首次 npm 发布、GitHub Release 和许可证选择仍需单独授权。生产凭证不进入仓库、CI 日志或外部贡献者 PR。操作和失败恢复见 [发行说明](RELEASING.md)。

CLI 发行使用独立的 `vX.Y.Z` tag；npm、二进制的 `ccneed version` 和随包 Skill 属于同一版发行，不沿用 Club 的服务版本。预发行使用 `vX.Y.Z-rc.N`，不能进入 npm `latest` 或普通用户更新提醒。推送开发提交不等于发行，也不自动打 tag。

发行门禁校验 tag 与程序/npm 版本一致、各平台产物及校验值完整、npm 通道与对应包内容一致后才能宣布成功。发行构建注入官方包名，更新提醒查询 npm `latest`；未注入包名的源码构建使用公开 GitHub 来源。两类缓存隔离，不能因 GitHub Release 存在而让 npm 安装的客户端提醒升级。当前 0.1.0-rc.1 只记录待验收源码，保持 private/UNLICENSED 禁发，实际首发尚未完成。
