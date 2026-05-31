# Integration Testing Makefile
# Manages integration tests for MCP.

.PHONY: all test clean setup test-marklabs test-golang-sdk test-protocol test-typescript-sdk coverage help

MODULES := marklabs-interop golang-sdk-interop protocol-interop typescript-sdk-interop

# Default target
all: setup test

# The integration modules carry explicit replace directives and should be
# tested with GOWORK=off so local workspaces do not affect release evidence.
setup:
	@echo "No workspace setup required; tests run with GOWORK=off."

# Run all integration tests
test: setup
	@echo "Running all integration tests..."
	@failed=0; \
	for dir in $(MODULES); do \
		if [ -d "$$dir" ]; then \
			echo ""; \
			echo "=== Testing $$dir ==="; \
			(cd $$dir && GOWORK=off go test -v ./...) || failed=$$((failed + 1)); \
		fi; \
	done; \
	if [ $$failed -gt 0 ]; then \
		echo ""; \
		echo "FAILED: $$failed test suite(s) failed"; \
		exit 1; \
	else \
		echo ""; \
		echo "SUCCESS: All tests passed"; \
	fi

# Test individual modules
test-marklabs: setup
	@echo "Testing marklabs-interop..."
	@cd marklabs-interop && GOWORK=off go test -v ./...

test-golang-sdk: setup
	@echo "Testing golang-sdk-interop..."
	@cd golang-sdk-interop && GOWORK=off go test -v ./...

test-protocol: setup
	@echo "Testing protocol-interop..."
	@cd protocol-interop && GOWORK=off go test -v ./...

test-typescript-sdk: setup
	@echo "Testing typescript-sdk-interop..."
	@cd typescript-sdk-interop && GOWORK=off go test -v ./...

# Run tests with coverage
coverage: setup
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	@failed=0; \
	for dir in $(MODULES); do \
		if [ -d "$$dir" ]; then \
			echo ""; \
			echo "=== Coverage for $$dir ==="; \
			(cd $$dir && GOWORK=off go test -v -coverprofile=../coverage/$$dir.out ./... && \
			 go tool cover -html=../coverage/$$dir.out -o ../coverage/$$dir.html) || failed=$$((failed + 1)); \
		fi; \
	done; \
	if [ $$failed -eq 0 ]; then \
		echo ""; \
		echo "Coverage reports generated in coverage/"; \
	fi

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -rf coverage
	@for dir in $(MODULES); do \
		if [ -f "$$dir/go.work" ]; then \
			echo "Removing $$dir/go.work"; \
			rm -f "$$dir/go.work"; \
		fi; \
		if [ -f "$$dir/go.work.sum" ]; then \
			echo "Removing $$dir/go.work.sum"; \
			rm -f "$$dir/go.work.sum"; \
		fi; \
	done

# Help target
help:
	@echo "Integration Testing Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Setup and run all tests (default)"
	@echo "  setup        - Reserved setup hook"
	@echo "  test         - Run all integration tests"
	@echo "  test-marklabs - Run marklabs-interop tests only"
	@echo "  test-golang-sdk - Run golang-sdk-interop tests only"
	@echo "  test-protocol - Run protocol-interop tests only"
	@echo "  test-typescript-sdk - Run TypeScript SDK interop smoke only"
	@echo "  coverage     - Run tests with coverage reports"
	@echo "  clean        - Remove generated files"
	@echo "  help         - Show this help message"
