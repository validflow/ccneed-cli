<div align="center">
  <img src="https://raw.githubusercontent.com/validflow/ccneed-cli/main/assets/ccneed-logo.png" alt="CCNeed mascot holding a magnifying glass" width="280">
  <h1>CCNeed CLI + Agent Plugins</h1>
  <p>Collect data or run cloud research from Claude Code, Codex, or WorkBuddy.</p>
  <p><a href="README.zh.md">中文</a> | <a href="README.md">English</a></p>
  <p>
    <a href="https://x.com/CCNeed_ai"><img src="https://img.shields.io/badge/X-CCNeed-333840?style=flat&amp;logo=x&amp;logoColor=white&amp;labelColor=15171A" alt="X: CCNeed"></a>
    <a href="https://medium.com/@ccneedai"><img src="https://img.shields.io/badge/Medium-Ccneedai-259B75?style=flat&amp;logo=medium&amp;logoColor=white&amp;labelColor=14684F" alt="Medium: Ccneedai"></a>
    <a href="https://dev.to/cc_need_be7954bac5ca859e1"><img src="https://img.shields.io/badge/DEV-CC%20Need-5666CE?style=flat&amp;logo=devdotto&amp;logoColor=white&amp;labelColor=303C98" alt="DEV: CC Need"></a>
  </p>
  <p>
    <a href="https://ccneed.ai">Website</a> ·
    <a href="INSTALL.md">Installation guide</a> ·
    <a href="skills/ccneed/SKILL.md">Agent Skill</a> ·
    <a href="https://github.com/validflow/ccneed-cli/issues">Issues</a>
  </p>
</div>

CCNeed combines a global CLI with native Agent plugins for Claude Code, Codex and WorkBuddy. Collect and export data for your own Agent to analyze, or let CCNeed's cloud Agent analyze your knowledge base and return a report. The plugins bundle the shared Skill and collection hooks; each host manages their loading and lifecycle.

## What you can do

- **Collect data for your own Agent:** prepare a collection plan, collect into a knowledge base, and export CSV or JSON for local analysis. Export filenames include the KB ID, name and timestamp.
- **Run cloud research:** give CCNeed's cloud Agent your question, knowledge bases and files, follow its progress, respond to confirmation requests, and download the finished report.
- **Review before collecting:** see the research requirement, every platform's entity and other keywords, quotas, estimated credits and balance. You can adjust the plan before confirming; starting collection also requires your host's native execution approval. Unspecified counts default to 300 items across all platforms.
- **Use your preferred Agent:** native plugins for Claude Code, Codex and WorkBuddy share one Skill and collection hooks. The installer manages the global CLI and host-specific plugin setup.

Collection and export require matching CCNeed service APIs. See [installation](INSTALL.md) for host support and service prerequisites.

## Install

Node.js 22.14+ and npm are required. Choose your agent: `codex`, `workbuddy` (local coding environment), or `claude`.

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

Replace `codex` with your agent, or use `--agent codex,workbuddy` for both. The installer updates the global CLI and registers its same-version native plugin with Claude Code or Codex. WorkBuddy desktop requires the native plugin installation step shown in the result. The plugin bundles the shared Skill and collection hooks. Restart your agent, review/trust the hooks, and keep manual approval enabled. See [installation](INSTALL.md) for requirements and per-host results.

```bash
ccneed --help
```

### Build from source

For development, use Git and Go 1.25+:

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
ccneed --help
```

This builds the development CLI only. The source Skill lives in `skills/ccneed`; `go install` does not register a plugin. Use the npm installer for coordinated CLI and native plugin updates. See the [installation guide](INSTALL.md) for PATH and source-build details.

### Ask your agent to install it

```text
Read https://github.com/validflow/ccneed-cli/blob/main/INSTALL.md.
Install the official CCNeed CLI and native plugin for the agent I am using.
Give me the website authorization link; I will approve the login myself.
```

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

For new data, your agent prepares a plan with `kb plan`, presents it conversationally and waits for your confirmation, then requests native permission before `kb start`. Once collection completes, the Agent normally exports and delivers CSV automatically; requested formats and destinations take priority. See [collection instructions](skills/ccneed/references/acquisition.md). The legacy `kb create` website entry remains available.

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

## Update and contribute

Run the same `npx @validflow/ccneed-cli@latest install --agent ...` command to update the CLI and native plugins together for every host you name. Check each host's result and complete any pending native installation or trust steps. Reload your agent afterward; update notifications never install software without your action.

Found a problem? [Open an issue](https://github.com/validflow/ccneed-cli/issues) with your OS, CLI version, and a redacted error message. Never post credentials or private files.

For product news and guides, follow [CCNeed on X](https://x.com/CCNeed_ai), [Ccneedai on Medium](https://medium.com/@ccneedai), or [CC Need on DEV](https://dev.to/cc_need_be7954bac5ca859e1).

[Installation](INSTALL.md) · [Contributing](CONTRIBUTING.md) · [Code style](CODE_STYLE.md) · [Security](SECURITY.md) · [Releasing](RELEASING.md) · [Changelog](CHANGELOG.md)
