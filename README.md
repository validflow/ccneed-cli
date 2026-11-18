<div align="center">
  <img src="https://raw.githubusercontent.com/validflow/ccneed-cli/main/assets/ccneed-logo.png" alt="CCNeed mascot holding a magnifying glass" width="280">
  <h1>CCNeed CLI + Agent Plugins</h1>
  <p>Social data collection and cloud research for your Agent.</p>
  <p><a href="README.zh.md">中文</a> | <a href="README.md">English</a></p>
  <p>
    <a href="https://x.com/CCNeed_ai"><img src="https://img.shields.io/badge/X-CCNeed-333840?style=flat&amp;logo=x&amp;logoColor=white&amp;labelColor=15171A" alt="X: CCNeed"></a>
    <a href="https://medium.com/@ccneedai"><img src="https://img.shields.io/badge/Medium-Ccneedai-259B75?style=flat&amp;logo=medium&amp;logoColor=white&amp;labelColor=14684F" alt="Medium: Ccneedai"></a>
    <a href="https://dev.to/cc_need_be7954bac5ca859e1"><img src="https://img.shields.io/badge/DEV-CC%20Need-5666CE?style=flat&amp;logo=devdotto&amp;logoColor=white&amp;labelColor=303C98" alt="DEV: CC Need"></a>
  </p>
  <p>
    <a href="https://ccneed.ai">Website</a> ·
    <a href="docs/en/INSTALL.md">Installation guide</a> ·
    <a href="skills/ccneed/SKILL.md">Agent Skill</a> ·
    <a href="https://github.com/validflow/ccneed-cli/issues">Issues</a>
  </p>
</div>

CCNeed connects your Agent to social data collection and cloud research through a CLI and native plugins.

## What you can do

- **Get data:** collect into a knowledge base and export CSV or JSON for your own Agent.
- **Run cloud research:** analyze knowledge bases and files with CCNeed's cloud Agent, then download the report.
- **Review before collecting:** adjust keywords, collection targets and budget, confirm the plan, then approve execution in your Agent.

## Install or update

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

Requires Node.js 22.14+ and npm.

### Ask your Agent

```text
Read https://github.com/validflow/ccneed-cli/blob/main/docs/en/INSTALL.md
and install or update CCNeed CLI and its native plugin for my current Agent.
```

### Install manually

Use one command to install or update the CLI and your chosen Agent's plugin:

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

Choose your Agent's identifier in the [installation guide](docs/en/INSTALL.md). Separate multiple identifiers with commas; only the hosts you select are configured.

Complete the native manager steps returned by the installer, then reload your Agent. Some hosts require an in-app import. See the [installation guide](docs/en/INSTALL.md) for compatibility and setup.

## Sign in

Use your existing CCNeed account. The CLI opens a website authorization flow; it does not need your password or browser cookies.

```bash
ccneed auth login
# Open the URL, check the confirmation code, and approve in your browser.
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
ccneed account credits
```

Without extra flags, every command connects to the production service at `https://ccneed.ai`. Regular users should not set `--api-url`. Only authorized developers may use it to connect to staging with a staging test account; staging and production credentials must stay separate.

## Work with CCNeed

In your agent, you can ask:

```text
Use my CCNeed knowledge base to research iPhone 18 discussions and bring me a PDF report.
Ask me before making any choice that needs my confirmation.
```

For existing data, export it without starting a research conversation:

```bash
ccneed kb export --kb-id KB_ID --format csv
ccneed kb export --kb-id KB_ID --format json
```

For new data, your agent prepares a plan with `kb plan`, presents it conversationally and waits for your confirmation, then requests native permission before `kb start`. Once collection and tagging finish, the Agent normally exports and delivers CSV automatically; requested formats and destinations take priority. See [collection instructions](skills/ccneed/references/acquisition.md). The legacy `kb create` website entry remains available.

For cloud analysis, start a conversation with your first question and the knowledge-base ID. To bind a file too, upload it first and use the returned file ID:

```bash
ccneed files upload ./research-brief.pdf
ccneed run "Research iPhone 18 discussions and prepare a report" --kb-id KB_ID --file-id FILE_ID
```

The [Agent Skill](skills/ccneed/SKILL.md) covers the remaining commands, including tasks, files, confirmations, and reports.

### At a glance

| What you need | Command |
| --- | --- |
| 💳 Check credits and subscription | `ccneed account overview` |
| 🌐 Discover collection platforms | `ccneed kb platforms` |
| 📚 Explore knowledge bases | `ccneed kb list` |
| 📝 Prepare a collection plan | `ccneed kb plan --plan-file plan.json --output prepared.json` |
| 📤 Export knowledge-base data | `ccneed kb export --kb-id KB_ID --format csv` |
| 🚀 Start cloud research | `ccneed run "Your question" --kb-id KB_ID` |

Use `ccneed <command> --help` for options and required arguments.

## Feedback and contributing

Found a problem? [Open an issue](https://github.com/validflow/ccneed-cli/issues) with your OS, CLI version, and a redacted error message. Never post credentials or private files.

For product news and guides, follow [CCNeed on X](https://x.com/CCNeed_ai), [Ccneedai on Medium](https://medium.com/@ccneedai), or [CC Need on DEV](https://dev.to/cc_need_be7954bac5ca859e1).

[Installation](docs/en/INSTALL.md) · [Contributing](CONTRIBUTING.md) · [Code style](docs/en/CODE_STYLE.md) · [Security](SECURITY.md) · [Releasing](docs/en/RELEASING.md) · [Changelog](docs/en/CHANGELOG.md)
