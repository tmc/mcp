# covtest - Individual Test Coverage Analysis Tool

`covtest` is a tool that runs each Go test in a package individually with separate coverage reports, allowing you to analyze how much coverage each test contributes.

## Features

- Run tests individually with isolated coverage collection
- Calculate coverage delta for each test
- Identify tests that provide unique coverage
- Generate detailed reports in both text and JSON formats
- Support for baseline coverage comparison

## Installation

```bash
go install github.com/tmc/mcp/exp/covtest@latest
```

## Usage

```bash
# Run coverage analysis on the current package
covtest

# Run analysis on a specific package
covtest -pkg github.com/tmc/mcp

# Generate JSON output
covtest -json -out coverage_data

# Run only tests matching a pattern
covtest -run TestSpecific

# Disable baseline comparison
covtest -baseline=false
```

## Options

- `-pkg`: Package path to test (default: ".")
- `-out`: Output directory for results (default: "coverage_analysis")
- `-v`: Verbose output
- `-timeout`: Timeout for each test (default: 10m)
- `-baseline`: Run baseline coverage with all tests (default: true)
- `-run`: Regex pattern for test selection
- `-json`: Output results in JSON format

## Output

The tool generates:

1. Console output showing top coverage contributors
2. A markdown report (`coverage_report.md`) with detailed analysis
3. Optional JSON output with full data
4. Coverage data directories for each test

## Example Output

```
Found 25 tests in github.com/tmc/mcp

Running baseline coverage (all tests)...
Running tests individually...

=== Coverage Analysis ===
Total test time: 2m30s
Tests run: 25

Top Coverage Contributors:
1. TestServerCore: +15.3% across 3 packages
2. TestClientAPI: +12.1% across 5 packages
3. TestTransport: +8.7% across 2 packages

Tests with Unique Coverage:
- TestEdgeCase: 2 unique packages
- TestErrorHandling: 1 unique package
```

## Integration with CI

The JSON output can be used in CI pipelines to:
- Track coverage trends over time
- Identify tests that can be run in parallel
- Optimize test execution order
- Find redundant tests