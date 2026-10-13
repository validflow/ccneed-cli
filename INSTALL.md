# Install CCNeed CLI and Skill

English · [中文](INSTALL.zh.md)

The CLI is installed globally; the companion Skill is linked into the selected Agent's current-user directory. Both use the same package version across projects. Website login still requires the user's explicit approval.

> The official npm package is published. Use `@latest` for the current stable version. Some cloud commands still depend on their matching production service APIs.

## Install or update

After checking the official source and asking the user to approve installation, choose the Agent host:

```bash
npx @validflow/ccneed-cli@latest install --agent codex
# Or: --agent workbuddy, --agent claude, or --agent codex,workbuddy
```

Requires Node.js 22.14+ and npm. The installer places `ccneed` in npm's global command directory, which must be on the Agent's PATH. A local install does not automatically make it available inside a remote sandbox. The installer does not read CCNeed account data, create tasks, or change Agent security settings.

`@latest` is npm's current stable release; replace it with an exact version if reproducibility matters. The installer globally installs that exact CLI and Skill version. Selected hosts receive links to the one installed Skill directory:

| Host | Current-user Skill path |
| --- | --- |
| Codex | `~/.agents/skills/ccneed` |
| WorkBuddy local coding environment | `~/.codebuddy/skills/ccneed` |
| Claude Code | `~/.claude/skills/ccneed` |

These are directory links, not copies: Windows uses junctions; other systems use symlinks. Updating the global package updates already-linked hosts too, even if the update command names only one host. Restart or reload the host afterward; an existing Agent conversation may retain the old Skill in memory.

The package includes the Chinese `SKILL.md` and its four reference files. The release and installer verify the whole Skill checksum, as well as the CLI binary version and checksum. A tag, binary, or directory link alone is not proof of a complete release or successful host loading.

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

Official host references: [Codex Skills](https://developers.openai.com/codex/skills) · [WorkBuddy settings](https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Setting) · [CodeBuddy directories](https://www.codebuddy.cn/docs/cli/codebuddy-dir) · [Claude Code Skills](https://code.claude.com/docs/en/skills). WorkBuddy's non-local Skill import is separate from npm-linked updates; actual WorkBuddy client loading still requires validation.
