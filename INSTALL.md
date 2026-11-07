# Install CCNeed CLI and Skill

English · [中文](INSTALL.zh.md)

The CLI is installed globally and links its companion Skill into the selected host's user configuration. This guide also describes the upcoming collection hooks and native ask rules, subject to the release requirements below. Website login still requires explicit user approval.

> Release status: this branch keeps version 0.1.6 without publishing it again. The new `kb plan`, `kb start`, `kb export` and collection plugin require a new CLI release containing these changes and deployment of the matching Club APIs. `@latest` installs the published stable package, not unreleased branch changes. Keep the CLI, Skill and hooks on the same package version; merging code does not publish npm automatically.

## Install or update

After checking the official source and asking the user to approve installation, choose the Agent host:

```bash
npx @validflow/ccneed-cli@latest install --agent codex
# Or: --agent workbuddy, --agent claude, or --agent codex,workbuddy
```

Requires Node.js 22.14+ and npm. The installer places `ccneed` in npm's global command directory, which must be on the Agent's PATH. A local install does not automatically make it available inside a remote sandbox. In the new release described above, it also adds collection hooks and ask rules only for selected hosts, preserves unrelated settings, and does not change the host's permission mode, read account data or create tasks.

If GitHub downloads require a proxy, set `HTTPS_PROXY` (or `HTTP_PROXY`) in the terminal before installing. The binary downloader honors these variables and `NO_PROXY`, including their lowercase forms; no machine-specific proxy address is built into the package.

`@latest` is npm's current stable release; replace it with an exact version if reproducibility matters. The installer globally installs that exact CLI and Skill version. Selected hosts receive links to the one installed Skill directory:

| Host | Current-user Skill path |
| --- | --- |
| Codex | `~/.agents/skills/ccneed` |
| WorkBuddy desktop | `~/.workbuddy/skills/ccneed` |
| Claude Code | `~/.claude/skills/ccneed` |

These are directory links, not copies: Windows uses junctions; other systems use symlinks. Updating the global package updates already-linked hosts too, even if the update command names only one host. Restart or reload the host afterward; an existing Agent conversation may retain the old Skill in memory.

The package includes the Chinese Skill, references and collection hooks. Release and installation validate the Skill and CLI binary. A directory link alone is not proof of successful host loading. Earlier `.codebuddy/skills/ccneed` links are preserved; WorkBuddy desktop now uses its actual `.workbuddy` directory.

## Collection approval setup

This setup applies to the new coordinated release described above. Check the installed version and command help first; the published 0.1.6 package does not include these collection hooks.

| Host | Configuration added | Required mode |
| --- | --- | --- |
| Claude Code | `~/.claude/settings.json`: PreToolUse/PostToolUse hooks and ask rules | Interactive default/acceptEdits; no automatic approvals |
| Codex | `~/.codex/hooks.json` and `~/.codex/rules/ccneed.rules` | `on-request`, `approvals_reviewer = "user"`; hooks and rules enabled |
| WorkBuddy desktop | `~/.workbuddy/settings.json`: hooks, ask rules and `sandbox.commandSafety.ask` | Default permission; paid collection through native Bash; allow once |

Restart each selected host and review/trust its new hooks. A plan command must display the research requirement, platform keywords/quotas, credit estimate and balance. Starting the same prepared plan must show a native permission prompt; reject it during acceptance testing and verify no collection starts. Unit tests and files on disk do not prove this UI acceptance passed.

WorkBuddy collection requires its native Bash tool (Git Bash on Windows). The installer preserves the sandbox and existing safety rules. Its current PowerShell sandbox wrapper does not reliably match collection prefixes, so paid PowerShell calls are blocked with instructions to use Bash in the same host. Choose **allow once** each time. Choosing a session-wide approval actively disables repeated confirmation for that prefix; start a new host session to restore it.

Use a standalone `ccneed kb start ...` command. Do not wrap it in another script or hide plan output through shell redirection. Codex needs absolute plan/answer paths because current hooks do not receive an overridden command working directory. Its PreToolUse hook cannot return `ask`; the installed native command policy supplies the prompt. Commands are rejected in known bypass/never/auto-review modes. Disabling hooks/rules or overriding approval settings invalidates this supported configuration.

Host protocols were checked against official [Claude hooks](https://code.claude.com/docs/en/hooks), [Codex hooks](https://learn.chatgpt.com/docs/hooks) and [rules](https://learn.chatgpt.com/docs/agent-configuration/rules), plus the installed WorkBuddy 5.3.8 desktop runtime. WorkBuddy desktop [permission modes](https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Permission-Modes) distinguish default from full access. Actual per-host native prompt acceptance still needs to be recorded on the deployed revisions.

## Existing files and update failures

- The installer never overwrites a custom Skill or unrelated link at the same path. If an official Skill was locally modified, preserve the customization elsewhere before upgrading.
- Missing links fail explicitly; the installer does not leave behind a copied Skill that will silently go stale. It changes only explicitly selected hosts, not every host in an npm `postinstall` hook.
- Failed network, checksum, or link validation must not be reported as success. A link-stage failure rolls back only links created during that attempt; npm package replacement is not promised to be transactional.
- A changed npm prefix or Node installation may change the global package path. The installer stops on a conflicting old link instead of rewriting an unknown location.
- The installed package's Skill is npm-managed. Store custom instructions under a different Skill name instead of editing the installed one.

## Verify and authorize

```bash
ccneed version --no-update-check
ccneed --help
ccneed auth login
# Show the returned website link and confirmation code to the user.
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
```

Verify the CLI version, PATH, and actual Skill loading in the selected host. `restart_required` means the host needs a reload; `auth: not_started` means login has not happened. Never approve website access on the user's behalf, read browser cookies, or print secrets.

By default the CLI uses production at `https://ccneed.ai`; ordinary users do not need `--api-url`. Only authorized developers may use that flag with a staging test account, passing the same origin on every command. Remote HTTP additionally requires `--allow-insecure-http` and exposes credentials over plaintext; never use it for production. The server must independently enforce staging access. See [Contributing](CONTRIBUTING.md).

## Build from source (developers)

In a new, clean directory with Git and Go 1.25+:

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
```

Put `go env GOBIN` on PATH, or GOPATH's `bin` when GOBIN is empty. Record the checked-out revision. This builds the development CLI only: `go install` does not install or update the companion Skill. If linking it manually, keep the source Skill at the same revision. Use npm for coordinated CLI + Skill updates. A source build is not an official npm release.

You can inspect installer help locally with `node scripts/npm/run.mjs install --help`.

Official host references: [Codex Skills](https://developers.openai.com/codex/skills) · [WorkBuddy settings](https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Setting) · [Claude Code Skills](https://code.claude.com/docs/en/skills). A separate client-imported Skill is outside npm-linked updates.
