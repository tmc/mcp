# MCP Testing Utilities

This package provides comprehensive testing utilities for MCP (Model Context Protocol) implementations.

## Package Structure

- `testutil.go` - Core testing utilities and configuration
- `mock.go` - Mock implementations for servers, clients, and transports
- `assertions.go` - Type-safe assertion helpers for MCP-specific types
- `helpers.go` - Helper functions for common testing scenarios

## Core Features

### Test Configuration

```go
config := testutil.DefaultTestConfig().
    WithTimeout(30 * time.Second).
    WithWorkingDir("/tmp/test").
    WithEnv("DEBUG", "true")

ctx, cancel := config.TestContext()
defer cancel()
```

### Connection Testing

```go
// Create pipe connections for client-server testing
conn := testutil.NewPipeConnection()
defer conn.Close()

// Use conn.Client and conn.Server for testing
```

### Temporary Resources

```go
// Create temporary directories
tempDir := testutil.CreateTempDir(t, "subdir1", "subdir2")

// Create temporary files
filePath := testutil.CreateTempFile(t, "test.json", `{"test": true}`)
```

## Mock Implementations

### Mock Server

```go
server := testutil.NewMockServer()
server.SetServerInfo(protocol.Implementation{
    Name:    "test-server",
    Version: "1.0.0",
})

// Add tools, resources, and prompts
server.AddTool(testutil.CreateSampleTool("test-tool", "A test tool"))
server.AddResource(testutil.CreateSampleResource("file://test.txt", "test", "Test resource", "text/plain"))

// Set custom request handlers
server.SetRequestHandler("tools/call", func(req interface{}) (interface{}, error) {
    return &protocol.CallToolResult{
        Content: []protocol.Content{
            testutil.CreateTextContent("Custom response"),
        },
    }, nil
})
```

### Mock Client

```go
client := testutil.NewMockClient()

// Set mock responses
client.SetResponse("tools/list", []*protocol.Tool{
    testutil.CreateSampleTool("mock-tool", "Mock tool"),
})

// Set mock errors
client.SetError("tools/call", errors.New("tool execution failed"))

// Check call history
history := client.GetCallHistory()
testutil.AssertStringSliceEqual(t, history, []string{"tools/list", "tools/call"})
```

### Mock Transport

```go
transport := testutil.NewMockTransportConn()
transport.SetReadData([]byte("test data"))
transport.SetReadDelay(100 * time.Millisecond)

// Use transport in your tests
data := make([]byte, 100)
n, err := transport.Read(data)
```

## Assertion Helpers

### Generic Assertions

```go
testutil.AssertNoError(t, err)
testutil.AssertError(t, err)
testutil.AssertEqual(t, actual, expected)
testutil.AssertNotEqual(t, actual, unexpected)
testutil.AssertContains(t, "hello world", "hello")
```

### MCP-Specific Assertions

```go
// Compare tools
testutil.AssertToolEqual(t, actualTool, expectedTool)

// Compare resources
testutil.AssertResourceEqual(t, actualResource, expectedResource)

// Compare prompts
testutil.AssertPromptEqual(t, actualPrompt, expectedPrompt)

// Compare content arrays
testutil.AssertContentEqual(t, actualContent, expectedContent)

// JSON comparison (handles field ordering)
testutil.JSONEqual(t, actualJSON, expectedJSON)
```

### Advanced Assertions

```go
// Error type checking
mcpErr := testutil.AssertErrorType[*protocol.MCPError](t, err)

// Interface implementation checking
testutil.AssertImplementsInterface[io.Reader](t, myReader)

// Panic testing
testutil.AssertPanic(t, func() { panic("test") })
testutil.AssertNoPanic(t, func() { /* safe code */ })
```

## Helper Functions

### Sample Data Creation

```go
// Create sample protocol objects
tool := testutil.CreateSampleTool("calculator", "A calculator tool")
resource := testutil.CreateSampleResource("file://data.json", "data", "Test data", "application/json")
prompt := testutil.CreateSamplePrompt("summarize", "Summarize text", 
    testutil.CreatePromptArgument("text", "Text to summarize", true))

// Create content objects
textContent := testutil.CreateTextContent("Hello, world!")
imageContent := testutil.CreateImageContent("base64data", "image/png")
resourceContent := testutil.CreateResourceContent("file://test.txt")
```

### Protocol Messages

```go
// Create initialize request/result
initReq := testutil.CreateInitializeRequest("test-client", "1.0.0")
initResult := testutil.CreateInitializeResult("test-server", "1.0.0")

// Create tool call request/result
toolReq := testutil.CreateCallToolRequest("calculator", map[string]interface{}{
    "operation": "add",
    "a": 5,
    "b": 3,
})
toolResult := testutil.CreateCallToolResult(
    testutil.CreateTextContent("Result: 8"),
)
```

### Utility Functions

```go
// JSON handling
data := testutil.ParseJSON(t, `{"test": true}`)
jsonStr := testutil.ToJSON(t, data)
prettyJSON := testutil.PrettyJSON(t, data)

// Environment variables
testutil.SetEnv(t, map[string]string{
    "TEST_MODE": "true",
    "DEBUG": "false",
})
value := testutil.RequireEnv(t, "REQUIRED_VAR")

// Conditional execution
testutil.WaitForCondition(t, func() bool {
    return server.IsReady()
}, 5*time.Second, "server not ready")

testutil.RetryUntilSuccess(t, func() error {
    return client.Connect()
}, 10*time.Second, "client connection failed")
```

## Table-Driven Tests

```go
// Anonymous table tests
tests := []struct {
    name     string
    input    string
    expected int
}{
    {"positive", "123", 123},
    {"negative", "-456", -456},
    {"zero", "0", 0},
}

testutil.TableTest(t, tests, func(t *testing.T, tc struct{...}) {
    result, err := ParseInt(tc.input)
    testutil.AssertNoError(t, err)
    testutil.AssertEqual(t, result, tc.expected)
})

// Named table tests
namedTests := map[string]TestCase{
    "simple_addition": {a: 1, b: 2, expected: 3},
    "with_negatives":  {a: -1, b: 1, expected: 0},
}

testutil.NamedTableTest(t, namedTests, func(t *testing.T, tc TestCase) {
    result := tc.a + tc.b
    testutil.AssertEqual(t, result, tc.expected)
})
```

## Script Testing

```go
engine := testutil.NewMCPScriptEngine()
engine.Test(t, "testdata/scripts/*.txt")
```

## Best Practices

1. **Use type-safe assertions** - Prefer `AssertEqual[T]` over generic comparisons
2. **Clean up resources** - Use `t.Cleanup()` or defer statements
3. **Use contexts with timeouts** - Always set reasonable timeouts for async operations
4. **Mock external dependencies** - Use mock implementations for isolated unit tests
5. **Test error conditions** - Ensure error paths are properly tested
6. **Use table-driven tests** - Organize multiple test cases efficiently
7. **Provide descriptive test names** - Make test failures easy to understand

## Examples

See the `examples/` directory for complete examples of using these testing utilities in different scenarios:

- Unit testing MCP servers and clients
- Integration testing with mock transports
- Protocol compliance testing
- Performance testing with timeouts and retries