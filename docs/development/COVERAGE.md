# Code Coverage Guide for MCP Project

This document explains how to generate and analyze code coverage for the MCP project.

## Basic Coverage Collection

For basic coverage analysis of packages you're actively working on:

```bash
# Generate coverage data
go test -coverprofile=coverage.out ./...

# View HTML coverage report
go tool cover -html=coverage.out
```

This approach provides coverage information for directly tested packages but doesn't capture coverage from integration tests or from packages called by your code.

## Comprehensive Coverage Collection

For complete coverage analysis across all packages:

### Method 1: Using -coverpkg=all

This approach captures coverage across all packages, including those called by tests in other packages.

```bash
# Generate comprehensive coverage data
go test -coverpkg=all -coverprofile=all-coverage.out ./...

# View HTML coverage report
go tool cover -html=all-coverage.out -o all-coverage.html
```

### Method 2: Using Binary Coverage (Go 1.20+)

Binary coverage provides more detailed information and better handling of concurrent tests:

```bash
# Create a directory for coverage data
mkdir -p coverage-dir

# Run tests with binary coverage collection
go test -test.gocoverdir=./coverage-dir ./...

# Generate coverage data in text format
go tool covdata textfmt -i=./coverage-dir -o coverage-dir.txt

# Convert to HTML format
go tool covdata func -i=./coverage-dir
```

### Advanced Coverage Options

For integration with CI/CD systems:

```bash
# Generate coverage data in JSON format
go tool covdata func -i=./coverage-dir -o=./coverage/func.txt -percent=true

# Generate package-level statistics
go tool covdata pkglist -i=./coverage-dir > ./coverage/packages.txt
```

## Coverage Analysis

### Understanding Coverage Reports

The coverage report highlights code with different colors:
- Green: Code that is covered by tests
- Red: Code that is not covered by tests
- Gray: Code that is not executable (e.g., comments, blank lines)

### Key Coverage Metrics

- **Function Coverage**: Percentage of functions executed during tests
- **Statement Coverage**: Percentage of statements executed during tests
- **Branch Coverage**: Percentage of branches executed during tests

### Improving Coverage

1. Focus on core functions first
2. Add tests for error handling paths
3. Use table-driven tests for comprehensive input testing
4. Add integration tests to cover interactions between components
5. Mock external dependencies for better isolation and coverage

## Current Coverage Status

The current coverage analysis can be found in `coverage/SUMMARY.md`, which provides:
- Overall coverage percentage
- Well-covered components
- Components needing more coverage
- Recommendations for improvement

## Related Documentation

- [Experimental Tools Documentation](../../EXPERIMENTAL_TOOLS.md): Information about tools moved to the experimental repository
- [Testing Documentation](../../docs/scripttest/README.md): General testing approach
EOF < /dev/null
