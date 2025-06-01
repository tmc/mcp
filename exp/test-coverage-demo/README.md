# Test Coverage Demo

This repository demonstrates various types of Go tests to showcase coverage visualization capabilities.

## Structure

```
test-coverage-demo/
├── basic/          # Basic unit tests
├── table/          # Table-driven tests
├── subtests/       # Tests with subtests
├── fuzzing/        # Fuzz tests
├── benchmark/      # Benchmark tests
├── integration/    # Integration tests
├── testdata/       # Test data files
└── mocks/          # Mock implementations
```

## Running Tests with Coverage

```bash
# Run all tests with coverage
go test -coverprofile=coverage.out ./...

# Generate HTML coverage report
go cover -html=coverage.out -o coverage.html

# Run with race detector
go test -race -coverprofile=coverage.out ./...

# Run specific package
go test -coverprofile=coverage.out ./basic/

# Run with verbose output
go test -v -coverprofile=coverage.out ./...
```

## Test Types Demonstrated

1. **Basic Unit Tests** - Simple function tests
2. **Table-Driven Tests** - Parameterized test cases
3. **Subtests** - Nested test organization
4. **Error Cases** - Testing error conditions
5. **Edge Cases** - Boundary value testing
6. **Concurrent Tests** - Testing goroutines and channels
7. **Benchmark Tests** - Performance testing
8. **Fuzz Tests** - Property-based testing
9. **Integration Tests** - Multi-component testing
10. **Mock Tests** - Testing with dependencies

## Coverage Goals

- Demonstrate various coverage patterns
- Show partially covered functions
- Include uncovered edge cases
- Highlight branch coverage scenarios