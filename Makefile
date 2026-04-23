# MCP Go Implementation - Development Makefile

.PHONY: test test-synctest test-race test-coverage build build-tools fmt vet lint pre-commit clean help security-scan security-quick security-baseline

# Default target
all: test

# Test targets
test:
	go test ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off go test ./...; fi

test-synctest:
	GOEXPERIMENT=synctest go test -tags=synctest ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off GOEXPERIMENT=synctest go test -tags=synctest ./...; fi

test-race:
	go test -race -timeout=10m ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off go test -race -timeout=10m ./...; fi

test-coverage:
	mkdir -p coverage
	go test -coverprofile=coverage/root.out -covermode=atomic ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off go test -coverprofile=../../coverage/cmd-mcp.out -covermode=atomic ./...; fi
	@{ head -n1 coverage/root.out; tail -n +2 coverage/root.out; if [ -f coverage/cmd-mcp.out ]; then tail -n +2 coverage/cmd-mcp.out; fi; } > coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Build targets
build:
	go build ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off go build ./...; fi

build-tools:
	@echo "Building core tools..."
	@if [ -d "cmd/mcp" ]; then \
		echo "Building mcp..."; \
		(cd cmd/mcp && GOWORK=off go build ./...); \
	fi
	@for tool in cmd/*; do \
		if [ -d "$$tool" ] && [ -f "$$tool/main.go" ]; then \
			if [ -f "$$tool/go.mod" ]; then \
				continue; \
			fi; \
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
	@excluded_patterns='temp/example_server_design_exploration|temp/mock_client_fix.go|exp/schema2go/generator.go|exp/cmd/mcp-tool-graph/main.go'; \
	files=$$(git ls-files '*.go' | grep -v -E "$$excluded_patterns" || true); \
	if [ -n "$$files" ]; then printf '%s\n' "$$files" | xargs gofmt -s -w; fi

vet:
	go vet ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off go vet ./...; fi

lint:
	golangci-lint run
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off golangci-lint run; fi

# Security targets
security-scan:
	@echo "Running comprehensive security scan..."
	@./scripts/security-scan.sh

security-quick:
	@echo "Running quick security scan (gosec only)..."
	@gosec -severity medium -confidence medium -exclude-generated ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off gosec -severity medium -confidence medium -exclude-generated ./...; fi

security-baseline:
	@echo "Establishing security baseline..."
	@mkdir -p security-reports
	@./scripts/security-scan.sh
	@echo "Baseline reports saved to security-reports/"

# Development workflow
pre-commit:
	./scripts/pre-commit.sh

tidy:
	go mod tidy
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off go mod tidy; fi
	@if [ -d "exp" ]; then cd exp && go mod tidy; fi

# CI/CD simulation
ci-local: fmt vet build test test-race security-quick
	@echo "✅ Local CI checks passed"

# Docker targets
docker-build:
	docker build -t mcp-go .

docker-run:
	docker run --rm mcp-go

# Cleanup
clean:
	go clean ./...
	@if [ -d "cmd/mcp" ]; then cd cmd/mcp && GOWORK=off go clean ./...; fi
	rm -rf coverage/
	rm -rf security-reports/
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
	@echo "  security-scan  - Run comprehensive security scan (gosec + govulncheck)"
	@echo "  security-quick - Run quick security scan (gosec only)"
	@echo "  security-baseline - Establish security baseline reports"
	@echo "  pre-commit     - Run pre-commit checks"
	@echo "  tidy           - Run go mod tidy"
	@echo "  ci-local       - Simulate CI pipeline locally"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Show this help"
