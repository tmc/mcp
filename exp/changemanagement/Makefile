.PHONY: all build install test clean demo

# Build all tools
all: build

build:
	go build -o bin/mcp-change-analyze ./cmd/mcp-change-analyze
	go build -o bin/mcp-test-find ./cmd/mcp-test-find
	go build -o bin/mcp-doc-gen ./cmd/mcp-doc-gen
	go build -o bin/mcp-test-mutate ./cmd/mcp-test-mutate
	go build -o bin/mcp-change-execute ./cmd/mcp-change-execute

install:
	go install ./cmd/mcp-change-analyze
	go install ./cmd/mcp-test-find
	go install ./cmd/mcp-doc-gen
	go install ./cmd/mcp-test-mutate
	go install ./cmd/mcp-change-execute

test:
	go test -v ./...

demo: build
	./demo.sh

clean:
	rm -rf bin/
	rm -rf demo-output/
	rm -rf change-output/

# Development helpers
run-analyzer:
	go run ./cmd/mcp-change-analyze/main.go -description "Add OAuth2 authentication"

run-test-finder:
	go run ./cmd/mcp-test-find/main.go -change demo-output/analysis.json -codebase .

run-doc-gen:
	go run ./cmd/mcp-doc-gen/main.go -change demo-output/analysis.json

run-mutator:
	go run ./cmd/mcp-test-mutate/main.go -test testdata/sample_test.txt

run-orchestrator:
	go run ./cmd/mcp-change-execute/main.go -description "Add OAuth2 authentication" -verbose