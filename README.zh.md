<div align="center">
  <img src="https://raw.githubusercontent.com/validflow/ccneed-cli/main/assets/ccneed-logo.png" alt="拿着放大镜的 CCNeed 小河马" width="280">
  <h1>CCNeed CLI</h1>
  <p>在你熟悉的 Agent 或终端里使用 CCNeed。</p>
  <p><a href="README.zh.md">中文</a> | <a href="README.md">English</a></p>
  <p>
    <a href="https://x.com/CCNeed_ai"><img src="https://img.shields.io/badge/X-CCNeed-333840?style=flat&amp;logo=x&amp;logoColor=white&amp;labelColor=15171A" alt="X：CCNeed"></a>
    <a href="https://medium.com/@ccneedai"><img src="https://img.shields.io/badge/Medium-Ccneedai-259B75?style=flat&amp;logo=medium&amp;logoColor=white&amp;labelColor=14684F" alt="Medium：Ccneedai"></a>
    <a href="https://dev.to/cc_need_be7954bac5ca859e1"><img src="https://img.shields.io/badge/DEV-CC%20Need-5666CE?style=flat&amp;logo=devdotto&amp;logoColor=white&amp;labelColor=303C98" alt="DEV：CC Need"></a>
  </p>
  <p>
    <a href="https://ccneed.ai">官网</a> ·
    <a href="INSTALL.md">安装指南</a> ·
    <a href="skills/ccneed/SKILL.md">Agent Skill</a> ·
    <a href="https://github.com/validflow/ccneed-cli/issues">反馈</a>
  </p>
</div>

把长程研究任务交给 CCNeed 云端执行，继续在自己的 Agent 里查看进度、回答确认问题并取回报告。CLI 与配套 Skill 通过同一个入口安装。

> 部分云端命令仍取决于 CCNeed 正式环境中对应接口的上线进度。

## 安装

需要 Node.js 22.14+ 和 npm。选择你使用的 Agent：`codex`、`workbuddy`（本地代码环境）或 `claude`。

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

把 `codex` 换成自己的 Agent；同时接入两个 Agent 可写 `--agent codex,workbuddy`。安装器会添加全局 `ccneed` 命令，并把同版本 Skill 链接到所选 Agent 的用户目录，不修改当前项目。安装后重新打开 Agent。

```bash
ccneed --help
```

### 从源码构建

想查看或修改源码时，需要 Git 和 Go 1.25+：

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
ccneed --help
```

这条路径只构建开发版 CLI。源码 Skill 位于 `skills/ccneed`，不会随 `go install` 自动安装或更新。要让 CLI 与 Skill 同步升级，使用上面的 npm 安装方式。PATH 与源码构建说明见[安装指南](INSTALL.md)。

### 让 Agent 帮你安装

```text
请阅读 https://github.com/validflow/ccneed-cli/blob/main/INSTALL.md。
为我当前使用的 Agent 安装官方 CCNeed CLI 和配套 Skill。
登录时把官网授权链接发给我，由我自己确认授权。
```

## 登录

使用已有的 CCNeed 官网账号。CLI 会发起网页授权，不需要把密码或浏览器 Cookie 交给 Agent。

```bash
ccneed auth login
# 打开返回的链接，核对确认码，在网页上同意授权。
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
ccneed account credits
```

不额外指定参数时，所有命令默认连接正式线上服务 `https://ccneed.ai`。普通用户不需要设置 `--api-url`；只有获授权的开发者才能用它连接 staging，并使用 staging 测试账号联调。测试和正式环境的凭证必须隔离。

## 使用 CCNeed

你可以直接对自己的 Agent 说：

```text
使用 CCNeed 分析我的 posts.csv，筛选高价值博主，并把 PDF 报告交给我。
需要我确认的地方，请先问我。
```

也可以在终端执行。下面的 ID 要换成命令返回的真实值：

```bash
ccneed kb list
ccneed files upload ./posts.csv
ccneed threads create --title "博主分析" --kb-id 7 --file-id FILE_UUID
ccneed run "分析已绑定资料并生成报告" --thread-id THREAD_UUID
```

上传文件后还需要绑定到会话；创建会话不会启动研究，`run` 才会提交任务。任务可能消耗官网积分。

`run` 返回后，研究仍在云端执行。可以继续观察同一轮、回答确认问题并取回结果：

```bash
ccneed tasks watch --thread-id THREAD_UUID --turn-id TURN_UUID --timeout 60s
ccneed artifacts list --thread-id THREAD_UUID
ccneed artifacts export-pdf --thread-id THREAD_UUID --artifact-id ARTIFACT_UUID --output ./report.pdf
```

需要可编辑的 Markdown 原文时，改用 `ccneed artifacts download`。`watch` 超时不会取消云端任务，稍后可以继续查看。完整参数用 `ccneed <命令> --help` 查询，Agent 调用规则见 [Skill](skills/ccneed/SKILL.md)。

### 功能速览

| 想做什么 | 命令 |
| --- | --- |
| 💳 查看积分、订阅 | `ccneed account credits` / `ccneed account overview` |
| 💬 查看会话 | `ccneed threads list` |
| 📚 查看知识库 | `ccneed kb list` |
| ➕ 在网页创建知识库 | `ccneed kb create` |
| 📎 查看、上传文件 | `ccneed files list` / `ccneed files upload ./posts.csv` |
| 🗂️ 查看、追加会话资料 | `ccneed threads resources` / `ccneed threads append` |
| 🕘 查看任务历史 | `ccneed tasks list --thread-id THREAD_UUID` |
| 🚪 退出登录 | `ccneed auth logout` |

知识库和文件绑定只追加、不解绑。具体参数可用 `ccneed <命令> --help` 查看。

## 更新与反馈

重新运行 `npx @validflow/ccneed-cli@latest install --agent ...`，就能一起更新 CLI 和已接入的 Skill。更新后重新打开 Agent；更新提醒不会自动替你安装。

遇到问题请 [提交 Issue](https://github.com/validflow/ccneed-cli/issues)，附上操作系统、CLI 版本和脱敏后的错误。不要上传凭证或私人文件。

产品动态和使用文章可关注 [X 上的 CCNeed](https://x.com/CCNeed_ai)、[Medium 上的 Ccneedai](https://medium.com/@ccneedai) 或 [DEV 上的 CC Need](https://dev.to/cc_need_be7954bac5ca859e1)。

[安装与排障](INSTALL.md) · [参与贡献](CONTRIBUTING.md) · [代码规范](CODE_STYLE.md) · [安全说明](SECURITY.md) · [版本发布](RELEASING.md)
