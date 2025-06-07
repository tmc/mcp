# MCP Test Utilities

The `testutil` package provides testing helpers that make it easy to create connected server/client pairs for testing MCP implementations. The helpers handle logging configuration to ensure clean test output.

## Features

- **Automatic logging configuration**: Server logs are redirected to `t.Log()` for clean test output
- **Respects `testing.Verbose()`**: Only error logs and above are shown unless `-v` is used
- **Easy server/client pair creation**: Simplifies setup for integration tests
- **Type-safe configurations**: Provides helpers for common test scenarios

## ServerClientPair Helper

The `NewServerClientPair` function creates a connected server and client pair using bidirectional pipes.

### Basic Usage

```go
package mypackage_test

import (
    "context"
    "testing"
    
    "github.com/tmc/mcp"
    "github.com/tmc/mcp/testutil"
)

func TestMyServer(t *testing.T) {
    // Create your server
    server := mcp.NewServer("my-server", "1.0.0")
    
    // Register tools, prompts, etc.
    server.RegisterTool(myTool, myHandler)
    
    // Create connected pair
    ctx := context.Background()
    pair, err := testutil.NewServerClientPair(t, ctx, server)
    if err != nil {
        t.Fatal(err)
    }
    defer pair.Cleanup() // Always clean up!
    
    // Use pair.Client to interact with server
    result, err := pair.Client.CallTool(ctx, request)
    // Test assertions...
}
```

### Logging Behavior

The test logger follows these rules:

- **Without `-v`**: Shows INFO, WARN, and ERROR logs
- **With `-v`**: Shows all logs (DEBUG and above)
- **With `MCP_TEST_DEBUG=1`**: Always shows DEBUG logs, regardless of `-v`
- Logs are formatted as: `[LEVEL] message`

You can control logging with:
```bash
# Run with verbose output
go test -v

# Always show debug logs
MCP_TEST_DEBUG=1 go test

# Combine both
MCP_TEST_DEBUG=1 go test -v
```

## TestServerConfig Helper

For more complex test scenarios:

```go
config := &testutil.TestServerConfig{
    Name:    "test-server",
    Version: "1.0.0",
    Options: []mcp.ServerOption{
        mcp.WithServerInstructions("Test instructions"),
    },
}

server := testutil.NewTestServer(t, config)
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
    }
    
    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Tool implementation
    }
    
    server.RegisterTool(tool, handler)
    
    // Test with client
    pair, err := testutil.NewServerClientPair(t, context.Background(), server)
    require.NoError(t, err)
    defer pair.Cleanup()
    
    // Use the client...
}
```

### 2. Testing with Custom Logger Level

```go
func TestWithCustomLogging(t *testing.T) {
    // Only show WARN and ERROR logs
    server := mcp.NewServer("server", "1.0.0",
        testutil.WithTestLogger(t, slog.LevelWarn),
    )

    // Or use environment variable
    // MCP_TEST_DEBUG=1 go test  # Forces debug logging

    // Continue with test...
}
```

### 3. Testing Error Scenarios

Error logs are always visible, ensuring you never miss important issues:

```go
func TestErrorHandling(t *testing.T) {
    server := mcp.NewServer("server", "1.0.0")
    
    pair, err := testutil.NewServerClientPair(t, context.Background(), server)
    require.NoError(t, err)
    defer pair.Cleanup()
    
    // This error will be logged even without -v
    _, err = pair.Client.CallTool(ctx, mcp.CallToolRequest{
        Name: "nonexistent",
    })
    
    // Error logs will appear in test output
    require.Error(t, err)
}
```

## Migration from Inline Helpers

If you previously had testing helpers in your main package:

1. Update imports:
   ```go
   import "github.com/tmc/mcp/testutil"
   ```

2. Update function calls:
   ```go
   // Old:
   pair, err := mcp.NewServerClientPair(t, ctx, server)
   
   // New:
   pair, err := testutil.NewServerClientPair(t, ctx, server)
   ```

3. The API remains the same, just the package has changed.

## Best Practices

1. **Always defer Cleanup()**: Ensures resources are freed
2. **Use meaningful server names**: Helps with debugging
3. **Let the logger handle verbosity**: Don't add custom logging logic
4. **Test both success and error paths**: Error logs will always show
5. **Use `-v` during development**: See all debug logs when needed

## Comparison with golang-tools-internal-mcp

This package provides similar functionality to golang-tools helpers:

- Easy server/client pair creation
- Automatic transport setup  
- Clean test output management
- Type-safe test configurations

The key enhancement is the automatic integration with Go's testing framework for proper log handling.