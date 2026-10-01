# Local toolkit development commands.

MODULES = . examples exp/cmd/mcp exp/mcp9p exp/cmd/mcptrace-to-otel

.PHONY: all build test test-race vet fmt gates check-deps

all: build

build:
	@set -e; for module in $(MODULES); do (cd "$$module" && GOWORK=off go build ./...); done

test:
	@set -e; for module in $(MODULES); do (cd "$$module" && GOWORK=off go test ./...); done

test-race:
	@set -e; for module in $(MODULES); do (cd "$$module" && GOWORK=off go test -race ./...); done

vet:
	@set -e; for module in $(MODULES); do (cd "$$module" && GOWORK=off go vet ./...); done

fmt:
	@git ls-files -z '*.go' | xargs -0 gofmt -s -w

gates: build vet test-race check-deps
	@test -z "$$(git ls-files '*.go' | xargs gofmt -s -l)"
	staticcheck ./...

check-deps:
	@set -e; for module in $(MODULES); do (cd "$$module" && GOWORK=off go mod verify); done
	@GOWORK=off go list -m -f '{{.Path}}' all | awk '/^github\.com\/charmbracelet\// { found = 1 } END { exit found }'
