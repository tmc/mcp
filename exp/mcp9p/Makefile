# Makefile for MCP9P - Plan9-inspired namespace tools

.PHONY: all build clean install test

# Build all tools
all: build

build:
	@echo "Building mcp-namespace..."
	cd mcp-namespace && go build -o ../bin/mcp-namespace .
	@echo "Building mcp-ns..."
	cd mcp-ns && go build -o ../bin/mcp-ns .
	@echo "Building mcp-mount..."
	cd mcp-mount && go build -o ../bin/mcp-mount .
	@echo "Building mcp-fs..."
	cd mcp-fs && go build -o ../bin/mcp-fs .
	@echo "Building mcp-tunnel-ns..."
	cd mcp-tunnel-ns && go build -o ../bin/mcp-tunnel-ns .
	@echo "All tools built in bin/"

# Install tools to GOPATH/bin
install:
	cd mcp-namespace && go install
	cd mcp-ns && go install
	cd mcp-mount && go install
	cd mcp-fs && go install
	cd mcp-tunnel-ns && go install

# Clean build artifacts
clean:
	rm -rf bin/
	find . -name "*.test" -delete
	find . -name "*.out" -delete

# Run tests
test:
	go test ./...

# Start namespace server
server:
	cd mcp-namespace && go run main.go

# Example registrations
demo:
	@echo "Starting demo..."
	@echo "Make sure namespace server is running (make server)"
	cd mcp-ns && go run main.go -c register /services/echo \
		-type local \
		-transport stdio \
		-command echo \
		-args "MCP Echo Service"
	cd mcp-ns && go run main.go -c list /services

# Development helpers
.PHONY: fmt vet lint

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run