# Integration Testing Makefile
# Manages integration tests and go.work files for MCP

.PHONY: all test clean setup test-marklabs test-golang-sdk test-protocol coverage help

# Default target
all: setup test

# Setup go.work files for each module
setup:
	@echo "Setting up go.work files..."
	@for dir in marklabs-interop golang-sdk-interop protocol-interop; do \
		if [ -d "$$dir" ]; then \
			if [ ! -f "$$dir/go.work" ]; then \
				echo "Creating go.work for $$dir"; \
				(cd $$dir && go work init .) || true; \
				(cd $$dir && go work use . ../../../) || true; \
			else \
				echo "go.work already exists for $$dir"; \
			fi; \
		fi; \
	done

# Run all integration tests
test: setup
	@echo "Running all integration tests..."
	@failed=0; \
	for dir in marklabs-interop golang-sdk-interop protocol-interop; do \
		if [ -d "$$dir" ]; then \
			echo ""; \
			echo "=== Testing $$dir ==="; \
			(cd $$dir && go test -v .) || failed=$$((failed + 1)); \
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
	@cd marklabs-interop && go test -v .

test-golang-sdk: setup
	@echo "Testing golang-sdk-interop..."
	@cd golang-sdk-interop && go test -v .

test-protocol: setup
	@echo "Testing protocol-interop..."
	@cd protocol-interop && go test -v .

# Run tests with coverage
coverage: setup
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	@failed=0; \
	for dir in marklabs-interop golang-sdk-interop protocol-interop; do \
		if [ -d "$$dir" ]; then \
			echo ""; \
			echo "=== Coverage for $$dir ==="; \
			(cd $$dir && go test -v -coverprofile=../coverage/$$dir.out . && \
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
	@for dir in marklabs-interop golang-sdk-interop protocol-interop; do \
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
	@echo "  setup        - Create go.work files for each module"
	@echo "  test         - Run all integration tests"
	@echo "  test-marklabs - Run marklabs-interop tests only"
	@echo "  test-golang-sdk - Run golang-sdk-interop tests only"
	@echo "  test-protocol - Run protocol-interop tests only"
	@echo "  coverage     - Run tests with coverage reports"
	@echo "  clean        - Remove generated files"
	@echo "  help         - Show this help message"