<div align="center">
  <img src="assets/ccneed-logo.png" alt="CCNeed：拿着放大镜的白色小河马" width="320">
  <h1>CCNeed CLI</h1>
  <p>在 Codex、WorkBuddy 或终端里使用 CCNeed。</p>
  <p>
    <a href="https://ccneed.ai">官网</a> ·
    <a href="INSTALL.md">安装指南</a> ·
    <a href="skills/ccneed/SKILL.md">Skill</a> ·
    <a href="https://github.com/validflow/ccneed-cli/issues">反馈</a>
  </p>
</div>

把研究任务交给 CCNeed 云端执行，在你常用的 Agent 里查看进度、回答确认问题、拿回报告。

> 🚧 当前为 `0.1.0-rc.1` 待验收源码，npm 尚未发布，配套服务尚未完成部署验收。下面的安装与使用流程面向正式发布后；目前不能作为可用的正式版安装。

## 安装

需要 Node.js 22.14+ 和 npm，无需安装 Go 或编译源码。**以下命令在官方 npm 包正式发布后才可执行。**

选择你使用的 Agent，一次安装 CLI 和 Skill：

```bash
# Codex
npx @validflow/ccneed-cli@latest install --agent codex

# WorkBuddy
npx @validflow/ccneed-cli@latest install --agent workbuddy
```

安装完成后，直接使用：

```bash
ccneed --help
```

CLI 全局安装，Skill 放在所选 Agent 的用户目录，不修改你的项目。两个 Agent 都用，可指定 `--agent codex,workbuddy`；Claude Code 使用 `--agent claude`。安装后重新打开 Agent，具体步骤见 [安装指南](INSTALL.md)。

### 让 Agent 帮你安装

也可以直接把这句话发给 Codex 或 WorkBuddy：

```text
请阅读 https://github.com/validflow/ccneed-cli 的 INSTALL.md，
确认官方正式版已发布后，帮我安装 CCNeed CLI 和当前 Agent 的 Skill。
登录时把官网授权链接发给我，由我完成授权。
```

## 登录

🔑 使用已有的 CCNeed 官网账号，不用再注册一套账号。

先发起登录：

```bash
ccneed auth login
```

打开返回的官网链接，核对账号和确认码，点击同意授权。回到终端后完成登录并验证：

```bash
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
ccneed account credits
```

第一步返回 `pending` 不是登录成功；完成授权后才能使用账号功能。不要把密码、Token 或浏览器 Cookie 发给 Agent。

## 使用

💬 安装 Skill 并登录后，在你的 Agent 里直接说：

```text
使用 CCNeed，先让我选择知识库和文件，
分析高价值博主，生成报告并下载到我指定的目录。
需要确认的地方请问我。
```

也可以直接在终端操作。下面的 ID 要替换为命令返回的真实值：

```bash
ccneed kb list
ccneed files upload ./posts.csv
ccneed threads create --title "博主分析" --kb-id 7 --file-id FILE_UUID
ccneed run "分析已绑定资料并生成报告" --thread-id THREAD_UUID
```

文件上传后还需要绑定到会话；创建会话不会启动研究，`run` 才会提交任务。任务可能消耗官网积分。

### 查看进度与报告

`run` 返回会话 ID 和本轮任务 ID 后就退出，研究继续在云端执行：

```bash
ccneed tasks watch --thread-id THREAD_UUID --turn-id TURN_UUID --timeout 60s
ccneed artifacts list --thread-id THREAD_UUID
ccneed artifacts download --thread-id THREAD_UUID --artifact-id ARTIFACT_UUID --output ./report.md
ccneed artifacts export-pdf --thread-id THREAD_UUID --artifact-id ARTIFACT_UUID --output ./report.pdf
```

`watch` 等待超时不取消任务，稍后查询同一轮即可。需要确认时，Agent 会把问题和选项交给你，再用 `tasks reply` 提交你的答案。任务完成后可查看本轮回答及 findings/reports，不展示隐藏推理或完整工具日志。阅读带图表与排版的完整报告，建议 `export-pdf`；需要原文或继续编辑时用 `download` 保存 MD。两者都不会覆盖已有文件。

### 常用命令

| 想做什么 | 命令 |
| --- | --- |
| 💳 查看积分、订阅 | `ccneed account credits` / `ccneed account overview` |
| 💬 查看会话 | `ccneed threads list` |
| 📚 查看知识库 | `ccneed kb list` |
| ➕ 创建知识库 | `ccneed kb create`，返回网页入口 |
| 📎 查看、上传文件 | `ccneed files list` / `ccneed files upload ./posts.csv` |
| 🗂️ 查看、追加会话资料 | `ccneed threads resources` / `ccneed threads append` |
| 🕘 查看任务历史 | `ccneed tasks list --thread-id THREAD_UUID` |
| 🚪 退出登录 | `ccneed auth logout` |

表中未展开的参数，可用 `ccneed <命令> --help` 查看。知识库与文件绑定只追加、不解绑；完整调用规则见 [Skill](skills/ccneed/SKILL.md)。

## 更新

🔄 正式发布后，重新执行上面的安装命令即可更新。CLI 和已接入的 Skill 一起升级，不需要分别下载；更新后重新打开 Agent。更新提醒不会替你自动升级。

## 反馈与贡献

遇到问题请 [提交 Issue](https://github.com/validflow/ccneed-cli/issues)，附上操作系统、CLI 版本和脱敏后的错误。不要上传凭证或私人文件。

[安装与排障](INSTALL.md) · [开发与贡献](CONTRIBUTING.md) · [代码规范](CODE_STYLE.md) · [安全说明](SECURITY.md) · [版本发布](RELEASING.md)

当前尚未授予开源许可证。开发者联调、源码构建及发行验收说明请查看贡献与安装指南。
