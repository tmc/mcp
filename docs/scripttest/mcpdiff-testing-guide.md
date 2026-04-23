# Testing MCP Trace Diffing with mcpscripttest

This guide documents best practices for testing the mcpdiff tool using mcpscripttest, a powerful testing framework specifically designed for MCP tools.

## Overview

The mcpdiff tool compares two MCP trace files and highlights their differences. It can:

- Identify differences in message content
- Compare responses, notifications, and other MCP components
- Handle formatting differences through semantic comparison
- Provide colorized output for easy visual inspection

Testing mcpdiff requires verifying its ability to correctly identify differences between MCP traces while properly handling edge cases.

## Test Structure

The mcpscripttest package provides a scripttest-based framework for writing declarative tests. For mcpdiff, we structure tests as follows:

### Basic Test Organization

```
/exp/cmd/mcpdiff/
  main.go             # Tool implementation
  main_test.go        # Test runner for scripttest files
  testdata/           # Test scripts and sample files
    basic_diff.txt    # Core functionality tests
    notification_diff_test.txt  # Specialized notification comparison tests
    sample1.mcp       # Sample MCP trace files for testing
    sample2.mcp
```

### Test Runner (main_test.go)

The test runner sets up the mcpscripttest environment and runs the scripttest files:

```go
package main

import (
    "testing"
    "github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPDiffBasic runs all script tests for mcpdiff
func TestMCPDiffBasic(t *testing.T) {
    // Setup coverage environment automatically
    mcpscripttest.SetupCoverageEnvironment(t)

    // Run all tests in the testdata directory
    mcpscripttest.Test(t, "testdata/*.txt")
}

// Run specific test categories in subtests
func TestMCPScripts(t *testing.T) {
    // Test scripts in the main testdata directory
    t.Run("MainTests", func(t *testing.T) {
        mcpscripttest.Test(t, "testdata/*.txt")
    })

    // Test scripts in subdirectories
    t.Run("SubdirTests", func(t *testing.T) {
        mcpscripttest.Test(t, "testdata/newtests/*.txt")
    })
}
```

## Writing Effective mcpdiff Tests

### Test Files Structure

A typical mcpdiff test file follows this structure:

```txt
# Description of the test case

# First create test files with known differences
-- file1.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550

-- file2.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":"updated"}}} # 1683000000.550

# Run mcpdiff and verify expected output
mcpdiff file1.mcp file2.mcp
stdout "server"
! stdout "No differences found"

# Test with specific flags
mcpdiff -ignore-timestamps file1.mcp file2.mcp
stdout "server"
! stderr "error"
```

### Test Categories

Organize mcpdiff tests into distinct categories:

1. **Basic Functionality**
   - Compare identical files (should report no differences)
   - Compare files with simple differences
   - Verify exit codes (0 for identical, 1 for differences, 2 for errors)

2. **Format Options**
   - Test different output formats (default, JSON)
   - Test colorization options

3. **Comparison Options**
   - Test with and without timestamp comparison
   - Test ignoring message order
   - Test semantic comparison vs. exact text comparison

4. **Notification Handling**
   - Test differences in notification content
   - Test differences in notification order
   - Test notification filtering

5. **Error Handling**
   - Test with malformed files
   - Test with empty files
   - Test with missing files

### Test File Patterns

#### Sample/Fixture Creation

Create test files using the txtar embedded file format:

```
-- identical1.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1000000000.100

-- identical2.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1000000000.100

-- different1.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1000000000.100

-- different2.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":false}}} # 1000000000.100
```

#### Command Execution

Execute mcpdiff with various options and verify output:

```
# Basic comparison (should show differences)
mcpdiff different1.mcp different2.mcp
stdout "server"
stdout "true"
stdout "false"
! stdout "No differences found"
! stderr "error"

# Identical file comparison (should not show differences)
mcpdiff identical1.mcp identical2.mcp
! stdout "Differences found"
! stderr "error"

# Test with flags
mcpdiff -ignore-timestamps -ignore-order different1.mcp different2.mcp
stdout "server"
```

#### Output Checking

Verify mcpdiff outputs using assertions:

- `stdout <pattern>`: Checks if stdout contains the pattern
- `! stdout <pattern>`: Checks if stdout does NOT contain the pattern
- `stderr <pattern>`: Checks if stderr contains the pattern
- `! stderr <pattern>`: Checks if stderr does NOT contain the pattern
- `stdout -count=N <pattern>`: Checks if stdout contains the pattern exactly N times
- `stdout -q <pattern>`: Quietly checks if stdout contains the pattern (no output)

### Advanced Testing Techniques

#### Pipeline Testing

Test mcpdiff in pipeline scenarios:

```
# Test with pipeline using bash
exec bash -c 'mcpdiff -json file1.mcp file2.mcp | jq .differences'
stdout "server"
```

#### Error Handling Testing

Verify mcpdiff properly handles errors:

```
# Test with nonexistent file
! mcpdiff file1.mcp nonexistent.mcp
stderr "error"

# Test with malformed file
! mcpdiff file1.mcp malformed.mcp
stderr "error parsing"
```

#### Format Testing

Verify various output formats:

```
# Test JSON output
mcpdiff --json file1.mcp file2.mcp
stdout '{"difference":'
stdout '"server"'
! stderr "error"
```

## Real-World Examples

### Notification Difference Testing

```
# Test notification differences
-- notif1.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} # 1000000000.100
mcp-send {"method":"notifications/message","params":{"level":"info","data":"Message 1"},"jsonrpc":"2.0"} # 1000000001.000

-- notif2.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} # 1000000000.100
mcp-send {"method":"notifications/message","params":{"level":"info","data":"Message 2"},"jsonrpc":"2.0"} # 1000000001.000

# Verify notification content differences
mcpdiff notif1.mcp notif2.mcp
stdout "Message 1"
stdout "Message 2"
stdout "notifications/message"
! stderr "error"
```

### Semantic Comparison Testing

```
# Test semantic JSON comparison
-- semantic1.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"a":1,"b":2}} # 1000000000.100

-- semantic2.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"b":2,"a":1}} # 1000000000.100

# Exact comparison should show differences due to field order
mcpdiff semantic1.mcp semantic2.mcp
stdout "Differences found"

# Semantic comparison should ignore field order
mcpdiff -semantic semantic1.mcp semantic2.mcp
! stdout "Differences found"
```

## Best Practices

1. **Use Self-Contained Tests**: Each test file should be independent and contain all needed fixtures.

2. **Test Both Positive and Negative Cases**: Verify correct behavior for both matching and differing files.

3. **Test Realistic Scenarios**: Use real-world MCP trace patterns to ensure practical functionality.

4. **Test All Options**: Create separate tests for each major flag/option.

5. **Test Error Handling**: Verify proper handling of edge cases, malformed inputs, etc.

6. **Keep Tests Declarative**: Scripttest is most effective when tests remain descriptive and declarative.

7. **Use Patterns in Assertions**: Use pattern matching in stdout/stderr assertions rather than exact matches when appropriate.

8. **Document Test Intent**: Add clear comments describing what each test is verifying.

## Extending Tests

When adding new functionality to mcpdiff:

1. First write tests that define the expected behavior (TDD approach)
2. Implement the feature to make the tests pass
3. Add any additional tests to cover edge cases

## Coverage Considerations

The mcpscripttest framework automatically integrates with Go's coverage tools:

```go
// Enable coverage reporting
mcpscripttest.SetupCoverageEnvironment(t)
```

Run tests with coverage:

```bash
cd exp && GOWORK=off go test -cover ./cmd/mcpdiff/...
# or for more detailed coverage
cd exp && GOWORK=off go test -coverprofile=cover.out ./cmd/mcpdiff/...
go tool cover -html=cover.out
```

## Conclusion

The scripttest approach provides a powerful, declarative way to test the mcpdiff tool. By creating comprehensive test scripts that cover various features, options, and edge cases, we ensure that mcpdiff reliably compares MCP trace files and accurately reports differences.
