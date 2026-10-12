# CX Review operator setup

CCNeed CLI uses the same CX Review execution pattern as `usight_agent`.

1. Give `validflow/ccneed-cli` access to the existing organization runner pool labelled `[self-hosted, macOS, ARM64, cc-review]`.
2. Ensure that runner has `codex` and `gh`. The workflow uses `gh` only in a separate read-only step to fetch the PR diff; the Codex process receives no GitHub token. An existing `codex login` session is sufficient; `CODEX_ACCESS_TOKEN` or `OPENAI_API_KEY` repository secrets are optional alternatives.
3. Same-repository PRs trigger `CX Review` on open, update, reopen, or ready-for-review. The trusted base revision supplies the workflow, code and review Skill; PR changes are fetched as untrusted diff data and never checked out or executed on the credentialed runner. A GitHub-hosted posting job comments the original report on the PR. Fork PRs are skipped; do not grant an untrusted fork access to the credentialed runner.
4. The required check is the terminal `AI Code Review` job, not `Review execution` or a custom commit status. It fails if review execution fails, the report is missing, or posting fails; findings in the report require maintainer judgment rather than format-based CI parsing. Verify a small same-repository PR and its review comment before making the job required in main branch protection. Keep `quality` required separately. A skipped fork job is not evidence of review; external contributions need a separate review policy before public release.

No `CX_REVIEW_READY`, `CX_AUTH_FILE`, or manual `workflow_dispatch` setup is required. Because this workflow uses `pull_request_target`, a PR that changes the workflow still runs the version on the base branch. Review the bootstrap PR manually; verify the new workflow on a subsequent same-repository PR before making its check required.

Run `npm test` and `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 -config-file scripts/cx-review/actionlint.yml` locally to validate scripts and workflow syntax. A real runner invocation is still needed to verify Codex login and GitHub comment permissions.
