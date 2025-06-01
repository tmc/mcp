# Coverage Analysis Demo

This demo shows how to use the new coverage tools to analyze per-test coverage contributions.

## 1. Running Individual Test Coverage Analysis

First, let's analyze a few individual tests:

```bash
# Navigate to the MCP package
cd /Volumes/tmc/go/src/github.com/tmc/mcp

# Run covtest to analyze individual test contributions
./exp/covtest/covtest -pkg . -out coverage_analysis -codecov codecov_output -per-test -run "TestClient.*"
```

This will:
- Run each test matching `TestClient.*` individually
- Generate per-test Codecov JSON files
- Calculate coverage contributions
- Create a test contribution report

## 2. Converting Coverage to Codecov Format

If you have existing coverage data in Go's binary format:

```bash
# Generate coverage data
mkdir coverage_data
GOCOVERDIR=coverage_data go test -v

# Convert to Codecov JSON
./exp/cov2codecov/cov2codecov -input coverage_data -output coverage.json -json
```

## 3. Comparing Coverage Between Runs

To see what coverage a new feature adds:

```bash
# Baseline coverage
GOCOVERDIR=coverage_before go test

# Feature branch coverage  
GOCOVERDIR=coverage_after go test

# Compare
./exp/covdiff/covdiff -base coverage_before -new coverage_after
```

## 4. Example Output

The Codecov JSON format shows detailed line-by-line coverage:

```json
{
  "coverage": {
    "github.com/tmc/mcp/client.go": [
      null,      // line 0 (always null)
      null,      // line 1: no code
      null,      // line 2: no code
      1,         // line 3: executed once
      0,         // line 4: not covered
      "2/3",     // line 5: 2 of 3 branches covered
      5          // line 6: executed 5 times
    ]
  },
  "messages": {
    "_metadata": {
      "test_name": "TestClientNotificationHandling",
      "generated": "2025-05-16T22:30:00-07:00"
    }
  }
}
```

## 5. Practical Usage

### Finding Untested Code
```bash
# Run all tests individually
./exp/covtest/covtest -pkg . -codecov coverage_reports -per-test

# Check which tests cover unique code
cat coverage_reports/test-contributions.json | jq '.[] | select(.unique_lines > 0)'
```

### CI Integration
```bash
# In your CI pipeline
export GOCOVERDIR=$PWD/coverage

# Run tests
go test -cover ./...

# Convert and upload to Codecov
./exp/cov2codecov/cov2codecov -input coverage -output coverage.json -json -upload
```

### Test Optimization
```bash
# Find tests with low unique coverage contribution
./exp/covtest/covtest -pkg . -codecov reports -per-test
cat reports/test-contributions.json | jq 'to_entries | sort_by(.value.unique_lines) | .[:10]'
```

## Key Insights

1. **Per-Test Isolation**: Running tests individually shows their true coverage contribution
2. **Unique Coverage**: Identifies which tests provide unique value vs redundant coverage
3. **Branch Coverage**: The fraction notation (e.g., "2/3") shows partial branch coverage
4. **Hit Counts**: Exact execution counts help identify hot paths and dead code

## Next Steps

1. Integrate with your CI pipeline for automatic coverage tracking
2. Use coverage data to optimize test suites
3. Identify gaps in test coverage with detailed line-by-line analysis
4. Track coverage trends over time using the JSON format