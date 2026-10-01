# cov2codecov - Go Coverage to Codecov Converter

Convert Go's binary coverage data (introduced in Go 1.20) to Codecov-compatible text format. This tool handles merging unit and integration test coverage and outputs in the format Codecov expects.

## Features

- Convert Go 1.20+ binary coverage data to text format
- Merge multiple coverage directories (unit tests, integration tests, etc.)
- Filter coverage by package
- Direct upload to Codecov
- Compatible with GitHub Actions and other CI/CD systems

## Installation

```bash
go install github.com/tmc/mcp/exp/cov2codecov@latest
```

## Usage

### Basic Conversion

```bash
# Convert a single coverage directory
cov2codecov -input coverage/dir -output coverage.txt

# Convert with package filtering
cov2codecov -input coverage/dir -pkg github.com/myproject -output coverage.txt
```

### Merge Unit and Integration Tests

```bash
# Merge unit and integration test coverage
cov2codecov -unit coverage/unit -integ coverage/integration -output coverage.txt

# Multiple directories
cov2codecov -input dir1,dir2,dir3 -output coverage.txt
```

### Upload to Codecov

```bash
# Convert and upload to Codecov
cov2codecov -input coverage/dir -output coverage.txt -upload

# With Codecov token and flags
cov2codecov -input coverage/dir -upload -token $CODECOV_TOKEN -flags unit,integration
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      # Run unit tests with coverage
      - name: Run unit tests
        run: |
          mkdir -p coverage/unit
          go test -cover ./... -args -test.gocoverdir="$PWD/coverage/unit"
      
      # Build and run integration tests
      - name: Run integration tests
        run: |
          go build -cover -o myapp ./cmd/myapp
          mkdir -p coverage/integration
          GOCOVERDIR=$PWD/coverage/integration ./myapp
      
      # Convert and upload to Codecov
      - name: Upload coverage
        run: |
          go install github.com/tmc/mcp/exp/cov2codecov@latest
          cov2codecov -unit coverage/unit -integ coverage/integration -output coverage.txt -upload
        env:
          CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}
```

### Makefile Example

```makefile
.PHONY: test coverage

COVERAGE_DIR := coverage
UNIT_COV := $(COVERAGE_DIR)/unit
INTEGRATION_COV := $(COVERAGE_DIR)/integration

test: test-unit test-integration

test-unit:
	@mkdir -p $(UNIT_COV)
	@go test -cover ./... -args -test.gocoverdir="$(UNIT_COV)"

test-integration:
	@go build -cover -o bin/myapp ./cmd/myapp
	@mkdir -p $(INTEGRATION_COV)
	@GOCOVERDIR=$(INTEGRATION_COV) ./bin/myapp

coverage: test
	@cov2codecov -unit $(UNIT_COV) -integ $(INTEGRATION_COV) -output coverage.txt
	@echo "Coverage report: coverage.txt"
	@go tool cover -func=coverage.txt | tail -n1

upload-coverage: coverage
	@cov2codecov -unit $(UNIT_COV) -integ $(INTEGRATION_COV) -upload
```

## Options

- `-unit`: Unit test coverage directory
- `-integ`: Integration test coverage directory
- `-input`: Comma-separated list of coverage directories
- `-output`: Output file for text coverage (default: "coverage.txt")
- `-merge`: Directory to store merged binary coverage
- `-pkg`: Filter packages (comma-separated)
- `-v`: Verbose output
- `-skip-merge`: Skip merge step and convert directly
- `-upload`: Upload to Codecov after conversion
- `-token`: Codecov upload token
- `-flags`: Codecov flags (comma-separated)

## How It Works

1. **Collect Coverage**: Go 1.20+ can generate binary coverage data when:
   - Running tests with `-args -test.gocoverdir=DIR`
   - Running binaries built with `-cover` and `GOCOVERDIR=DIR`

2. **Merge**: If multiple directories are provided, `go tool covdata merge` combines them

3. **Convert**: `go tool covdata textfmt` converts binary data to text format

4. **Upload**: Optionally upload to Codecov using their CLI or bash uploader

## Troubleshooting

### No coverage data found
- Ensure Go 1.20+ is being used
- Check that tests/binaries were run with coverage enabled
- Verify the coverage directories contain `.covcounters.*` and `.covmeta.*` files

### Package filtering not working
- Use full package paths (e.g., `github.com/user/project/pkg`)
- Separate multiple packages with commas

### Upload fails
- Ensure Codecov token is set (for private repos)
- Check network connectivity
- Try verbose mode (`-v`) for detailed error messages