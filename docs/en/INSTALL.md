# Install CCNeed

English · [中文](../zh/INSTALL.md)

The global CLI runs commands. Native plugins and extensions load one shared Skill and plan context through host events.

## Install or update

Requires Node.js 22.14+, npm and the selected host. After reviewing the official source and obtaining permission to install:

```bash
npx @validflow/ccneed-cli@latest install --agent codex
```

The installer installs the exact global CLI version and coordinates native registration only for named hosts. Include every host you want to update. It does not start login or cloud tasks. Keep npm's global command directory on the Agent's PATH; local installation does not install commands inside a remote sandbox.

| Host | Agent ID | Native registration |
| --- | --- | --- |
| Claude Code | `claude` | Claude marketplace/plugin commands; verified user-scope version and enabled state |
| Codex | `codex` | Codex marketplace/plugin commands; verified installed version and enabled state |
| WorkBuddy desktop | `workbuddy` | Complete the native marketplace/plugin steps returned by the installer |
| Kimi Code | `kimi` | Native `/plugins` import steps |
| OpenClaw | `openclaw` | Native linked plugin |
| Pi | `pi` | Native Pi package and extension |
| ZCode | `zcode` | Local marketplace in the desktop plugin manager |
| Qoder | `qoder` | Native local plugin |
| Trae CLI 2.0 | `trae` | Dedicated native plugin |
| DeepSeek Harness | `dsh` | Native Cordis bundle |

Claude Code, Codex and WorkBuddy use the local marketplace `ccneed` and plugin `ccneed@ccneed`; Kimi uses manifest ID `ccneed`. Native managers load the shared Skill and hooks without a second global Skill or user-level hook registration. Rerun the installer and complete each host's update steps; `npm install -g` alone does not refresh native caches.

Check `plugins[]` in the JSON result. `installed` means the native manager reported the requested version enabled; hooks still require trust and a running session may need reloading. `action_required` or `failed` means integration is incomplete. WorkBuddy desktop currently has no verified public CLI registration command, so it remains pending until native desktop installation is completed. CodeBuddy CLI registration does not establish WorkBuddy desktop integration.

Additional hosts return `action_required` with native manager commands or in-app import steps. The installer prepares the whole matching plugin payload; it does not edit their profiles or claim they are enabled. Complete those steps in the intended host/profile, verify the version and reload. Select multiple hosts with commas, for example `--agent codex,pi`.

Restart the host, review/trust hooks and confirm the Skill loads. Use native plugin controls for enable, disable and removal. Removing a plugin does not uninstall the global CLI or automatically remove extra Codex/WB permission rules. Retain those conservative rules until deliberately removing the corresponding CCNeed entries; preserve unrelated rules.

If downloads require a proxy, set `HTTPS_PROXY` or `HTTP_PROXY`; the downloader also honors `NO_PROXY` and lowercase forms. Use an exact version instead of `@latest` for reproducibility. RC releases use `@next`.

## Kimi Code

Run `npx @validflow/ccneed-cli@latest install --agent kimi`, then follow the returned `/plugins install "PACKAGE_PATH"` instructions in **Kimi Code CLI 2.5.3** or a compatible newer native release. Enable `ccneed`, verify the requested version, and `/reload` or start a new session. Repeat the import after every update: the native manager copies the plugin. The installer leaves Kimi profiles, `KIMI_CODE_HOME` and user configuration untouched and reports `action_required` for this manual setup.

The bundled manifest follows the [documented plugin format](https://www.kimi.com/code/docs/en/kimi-code-cli/customization/plugins). Native 2.5.3 was tested in an isolated profile: installation, enable/disable and update to 0.1.12 loaded one Skill and three hooks without diagnostics. The deprecated Python `kimi-cli` 1.x is a different product and cannot load this bundle; use the [official native installer](https://www.kimi.com/code/docs/en/kimi-code-cli/guides/getting-started).

Kimi loads instructions through `sessionStart.skill`; SessionStart/PostToolUse do not inject returned hook text. The Agent presents the full CLI result and saved JSON. For `kb plan` and HITL `tasks get/watch`, use a new absolute `--output` file so the Agent can read complete results when tool output is truncated. Existing files are never overwritten.

## Additional host requirements

- **OpenClaw and Pi:** load CCNeed through the native plugin manager.
- **ZCode/Qoder:** load the full native plugin with its explicit host hook file. Hooks use native argv execution, preserving spaces and Unicode paths. Complete the returned marketplace/local-plugin steps; desktop UI acceptance remains separate from protocol tests.
- **Trae CLI 2.0:** use native 0.215.0 or a compatible release. Install `traecli plugin install "PACKAGE_PATH/plugins/trae" --name ccneed`, then check `traecli plugin validate ccneed` reports **1 Skill and 2 hooks**. After updating the global CLI, run `traecli plugin upgrade ccneed` and reload the session. The native manager switches the complete active cache version; inactive caches remain under its control. This payload uses Trae CLI's supported `.codex-plugin` format in a separate plugin root, with the same-source Skill and hooks assembled at release time. It does not establish Trae IDE support.
- **DeepSeek Harness:** use 0.2.2-rc.3 or a compatible 0.2.x release with **Node.js 24.9+**. The current npm `latest` tag can point to an older incompatible runtime. Select the profile used by your running host: `dsh plugin --profile <name> add "PACKAGE_PATH"`. Verify with `dsh --profile <name> --dump-config` and Plugins. The bundle registers one shared Skill and result-context handlers. Remove with `dsh plugin --profile <name> remove @validflow/ccneed-cli`.

OpenClaw, Pi and DSH use native extension events rather than shell hook names. Paths must be accessible to the process running the CLI. A truncated result is not a valid displayed plan.

Protocol references: [OpenClaw tool policy](https://docs.openclaw.ai/plugins/hooks/tool-policy), [Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md), [Pi extensions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md), [ZCode plugins](https://zcode.z.ai/cn/docs/plugin), [Qoder plugins](https://docs.qoder.com/cli/plugins), [Trae CLI 2.0](https://docs.trae.cn/cli_get-started-with-trae-cli-2), [DSH bundles](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/user/develop/basic/publish.md).

## Collection confirmation

The agent displays the complete plan and waits for user confirmation. Changes require a fresh validated plan and confirmation. After confirmation it runs `kb start`. The CLI checks the prepared file against the plan token; the server is responsible for validating the signature and expiry. Submit cloud HITL answers according to `resume_schema`.

On upgrade, only an exact legacy installer-generated Codex `ccneed.rules` file is migrated; custom rules are preserved. Review rules in other hosts or modified files yourself. Restart and trust the updated hooks.

## Existing files and failures

The installer preserves unrelated settings and custom Skills. It migrates only recognized installer-owned legacy Skill links/hooks; unknown or edited files require manual resolution. A modified npm-managed Skill blocks replacement until its customization is preserved elsewhere. Keep custom instructions under another Skill name.

Native failures are reported per host after CLI installation, never as overall success. Fix the reported cause and retry. npm replacement and native managers are not one transaction. Conflicting same-name marketplaces are not redirected. Ordinary npm `postinstall` does not modify host configuration.

Installation errors identify the failed stage and a safe error code. A failed native plugin also includes `plugins[].diagnostic`; the CLI may already be updated, so check each host's status before using it.

| Error | What to do |
| --- | --- |
| `EPERM` / `EACCES` | Check the path in the local npm log. The current terminal or Agent session needs write access to that cache or installation directory. Retry in a session with that access. |
| `EBUSY` | A file is occupied. Close the process using it and retry. If the occupied path is in the npm cache, a fresh writable cache can help. |
| `ENOENT` | Check Node.js, npm and the selected host are installed and available on PATH. |
| `MARKETPLACE_UNAVAILABLE` | Run the host's marketplace list command locally and repair the source it reports, which may belong to another plugin. |
| Other failures | Inspect the local npm log or reported host operation; network errors also require checking registry and proxy access. Raw child output is withheld to avoid exposing credentials. |

An npm/npx error **before the CCNeed installer starts** cannot be handled by this diagnostic code. For a cache problem, rerun the same installation command with `npx --cache <new-writable-cache-directory> ...`. Changing the cache does not grant permission to write the global npm directory. The installer does not automatically elevate privileges, delete caches or stop processes.

Non-default `CODEX_HOME` or `CLAUDE_CONFIG_DIR` is reported as requiring manual profile setup; the installer does not register plugins in a different profile. Use the default profile for coordinated installation, or configure and verify the native plugin in your chosen profile.

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
