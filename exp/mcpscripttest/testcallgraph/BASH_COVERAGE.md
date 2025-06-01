# Bash Script Coverage with TestCallGraph

TestCallGraph has been extended to analyze bash script executions and coverage within mcpscripttest files.

## Features

### Bash Script Detection
- Automatically detects bash script executions (`exec bash`, `bash`, `sh` commands)
- Recognizes coverage-enabled executions (using `kcov`, `bashcov`, etc.)
- Tracks script arguments and execution context

### Script Analysis  
- Parses bash scripts to identify function definitions
- Maps function locations (start and end lines)
- Tracks which functions are called during execution

### Coverage Integration
- Identifies coverage collection commands
- Provides coverage reports showing:
  - Lines executed vs total lines
  - Function call counts
  - Uncovered code sections

### Call Graph Generation
- Creates edges from test files to bash scripts
- Shows internal function calls within scripts
- Integrates with existing Go program analysis

## Usage

### Enable Bash Analysis

```bash
# Run with bash mode enabled
testcallgraph -bash test.txt

# Generate visual call graph including bash scripts
testcallgraph -bash -format dot test.txt | dot -Tsvg > graph.svg

# Show bash coverage statistics
testcallgraph -bash -stats test.txt
```

### Example Test File

```txt
# test.txt - mcpscripttest file with bash executions

# Execute a bash script
exec bash deploy.sh production

# Run with coverage collection
exec kcov --exclude-pattern=/usr coverage_out ./test.sh

# Custom command that runs bash
mcp-server-start server -- ./start_server.sh
```

### Coverage Report Output

```
=== Bash Script Coverage Report ===

deploy.sh:
  Lines: 45/120 (37.5%)
  Functions:
    setup (lines 5-15): called 1 times
    deploy (lines 20-50): called 1 times  
    cleanup (lines 55-65): not called

test.sh:
  Lines: 89/100 (89.0%)
  Functions:
    run_tests (lines 10-80): called 3 times
    report (lines 85-95): called 1 times
```

## Implementation Details

### BashStitcher Class

The `BashStitcher` extends `EnhancedStitcher` with bash-specific functionality:

```go
type BashStitcher struct {
    *EnhancedStitcher
    BashScriptMap    map[string][]BashExecution  
    BashCoverageMap  map[string]*BashCoverage
}
```

### Bash Execution Tracking

```go
type BashExecution struct {
    ScriptPath   string
    Command      string
    Line         int
    ExecutedBy   string
    Arguments    []string
    WithCoverage bool
}
```

### Coverage Data Structure

```go
type BashCoverage struct {
    ScriptPath      string
    TotalLines      int
    ExecutedLines   map[int]bool
    CoveragePercent float64
    Functions       map[string]*BashFunction
}
```

## Integration with Coverage Tools

TestCallGraph can integrate with popular bash coverage tools:

### kcov
```bash
exec kcov --exclude-pattern=/usr coverage_out ./script.sh
```

### bashcov
```bash
exec bashcov ./script.sh
```

### Custom Coverage
Implement custom coverage collection by parsing tool output formats.

## Future Enhancements

1. **Real-time Coverage Collection**
   - Integrate with coverage tool APIs
   - Parse coverage data formats (lcov, cobertura)

2. **Advanced Script Analysis**
   - Track variable usage
   - Identify conditional branches
   - Map source locations to test cases

3. **Coverage Suggestions**
   - Identify uncovered functions
   - Suggest test modifications
   - Generate coverage improvement reports

4. **Multi-language Support**
   - Python script analysis
   - Shell script variants (zsh, fish)
   - Other interpreted languages

## Example Workflow

1. Write mcpscripttest files that execute bash scripts
2. Run testcallgraph with `-bash` flag
3. Analyze coverage report
4. Generate visual call graphs
5. Identify gaps in test coverage
6. Improve tests based on analysis

This enhancement makes TestCallGraph a comprehensive tool for analyzing both compiled Go programs and interpreted bash scripts within the MCP testing framework.