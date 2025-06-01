# TestCallGraph vs Standard Callgraph Comparison

## Overview

This document compares the enhanced `testcallgraph` with the standard `golang.org/x/tools/cmd/callgraph` tool.

## Key Differences

| Feature | Standard Callgraph | TestCallGraph |
|---------|-------------------|---------------|
| Analysis Type | Static only | Static + Dynamic |
| Input | Go packages | Script test files |
| Context | General purpose | Test-specific |
| Coverage | No integration | Full integration |
| Proximity | Not supported | Built-in |
| Multi-binary | Single binary | Multiple tools |
| Suggestions | None | Test modifications |

## Standard Callgraph

### Purpose
- Analyze static call relationships in Go programs
- Understand potential execution paths
- Support different algorithms (CHA, RTA, VTA)

### Usage
```bash
callgraph -algo=rta -format=graphviz package/...
```

### Output
```
github.com/example/pkg.main --static:42:5--> github.com/example/pkg.doWork
github.com/example/pkg.doWork --static:17:8--> fmt.Printf
```

### Limitations
- No runtime information
- No test awareness
- Single package at a time
- No coverage integration

## TestCallGraph

### Purpose
- Analyze how tests exercise code
- Find paths to uncovered code
- Optimize test coverage
- Support scripttest format

### Usage
```bash
testcallgraph -proximity parser.go:89 coverage_test.txt
testcallgraph -format json -packages ./cmd/... test.txt
```

### Output
```json
{
  "test": "coverage_test.txt:5",
  "command": "exec mcpdiff file1.mcp file2.mcp",
  "execution": {
    "duration": "134ms",
    "coverage": "23.4%"
  },
  "calls": [
    {
      "from": "test",
      "to": "mcpdiff.main",
      "type": "dynamic",
      "actual": true
    },
    {
      "from": "mcpdiff.main",
      "to": "parser.loadFile",
      "type": "static",
      "actual": true,
      "line": 45
    }
  ],
  "proximity": {
    "target": "parser.go:89",
    "distance": 2,
    "suggestion": "Change input to invalid JSON"
  }
}
```

### Enhancements
- Dynamic execution tracing
- Test-to-code mapping
- Proximity analysis
- Coverage integration
- Multi-tool support

## Use Cases

### Standard Callgraph Use Cases
1. Understanding code structure
2. Refactoring impact analysis
3. Dead code detection
4. API usage analysis

### TestCallGraph Use Cases
1. Coverage optimization
2. Test redundancy detection
3. Finding paths to uncovered code
4. Test modification suggestions
5. CI/CD integration

## Algorithm Comparison

### Standard Algorithms
- **Static**: Only direct calls
- **CHA**: Class Hierarchy Analysis
- **RTA**: Rapid Type Analysis
- **VTA**: Variable Type Analysis

### TestCallGraph Algorithms
- All standard algorithms PLUS:
- **Dynamic**: Actual execution paths
- **Proximity**: Graph distance calculation
- **Coverage**: Integration with Go coverage

## Example Comparison

### Finding Uncovered Code

**Standard Callgraph**: Not supported

**TestCallGraph**:
```bash
$ testcallgraph -proximity parser.go:89 test.txt

Closest test: test.txt:5 (exec mcpdiff valid.mcp)
Distance: 2 calls
Path: mcpdiff.main -> parser.loadFile -> parser.handleError

Suggestion: Replace valid.mcp with invalid JSON to trigger error path
```

### Multi-Tool Analysis

**Standard Callgraph**: Must analyze each tool separately

**TestCallGraph**: Analyzes entire test execution
```
Test line: exec mcpspy -- mcpdiff file1.mcp file2.mcp
Traces calls through:
  - mcpspy.main (process monitoring)
  - mcpdiff.main (file comparison)
  - Both tools' internal calls
```

## Performance Comparison

| Operation | Standard | TestCallGraph |
|-----------|----------|---------------|
| Static Analysis | Fast | Fast |
| Dynamic Tracing | N/A | Moderate |
| Large Codebases | Good | Good |
| Test Suites | N/A | Optimized |

## Integration

### Standard Callgraph
- Standalone tool
- Basic text/graph output
- Manual interpretation

### TestCallGraph
- Integrated with mcpscripttest
- Multiple output formats
- IDE plugins possible
- CI/CD ready

## Conclusion

TestCallGraph extends standard callgraph specifically for test analysis:
- **Standard**: General-purpose static analysis
- **TestCallGraph**: Test-specific with dynamic tracing

Both tools serve different purposes and can be used complementarily.