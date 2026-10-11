# CX Review operator setup

CCNeed CLI uses the same CX Review execution pattern as `usight_agent`.

1. Give `validflow/ccneed-cli` access to the existing organization runner pool labelled `[self-hosted, macOS, ARM64, cc-review]`.
2. Ensure that runner has `codex` and `gh`. An existing `codex login` session is sufficient; `CODEX_ACCESS_TOKEN` or `OPENAI_API_KEY` repository secrets are optional alternatives, as in `usight_agent`.
3. PRs from branches in this repository trigger `CX Review` automatically on open, update, reopen, or ready-for-review. The workflow checks out the merge ref, runs `$review-pipeline` using `codex exec`, and posts the report as a PR comment. Fork PRs are skipped, matching `usight_agent`; do not grant an untrusted fork access to the credentialed runner.
4. The review check is the `AI Code Review` job, not a custom `cx-review` commit status. Verify a small same-repository PR and its review comment before making the job required in main branch protection. Keep `quality` required separately. A skipped fork job is not evidence of review; external contributions need a separate review policy before public release.

No `CX_REVIEW_READY`, `CX_AUTH_FILE`, or manual `workflow_dispatch` setup is required. The first PR adding this workflow cannot be reviewed automatically by the new workflow until the workflow reaches `main`; review that bootstrap PR manually.

Run `npm test` and `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 -config-file scripts/cx-review/actionlint.yml` locally to validate scripts and workflow syntax. A real runner invocation is still needed to verify Codex login and GitHub comment permissions.
