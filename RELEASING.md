# CLI 发行

维护者决定版本号；版本变更经 PR 合入 main 后，CI 自动执行后续步骤。普通功能合并不发版，不按 feat/fix、分支名或提交数量计算版本。

维护者在发布 PR 中确定版本并更新 `package.json`、锁文件和变更记录。审查通过、Squash 合入 main 后，Action 先在**无发布权限**的 job 中预检、测试、构建六个平台产物及同版本 npm/Skill 包；全部通过后，独立发布 job 才获得短时发布身份，下载这次构建的产物，打不可移动的 Git tag，发布 GitHub Release 和 npm 包，最后验证 npm 通道。中途失败不会伪装成发布成功。

Git tag 不等于 npm 已发布。用户通过 npm registry 安装，不直接读取 GitHub main。发行通过一个工作流连续执行；不依赖 GitHub token 创建的 tag 再触发第二个工作流。

## 首次启用前

当前 `package.json` 与锁文件使用 `0.1.0-rc.1`、`private: true`、`UNLICENSED`。这是待验收源码里程碑，尚不能用于用户安装。合并后经维护者确认，可手动将 `v0.1.0-rc.1` 绑定该 main 提交；不创建公开 Release 或 npm 包，也不移动已有 tag。

私有包更新版本只记录源码，不生成发行计划。只有明确解除 `private` 禁发、通过包身份/许可/材料预检并开启发行开关后才进入发行；首次启用可以使用同一版本，但须校验既有 tag 绑定的提交，若源码已经变化则使用新版本。发行后的 RC 可通过 npm `next` 明确试装，安装器仍会全局安装同一精确版本的 CLI 与 Skill；私有源码里程碑不可试装。

1. 完成真实 staging 联调，确认兼容服务已上线 sg-prod；CLI 发行不部署 Club。
2. 经所有者确认后将 GitHub 仓库改为 public，让公开 npm 包的安装器能够下载 GitHub Release 二进制；发行预检会在仓库仍为 private 时拒绝打 tag。确认 npm 包名、首发版本与开源许可；添加实际 `LICENSE`、`CHANGELOG.md`、可用的 `skills/ccneed/SKILL.md`，不是占位说明。公开前检查提交历史是否含私密数据。
3. 确认 main 的 PR、CX Review 和质量保护已启用。仅存在 workflow 文件不代表这些设置已生效。
4. 先解决**首包引导**：npm 的 trusted publisher 设置在已有包的设置页；`npm trust` 也要求包已存在。因此 `@validflow/ccneed-cli` 尚未建立时，不能只打开 Action 就期待 OIDC 完成首发。维护者须在正式发布前单独批准首包的创建/人工发布方案，核对精确版本与产物；不得临时把长期 npm token 塞进 CI。首包建好后，再在 npm 包设置中配置 trusted publisher：组织 `validflow`、仓库 `ccneed-cli`、工作流 `release.yml`、环境 `npm-release`，分别允许 `npm publish` 和 `npm dist-tag`。
5. GitHub 的 `npm-release` 环境仅允许 main；若希望全自动，不设置每次发布都需人工批准的 environment reviewer。生产资源凭证不进入此环境。
6. 确认首包、可信发布者和 Club 生产兼容都已完成后，维护者才将仓库变量 `CCNEED_RELEASE_ENABLED` 设为 `true`。本次开发不修改该变量，也不选择许可证或真实发布。新建 trusted publisher 配置必须在 2 天内完成首次成功发布；不要提前配置后搁置。详见 [npm 官方 Trusted Publishing 文档](https://docs.npmjs.com/trusted-publishers/)。

npm OIDC 要求受支持的 Node/npm 与 GitHub-hosted runner；`dist-tag` 的 OIDC 恢复至少需要 npm 11.21.0（12.x 至少 12.2.0）。工作流使用 Node 24 并固定 npm 11.21.0，不依赖 Node 自带的 npm 版本。发行要求 GitHub 仓库已公开；满足条件时 trusted publishing 自动生成 provenance，无须在命令里强制 `--provenance`。npm 页面上的 `npm publish` 和 `npm dist-tag` 是独立授权，缺一个都可能导致恢复失败。

## 每次发行

在新功能分支中执行，不直接修改 main：

```bash
# 仅更新 manifest/锁文件，不在本地打 tag
npm version 0.1.1 --no-git-tag-version --ignore-scripts
# 同步 CHANGELOG.md 后检查
npm test
bash scripts/check.sh
```

发布 PR 经审查后 Squash 合并。CI 对该合并提交先执行预检、测试与构建，成功后才打 tag、发布并验证。保持一次发行完成后再合下一次版本 PR；GitHub concurrency 只串行化发布 job，不将它当作无限队列。

版本唯一来源是 `package.json.version`。锁文件根版本及根包版本要同步，二进制注入同一版本/提交/包名。稳定发行使用 `vX.Y.Z` 与 npm `latest`；预发布使用 `vX.Y.Z-rc.N` 与 npm `next`，不进入普通用户更新提醒。`latest` / `next` 是 npm 的可移动安装入口，不是 Git tag；Action 通过 `npm publish --tag` 明确选择，不能只凭版本号后缀期待 npm 自动分流。不兼容变更的版本选择由维护者审查，0.x 的具体约定待首发前确认。

普通功能 PR 可以积累一周，不触发发行。准备发行时由维护者单独提交版本 PR：需要试装才先发 `X.Y.Z-rc.1`，修复后递增 RC 序号；验收完成再发 `X.Y.Z` 正式版。RC 不是必经步骤。`npm version X.Y.Z --no-git-tag-version --ignore-scripts` 同步修改 manifest 与锁文件，合并后 Action 才自动打 Git tag、构建及发布 npm；仅改 `package.json` 而遗漏锁文件会被发行预检拒绝。合并时不自动计算版本号，也不按周定时强制发布。

产物支持 Linux、macOS、Windows 的 amd64/arm64。每个平台提供独立可执行文件，npm 安装器按自身精确版本下载，校验 SHA-256 后才安装；不要求用户安装 Go，也不解压不可信归档。npm 包含同一源码版本的完整 Skill 和统一安装器模块；校验和另包含 Skill 完整文件摘要。统一入口全局安装精确版本 CLI，将所选宿主的用户级目录链接到同包 Skill，并校验程序实际版本和 Skill 摘要。已有宿主入口随全局包一起更新，不维护独立复制件；真实宿主加载仍需验收。

## 失败与重试

- 缺失版本、许可、Skill、锁文件或质量门禁失败：在打 tag 前停止，发布身份不会进入运行测试/构建的 job。
- tag 已存在：只有指向同一 main SHA 才能复用，绝不移动 tag。
- GitHub 草稿只补缺失文件；已有文件逐个下载比对，禁止覆盖不同字节。公开 Release 缺产物时停止，不悄悄补发改变既有版本。
- npm 已存在该版本：只有精确 tarball integrity 相同才能复用；不覆盖版本，不把稳定通道倒退。
- Actions 保存构建产物 30 天。临时网络/权限错误可重跑同一次 Release workflow；每次重跑都会重新通过门禁并使用该次构建产物。源码、构建工具或内容改变导致字节不同，应使用保留的原始产物或新版本，不强行复用旧号。
- GitHub 已公开、npm 未完成：npm 安装渠道不会因 GitHub tag 单独提醒更新。修复权限/网络后恢复该发行，不向用户宣告成功。

发布成功以 GitHub 产物、npm 版本内容及目标通道均校验通过为准。更新检查按 npm 稳定 latest，缓存 24 小时、请求最多 2 秒，失败不阻塞命令且提示不写 stdout。提醒不会静默升级；用户同意后运行统一安装入口，CLI 与已接入宿主的 Skill 同包换版。旧 Skill 修改会阻止统一升级，客户端需重载，手工复制或界面上传不在联动范围；完整规则见 [安装指南](INSTALL.md)。此机制的源码测试通过不代表 npm 已发行。
