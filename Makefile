# MCP Go Implementation - Development Makefile

.PHONY: test test-synctest test-race test-coverage build build-tools fmt vet lint pre-commit clean help

# Default target
all: test

# Test targets
test:
	go test ./...

test-synctest:
	GOEXPERIMENT=synctest go test -tags=synctest ./...

test-race:
	go test -race -timeout=10m ./...

test-coverage:
	mkdir -p coverage
	go test -coverprofile=coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Build targets
build:
	go build ./...

build-tools:
	@echo "Building core tools..."
	@for tool in cmd/*; do \
		if [ -d "$$tool" ] && [ -f "$$tool/main.go" ]; then \
			echo "Building $$(basename $$tool)..."; \
			go build "$$tool"; \
		fi \
	done
	@if [ -d "exp/cmd" ]; then \
		echo "Building experimental tools..."; \
		cd exp && go build -tags=k8s ./cmd/... 2>/dev/null || echo "Some experimental tools failed to build (non-critical)"; \
	fi

# Quality targets
fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint:
	golangci-lint run

# Development workflow
pre-commit:
	./scripts/pre-commit.sh

tidy:
	go mod tidy
	@if [ -d "exp" ]; then cd exp && go mod tidy; fi

# CI/CD simulation
ci-local: fmt vet build test test-race
	@echo "✅ Local CI checks passed"

# Docker targets
docker-build:
	docker build -t mcp-go .

docker-run:
	docker run --rm mcp-go

# Cleanup
clean:
	go clean ./...
	rm -rf coverage/
	rm -f mcp-*
	@if [ -d "exp" ]; then cd exp && go clean ./...; fi

# Help
help:
	@echo "Available targets:"
	@echo "  test           - Run all tests"
	@echo "  test-synctest  - Run tests with synctest"
	@echo "  test-race      - Run tests with race detector"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  build          - Build all packages"
	@echo "  build-tools    - Build all tools"
	@echo "  fmt            - Format code with gofmt"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  pre-commit     - Run pre-commit checks"
	@echo "  tidy           - Run go mod tidy"
	@echo "  ci-local       - Simulate CI pipeline locally"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Show this help"
