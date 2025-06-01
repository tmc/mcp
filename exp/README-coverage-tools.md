# Coverage Analysis Tools

This directory contains experimental tools for analyzing Go test coverage, particularly focused on per-test coverage analysis and Codecov integration.

## Tools

### 1. covtest - Per-Test Coverage Analysis

Runs tests individually to analyze coverage contribution per test.

```bash
# Basic usage
go run covtest/main.go -pkg ./path/to/package

# Output per-test Codecov JSON files
go run covtest/main.go -pkg ./mcp -codecov ./coverage/json -per-test

# Combined coverage report
go run covtest/main.go -pkg ./mcp -codecov ./coverage/json
```

Features:
- Runs each test individually with isolated coverage
- Calculates coverage delta per test
- Identifies uniquely tested code per test
- Outputs in Codecov JSON format for detailed line-by-line coverage
- Generates test contribution reports

### 2. cov2codecov - Coverage Format Converter

Converts Go's binary coverage data to Codecov-compatible formats.

```bash
# Convert to text format (traditional)
go run cov2codecov/main.go -input coverage/dir -output coverage.txt

# Convert to Codecov JSON format
go run cov2codecov/main.go -input coverage/dir -output coverage.json -json

# Add test information to JSON
go run cov2codecov/main.go -input coverage/dir -output coverage.json -json -test-info "TestFoo"

# Merge multiple coverage directories
go run cov2codecov/main.go -input dir1,dir2,dir3 -output coverage.txt
```

Features:
- Supports both text and JSON output formats
- Merges multiple coverage directories
- Filters by package patterns
- Direct upload to Codecov
- Preserves hit counts in JSON format

### 3. covdiff - Coverage Difference Analysis

Analyzes coverage differences between test runs.

```bash
# Compare two coverage directories
go run covdiff/main.go -base coverage/baseline -new coverage/feature

# Show only added coverage
go run covdiff/main.go -base coverage/before -new coverage/after -added-only

# Filter by packages
go run covdiff/main.go -base old -new new -pkg github.com/tmc/mcp
```

## Codecov JSON Format

The tools support Codecov's JSON format for detailed coverage reporting:

```json
{
  "coverage": {
    "path/to/file.go": [null, 1, 0, null, true, 0, 0, 1, 1],
    "path/to/other.go": [null, 0, 1, 1, "1/3", null]
  },
  "messages": {
    "_metadata": {
      "test_name": "TestFoo",
      "generated": "2023-10-15T15:30:00Z"
    }
  }
}
```

Coverage array values:
- `null`: No code on this line
- `0`: Line not covered
- `1+`: Hit count (times executed)
- `"1/3"`: Branch coverage (1 of 3 branches covered)

## Use Cases

### 1. Identify Test Coverage Gaps

```bash
# Run all tests individually to see what each covers
go run covtest/main.go -pkg ./mcp -codecov ./coverage -per-test

# Look at test-contributions.json to see unique coverage per test
cat coverage/test-contributions.json
```

### 2. Analyze Feature Branch Coverage

```bash
# Generate baseline coverage
GOCOVERDIR=coverage/main go test ./...

# Generate feature branch coverage  
GOCOVERDIR=coverage/feature go test ./...

# Compare
go run covdiff/main.go -base coverage/main -new coverage/feature
```

### 3. Generate Codecov Reports

```bash
# Convert Go coverage to Codecov JSON with per-test info
go run covtest/main.go -pkg ./mcp -codecov reports -per-test

# Upload to Codecov
cd reports
for f in *.json; do
  codecov --file "$f" --flag "${f%.json}"
done
```

### 4. Continuous Integration

```bash
# In CI pipeline
export GOCOVERDIR=$PWD/coverage

# Run tests
go test -cover ./...

# Convert and upload
go run cov2codecov/main.go -input coverage -output coverage.json -json -upload
```

## Best Practices

1. **Isolate Test Coverage**: Run tests individually to understand true contribution
2. **Use JSON Format**: Provides line-by-line detail including partial branch coverage
3. **Track Coverage Over Time**: Use git hooks or CI to monitor coverage trends
4. **Focus on Unique Coverage**: Identify tests that provide unique value
5. **Merge Strategically**: Combine unit and integration test coverage appropriately

## Future Enhancements

- Support for branch coverage visualization
- Integration with IDE plugins
- Real-time coverage monitoring
- Test impact analysis based on code changes
- Coverage trend analysis over time