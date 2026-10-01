# covdiff - Coverage Difference Analysis Tool

`covdiff` analyzes coverage differences between test runs and provides various operations for manipulating coverage data using Go's built-in `covdata` tool.

## Features

- Compare coverage between two test runs
- Merge coverage data from multiple sources
- Find intersection of coverage
- Subtract coverage to find unique contributions
- Generate detailed difference reports

## Installation

```bash
go install github.com/tmc/mcp/exp/covdiff@latest
```

## Usage

### Difference Analysis

```bash
# Compare coverage between two directories
covdiff -base baseline_coverage -compare new_coverage

# Generate JSON output
covdiff -base baseline_coverage -compare new_coverage -json

# Show verbose output
covdiff -base baseline_coverage -compare new_coverage -v
```

### Coverage Operations

```bash
# Merge coverage from two directories
covdiff -mode merge -base dir1 -compare dir2 -out merged_coverage

# Find intersection of coverage
covdiff -mode intersect -base dir1 -compare dir2 -out intersection

# Subtract coverage (what's in base but not in compare)
covdiff -mode subtract -base dir1 -compare dir2 -out unique_to_base
```

## Options

- `-base`: Base coverage directory
- `-compare`: Directory to compare against base
- `-mode`: Operation mode: diff, merge, intersect, subtract (default: "diff")
- `-out`: Output directory (default: "coverage_diff")
- `-v`: Verbose output
- `-json`: Output in JSON format

## Output

### Diff Mode

Generates:
- Console output showing improvements and regressions
- Markdown report (`diff_report.md`) with detailed analysis
- Optional JSON output with full difference data

### Other Modes

- **merge**: Creates merged coverage data
- **intersect**: Creates coverage data for code covered by both
- **subtract**: Creates coverage data for code covered only by base

## Example Output

```
=== Coverage Difference Analysis ===
Base: baseline_coverage
Compare: new_coverage

Coverage Improvements:
  github.com/tmc/mcp/client: +5.2%
  github.com/tmc/mcp/server: +3.1%
  github.com/tmc/mcp/transport: +8.4%

Coverage Regressions:
  github.com/tmc/mcp/utils: -2.1%

Newly Covered Packages:
  github.com/tmc/mcp/experimental: 15.3%
```

## Use Cases

1. **CI/CD Integration**: Compare coverage between commits
2. **Test Optimization**: Identify which changes affect coverage
3. **Coverage Tracking**: Monitor coverage trends over time
4. **Test Suite Analysis**: Find gaps in test coverage

## Working with covtest

`covdiff` works well with `covtest` output:

```bash
# Run individual test analysis
covtest -out individual_tests

# Compare specific test coverage to baseline
covdiff -base individual_tests/covdata/baseline \
        -compare individual_tests/covdata/TestSpecific
```