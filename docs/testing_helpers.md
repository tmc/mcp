# MCP Testing Helpers

The MCP Go SDK provides testing helpers through the `testutil` package that make it easy to create connected server/client pairs for testing. This pattern is inspired by similar helpers in golang-tools-internal-mcp.

**Note**: The testing helpers have been moved to the `github.com/tmc/mcp/testutil` package for better organization. See [testutil documentation](testutil.md) for details.

## ServerClientPair Helper

The `NewServerClientPair` function creates a connected server and client pair using bidirectional pipes, which is useful for testing server implementations with real client interactions.

### Basic Usage

```go
package mcp_test

import (
    "context"
    "testing"
    
    "github.com/tmc/mcp"
)

func TestMyServer(t *testing.T) {
    // Create your server
    server := mcp.NewServer("my-server", "1.0.0")
    
    // Register tools, prompts, etc.
    server.RegisterTool(myTool, myHandler)
    
    // Create connected pair
    ctx := context.Background()
    pair, err := mcp.NewServerClientPair(ctx, server)
    if err != nil {
        t.Fatal(err)
    }
    defer pair.Cleanup() // Always clean up!
    
    // Use pair.Client to interact with server
    result, err := pair.Client.CallTool(ctx, request)
    if err != nil {
        t.Fatal(err)
    }
    
    // Test assertions...
}
```

### Features

1. **Automatic Connection Setup**: The helper handles all the transport setup and connection initialization.

2. **Bidirectional Communication**: Uses `net.Pipe()` to create fully bidirectional communication between server and client.

3. **Proper Cleanup**: The `Cleanup()` method ensures all resources are properly released.

4. **Context Support**: Full context support for cancellation and timeouts.

## TestServerConfig Helper

For more complex test scenarios, use `TestServerConfig` to configure servers:

```go
config := &mcp.TestServerConfig{
    Name:    "test-server",
    Version: "1.0.0",
    Options: []mcp.ServerOption{
        mcp.WithServerInstructions("Test server instructions"),
        mcp.WithCapabilities(capabilities),
    },
}

server := mcp.NewTestServer(config)
```

## Testing Patterns

### 1. Simple Tool Testing

```go
func TestToolExecution(t *testing.T) {
    server := mcp.NewServer("tool-server", "1.0.0")
    
    // Register a tool
    tool := mcp.Tool{
        Name:        "add",
        Description: "Add two numbers",
        InputSchema: []byte(`{"type": "object", "properties": {"a": {"type": "number"}, "b": {"type": "number"}}}`),
    }
    
    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Tool implementation
    }
    
    server.RegisterTool(tool, handler)
    
    // Test with client
    pair, err := mcp.NewServerClientPair(context.Background(), server)
    require.NoError(t, err)
    defer pair.Cleanup()
    
    result, err := pair.Client.CallTool(ctx, mcp.CallToolRequest{
        Name: "add",
        Arguments: []byte(`{"a": 5, "b": 3}`),
    })
    require.NoError(t, err)
    
    // Validate result
}
```

### 2. Testing Cancellation

```go
func TestCancellation(t *testing.T) {
    server := mcp.NewServer("slow-server", "1.0.0")
    
    // Register a slow operation
    server.RegisterTool(slowTool, slowHandler)
    
    pair, err := mcp.NewServerClientPair(context.Background(), server)
    require.NoError(t, err)
    defer pair.Cleanup()
    
    // Create cancellable context
    ctx, cancel := context.WithCancel(context.Background())
    
    // Start operation
    go func() {
        pair.Client.CallTool(ctx, request)
    }()
    
    // Cancel after delay
    time.Sleep(100 * time.Millisecond)
    cancel()
    
    // Verify cancellation handled properly
}
```

### 3. Testing with CancelCause

```go
func TestCancellationCause(t *testing.T) {
    server := mcp.NewServer("server", "1.0.0")
    
    pair, err := mcp.NewServerClientPair(context.Background(), server)
    require.NoError(t, err)
    defer pair.Cleanup()
    
    ctx, cancel := context.WithCancelCause(context.Background())
    
    go func() {
        pair.Client.CallTool(ctx, request)
    }()
    
    // Cancel with specific reason
    cancel(errors.New("user requested cancellation"))
    
    // The cancellation reason is automatically propagated
}
```

## Content Type Handling

When using the testing helpers, be aware that content types may be returned as maps rather than specific structs:

```go
// Tool handler returns typed content
return &mcp.CallToolResult{
    Content: []any{
        mcp.TextContent{
            Type: "text",
            Text: "Result",
        },
    },
}

// But client may receive it as map
if contentMap, ok := result.Content[0].(map[string]interface{}); ok {
    text := contentMap["text"].(string)
    // Use text...
}
```

## Best Practices

1. **Always defer Cleanup()**: This ensures resources are freed even if tests fail.

2. **Use meaningful server names**: This helps with debugging test output.

3. **Test error cases**: The helper makes it easy to test both success and failure scenarios.

4. **Use context for timeouts**: Pass contexts with timeouts to prevent hanging tests.

5. **Verify initialization**: The helper automatically initializes the client, but you can verify server capabilities if needed.

## Comparison with golang-tools-internal-mcp

This helper pattern is inspired by the golang-tools-internal-mcp approach:

- **Similar**: Both provide easy server/client pair creation for testing
- **Similar**: Both handle transport setup automatically
- **Different**: MCP SDK uses `net.Pipe()` while golang-tools may use different transports
- **Different**: Our helper includes automatic client initialization

The goal is to provide the same ease of testing while adapting to the specific needs of the MCP SDK.