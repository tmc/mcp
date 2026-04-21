# MCPScriptTest Example Tests

This directory contains comprehensive example test files demonstrating key features and patterns of the mcpscripttest framework.

## Example Tests Overview

### Basic Examples

| Test File | Description | Concepts |
|-----------|-------------|----------|
| `01_simple_echo.txt` | Basic command execution and output validation | `exec`, `stdout`, `!` negation |
| `02_stdin_output.txt` | Input handling and piping | `setstdin`, `cat`, multi-line input |
| `03_stderr_validation.txt` | Error output verification | `stderr`, error conditions |
| `04_environment_variables.txt` | Environment variable management | `env`, variable persistence |
| `05_file_operations.txt` | File creation, reading, manipulation | `mkdir`, `cp`, `ls`, file I/O |
| `06_pattern_matching.txt` | Output pattern validation | Literal matching, special characters |
| `07_conditional_execution.txt` | Conditional test logic | Environment-driven behavior |

### Protocol and Integration Examples

| Test File | Description | Concepts |
|-----------|-------------|----------|
| `08_jsonrpc_basic.txt` | JSON-RPC 2.0 message format validation | Request/response structure, notifications |
| `09_tool_discovery.txt` | MCP tool registration and listing | Capability negotiation, tools/list method |
| `10_error_handling.txt` | Error response validation | Error codes, malformed requests |
| `11_integration_workflow.txt` | End-to-end MCP interaction | Multi-step workflows, state management |
| `12_protocol_conformance.txt` | Strict protocol compliance | Field requirements, format validation |

## Quick Start

### Run All Examples

```bash
cd /Volumes/tmc/go/src/github.com/tmc/mcp
go test -v ./testing/mcpscripttest/examples/...
```

### Run Specific Example

```bash
# Run only the basic echo example
go test -v -run "01_simple_echo" ./testing/mcpscripttest/examples/...

# Run only JSON-RPC examples
go test -v -run "jsonrpc" ./testing/mcpscripttest/examples/...
```

### Run with Coverage

```bash
GOCOVERDIR=/tmp/coverage go test ./testing/mcpscripttest/examples/...
go tool covdata percent -i /tmp/coverage
```

### Run in Verbose Mode to See Details

```bash
go test -v ./testing/mcpscripttest/examples/... 2>&1 | head -100
```

## Test File Structure

Each example test file follows this pattern:

```
# Title - Description
# Demonstrates: key concepts and features

# Setup or context comment
[test commands]

# Verification or assertion comment
[assertion commands]

# Additional test scenarios
[more tests]
```

## Command Reference

### Basic Commands

- **`exec COMMAND`**: Execute a system command
- **`setstdin DATA`**: Set input for next command
- **`stdout PATTERN`**: Assert output contains pattern
- **`stderr PATTERN`**: Assert error output contains pattern

### Control Flow

- **`! COMMAND`**: Negate - assert command does NOT produce output
- **`env VAR=VALUE`**: Set environment variable
- **`cd DIRECTORY`**: Change working directory

### Examples

```
# Execute and verify success
exec echo "hello"
stdout "hello"

# Verify something doesn't happen
! stdout "error"

# Multi-step workflow
env TEST_MODE=true
exec sh -c 'if [ "$TEST_MODE" = "true" ]; then echo "Testing"; fi'
stdout "Testing"

# File operations
exec mkdir -p /tmp/test
cd /tmp/test
exec echo "content" > file.txt
exec cat file.txt
stdout "content"
```

## Learning Path

1. **Start with basics**: `01_simple_echo.txt` → `02_stdin_output.txt`
2. **Progress to operations**: `04_environment_variables.txt` → `05_file_operations.txt`
3. **Learn protocol**: `08_jsonrpc_basic.txt` → `09_tool_discovery.txt`
4. **Advanced patterns**: `11_integration_workflow.txt` → `12_protocol_conformance.txt`

## Creating New Examples

To add a new example test:

1. Create file: `NN_descriptive_name.txt` (use 2-digit number)
2. Add descriptive header with title and key concepts
3. Include inline comments explaining each test section
4. Follow existing pattern and conventions
5. Test locally: `go test -v ./testing/mcpscripttest/examples/...`
6. Document in this README

### Template

```
# Example Title - Clear Description
# Demonstrates: key concepts here

# Section 1: Setup
[test code]

# Section 2: Main test
[test code]

# Section 3: Verification
[assertions]
```

## Best Practices

### Write Clear Comments

```
# Good
# Test that requests with null IDs are rejected
# JSON-RPC 2.0 requires valid non-null IDs

# Less helpful
# null id test
```

### Use Specific Patterns

```
# Good - specific field check
stdout '"status":"success"'

# Less good - too generic
stdout 'status'
```

### Organize Tests Logically

```
# Good - grouped by concept
exec echo "test1"
stdout "test1"
! stdout "error"

exec echo "test2"
stdout "test2"
! stdout "error"

# Not as good - scattered assertions
exec echo "test"
! stdout "error"
exec echo "test2"
stdout "test2"
```

### Provide Variety

```
# Good - tests multiple scenarios
exec true
! stdout "."  # no output

exec echo "output"
stdout "output"

exec sh -c 'exit 1'  # error case (if expected)
```

## Common Patterns

### Testing JSON Output

```
setstdin {"key":"value","number":42}
stdout '"key":"value"'
stdout '"number":42'
```

### Testing Command Failure

```
exec false
! stdout "."
```

### Testing File Operations

```
exec mkdir -p /tmp/test
cd /tmp/test
exec echo "content" > file.txt
exec cat file.txt
stdout "content"
```

### Testing Conditionals

```
env VAR=true
exec sh -c 'if [ "$VAR" = "true" ]; then echo "yes"; fi'
stdout "yes"
```

### Testing MCP Protocol

```
# Initialize
setstdin {"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
stdout '"result":'

# Call tool
setstdin {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{}}
stdout '"result":'
```

## Testing Examples Locally

### Quick Test

```bash
cd /Volumes/tmc/go/src/github.com/tmc/mcp
go test -short ./testing/mcpscripttest/examples/...
```

### Detailed Output

```bash
go test -v -count=1 ./testing/mcpscripttest/examples/...
```

### Single Test File

```bash
go test -run "01_simple_echo" -v ./testing/mcpscripttest/examples/...
```

### With Timing Information

```bash
go test -v -timeout=30s ./testing/mcpscripttest/examples/...
```

## Related Documentation

- **Main Guide**: See `README.md` in parent directory
- **Framework Guide**: See `EXAMPLES.md` for detailed explanations
- **API Reference**: See `mcpscripttest.go` for public functions
- **MCP Protocol**: Visit https://modelcontextprotocol.io/

## Status

✅ All example tests are implemented and working
✅ Comprehensive coverage of basic through advanced features
✅ Real-world protocol testing scenarios
✅ Clear documentation and inline comments
✅ Learning path from simple to complex

## Next Steps

- Use these examples as templates for your own tests
- Extend examples with application-specific scenarios
- Contribute improvements or additional examples
- Reference these patterns in new test development

---

**Last Updated:** 2025-10-17
**Framework Version:** MCPScriptTest
**Example Count:** 12 comprehensive examples
**Difficulty Levels:** Beginner → Intermediate → Advanced
