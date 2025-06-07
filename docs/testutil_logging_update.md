# Test Logging Update

## Summary of Changes

The test logging system has been updated to provide a better default experience:

### Previous Behavior
- Without `-v`: Only ERROR logs visible
- With `-v`: All logs visible

### New Behavior  
- Without `-v`: INFO, WARN, and ERROR logs visible
- With `-v`: All logs visible (DEBUG and above)
- With `MCP_TEST_DEBUG=1`: Forces DEBUG logging regardless of `-v`

### Rationale
- INFO logs are useful for understanding test flow and major events
- DEBUG logs can be overwhelming but are valuable for troubleshooting
- The environment variable provides flexibility for debugging production issues

### New Helper Function

Added `TestLogger(t *testing.T)` for creating loggers in test handlers:

```go
func TestWithCustomLogger(t *testing.T) {
    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        logger := testutil.TestLogger(t)
        logger.Debug("Debug info")  // Shown with -v or MCP_TEST_DEBUG=1
        logger.Info("Important event")  // Always shown
        logger.Error("Problem occurred")  // Always shown
        return result, nil
    }
}
```

### Usage Examples

```bash
# Default behavior - see INFO and above
go test ./mypackage

# Verbose mode - see all logs including DEBUG
go test -v ./mypackage

# Force debug logs without verbose test output
MCP_TEST_DEBUG=1 go test ./mypackage

# Combine both for maximum output
MCP_TEST_DEBUG=1 go test -v ./mypackage
```

### Migration

No code changes required for existing tests. The new behavior is automatically applied to all tests using `testutil.NewServerClientPair` or `testutil.WithTestLogger`.