# MCP Go Implementation - API Updates

This document tracks recent API changes and updates in the MCP Go implementation.

## Recent Changes (Latest)

### Handler Function Signatures

#### Resource Handlers
**Before:**
```go
func(ctx context.Context, req ReadResourceRequest) (*ReadResourceResult, error)
```

**After:**
```go
func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error)
```

The resource handler now returns a slice of `ResourceContents` directly instead of wrapping them in a `ReadResourceResult`.

#### Notification Handlers
**Before:**
```go
func(method string, params json.RawMessage) error
```

**After:**
```go
func(notification JSONRPCNotification)
```

Notification handlers now receive a `JSONRPCNotification` struct and don't return errors.

### New Error Constants

Added `ErrTransportClosed` to the common error values:
```go
var ErrTransportClosed = errors.New("mcp: transport closed")
```

### Custom JSON Marshaling

Added custom JSON unmarshaling for `ReadResourceResult` to handle polymorphic `ResourceContents` interface:
- Automatically detects content type based on JSON structure
- Supports `TextResourceContents`, `BlobResourceContents`, and `ImageResourceContents`

### Transport Updates

- `ReadWriteCloserTransport.Dial()` now returns `io.ErrClosedPipe` when the transport is nil
- Removed non-existent `String()` method references from transport types
- Added experimental WebSocket transport support

## Test Coverage Improvements

- Test coverage improved from 49.2% to 49.4%
- Fixed and enabled multiple comprehensive test suites:
  - `integration_comprehensive_test.go`
  - `transport_comprehensive_test.go`
  - `high_coverage_test.go`
  - `error_handling_comprehensive_test.go`

### Still Disabled Tests

Some tests remain disabled due to various issues:
- `notify_test.go.skip` - Dispatcher API changes
- `optimized_coverage_test.go.skip` - Uses unexported mcpscripttest types
- `ultra_aggressive_scripttest_coverage_test.go.skip.old` - Uses unexported types

## Handler Type Reference

### Core Handler Types
- `CallToolHandlerFunc`: `func(ctx context.Context, request CallToolRequest) (*CallToolResult, error)`
- `ReadResourceHandlerFunc`: `func(ctx context.Context, request ReadResourceRequest) ([]ResourceContents, error)`
- `GetPromptHandlerFunc`: `func(ctx context.Context, request GetPromptRequest) (*GetPromptResult, error)`

### Client Options
- `WithNotificationHandler(handler func(notification JSONRPCNotification))` - Sets notification handler

## Migration Guide

### Updating Resource Handlers

If you have existing resource handlers, update them to return `[]ResourceContents` directly:

```go
// Old
resourceHandler := func(ctx context.Context, req ReadResourceRequest) (*ReadResourceResult, error) {
    return &ReadResourceResult{
        Contents: []ResourceContents{
            TextResourceContents{
                URI:  req.URI,
                Text: "content",
            },
        },
    }, nil
}

// New
resourceHandler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
    return []ResourceContents{
        TextResourceContents{
            URI:  req.URI,
            Text: "content",
        },
    }, nil
}
```

### Updating Notification Handlers

Update notification handlers to use the new signature:

```go
// Old
client, err := NewClient(transport, WithNotificationHandler(
    MethodProgress,
    func(method string, params json.RawMessage) error {
        // handle notification
        return nil
    },
))

// New
client, err := NewClient(transport, WithNotificationHandler(func(notif JSONRPCNotification) {
    // handle notification
}))
```