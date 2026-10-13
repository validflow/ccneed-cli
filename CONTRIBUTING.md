# Contributing

English · [中文](CONTRIBUTING.zh.md)

## Workflow

We maintain one `main` branch, not `develop`. Work on a short-lived branch, open a PR, pass the required `quality` check, obtain maintainer review, then **Squash and merge**. Merging ordinary code changes does not publish a release.

```bash
git switch main
git pull --ff-only
git switch -c feat/short-description
make check
```

Start a fresh branch from updated `main` after each Squash merge; do not reuse an already-merged branch. Read the [code style guide](CODE_STYLE.md) before changing code. Add a test that fails for a new behavior or bug before implementing the fix.

## Branches, PRs, and commits

Naming is guidance, not an enforced GitHub or CI restriction. Maintainers tidy the final PR title before merging.

| Item | Suggested format | Example |
| --- | --- | --- |
| Branch | `type/short-description` | `feat/browser-login`, `fix/report-download` |
| PR title | `type(scope): description` | `feat(auth): add browser login` |
| Branch commits | Same pattern, but interim commits are fine | `fix(files): prevent download overwrite` |

Use readable branch names; Agent-generated names are acceptable. Suggested types: `feat`, `fix`, `docs`, `ci`, `test`, `refactor`, `perf`, `build`, `chore`. Scope is optional. Explain what changed, why, and how it was verified. Use `!` for a breaking change and describe migration. Internal branches may include a Multica issue ID; external contributors do not need one. Link relevant GitHub issues in the PR description, not product code or comments.

The Squash commit normally uses the reviewed PR title. Version numbers are chosen by maintainers, not inferred from titles or branch names. Skill behavior changes use `feat(skill)` or `fix(skill)`; copy-only changes use `docs(skill)`.

## Repository structure and Skill

`cmd/ccneed` contains only process entry. Command parsing belongs in `internal/cli`; API, credentials, output, and downloads are separate modules added as needed. Development tools in `tools/` are not CLI runtime dependencies.

The host Skill is `skills/ccneed/SKILL.md` with Chinese guidance and references; it has no separate English translation. The npm installer links the same global package Skill into each selected host, rather than copying independent versions. It must not overwrite a custom Skill; a modified managed Skill blocks upgrade until resolved. Check actual host loading, not just link existence.

## Local tests before staging integration

Validate CLI and Club locally first. Then deploy the relevant Club branch to staging, point the local CLI build at that deployment, and record both deployed revisions and results. A CLI PR does not deploy Club; full production functionality requires Club deployment and controlled production smoke testing.

- Staging and production (`sg-prod`) must use separate data, storage, queues, task execution, credit ledgers, and credentials.
- The CLI defaults to production `https://ccneed.ai`. Ordinary users should not set `--api-url`. Authorized developers may explicitly select a staging origin with a staging test account. The flag does not grant staging access; the server must enforce it. The CLI accepts HTTPS or loopback HTTP by default. A remote HTTP preview requires `--allow-insecure-http` on **each** command and must be a trusted preview IP; credentials travel in plaintext, so never use this with production.
- API, website authorization, callback, and conversation/report links must stay in the same environment. Credentials are isolated by trusted origin; the server must reject cross-environment credentials.
- Record the Club revision **actually deployed**, CLI revision, origin, fixture IDs, results, and cleanup notes—never tokens. Reuse a healthy compatible staging deployment for CLI-only changes; redeploy after backend changes or preview expiration. The current preview lifetime is four hours.
- Local HTTP test doubles cover contracts and errors, not the real staging login, resources, task, HITL, and report workflow.
- Do not change Club's separate develop → release/hotfix → main → tag process. CLI feature merges do not deploy that service or publish npm.

An integration pass starts with local Go checks (`npm test` too when changing npm/release/CI), followed by a staging Club deployment and isolation check. Build the CLI locally; pass the deployment origin with `--api-url`, not an endpoint path. Verify website login and credits, then knowledge bases, files, tasks, HITL, and downloads. A deployment instance alone does not prove database isolation. Do not claim production readiness from local contract tests.

## Quality gate

`make check` and `bash scripts/check.sh` run the same non-mutating checks: `gofmt`, dependency consistency, Google-style comment structure, `go vet`, pinned `golangci-lint`, tests, and build. Linux CI also runs race tests; CI tests and builds on Linux, macOS, and Windows. For npm, release, or CI changes, run `npm test` with Node.js 22.14+.

`main` is protected against deletion and force pushes and accepts PR Squash merges only. `quality` is required. There is no mandatory approval count; automated CX Review is paused, so it is **not** a passing gate. External users may open PRs, but only repository members with Write/Maintain/Admin permissions may merge them. Maintainers still review code, dependencies, and permissions. We do not impose an arbitrary coverage percentage; tests must detect real regressions.

## PR and release

Use the PR template to describe scope, evidence, risk, compatibility, and any maintainer prerequisites. Update README, installation guidance, and Skill when behavior changes. PR checks run without production credentials. The CX Review workflow and instructions remain in the repository but its job is disabled; do not run an external PR on a credential-bearing self-hosted runner.

CLI versions use independent `vX.Y.Z` tags. npm, `ccneed version`, binary assets, and the bundled Skill share one release version. RCs use `vX.Y.Z-rc.N` and npm `next`; they do not enter `latest` or ordinary update notices. Only an explicit version-change PR merged into `main` triggers automated publishing; ordinary merges do not. Before a fully usable public release, confirm Club production compatibility and a controlled login/task smoke test, or clearly describe remaining server limitations. See [Releasing](RELEASING.md).
