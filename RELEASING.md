# 发布 CCNeed CLI

CLI、npm 包和配套 Skill 使用同一个版本。维护者决定何时发行、发行什么版本；普通功能 PR 可以先合到 `main` 积累，**不要提前改 `package.json.version`**。不按 PR 标题、分支名、提交数量或日期自动计算版本。

只有版本变更经 PR Squash 合入 `main`，才会产生自动发行计划。Action 先在无发布权限的 job 中预检、测试并构建六个平台产物与同版本 npm/Skill 包；全部通过后，独立发布 job 才打 Git tag、发布 GitHub Release 和 npm 包，并核对 npm 安装通道。Git tag 存在不代表 npm 已发布成功。

## 当前状态与发布前检查

首个正式版 `0.1.0` 已发布，`0.1.1` 已修复首次发行流程；npm 包、MIT 许可和公开仓库均已就位。不要再执行首包手工引导流程。后续发布使用现有 [Release 工作流](.github/workflows/release.yml)；仓库变量 `CCNEED_RELEASE_ENABLED` 必须保持 `true`。npm Trusted Publisher 应绑定 `validflow/ccneed-cli`、`release.yml` 和 `npm-release` 环境，并允许 `npm publish` 与 `npm dist-tag`；下一版首次全自动发布时仍需核对 OIDC 是否生效。工作流使用 Node 24 和固定版本的 npm，在 GitHub-hosted runner 上发布，不放长期 npm token。

发布前确认：

1. 本次 PR 的必需 `quality` 检查通过；CX Review 当前暂停，不作为合并或发行门禁。维护者核对版本变更、变更记录和风险。
2. `package.json`、`package-lock.json` 根版本与 `CHANGELOG.md` 一致；Skill、安装入口和对应平台产物随同一版本发布。
3. 获授权的开发者使用 staging 测试账号完成真实联调。若 Club 正式服务尚未提供对应接口，在发行说明中明确限制；CLI 发行本身不会部署 Club，普通用户默认仍连接 `https://ccneed.ai`。
4. `npm-release` 环境仅允许 `main`，不配置每次发行都需要手工批准的 reviewer。保持仓库公开，发行二进制才能被安装器下载。

## 每次发行

从最新 `main` 新建发行分支；将下面的版本示例换成维护者选定的**新版本**，不要直接改 `main` 或手工打 tag：

```bash
# 示例版本；发行时替换为本次选定的版本
npm version 0.1.3 --no-git-tag-version --ignore-scripts
# 同步 CHANGELOG.md 后检查
npm test
bash scripts/check.sh
```

发布 PR 的 `quality` 通过且维护者确认后 Squash 合并。Release Action 会对该合并提交重新执行发行门禁，成功后才打 tag、发布并验证。上一版完成或明确失败前，不合入下一版的版本 PR。

版本唯一来源是 `package.json.version`。锁文件根版本及根包版本要同步，二进制注入同一版本/提交/包名。稳定发行使用 `vX.Y.Z` 与 npm `latest`；预发布使用 `vX.Y.Z-rc.N` 与 npm `next`，不进入普通用户更新提醒。`latest` / `next` 是 npm 的可移动安装入口，不是 Git tag；Action 通过 `npm publish --tag` 明确选择，不能只凭版本号后缀期待 npm 自动分流。不兼容变更由维护者明确标注并说明升级影响。

普通功能 PR 可以积累一批，不触发发行。需要试装才先发 `X.Y.Z-rc.1`，修复后递增 RC 序号；验收完成再发 `X.Y.Z` 正式版。RC 不是必经步骤。仅改 `package.json` 而遗漏锁文件会被发行预检拒绝；`package.json.version` 不变的普通合并不会重打已有 tag，也不会重发 npm。合并时不自动计算版本号，也不按周定时强制发布。

产物支持 Linux、macOS、Windows 的 amd64/arm64。每个平台提供独立可执行文件，npm 安装器按自身精确版本下载，校验 SHA-256 后才安装；不要求用户安装 Go，也不解压不可信归档。npm 包含同一源码版本的完整 Skill 和统一安装器模块；校验和另包含 Skill 完整文件摘要。统一入口全局安装精确版本 CLI，将所选宿主的用户级目录链接到同包 Skill，并校验程序实际版本和 Skill 摘要。已有宿主入口随全局包一起更新，不维护独立复制件；真实宿主加载仍需验收。

## 失败与重试

- 缺失版本、许可、Skill、锁文件或质量门禁失败：在打 tag 前停止，发布身份不会进入运行测试/构建的 job。
- tag 已存在：只有指向同一 main SHA 才能复用，绝不移动 tag。
- GitHub 草稿只补缺失文件；已有文件逐个核对大小与 SHA-256 摘要，禁止覆盖不同字节。公开 Release 缺产物时停止，不悄悄补发改变既有版本。
- npm 已存在该版本：只有精确 tarball integrity 相同才能复用；不覆盖版本，不把稳定通道倒退。
- `npm publish` 成功后，registry 的版本和 `latest`/`next` 查询可能稍后才可见；发布 job 以 10 秒间隔最多核对 19 次两者及 tarball integrity，不因一次临时 404 误报失败。内容不同立即停止，超过时限仍不可见则明确失败，不重复发布或移动 tag。
- Actions 保存构建产物 30 天。同一次运行若只重跑失败的 publish job，构建产物名中的 `run_attempt` 会变化而找不到原产物；需要重跑整个工作流。重跑旧运行仍使用旧提交，不会带入后来合并的脚本修复。
- 若发布失败后已修复发行脚本、但 `package.json.version` 未改变，在 Release 工作流页面选择 main，手动运行并填写与当前 `package.json` 完全一致的 `retry_version`。它会从当前 main 重新预检、构建、打 tag 和发布；普通代码合并仍不会自动重发同版本。已存在的 tag 只能指向当前 main 提交；旧草稿也必须指向相同提交，否则先核对并清理失败草稿和未发布 tag，不能移动已成功发布的版本。
- 发行脚本使用新建草稿响应中的 Release ID 上传资产并公开草稿，不假定草稿可被立即按 tag 或列表再次查到。重试复用草稿时仍核对 tag、目标提交与每项资产的 SHA-256 摘要；任何不一致都停止，不覆盖旧产物。
- GitHub 已公开、npm 未完成：npm 安装渠道不会因 GitHub tag 单独提醒更新。修复权限/网络后恢复该发行，不向用户宣告成功。

发布成功以 GitHub 产物、npm 版本内容及目标通道均校验通过为准。更新检查按 npm 稳定 latest，缓存 24 小时、请求最多 2 秒，失败不阻塞命令且提示不写 stdout。提醒不会静默升级；用户同意后运行统一安装入口，CLI 与已接入宿主的 Skill 同包换版。旧 Skill 修改会阻止统一升级，客户端需重载，手工复制或界面上传不在联动范围；完整规则见 [安装指南](INSTALL.md)。此机制的源码测试通过不代表 npm 已发行。

## 正式版群通知

仅当 `publish` job 已验证 Git tag、GitHub Release、npm 包内容及 `latest` 后，独立的 `notify` job 才通过仓库 Secret `MULTICA_CLI_RELEASE_WEBHOOK_URL` 触发 Multica Autopilot。每个稳定版使用固定幂等键 `ccneed-cli:vX.Y.Z`，重试不会重复启动通知；RC 和未改版本的普通合并不通知。Webhook URL 是凭证，只存 GitHub Secret，不写入仓库或日志。

Autopilot 独立复核公开发行状态，再向「上线即开奖」发送版本、tag、npm `latest` 与 Release 链接，并原生 @所有人。Webhook 返回 `accepted` 只表示 Multica 接收，**不代表飞书消息送达**；需要在 Autopilot 运行记录核对实际消息回执。若 `notify` 失败而 `publish` 成功，发行已经完成，不要重打 tag 或重发 npm，只修复通知并重跑通知 job；群通知失败不能回滚已发行版本。
