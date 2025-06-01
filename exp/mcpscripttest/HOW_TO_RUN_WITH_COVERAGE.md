# How to Run Scripttest with Coverage

This guide shows the simplest way to run a scripttest and collect coverage data from the tools it executes.

## Quick Start

### 1. Set Coverage Directory

```bash
export GOCOVERDIR=/tmp/mcp-coverage
mkdir -p $GOCOVERDIR
```

### 2. Run a Test

```bash
cd /path/to/mcpscripttest
go test -v -run TestScripttestCoverageAcrossBinaries
```

### 3. Analyze Coverage

```bash
# View coverage percentages
go tool covdata percent -i $GOCOVERDIR

# Generate detailed report
go tool covdata textfmt -i $GOCOVERDIR -o coverage.txt
```

## Complete Example

Here's a complete script that demonstrates coverage collection:

```bash
#!/bin/bash

# Create a temporary coverage directory
COVERDIR=$(mktemp -d)
echo "Coverage directory: $COVERDIR"

# Set the coverage directory
export GOCOVERDIR=$COVERDIR

# Run the test
go test -v -run TestScripttestCoverageAcrossBinaries

# Check what was collected
echo "Coverage files:"
find $COVERDIR -name "covcounters.*" -o -name "covmeta.*"

# Analyze coverage
echo "Coverage analysis:"
go tool covdata percent -i $COVERDIR
```

## Writing Your Own Test

Here's a minimal test that runs a scripttest with coverage:

```go
package mypackage

import (
    "testing"
    "github.com/tmc/mcp/exp/mcpscripttest"
)

func TestWithCoverage(t *testing.T) {
    // Coverage is automatically detected from GOCOVERDIR
    
    // Install coverage-enabled tools
    cleanup := mcpscripttest.InstallMCPTools(t, nil)
    defer cleanup()
    
    // Run your scripttest
    opts := mcpscripttest.DefaultOptions()
    opts.AdditionalEnvVars = []string{"GOCOVERDIR"}
    mcpscripttest.Test(t, "testdata/mytest.txt", opts)
}
```

## Script Test File

Create a test file that uses MCP tools:

```
# testdata/mytest.txt
exec mcpdiff file1.mcp file2.mcp
stdout 'Files match'

exec mcpcat file1.mcp
stderr 'mcp-send'

-- file1.mcp --
mcp-send {"jsonrpc":"2.0","method":"test","id":1}

-- file2.mcp --
mcp-send {"jsonrpc":"2.0","method":"test","id":1}
```

## Important Notes

1. **GOCOVERDIR must be set** before running tests for coverage collection
2. **Tools are automatically built with coverage** when GOCOVERDIR is detected
3. **Coverage is collected from all tools** executed during the test
4. **Binary names are hashed** in coverage files, so you'll see hex strings
5. **Use go tool covdata** to analyze the collected coverage

## Troubleshooting

If no coverage is collected:

1. Ensure GOCOVERDIR is set and the directory exists
2. Check that tools are being installed (look for "Installing X with coverage: true")
3. Verify tools are executing in your test
4. Look for covcounters.* and covmeta.* files in the coverage directory

## Full Working Example

Save this as `run_coverage_test.sh`:

```bash
#!/bin/bash

# Setup
COVERDIR=$(mktemp -d)
export GOCOVERDIR=$COVERDIR

# Run test
echo "Running test with coverage..."
go test -v -run TestScripttestCoverageAcrossBinaries

# Find coverage files
echo -e "\nCoverage files:"
find $COVERDIR -type f | head -10

# Analyze
echo -e "\nCoverage summary:"
go tool covdata percent -i $COVERDIR || echo "No coverage data found"

# Cleanup
echo -e "\nCoverage data saved in: $COVERDIR"
```

Make it executable and run:

```bash
chmod +x run_coverage_test.sh
./run_coverage_test.sh
```

This will show you exactly how coverage collection works with scripttest.