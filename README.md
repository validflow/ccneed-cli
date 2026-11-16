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

<p>
  <img src="assets/agents/claude-code.svg" width="28" height="28" align="absmiddle" alt="">&nbsp;Claude&nbsp;Code
  &nbsp;·&nbsp;
  <img src="assets/agents/codex.svg" width="28" height="28" align="absmiddle" alt="">&nbsp;Codex
  &nbsp;·&nbsp;
  <img src="assets/agents/workbuddy.svg" width="28" height="28" align="absmiddle" alt="">&nbsp;WorkBuddy
  &nbsp;·&nbsp;
  <picture><source media="(prefers-color-scheme: dark)" srcset="assets/agents/kimi-dark.svg"><img src="assets/agents/kimi.svg" width="28" height="28" align="absmiddle" alt=""></picture>&nbsp;Kimi&nbsp;Code
</p>

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

Replace `codex` as needed: `claude` (Claude Code), `codex`, `workbuddy` or `kimi`. Select multiple with commas, e.g. `--agent claude,codex`.

Follow the installer result to finish plugin setup, then restart your Agent. See the [installation guide](docs/en/INSTALL.md) for host requirements, native import steps and troubleshooting.

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
| 💳 Check credits and subscription | `ccneed account credits` / `ccneed account overview` |
| 💬 Find conversations | `ccneed threads list` |
| 🚀 Start research | `ccneed run "Your question" --kb-id KB_ID` |
| 📚 Explore knowledge bases | `ccneed kb list` |
| 📊 Check a knowledge base's data mix | `ccneed kb stats --kb-id KB_ID` |
| Export data for your own agent | `ccneed kb export --kb-id KB_ID --format csv` |
| Prepare a collection plan | `ccneed kb plan --plan-file ./plan.json --output ./prepared.json` |
| ➕ Create a knowledge base in the browser | `ccneed kb create` |
| 📎 List or upload your own files | `ccneed files list` / `ccneed files upload ./research-brief.pdf` |
| 🗂️ View or add conversation resources | `ccneed threads resources` / `ccneed threads append` |
| 🕘 Review task history | `ccneed tasks list --thread-id THREAD_UUID` |
| 🔎 Follow a research turn | `ccneed tasks get --thread-id THREAD_UUID --turn-id TURN_UUID` |
| ⏳ Wait for a result or question | `ccneed tasks watch --thread-id THREAD_UUID --turn-id TURN_UUID --timeout 60s` |
| 🚪 Sign out | `ccneed auth logout` |

Knowledge-base and file bindings can be added, but not removed. Use `ccneed <command> --help` for the required arguments.

## Feedback and contributing

Found a problem? [Open an issue](https://github.com/validflow/ccneed-cli/issues) with your OS, CLI version, and a redacted error message. Never post credentials or private files.

For product news and guides, follow [CCNeed on X](https://x.com/CCNeed_ai), [Ccneedai on Medium](https://medium.com/@ccneedai), or [CC Need on DEV](https://dev.to/cc_need_be7954bac5ca859e1).

[Installation](docs/en/INSTALL.md) · [Contributing](CONTRIBUTING.md) · [Code style](docs/en/CODE_STYLE.md) · [Security](SECURITY.md) · [Releasing](docs/en/RELEASING.md) · [Changelog](docs/en/CHANGELOG.md)
