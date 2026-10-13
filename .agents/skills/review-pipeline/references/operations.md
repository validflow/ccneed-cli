# CX Review operator setup

CX Review is currently paused for this public repository: its execution job is disabled and `AI Code Review` is not a required check. `quality` remains required. The workflow and review Skill are retained for local review and a possible future reactivation; the steps below are **not** current PR requirements.

Before reactivation, review the public-repository trust boundary and decide how fork contributions will be reviewed. Then validate the runner and workflow:

1. Give `validflow/ccneed-cli` access to the existing organization runner pool labelled `[self-hosted, macOS, ARM64, cc-review]`.
2. Ensure that runner has `codex` and `gh`. The workflow uses `gh` only in a separate read-only step to fetch the PR diff; the Codex process receives no GitHub token. An existing `codex login` session is sufficient; `CODEX_ACCESS_TOKEN` or `OPENAI_API_KEY` repository secrets are optional alternatives.
3. If the execution job is re-enabled, same-repository PRs can trigger `CX Review` on open, update, reopen, or ready-for-review. The trusted base revision supplies the workflow, code and review Skill; PR changes are fetched as untrusted diff data and never checked out or executed on the credentialed runner. A GitHub-hosted posting job comments the original report on the PR. Fork PRs are skipped; do not grant an untrusted fork access to the credentialed runner.
4. After reactivation and a successful real PR trial, the terminal `AI Code Review` job may be considered for branch protection; it is **not required today**. It must fail if review execution fails, the report is missing, or posting fails; findings require maintainer judgment rather than format-based CI parsing. A skipped fork job is not evidence of review.

No `CX_REVIEW_READY`, `CX_AUTH_FILE`, or manual `workflow_dispatch` setup is required. Because this workflow uses `pull_request_target`, a PR that changes the workflow still runs the version on the base branch. Review any reactivation PR manually; verify the new workflow on a subsequent same-repository PR before making its check required.

Run `npm test` and `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 -config-file scripts/cx-review/actionlint.yml` locally to validate scripts and workflow syntax. A real runner invocation is still needed to verify Codex login and GitHub comment permissions.
