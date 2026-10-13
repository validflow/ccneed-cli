# Code style

English · [中文](CODE_STYLE.zh.md)

Follow the repository's modularity, Google-style documentation, real-behavior testing, and clean-code rules. For Go syntax, use [Google Go Style](https://google.github.io/styleguide/go/) and `gofmt`; do not copy Python's inheritance or formatting patterns mechanically.

## Modules and dependencies

- Give each package one responsibility. Avoid catch-all `utils`, universal services, and oversized command files.
- Keep dependencies inward-facing: process entry point → commands → API/resource capabilities. Lower layers must not import the command layer or Club/`usight_agent` runtime.
- Inject external I/O boundaries such as HTTP clients, output writers, credential stores, and clocks when needed. Define small interfaces at actual consumers; do not create a base interface for every type.
- Keep login, HTTP, terminal output, and task state handling separate. Lint rejects cyclomatic complexity above 10; review also checks file size and responsibility.
- Do not add a background daemon, implicit global state, or automatic fallback between environments.

## Documentation contract

Every production function, method, and type needs an adjacent comment beginning with its symbol name. Explain non-obvious test helpers; test functions need not repeat their signatures. Code and comments must not contain emoji, issue numbers, or historical migration markers. Existing code comments are primarily in Chinese, with symbol names and technical terms kept in English.

Use Google-style sections: `Args:` when there are parameters, `Returns:` when there is a return value, and `Errors:` when returning `error`. Describe each named parameter's meaning, constraints, and units. AST checks enforce the structure; review checks whether the content matches behavior. For example:

```go
// Submit sends a task to the authenticated service without waiting for cloud completion.
//
// Args:
//   ctx: Cancels the HTTP request; cancellation does not stop the cloud task.
//   request: Validated prompt, resource IDs, and idempotency request ID.
//
// Returns:
//   The created turn, or a submission error.
//
// Errors:
//   Returns authentication, validation, or network errors; does not blindly retry.
func (c *Client) Submit(ctx context.Context, request SubmitRequest) (Turn, error)
```

The example shows comment structure, not an implemented API. Focus on side effects, ownership, idempotency, timeouts, and cancellation rather than narrating obvious lines. Node.js ES-module release and installation scripts also document exported production functions; invoke external processes with argument arrays, never assembled shell commands.

## Errors, concurrency, and output

- Propagate errors explicitly and wrap with `%w`. The command layer chooses display and exit codes; lower layers must not call `os.Exit`.
- Pass `context.Context` from callers and bound all network operations. A local timeout must not be reported as cancellation of a cloud task.
- Do not blindly retry non-idempotent writes. The server validates ownership and idempotency for task submissions and HITL replies.
- Write JSON results to stdout and diagnostics to stderr. Never print tokens, complete authentication responses, or sensitive headers.
- Stream downloads, validate artifact ownership and output paths, and never silently overwrite existing files.
- Keep credentials isolated by environment; website cookies and CLI credentials are not the same files.

## Tests and checks

Use Go's `testing` and `httptest` with table-driven tests. Cover success, boundaries, errors, and side effects. Prefer real cross-module objects and replace only external network boundaries. Do not use fixed sleeps for concurrency tests or weaken assertions to get a green build.

`gofmt` controls formatting. `golangci-lint` checks errcheck, govet, staticcheck, revive, ineffassign, unused, gocyclo, and nolintlint. Avoid blanket `nolint`; any targeted exception needs a reason.
