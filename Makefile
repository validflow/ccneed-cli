.PHONY: check format test build

check:
	bash scripts/check.sh

format:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -not -path './.git/*')

test:
	go test -count=1 ./...

build:
	go build ./...
