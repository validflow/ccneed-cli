# CX Review operator setup

No remote settings or runner permissions are changed by these files. Before calling CX Review active:

1. Merge the initial trusted review controls into `main` after manual four-role review. `pull_request_target` intentionally cannot bootstrap controls from its own untrusted PR branch.
2. Grant this repository access to a restricted organization `cc-review` pool with `[self-hosted, macOS, ARM64, cc-review]`. Use ephemeral single-job machines destroyed after completion, not a reused developer machine or a publicly available persistent credentialed runner. Preparing/reviewing PR content never executes PR source, including after maintainer approval.
3. Preinstall a current Codex CLI supporting `exec --strict-config --ignore-user-config --ignore-rules --disable shell_tool --output-schema`, and authenticate its dedicated review identity. No personal host MCP/plugins/hooks or production account should exist on this image. Optional `CX_CODEX_BINARY` and `CX_AUTH_FILE` repository variables are trusted operator-set paths; they are not credentials. Leave empty for `codex` and `$HOME/.codex/auth.json` defaults.
4. Set repository variables `CX_REVIEW_READY=true` and `CX_REVIEW_EPHEMERAL=true` only after the runner/auth/tool-disable contract is verified. Without both, admitted reviews fail the required status rather than silently pass. No API secret or GitHub publishing token is given to model tools.
5. Create the main ruleset requiring `quality` and the **commit status context `cx-review`** from GitHub Actions, up-to-date main, PR review and squash-only merges. There is deliberately no job named `cx-review`: a skipped fork job must never satisfy the required review. Restrict workflow/control-file edits to maintainer review (CODEOWNERS can be added after choosing an actual owner/team).

## Internal and public contributions

- An internal PR runs automatically only for a same-repository head and an author with current write/maintain/admin access plus OWNER/MEMBER/COLLABORATOR association. Every synchronize marks the new head's status pending.
- A fork/untrusted-author PR stays **pending** without entering the credentialed pool. A maintainer with write/maintain/admin permission dispatches **CX Review from main**, supplying PR number and exact current 40-character head SHA. Labels are not authorization. New commits invalidate that approval; stale dispatch is rejected.
- Publication rechecks head and base SHA before writes and again before publishing the final status. Changed revisions require a fresh review. Require branches to be up to date because a commit status itself is head-bound, not base-bound after the fact.
- A skipped/unconfigured runner, model failure, missing/malformed JSON, missing pass/verifier, missing/truncated/binary patch or over-768-KiB total context cannot produce green. Split oversized PRs; binary changes require a separate explicit maintainer review/exception process, never an automatic fake clean result.
- Quality tests execute contributor code only on GitHub-hosted unprivileged machines, with read-only contents token, no persisted checkout credentials, no production secrets and no shared dependency cache. Linux runs deterministic quality/race/workflow validation; Linux/macOS/Windows test and build.

## Local verification

```bash
node --test scripts/cx-review/*.test.mjs
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 -config-file scripts/cx-review/actionlint.yml
```

The helper tests exercise authorization, exact-SHA approvals, stale publication, independent-pass completion, verification and tool-free environment boundaries without calling paid models or writing to GitHub. Real runner review, fork approval and required-status enforcement still need a controlled GitHub integration test after configuration.
