## Purpose and scope

Explain the problem and the change. Link the relevant issue; do not add unrelated identifiers. [中文模板](PULL_REQUEST_TEMPLATE.zh.md)

## Verification

- [ ] `make check` or `bash scripts/check.sh` passed.
- [ ] New behavior has evidence of a failing test that passes after the change.
- [ ] Comments, module boundaries, errors, and output contracts follow `CODE_STYLE.md`.
- [ ] Correctness, architecture, security, and requirement fit were reviewed.
- [ ] Required `quality` passed. If a local CX Review was also run, attach its report and resolve confirmed findings; automated CX Review is currently paused.

For real-service changes, record the deployed Club SHA, CLI SHA, staging origin, and end-to-end result without credentials.

## Risk and release verification

Explain user impact, rollback, and how the result will be verified after release. If there is no runtime impact, say why. Do not present local test-double results as production verification.
