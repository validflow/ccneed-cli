# Install CCNeed

English · [中文](../zh/INSTALL.md)

The global CLI runs commands. Native plugins supply a shared Skill and collection hooks for Claude Code, Codex, WorkBuddy and Kimi Code.

## Install or update

Requires Node.js 22.14+, npm and the selected host. After reviewing the official source and obtaining permission to install:

```bash
npx @validflow/ccneed-cli@latest install --agent claude
# Or: --agent codex, --agent workbuddy, --agent claude,codex,workbuddy
```

The installer installs the exact global CLI version and coordinates native registration only for named hosts. Include every host you want to update. It does not start login or cloud tasks. Keep npm's global command directory on the Agent's PATH; local installation does not install commands inside a remote sandbox.

| Host | Native registration | Additional approval configuration |
| --- | --- | --- |
| Claude Code | Claude marketplace/plugin commands; verified user-scope version and enabled state | Plugin PreToolUse requests native `ask` |
| Codex | Codex marketplace/plugin commands; verified installed version and enabled state | `~/.codex/rules/ccneed.rules` prompts for paid prefixes |
| WorkBuddy desktop | Complete the native marketplace/plugin steps returned by the installer | `~/.workbuddy/settings.json`: compatible ask rules and `sandbox.commandSafety.ask` |
| Kimi Code | `--agent kimi` returns native `/plugins` import steps | Follows the user's native permission mode and session authorization |

Claude Code, Codex and WorkBuddy use the local marketplace `ccneed` and plugin `ccneed@ccneed`; Kimi uses manifest ID `ccneed`. Native managers load the shared Skill and hooks without a second global Skill or user-level hook registration. Rerun the installer and complete each host's update steps; `npm install -g` alone does not refresh native caches.

Check `plugins[]` in the JSON result. `installed` means the native manager reported the requested version enabled; hooks still require trust and a running session may need reloading. `action_required` or `failed` means integration is incomplete. WorkBuddy desktop currently has no verified public CLI registration command, so it remains pending until native desktop installation is completed. CodeBuddy CLI registration does not establish WorkBuddy desktop integration.

Restart the host, review/trust hooks and confirm the Skill loads. Installation trust is separate from permission to start each collection. Use native plugin controls for enable, disable and removal. Removing a plugin does not uninstall the global CLI or automatically remove extra Codex/WB permission rules. Retain those conservative rules until deliberately removing the corresponding CCNeed entries; preserve unrelated rules.

If downloads require a proxy, set `HTTPS_PROXY` or `HTTP_PROXY`; the downloader also honors `NO_PROXY` and lowercase forms. Use an exact version instead of `@latest` for reproducibility. RC releases use `@next`.

## Kimi Code

Run `npx @validflow/ccneed-cli@latest install --agent kimi`, then follow the returned `/plugins install "PACKAGE_PATH"` instructions in **Kimi Code CLI 2.5.3** or a compatible newer native release. Enable `ccneed`, verify the requested version, and `/reload` or start a new session. Repeat the import after every update: the native manager copies the plugin. The installer leaves Kimi profiles, `KIMI_CODE_HOME` and user configuration untouched and reports `action_required` for this manual setup.

The bundled manifest follows the [documented plugin format](https://www.kimi.com/code/docs/en/kimi-code-cli/customization/plugins). Native 2.5.3 was tested in an isolated profile: installation, enable/disable and update to 0.1.12 loaded one Skill and three hooks without diagnostics. The deprecated Python `kimi-cli` 1.x is a different product and cannot load this bundle; use the [official native installer](https://www.kimi.com/code/docs/en/kimi-code-cli/guides/getting-started).

Collection and HITL execution follow Kimi's native permission policy. Manual/Ask When Needed modes prompt as configured; automatic mode and session authorization remain the user's choice. CCNeed does not force another prompt or reject those modes. Hooks still reject invalid, expired, changed or unaffordable plans.

Kimi loads instructions through `sessionStart.skill`; SessionStart/PostToolUse do not inject returned hook text. The Agent presents the full CLI result and saved JSON. For `kb plan` and HITL `tasks get/watch`, use a new absolute `--output` file so hooks can read the complete plan when Kimi truncates tool output to 2000 characters. Existing files are never overwritten. Hooks use local files; remote Bash output paths are not supported.

## Collection confirmation

1. The Agent prepares a plan; CCNeed validates it and compares estimated credits with balance.
2. The Agent reads the complete plan through PostToolUse context in Claude Code/Codex/WorkBuddy, or CLI output and the saved snapshot in Kimi. Each confirmation presents the requirement, platform entity/other keywords, quota, total, credits, balance and adjustments. Both keyword groups are editable; retain a nonblank entity per platform, revalidate changes and show the full result. Internal T1/T2/T3 allocation stays out of the summary.
3. PreToolUse checks that exact plan; the Agent gives a short start reminder without repeating it. Native permissions govern execution, including Kimi's existing automatic/session authorization. Changed or expired plans require preparation and display again. Conversation HITL uses the same business checks.
4. After collection finishes, the Agent normally exports CSV and delivers the local file. User-requested formats, destinations and analysis-only workflows take priority.

| Host | Supported collection mode |
| --- | --- |
| Claude Code | Interactive default/acceptEdits, manual approvals |
| Codex | `on-request`, `approvals_reviewer = "user"`, hooks/rules enabled |
| WorkBuddy desktop | Default permission, native Bash, **allow once** |
| Kimi Code | User-selected native permission mode, including automatic/session authorization |

Codex PreToolUse cannot return `ask`; native command rules supply confirmation. WorkBuddy's current PowerShell sandbox wrapper does not reliably match paid prefixes, so paid PowerShell calls are blocked. Use Bash, including Git Bash on Windows. Session-wide approval suppresses later confirmations for that prefix; open a fresh session to restore them. Known bypass/never/automatic-review modes are rejected.

Use a standalone `ccneed kb start ...`, without script wrappers, composite commands or redirects. Codex and Kimi plan/answer paths must be absolute. Keep business hooks enabled; Claude Code, Codex and WorkBuddy also require their documented manual-approval configuration. Kimi follows its native permission policy.

For acceptance, prepare a plan and verify its display; deny the native start prompt and confirm no collection starts, then approve a fresh attempt. Repeat for conversation HITL. Unit tests and files on disk do not establish desktop UI acceptance.

## Existing files and failures

The installer preserves unrelated settings and custom Skills. It migrates only recognized installer-owned legacy Skill links/hooks; unknown or edited files require manual resolution. A modified npm-managed Skill blocks replacement until its customization is preserved elsewhere. Keep custom instructions under another Skill name.

Native failures are reported per host after CLI installation, never as overall success. Fix the reported cause and retry. npm replacement and native managers are not one transaction. Conflicting same-name marketplaces are not redirected. Ordinary npm `postinstall` does not modify host configuration.

Non-default `CODEX_HOME` or `CLAUDE_CONFIG_DIR` is reported as requiring manual profile setup; the installer does not register plugins or write approval rules into a different profile. Use the default profile for coordinated installation, or configure and verify the native plugin and its permission rules together in your chosen profile.

## Verify and authorize

```bash
ccneed version --no-update-check
ccneed --help
ccneed auth login
# Show the website link and confirmation code; the user approves in the browser.
ccneed auth login --complete --timeout 60s
ccneed auth status --verify
```

`auth: not_started` means login has not happened. Never approve website access for the user, read browser cookies or print secrets. Commands default to production `https://ccneed.ai`; only authorized developers use `--api-url` with isolated staging accounts. See [Contributing](../../CONTRIBUTING.md).

## Build from source

With Git and Go 1.25+:

```bash
git clone https://github.com/validflow/ccneed-cli.git
cd ccneed-cli
go install ./cmd/ccneed
```

Put `go env GOBIN` on PATH, or GOPATH's `bin` when empty. This builds the development command only; it does not register native plugins. Record the revision and keep plugin source at that revision. Offline installer help: `node scripts/npm/run.mjs install --help`.

References: [Claude plugins](https://code.claude.com/docs/en/plugins-reference), [Codex plugin hooks](https://learn.chatgpt.com/docs/hooks#plugin-bundled-hooks), [Codex rules](https://learn.chatgpt.com/docs/agent-configuration/rules), [WorkBuddy permissions](https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Permission-Modes). Protocol checks used Claude Code 2.1.193, Codex 0.165.0 and WorkBuddy desktop 5.3.8; host updates still require acceptance testing.
