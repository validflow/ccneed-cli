#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# 固定工具版本；仅使用 Go 模块缓存，不修改本仓依赖。
lint_version=v2.14.0
unformatted=$(find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' -print0 | xargs -0 gofmt -l)
if [[ -n "$unformatted" ]]; then
  printf 'Run make format. Unformatted files:\n%s\n' "$unformatted" >&2
  exit 1
fi

go mod tidy -diff
go run ./tools/checkdocs
go vet ./...
go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${lint_version}" run
go test -count=1 -cover ./...
go build ./...
