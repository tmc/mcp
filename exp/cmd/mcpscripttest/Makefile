.PHONY: build install run run-all run-verbose clean test coverage

# Build the mcpscripttest binary
build:
	go build -o mcpscripttest .

# Install the mcpscripttest binary
install:
	go install .

# Run a specific test
run:
	./mcpscripttest -test $(TEST)

# Run all tests
run-all:
	./mcpscripttest -all

# Run all tests with verbose output
run-verbose:
	./mcpscripttest -all -v

# Run all tests with coverage
coverage:
	./mcpscripttest -all -coverage
	@echo "Coverage data collected. Process with:"
	@echo "go tool covdata textfmt -i /tmp/mcp-coverage-* -o coverage.txt"
	@echo "go tool cover -html=coverage.txt -o coverage.html"

# Clean build artifacts
clean:
	rm -f mcpscripttest

# Run Go tests
test:
	go test -v ./...

# Default target
all: build