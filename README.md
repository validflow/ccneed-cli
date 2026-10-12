<div align="center">
  <img src="assets/ccneed-logo.png" alt="CCNeed：拿着放大镜的白色小河马" width="320">
  <h1>CCNeed CLI</h1>
  <p>在你自己的 Agent 或终端中使用 CCNeed。</p>
  <p>
    <a href="https://ccneed.ai">官网</a> ·
    <a href="INSTALL.md">安装指南</a> ·
    <a href="skills/ccneed/SKILL.md">Skill</a> ·
    <a href="CONTRIBUTING.md">参与贡献</a> ·
    <a href="https://github.com/validflow/ccneed-cli/issues">反馈</a>
  </p>
</div>

把研究任务交给 CCNeed 云端执行，继续在你自己的 Agent 里查看进度、回答确认问题、拿回报告。

第一版源码已完成。npm 包尚未发布；下面是首发后的安装方式。现在需要从源码使用，可参阅 [安装指南](INSTALL.md)。

## 安装

需要 Node.js 22.14+ 和 npm；通过 npm 安装时无需安装 Go 或编译源码。

在自己的电脑上一次安装 CLI 和配套 Skill。将 `AGENT` 换成你使用的 Agent 类型（`codex`、`workbuddy` 或 `claude`）：

```bash
npx @validflow/ccneed-cli@latest install --agent AGENT
```

安装完成后，直接使用：

```bash
ccneed --help
```

安装器会把 CLI 装为全局 `ccneed` 命令，并把同版本 Skill 接入你选择的 Agent；不会修改当前项目。使用多个 Agent 时，可用逗号分隔类型。安装后重新打开 Agent，详见 [安装指南](INSTALL.md)。

### 让 Agent 帮你安装

也可以把这句话发给你正在使用的 Agent：

```text
请阅读 https://github.com/validflow/ccneed-cli 的 INSTALL.md，
确认官方 npm 包已发布、当前 Agent 在支持列表中后，
帮我在这台电脑上安装 CCNeed CLI 和对应的 Skill；不支持时先告诉我。
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

第一步返回 `pending` 只是等待授权；完成授权后才能使用账号功能。CLI 默认连接 CCNeed 正式环境的 HTTPS 地址 `https://ccneed.ai`。开发者可用 `--api-url` 显式连接 staging；不要把测试环境凭证用于正式环境，也不要把密码、Token 或浏览器 Cookie 发给 Agent。

## 使用

💬 安装并登录后，在你自己的 Agent 里直接说：

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

🔄 以后重新执行安装命令即可更新。CLI 和已接入的 Skill 一起升级，不需要分别下载；更新后重新打开 Agent。更新提醒不会替你自动升级。

## 反馈与贡献

遇到问题请 [提交 Issue](https://github.com/validflow/ccneed-cli/issues)，附上操作系统、CLI 版本和脱敏后的错误。不要上传凭证或私人文件。

[安装与排障](INSTALL.md) · [开发与贡献](CONTRIBUTING.md) · [代码规范](CODE_STYLE.md) · [安全说明](SECURITY.md) · [版本发布](RELEASING.md)

源码构建与开发联调见 [贡献指南](CONTRIBUTING.md)；npm 首发和仓库公开将单独进行。
