# Coverage Guide for MCPScripttest

This guide explains how coverage collection works in mcpscripttest, particularly for capturing coverage data across multiple binary executions.

## Overview

MCPScripttest automatically detects when coverage is enabled (via `GOCOVERDIR` environment variable) and builds all MCP tools with coverage instrumentation. This allows comprehensive coverage collection across all tool invocations during script tests.

## Features

1. **Automatic Coverage Detection**: When `GOCOVERDIR` is set, tools are automatically built with the `-cover` flag
2. **Multi-Binary Coverage**: Coverage data is collected from all MCP tools called during tests
3. **Tool Installation**: Coverage-enabled versions of tools are installed in a temporary directory and added to PATH
4. **Cross-Binary Analysis**: Coverage data from different tools can be merged and analyzed together

## How It Works

### Tool Building with Coverage

When tests are run with `GOCOVERDIR` set:

```go
// Auto-detection is enabled by default
opts := &ToolsOptions{
    AutoDetectCoverage: true,  // Checks GOCOVERDIR automatically
    Tools: []string{"mcpdiff", "mcpspy", "mcpcat"},
}

// Tools are built with -cover flag when coverage is detected
cleanup := InstallMCPTools(t, opts)
defer cleanup()
```

### Coverage Collection Across Binaries

Script tests that use multiple tools will collect coverage from each:

```
# testdata/coverage_test.txt
# Use mcpdiff to compare files
exec mcpdiff file1.mcp file2.mcp

# Use mcpcat to display content  
exec mcpcat -color=never file1.mcp

# Use mcpspy to spy on communication
exec bash -c 'echo "test" | mcpspy'
```

Each tool execution adds coverage data to the `GOCOVERDIR` directory.

## Usage Examples

### Basic Coverage Test

```go
func TestWithCoverage(t *testing.T) {
    // Set up coverage directory
    coverDir := t.TempDir()
    t.Setenv("GOCOVERDIR", coverDir)
    
    // Install coverage-enabled tools
    opts := &ToolsOptions{
        AutoDetectCoverage: true,
    }
    cleanup := InstallMCPTools(t, opts)
    defer cleanup()
    
    // Run script tests
    Test(t, "testdata/my_test.txt")
    
    // Coverage data is now in coverDir
}
```

### Collecting Coverage from Multiple Tools

```go
func TestMultiToolCoverage(t *testing.T) {
    // Create coverage directory
    coverDir := t.TempDir()
    t.Setenv("GOCOVERDIR", coverDir)
    
    // Install multiple tools with coverage
    opts := &ToolsOptions{
        AutoDetectCoverage: true,
        Tools: []string{"mcpdiff", "mcpspy", "mcpcat"},
    }
    cleanup := InstallMCPTools(t, opts)
    defer cleanup()
    
    // Run tests that use multiple tools
    Test(t, "testdata/multi_tool_test.txt")
    
    // Analyze coverage from all tools
    entries, _ := os.ReadDir(coverDir)
    toolCount := 0
    for _, entry := range entries {
        if strings.HasPrefix(entry.Name(), "covcounters.") {
            toolCount++
        }
    }
    t.Logf("Collected coverage from %d tools", toolCount)
}
```

## Coverage Analysis

After tests complete, analyze coverage using Go's coverage tools:

```bash
# Set coverage directory
export GOCOVERDIR=/tmp/mcp-coverage

# Run tests
go test ./...

# View coverage percentage
go tool covdata percent -i $GOCOVERDIR

# Generate detailed report
go tool covdata textfmt -i $GOCOVERDIR -o coverage.txt

# Create HTML report
go tool cover -html=coverage.txt -o coverage.html
```

## Demo Scripts

Two demo scripts are provided:

1. `coverage_demo.sh` - Basic coverage collection demo
2. `scripttest_coverage_demo.sh` - Advanced demo showing multi-tool coverage

Run them to see coverage collection in action:

```bash
./scripttest_coverage_demo.sh
```

## Best Practices

1. **Always Use Auto-Detection**: Let the tools automatically detect `GOCOVERDIR`
2. **Install Tools Once**: Install coverage-enabled tools once per test suite
3. **Clean Up**: Use the cleanup function to restore the original PATH
4. **Merge Coverage**: Use `go tool covdata merge` to combine coverage from different test runs
5. **Filter Results**: Focus on packages of interest when analyzing coverage

## Troubleshooting

If coverage isn't being collected:

1. Ensure `GOCOVERDIR` is set and the directory exists
2. Check that tools are being built with `-cover` flag (enable verbose output)
3. Verify tools are executing (check test output)
4. Look for coverage files: `covcounters.*` and `covmeta.*`

## Examples

See the test files for complete examples:
- `scripttest_meta_test.go` - Tests coverage collection across binaries
- `tools_coverage_test.go` - Tests tool installation with coverage
- `testdata/scripttest_coverage_test.txt` - Example script test using multiple tools