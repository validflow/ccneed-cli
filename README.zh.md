<div align="center">
  <p><picture>
    <source media="(prefers-reduced-motion: reduce) and (prefers-color-scheme: dark)" srcset="assets/wordmark-dark.png">
    <source media="(prefers-reduced-motion: reduce)" srcset="assets/wordmark-light.png">
    <source media="(prefers-color-scheme: dark)" srcset="assets/wordmark-dark.gif">
    <img src="assets/wordmark-light.gif" width="550" alt="CCNEED CLI">
  </picture></p>
  <p>让你的 Agent 获取社交数据，或交给云端 Agent 分析。</p>
  <p><a href="README.zh.md">中文</a> | <a href="README.md">English</a></p>
  <p>
    <a href="https://x.com/CCNeed_ai"><img src="https://img.shields.io/badge/X-CCNeed-333840?style=flat&amp;logo=x&amp;logoColor=white&amp;labelColor=15171A" alt="X：CCNeed"></a>
    <a href="https://medium.com/@ccneedai"><img src="https://img.shields.io/badge/Medium-Ccneedai-259B75?style=flat&amp;logo=medium&amp;logoColor=white&amp;labelColor=14684F" alt="Medium：Ccneedai"></a>
    <a href="https://dev.to/cc_need_be7954bac5ca859e1"><img src="https://img.shields.io/badge/DEV-CC%20Need-5666CE?style=flat&amp;logo=devdotto&amp;logoColor=white&amp;labelColor=303C98" alt="DEV：CC Need"></a>
  </p>
  <p>
    <a href="https://ccneed.ai">官网</a> ·
    <a href="docs/zh/INSTALL.md">安装指南</a> ·
    <a href="skills/ccneed/SKILL.md">Agent Skill</a> ·
    <a href="https://github.com/validflow/ccneed-cli/issues">反馈</a>
  </p>
</div>

通过 CLI 和原生插件，把你的 Agent 连接到 CCNeed 的社媒数据采集与云端研究能力。

## 可以做什么

- **获取数据：**采集到知识库，导出 CSV 或 JSON，交给自己的 Agent 分析。
- **云端研究：**让 CCNeed 云端 Agent 分析知识库和文件，下载研究报告。
- **采集前确认：**调整关键词、采集目标和预算，在对话中确认方案后执行采集。

## 安装与更新

<table align="center">
  <tr>
    <td width="160" align="center" valign="top"><img src="assets/agents/claude-code.svg" width="28" height="28" alt=""><br>Claude&nbsp;Code</td>
    <td width="160" align="center" valign="top"><img src="assets/agents/codex.svg" width="28" height="28" alt=""><br>Codex</td>
    <td width="160" align="center" valign="top"><img src="assets/agents/workbuddy.svg" width="28" height="28" alt=""><br>WorkBuddy</td>
    <td width="160" align="center" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="assets/agents/kimi-dark.svg"><img src="assets/agents/kimi.svg" width="28" height="28" alt=""></picture><br>Kimi&nbsp;Code</td>
    <td width="160" align="center" valign="top"><img src="assets/agents/openclaw.svg" width="28" height="28" alt=""><br>OpenClaw</td>
  </tr>
  <tr>
    <td width="160" align="center" valign="top"><img src="assets/agents/pi.svg" width="28" height="28" alt=""><br>Pi</td>
    <td width="160" align="center" valign="top"><img src="assets/agents/zcode.png" width="28" height="28" alt=""><br>ZCode</td>
    <td width="160" align="center" valign="top"><img src="assets/agents/qoder.svg" width="28" height="28" alt=""><br>Qoder</td>
    <td width="160" align="center" valign="top"><img src="assets/agents/trae.png" width="28" height="28" alt=""><br>Trae&nbsp;CLI</td>
    <td width="160" align="center" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="assets/agents/dsh-dark.svg"><img src="assets/agents/dsh.svg" width="28" height="28" alt=""></picture><br>DeepSeek&nbsp;Harness</td>
  </tr>
</table>

需要 Node.js 22.14+ 和 npm。

### 让 Agent 帮你安装

```text
请阅读 https://github.com/validflow/ccneed-cli/blob/main/docs/en/INSTALL.md，
为我当前使用的 Agent 安装或更新 CCNeed CLI 和原生插件。
```

### 手动安装

用同一条命令安装或更新 CLI 和所选 Agent 的插件：

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

在[安装指南](docs/zh/INSTALL.md)中选择所用 Agent 的安装标识；多个标识用逗号组合，只配置你选择的宿主。

按安装结果完成原生管理器中的操作，再重载 Agent；部分宿主需要在应用内导入插件。兼容要求和接入步骤见[安装指南](docs/zh/INSTALL.md)。

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

需要新数据时，Agent 通过 `kb plan` 准备方案，用自然语言展示并等待对话确认，确认后按宿主已有权限执行 `kb start`。采集、打标完成后，Agent 通常自动导出 CSV 并交付本地文件；指定格式与目录优先。详见[采集说明](skills/ccneed/references/acquisition.md)；原来的 `kb create` 网页入口继续保留。

需要云端分析时，带上第一句话和知识库 ID 开启会话。如果还要绑定文件，先上传，再使用返回的文件 ID：

```bash
ccneed files upload ./research-brief.pdf
ccneed run "研究 iPhone 18 的相关讨论并生成报告" --kb-id KB_ID --file-id FILE_ID
```

提交任务、文件绑定、进度跟进、确认和报告的具体命令，交给 [Agent Skill](skills/ccneed/SKILL.md) 说明。

### 功能速览

| 想做什么 | 命令 |
| --- | --- |
| 💳 查看积分、订阅 | `ccneed account overview` |
| 🌐 查询可采集平台 | `ccneed kb platforms` |
| 📚 查看知识库 | `ccneed kb list` |
| 📝 准备采集方案 | `ccneed kb plan --plan-file plan.json --output prepared.json` |
| 📤 导出知识库数据 | `ccneed kb export --kb-id KB_ID --format csv` |
| 🚀 开启云端研究 | `ccneed run "你的研究问题" --kb-id KB_ID` |

具体参数可用 `ccneed <命令> --help` 查看。

## 反馈与贡献

遇到问题请 [提交 Issue](https://github.com/validflow/ccneed-cli/issues)，附上操作系统、CLI 版本和脱敏后的错误。不要上传凭证或私人文件。

产品动态和使用文章可关注 [X 上的 CCNeed](https://x.com/CCNeed_ai)、[Medium 上的 Ccneedai](https://medium.com/@ccneedai) 或 [DEV 上的 CC Need](https://dev.to/cc_need_be7954bac5ca859e1)。

[安装与排障](docs/zh/INSTALL.md) · [参与贡献](docs/zh/CONTRIBUTING.md) · [代码规范](docs/zh/CODE_STYLE.md) · [安全说明](docs/zh/SECURITY.md) · [版本发布](docs/zh/RELEASING.md) · [更新记录](docs/zh/CHANGELOG.md)
