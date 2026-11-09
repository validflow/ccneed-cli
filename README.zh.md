<div align="center">
  <img src="https://raw.githubusercontent.com/validflow/ccneed-cli/main/assets/ccneed-logo.png" alt="拿着放大镜的 CCNeed 小河马" width="280">
  <h1>CCNeed CLI + Agent 插件</h1>
  <p>在 Claude Code、Codex、WorkBuddy 中采集数据或调用云端分析。</p>
  <p><a href="README.zh.md">中文</a> | <a href="README.md">English</a></p>
  <p>
    <a href="https://x.com/CCNeed_ai"><img src="https://img.shields.io/badge/X-CCNeed-333840?style=flat&amp;logo=x&amp;logoColor=white&amp;labelColor=15171A" alt="X：CCNeed"></a>
    <a href="https://medium.com/@ccneedai"><img src="https://img.shields.io/badge/Medium-Ccneedai-259B75?style=flat&amp;logo=medium&amp;logoColor=white&amp;labelColor=14684F" alt="Medium：Ccneedai"></a>
    <a href="https://dev.to/cc_need_be7954bac5ca859e1"><img src="https://img.shields.io/badge/DEV-CC%20Need-5666CE?style=flat&amp;logo=devdotto&amp;logoColor=white&amp;labelColor=303C98" alt="DEV：CC Need"></a>
  </p>
  <p>
    <a href="https://ccneed.ai">官网</a> ·
    <a href="INSTALL.zh.md">安装指南</a> ·
    <a href="skills/ccneed/SKILL.md">Agent Skill</a> ·
    <a href="https://github.com/validflow/ccneed-cli/issues">反馈</a>
  </p>
</div>

通过 CLI 和原生插件，在 Claude Code、Codex、WorkBuddy 中使用 CCNeed 的社媒数据采集与云端研究能力。

## 可以做什么

- **获取数据：**采集到知识库，导出 CSV 或 JSON，交给自己的 Agent 分析。
- **云端研究：**让 CCNeed 云端 Agent 分析知识库和文件，下载研究报告。
- **采集前确认：**调整关键词、采集目标和预算，确认方案后通过 Agent 原生权限界面批准执行。

## 安装

需要 Node.js 22.14+ 和 npm。选择你使用的 Agent：`codex`、`workbuddy`（本地代码环境）或 `claude`。

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

把 `codex` 换成自己的 Agent；同时接入两个 Agent 可写 `--agent codex,workbuddy`。安装器更新全局 CLI，并通过 Claude Code、Codex 的原生插件管理器注册同版本插件。WorkBuddy 桌面版需完成返回结果里的原生安装步骤。插件携带统一 Skill 与采集 hooks。安装后重新打开 Agent、审阅并信任 hooks，保持人工审批模式；逐宿主结果与要求见[安装指南](INSTALL.zh.md)。

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

这条路径只构建开发版 CLI。源码 Skill 位于 `skills/ccneed`；`go install` 不注册插件。CLI 与原生插件的配套升级使用上面的 npm 安装方式。PATH 与源码构建说明见[安装指南](INSTALL.zh.md)。

### 让 Agent 帮你安装

```text
请阅读 https://github.com/validflow/ccneed-cli/blob/main/INSTALL.md。
为我当前使用的 Agent 安装官方 CCNeed CLI 和原生插件。
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
使用我在 CCNeed 的知识库，研究 iPhone 18 的相关讨论，并把 PDF 报告交给我。
需要我确认的地方，请先问我。
```

已有数据可直接导出，不必开启研究会话：

```bash
ccneed kb export --kb-id KB_ID --format csv
ccneed kb export --kb-id KB_ID --format json
```

需要新数据时，Agent 通过 `kb plan` 准备方案，用自然语言展示并等待对话确认，再在 `kb start` 前请求原生权限。采集、打标完成后，Agent 通常自动导出 CSV 并交付本地文件；指定格式与目录优先。详见[采集说明](skills/ccneed/references/acquisition.md)；原来的 `kb create` 网页入口继续保留。

需要云端分析时，带上第一句话和知识库 ID 开启会话。如果还要绑定文件，先上传，再使用返回的文件 ID：

```bash
ccneed files upload ./research-brief.pdf
ccneed run "研究 iPhone 18 的相关讨论并生成报告" --kb-id KB_ID --file-id FILE_ID
```

提交任务、文件绑定、进度跟进、确认和报告的具体命令，交给 [Agent Skill](skills/ccneed/SKILL.md) 说明。

### 功能速览

| 想做什么 | 命令 |
| --- | --- |
| 💳 查看积分、订阅 | `ccneed account credits` / `ccneed account overview` |
| 💬 查看会话 | `ccneed threads list` |
| 🚀 开启研究 | `ccneed run "你的研究问题" --kb-id KB_ID` |
| 📚 查看知识库 | `ccneed kb list` |
| 📊 查看知识库数据分布 | `ccneed kb stats --kb-id KB_ID` |
| 导出数据交给自己的 Agent | `ccneed kb export --kb-id KB_ID --format csv` |
| 准备采集方案 | `ccneed kb plan --plan-file ./plan.json --output ./prepared.json` |
| ➕ 在网页创建知识库 | `ccneed kb create` |
| 📎 查看、上传自己的文件 | `ccneed files list` / `ccneed files upload ./research-brief.pdf` |
| 🗂️ 查看、追加会话资料 | `ccneed threads resources` / `ccneed threads append` |
| 🕘 查看任务历史 | `ccneed tasks list --thread-id THREAD_UUID` |
| 🔎 查看本轮研究进度 | `ccneed tasks get --thread-id THREAD_UUID --turn-id TURN_UUID` |
| ⏳ 等待本轮结果或确认 | `ccneed tasks watch --thread-id THREAD_UUID --turn-id TURN_UUID --timeout 60s` |
| 🚪 退出登录 | `ccneed auth logout` |

知识库和文件绑定只追加、不解绑。具体参数可用 `ccneed <命令> --help` 查看。

## 更新与反馈

重新运行 `npx @validflow/ccneed-cli@latest install --agent ...`，列出所有需要更新的宿主，一起更新 CLI 与原生插件。检查各宿主的结果，完成待办的原生安装或信任步骤。更新后重载 Agent；更新提醒不会自动安装。

遇到问题请 [提交 Issue](https://github.com/validflow/ccneed-cli/issues)，附上操作系统、CLI 版本和脱敏后的错误。不要上传凭证或私人文件。

产品动态和使用文章可关注 [X 上的 CCNeed](https://x.com/CCNeed_ai)、[Medium 上的 Ccneedai](https://medium.com/@ccneedai) 或 [DEV 上的 CC Need](https://dev.to/cc_need_be7954bac5ca859e1)。

[安装与排障](INSTALL.zh.md) · [参与贡献](CONTRIBUTING.zh.md) · [代码规范](CODE_STYLE.zh.md) · [安全说明](SECURITY.zh.md) · [版本发布](RELEASING.zh.md) · [更新记录](CHANGELOG.zh.md)
