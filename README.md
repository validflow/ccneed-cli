<div align="center">
  <img src="https://raw.githubusercontent.com/validflow/ccneed-cli/main/assets/ccneed-logo.png" alt="CCNeed mascot holding a magnifying glass" width="280">
  <h1>CCNeed CLI</h1>
  <p>Run research with CCNeed from your own AI agent or terminal.</p>
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

CCNeed runs long research tasks in the cloud. Its CLI lets you prepare research sources, submit work, follow progress, answer questions, and bring reports back into the agent you already use. The CLI and its companion Skill are installed together.

> Some cloud commands depend on the corresponding CCNeed service APIs being available in production.

## Install

Node.js 22.14+ and npm are required. Choose your agent: `codex`, `workbuddy` (local coding environment), or `claude`.

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

Replace `codex` with your agent, or use `--agent codex,workbuddy` for both. The installer adds a global `ccneed` command and links the same-version Skill into the selected agent's user directory; it does not modify the current project. Restart your agent after installation.

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

This builds the development CLI only. The source Skill lives in `skills/ccneed` and is **not** installed or updated automatically by `go install`. For a linked CLI + Skill that update together, use the npm method above. See the [installation guide](INSTALL.md) for PATH and source-build details.

### Ask your agent to install it

```text
Read https://github.com/validflow/ccneed-cli/blob/main/INSTALL.md.
Install the official CCNeed CLI and Skill for the agent I am using.
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

Or use the CLI directly. For each new research conversation, choose an existing knowledge base **or** upload your own file, then bind the chosen resource when creating the conversation. You can bind both. Replace example IDs with values returned by your commands:

```bash
ccneed kb list
ccneed threads create --title "iPhone 18 research" --kb-id 7
ccneed run "Analyze iPhone 18 discussions in the bound knowledge base and prepare a report" --thread-id THREAD_UUID
```

If you need a new knowledge base, run `ccneed kb create`, complete the crawl setup on the returned website, then use `ccneed kb list` / `ccneed kb get --kb-id 7` to find an available ID. CCNeed handles collection, cleaning, and analysis-ready data in the cloud; you do not need to manage its internal CSV files.

If you want to start from your own file instead, run `ccneed files upload ./research-brief.pdf`, then `ccneed threads create --title "Brief analysis" --file-id FILE_UUID`. Uploading alone does not bind the file. Creating a conversation does not start a task; `run` does. A task may use your CCNeed credits.

`run` returns immediately while the cloud task continues. You can watch the same turn, respond to a confirmation request, and retrieve its results:

```bash
ccneed tasks watch --thread-id THREAD_UUID --turn-id TURN_UUID --timeout 60s
ccneed artifacts list --thread-id THREAD_UUID
ccneed artifacts export-pdf --thread-id THREAD_UUID --artifact-id ARTIFACT_UUID --output ./report.pdf
```

For an editable Markdown report, use `ccneed artifacts download` instead. `watch` timing out does not cancel the cloud task; you can check again later. For all options, run `ccneed <command> --help` or read the [Skill](skills/ccneed/SKILL.md).

### At a glance

| What you need | Command |
| --- | --- |
| 💳 Check credits and subscription | `ccneed account credits` / `ccneed account overview` |
| 💬 Find conversations | `ccneed threads list` |
| 📚 Explore knowledge bases | `ccneed kb list` |
| ➕ Create a knowledge base in the browser | `ccneed kb create` |
| 📎 List or upload your own files | `ccneed files list` / `ccneed files upload ./research-brief.pdf` |
| 🗂️ View or add conversation resources | `ccneed threads resources` / `ccneed threads append` |
| 🕘 Review task history | `ccneed tasks list --thread-id THREAD_UUID` |
| 🚪 Sign out | `ccneed auth logout` |

Knowledge-base and file bindings can be added, but not removed. Use `ccneed <command> --help` for the required arguments.

## Update and contribute

Run the same `npx @validflow/ccneed-cli@latest install --agent ...` command to update both the CLI and linked Skill. Reload your agent afterward; update notifications never install software without your action.

Found a problem? [Open an issue](https://github.com/validflow/ccneed-cli/issues) with your OS, CLI version, and a redacted error message. Never post credentials or private files.

For product news and guides, follow [CCNeed on X](https://x.com/CCNeed_ai), [Ccneedai on Medium](https://medium.com/@ccneedai), or [CC Need on DEV](https://dev.to/cc_need_be7954bac5ca859e1).

[Installation](INSTALL.md) · [Contributing](CONTRIBUTING.md) · [Code style](CODE_STYLE.md) · [Security](SECURITY.md) · [Releasing](RELEASING.md) · [Changelog](CHANGELOG.md)
