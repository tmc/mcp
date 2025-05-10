# Testing MCP Code with Scripttest

This document provides a guide to testing MCP clients and servers using the scripttest approach.

## Overview

The `mcp-scripttest-server` provides a way to create mock MCP servers with precisely controlled behavior. Combined with the scripttest testing framework, this allows for powerful, declarative testing of MCP components.

## Testing Pipeline

Here's a typical MCP testing pipeline using scripttest:

1. **Recording**: Capture real MCP traffic using `mcp-spy`
2. **Test Generation**: Convert recordings to scripttest files
3. **Validation**: Test MCP clients against scripttest server
4. **Regression Testing**: Ensure changes don't break existing functionality

## Basic Testing Workflow

### 1. Create a Scripttest File

Create a `.txt` file with the expected requests and responses:

```
# Initialize response
expect-recv {"jsonrpc":"2.0","method":"initialize"}
send {"jsonrpc":"2.0","id":${ID},"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"test-server","version":"1.0.0"}}}

# Echo functionality
expect-recv {"jsonrpc":"2.0","method":"echo","params":{"message":"*"}}
send {"jsonrpc":"2.0","id":${ID},"result":{"message":"${REQUEST.params.message}"}}
```

### 2. Start the Scripttest Server

```bash
mcp-scripttest-server -script test_server.txt -port 8080
```

### 3. Test Your Client Against the Server

```bash
# Test client initialization
mcp-send -addr localhost:8080 {"jsonrpc":"2.0","id":1,"method":"initialize"}

# Test echo functionality
mcp-send -addr localhost:8080 {"jsonrpc":"2.0","id":2,"method":"echo","params":{"message":"Hello"}}
```

### 4. Automate with Scripttest

For automated testing, create a scripttest file that tests your client against the server:

```
# Start the scripttest server
mcp-scripttest-server -script test_server.txt -port 8080 &
sleep 1

# Run client tests
mcp-send -addr localhost:8080 {"jsonrpc":"2.0","id":1,"method":"initialize"}
stdout '"name":"test-server"'

mcp-send -addr localhost:8080 {"jsonrpc":"2.0","id":2,"method":"echo","params":{"message":"Hello"}}
stdout '"message":"Hello"'

# Clean up
killall mcp-scripttest-server
```

Run this with:

```bash
mcp-test testfile.txt
```

## Advanced Testing Patterns

### Converting MCP Recordings to Scripttest Files

To convert recordings to scripttest files:

```bash
# Record an MCP conversation
mcp-spy -f recording.mcp -- mcp-some-client

# Convert recording to scripttest file
mcp-replay -f recording.mcp --generate-scripttest > test_server.txt
```

### Creating Fault Injection Tests

Testing error handling:

```
# Simulate a timeout
expect-recv {"jsonrpc":"2.0","method":"slow_operation"}
delay 5000
send {"jsonrpc":"2.0","id":${ID},"result":{"status":"success"}}

# Simulate a rate limit
expect-recv {"jsonrpc":"2.0","method":"rate_limited"}
counter RATE_LIMIT
if RATE_LIMIT > 3
  send {"jsonrpc":"2.0","id":${ID},"error":{"code":-32429,"message":"Rate limit exceeded"}}
else
  send {"jsonrpc":"2.0","id":${ID},"result":{"status":"success"}}
endif
```

### Testing Stateful Behavior

For tests with state:

```
# Write operation creates state
expect-recv {"jsonrpc":"2.0","method":"write","params":{"key":"*","value":"*"}}
set-var ${REQUEST.params.key} = ${REQUEST.params.value}
send {"jsonrpc":"2.0","id":${ID},"result":{"status":"success"}}

# Read operation uses state
expect-recv {"jsonrpc":"2.0","method":"read","params":{"key":"*"}}
send {"jsonrpc":"2.0","id":${ID},"result":{"value":"${${REQUEST.params.key}}"}}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: MCP Client Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install MCP tools
        run: |
          go install github.com/tmc/mcp/cmd/mcp-scripttest-server@latest
          go install github.com/tmc/mcp/cmd/mcp-test@latest
      
      - name: Run MCP tests
        run: mcp-test tests/*.txt
```

## Best Practices

1. **Organize Tests by Feature**: Create separate scripttest files for different functionality
2. **Use Variables Carefully**: Variables are powerful but can make tests harder to debug
3. **Include Setup and Cleanup**: Always clean up after tests, especially when using real resources
4. **Test Different Request Patterns**: Ensure your script handles variations in request formats
5. **Add Clear Comments**: Document what each test is checking
6. **Keep Tests Deterministic**: Avoid random values or time-dependent tests

## Troubleshooting

- **Hanging Tests**: Check if you're waiting for a request that never comes
- **Variable Substitution Issues**: Verify variable names and formats with `-v` mode
- **Path Problems**: Ensure paths are correct between different environments

## Advanced Scripttest Server Features

The scripttest server supports more advanced features:

- **Pattern Matching**: Use wildcards in request patterns
- **Dynamic Responses**: Generate responses based on request data
- **External Script Execution**: Run shell commands to manipulate the environment
- **Conditional Logic**: Respond differently based on conditions
- **Delay Simulation**: Simulate slow responses or timeouts

By using these features, you can create comprehensive tests for your MCP clients that cover a wide range of scenarios and edge cases.