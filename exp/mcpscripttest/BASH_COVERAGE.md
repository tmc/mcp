# Bash Script Coverage in MCPScriptTest

This document describes the bash script coverage feature added to mcpscripttest, which allows tracking execution paths through bash scripts and integrating them into the call graph analysis.

## Overview

The bash coverage feature uses the `BASH_XTRACEFD` environment variable to capture execution traces from bash scripts. This trace data is then parsed to:

1. Track which lines in bash scripts are executed
2. Identify script-to-script calls
3. Build call graph edges between tests, bash scripts, and tools

## Usage

### Enable Bash Coverage

Set environment variables in your test:

```
env MCP_BASH_COVERAGE=1
env MCP_BASH_COVERAGE_DIR=$WORK/bash-coverage
```

### Running Bash Scripts

Use the `bash` command with coverage enabled:

```
bash 'echo "Direct command"'
bash './script.sh arg1 arg2'
```

### Example Test

```
# Test with bash coverage
env MCP_BASH_COVERAGE=1
env MCP_BASH_COVERAGE_DIR=$WORK/bash-coverage

# Run a script
exec chmod +x test.sh
bash './test.sh'

# Check coverage was collected
exec ls $WORK/bash-coverage/
stdout 'bash-.*\.trace'

-- test.sh --
#!/bin/bash
echo "Hello from test.sh"
mcpdiff file1.mcp file2.mcp
```

## Integration with testcallgraph

The testcallgraph tool has been enhanced with a `-bash` flag to enable bash script analysis:

```
testcallgraph -bash -format json test.txt
```

This will:
1. Parse bash scripts in the test files
2. Process bash trace data
3. Generate call graph edges including bash script executions

### Call Graph Format

Bash script edges appear in the call graph with special prefixes:

- `bash:bash` - Direct bash command execution
- `bash:exec` - Script executed via exec
- `bash:function` - Function calls within scripts

Example output:
```
test.txt:5 -> test.sh:1 (bash:bash)
test.sh:3 -> mcpdiff (exec)
```

## Implementation Details

### Trace Format

The bash trace format includes:
- Script name and line number: `+(script.sh:5): command`
- Direct execution tracing: `+(bash:1): ./script.sh`

### Coverage Data Structure

```go
type BashCoverage struct {
    ScriptPath     string
    TotalLines     int
    ExecutedLines  map[int]bool
    CoveragePercent float64
    Functions      map[string]*BashFunction
}
```

### Trace File Locations

Trace files are created in the specified coverage directory with names like:
- `bash-TestName-1747557695926786000.trace`

## Future Enhancements

1. Support for function-level coverage within bash scripts
2. Integration with standard code coverage tools
3. Visualization of bash script execution paths
4. Support for other shell types (zsh, sh)

## Limitations

1. Requires bash 4.0+ for line number tracking
2. Coverage data is per-execution (not cumulative)
3. Complex bash constructs may not be fully tracked

## Contributing

When adding new features:
1. Update the BashStitcher in testcallgraph
2. Add tests in bash_coverage_test.go
3. Update this documentation