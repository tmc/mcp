# mcp-probe Implementation Notes

## JSON Marshaling Issue

The current implementation has a marshaling issue with jsonrpc2.Request types. When marshaling requests like:

```go
initReq := &jsonrpc2.Request{
    ID:     jsonrpc2.Int64ID(1),
    Method: "initialize", 
    Params: json.RawMessage(`{...}`),
}
```

The output incorrectly shows empty objects for ID:
```json
{"ID":{},"Method":"initialize","Params":{...}}
```

Instead of the expected:
```json
{"id":1,"method":"initialize","params":{...}}
```

### Root Cause
The golang.org/x/exp/jsonrpc2 package uses custom ID types that don't marshal correctly with standard json.Marshal(). The ID field should be lowercase "id" and should render the actual value (1) rather than an empty object.

### TODO: Fix Required
1. Check if we're using the correct JSON tags on the Request struct
2. Verify the golang.org/x/exp/jsonrpc2.ID type marshaling behavior
3. Consider using a custom marshaler or switching to the internal jsonrpc2 implementation
4. Ensure JSON-RPC field names are lowercase as per spec (id, method, params)

### Affected Functionality
- mcp-probe sample output when run with no arguments
- Any test that validates JSON output format
- Integration with servers expecting proper JSON-RPC format