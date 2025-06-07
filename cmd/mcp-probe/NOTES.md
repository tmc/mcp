# mcp-probe Implementation Notes

## JSON Marshaling Issue (FIXED)

~~The current implementation has a marshaling issue with jsonrpc2.Request types.~~ **This issue has been fixed.**

### Problem Description
The implementation was marshaling requests without the required `jsonrpc: "2.0"` field and the internal jsonrpc2 package already had proper JSON tags.

### Solution Implemented
Updated the Send methods in both StdioTransport and HTTPTransport to wrap requests in a proper JSON-RPC 2.0 message structure:

```go
msg := map[string]interface{}{
    "jsonrpc": "2.0",
    "id":      req.ID,
    "method":  req.Method,
    "params":  req.Params,
}
```

The output now correctly shows:
```json
{"id":1,"jsonrpc":"2.0","method":"initialize","params":{...}}
```

### Fixed Functionality
- ✅ mcp-probe sample output when run with no arguments
- ✅ Proper JSON-RPC 2.0 format with lowercase field names
- ✅ ID field properly marshaled as number instead of empty object
- ✅ Integration with servers expecting proper JSON-RPC format