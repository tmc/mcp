# MCPScriptTest

A powerful testing framework for MCP (Model Context Protocol) tools that provides script-based testing with coverage analysis and call graph generation.

## Features

- **Script-based Testing**: Write tests as simple text scripts
- **Tool Installation**: Automatic MCP tool installation with coverage support
- **Coverage Analysis**: Track code coverage across tool executions
- **Call Graph Generation**: Visualize relationships between tests and tools
- **Bash Script Coverage**: Track execution through bash scripts (NEW!)
- **Flexible Commands**: Built-in commands plus custom tool support

## Quick Start

```go
package mytest

import (
    "testing"
    "github.com/tmc/mcp/exp/mcpscripttest"
)

func TestMyTool(t *testing.T) {
    mcpscripttest.Test(t, "testdata/mytool.txt")
}
```

Example test script (`testdata/mytool.txt`):
```
# Test my MCP tool
exec mytool analyze file.go
stdout 'Analysis complete'

# Test with stdin
stdin 'test data'
exec mytool process
stdout 'Processed'
```

## Bash Script Coverage

Track execution through bash scripts with the new bash coverage feature:

```
# Enable bash coverage
env MCP_BASH_COVERAGE=1
env MCP_BASH_COVERAGE_DIR=$WORK/bash-coverage

# Run bash scripts with coverage tracking
bash './test.sh --flag'

# Analyze with testcallgraph
testcallgraph -bash -format json test.txt
```

See [BASH_COVERAGE.md](BASH_COVERAGE.md) for details.

## Built-in Commands

- `exec`: Execute a command
- `stdin`/`setstdin`: Set stdin content
- `stdout`/`stderr`: Assert output content
- `env`: Set environment variables
- `cd`: Change directory
- `cp`/`rm`/`mkdir`: File operations
- `bash`: Execute bash commands with coverage support
- `grep`: Search file contents
- `wait`: Wait for background processes

## Tool Installation

Automatically install MCP tools with coverage support:

```go
cleanup := mcpscripttest.InstallMCPTools(t, &mcpscripttest.ToolsOptions{
    Tools: []string{"mcpdiff", "mcp-serve"},
    WithCoverage: true,
})
defer cleanup()
```

## Coverage Analysis

Enable coverage collection:

```go
cleanup := mcpscripttest.SetupTestCoverage(t, &mcpscripttest.CoverageOptions{
    Enabled: true,
    OutputDir: "coverage",
})
defer cleanup()
```

## Call Graph Analysis

Generate call graphs showing relationships between tests and tools:

```bash
testcallgraph -format dot -o graph.dot tests/
dot -Tpng graph.dot -o graph.png
```

With bash script support:
```bash
testcallgraph -bash -format json tests/ > callgraph.json
```

## Documentation

- [Conditional Testing](docs/conditional-testing.md)
- [Bash Coverage](BASH_COVERAGE.md)
- [Testing Helpers](../../../docs/testing_helpers.md)
- [Coverage Guide](COVERAGE_GUIDE.md)

## Contributing

See the main MCP repository for contribution guidelines.

## License

Same as the main MCP repository.