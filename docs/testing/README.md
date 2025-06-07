# MCP Testing Guide

Comprehensive testing is crucial for building reliable MCP implementations. This guide covers various testing approaches and tools available in the MCP ecosystem.

## Testing Overview

MCP provides multiple testing strategies:

1. **[Unit Testing](./unit-testing.md)** - Test individual components
2. **[Integration Testing](./integration-testing.md)** - Test component interactions
3. **[Scripttest Testing](../scripttest/README.md)** - Behavior-driven testing
4. **[Mock Testing](./mocking.md)** - Test with mock servers/clients
5. **[Conformance Testing](./conformance.md)** - Protocol compliance testing

## Quick Start

### Basic Server Test

```go
func TestServer(t *testing.T) {
    server := mcp.NewServer(
        mcp.WithTool("echo", EchoTool{}),
    )
    
    // Test tool execution
    result, err := server.ExecuteTool(ctx, "echo", map[string]any{
        "message": "test",
    })
    
    assert.NoError(t, err)
    assert.Equal(t, "test", result)
}
```

### Integration Test with Tools

```bash
# Record reference behavior
mcp-spy -f reference.mcp -- ./reference-server

# Test new implementation
mcp-replay -mock-client reference.mcp | ./new-server | mcpdiff - reference.mcp
```

### Scripttest Example

```txtar
# Test echo server
exec mcp-connect -cmd="go run ./server/main.go"
stdin echo-request.json
stdout expected-response.json

-- echo-request.json --
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/execute",
  "params": {
    "name": "echo",
    "arguments": {"message": "hello"}
  }
}

-- expected-response.json --
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {"output": "hello"}
}
```

## Testing Tools

### Core Testing Tools

- **[mcp-spy](../tools/mcp-spy.md)** - Record protocol interactions
- **[mcp-replay](../tools/mcp-replay.md)** - Replay and mock sessions
- **[mcpdiff](../tools/mcpdiff.md)** - Compare protocol behaviors
- **[mcp-test](../tools/mcp-test.md)** - Run conformance tests

### Test Utilities

```go
// Test client
client := mcp.NewTestClient(t)
response := client.Call("method", params)

// Test server
server := mcp.NewTestServer(t)
server.HandleTool("calculate", func(args map[string]any) (any, error) {
    return args["a"].(float64) + args["b"].(float64), nil
})
```

## Testing Strategies

### 1. Protocol Testing

Test JSON-RPC compliance:
```go
func TestProtocol(t *testing.T) {
    // Test valid request
    resp := sendRequest(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
    assert.Contains(t, resp, `"jsonrpc":"2.0"`)
    
    // Test error handling
    resp = sendRequest(`{"invalid":"json"}`)
    assert.Contains(t, resp, `"error"`)
}
```

### 2. Transport Testing

Test different transports:
```go
func TestTransports(t *testing.T) {
    transports := []Transport{
        NewStdioTransport(),
        NewHTTPTransport(":0"),
        NewWebSocketTransport(":0"),
    }
    
    for _, transport := range transports {
        t.Run(transport.Name(), func(t *testing.T) {
            testTransport(t, transport)
        })
    }
}
```

### 3. Capability Testing

Test server capabilities:
```go
func TestCapabilities(t *testing.T) {
    server := createServer()
    caps := server.GetCapabilities()
    
    assert.Contains(t, caps.Tools, "calculate")
    assert.Contains(t, caps.Resources, "file:///data/*")
}
```

### 4. Error Testing

Test error scenarios:
```go
func TestErrors(t *testing.T) {
    // Test method not found
    resp := client.Call("nonexistent", nil)
    assert.Equal(t, -32601, resp.Error.Code)
    
    // Test invalid params
    resp = client.Call("calculate", map[string]any{"invalid": true})
    assert.Equal(t, -32602, resp.Error.Code)
}
```

## Test Patterns

### Golden File Testing

```go
func TestGolden(t *testing.T) {
    actual := server.Process(request)
    golden := filepath.Join("testdata", "golden.json")
    
    if *update {
        os.WriteFile(golden, actual, 0644)
    }
    
    expected, _ := os.ReadFile(golden)
    assert.Equal(t, expected, actual)
}
```

### Table-Driven Tests

```go
func TestCalculate(t *testing.T) {
    tests := []struct {
        name     string
        input    map[string]any
        expected float64
        wantErr  bool
    }{
        {"add", map[string]any{"a": 1, "b": 2}, 3, false},
        {"subtract", map[string]any{"a": 5, "b": 3}, 2, false},
        {"invalid", map[string]any{"a": "x"}, 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := calculate(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Mock Testing

```go
func TestWithMockServer(t *testing.T) {
    // Create mock server from trace
    mock := mcp.NewMockServer("testdata/server.mcp")
    
    // Test client against mock
    client := mcp.NewClient(mock.Transport())
    resp := client.Initialize()
    
    assert.Equal(t, "1.0", resp.Version)
}
```

## Coverage Analysis

### Generate Coverage

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

### Coverage Scripts

```bash
# Use coverage script
./scripts/run-coverage-tests.sh

# For specific packages
go test -cover -coverprofile=pkg.out ./pkg/...
```

See [Coverage Guide](../development/COVERAGE.md) for details.

## CI/CD Integration

### GitHub Actions

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
      - run: go test -race ./...
      - run: ./scripts/run-conformance-tests.sh
```

### Makefile Integration

```makefile
.PHONY: test
test:
    go test ./...
    
.PHONY: test-integration
test-integration:
    cd testdata && scripttest *.txt
    
.PHONY: test-conformance
test-conformance:
    ./run-conformance-tests.sh
```

## Best Practices

1. **Test at multiple levels** - Unit, integration, and system
2. **Use golden files** for complex outputs
3. **Mock external dependencies** for isolation
4. **Test error paths** thoroughly
5. **Use table-driven tests** for clarity
6. **Record real sessions** for regression testing
7. **Automate testing** in CI/CD pipelines

## Next Steps

- Read [Unit Testing Guide](./unit-testing.md)
- Learn [Integration Testing](./integration-testing.md)
- Explore [Scripttest](../scripttest/README.md)
- Try [Mock Testing](./mocking.md)

## See Also

- [Development Guide](../development/README.md)
- [Tool Documentation](../tools/README.md)
- [Examples](../examples/README.md)