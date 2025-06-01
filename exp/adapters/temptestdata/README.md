# MCP API Migration with Adapters

This directory demonstrates how to migrate servers written for different MCP implementations (mark3labs-mcp-go and golang-tools-internal-mcp) to the standard SDK using adapters.

## Single-Line Import Change

The key benefit of our adapter pattern is that you can migrate existing servers with just an import change:

### Example: mark3labs API Migration

**Before (using mark3labs API directly):**
```go
import (
    mark3labs "github.com/mark3labs/mcp-go/server"
    mark3labsmcp "github.com/mark3labs/mcp-go/mcp"
)

func main() {
    s := mark3labs.NewMCPServer("My Server", "1.0.0")
    // ... existing code ...
}
```

**After (using adapter with same API):**
```go
import (
    mark3labs "github.com/tmc/mcprepos/mcp/adapters/mark3labs"  // <- Single line change
    mark3labsmcp "github.com/tmc/mcprepos/mcp/protocol"        // <- Single line change
)

func main() {
    s := mark3labs.NewMCPServer("My Server", "1.0.0")
    // ... existing code works without changes ...
}
```

## Files in this Directory

1. **mark3labs_server.go** - Server written using original mark3labs API
2. **golang_tools_server.go** - Server written using golang-tools API
3. **api_server_with_adapter.go** - Example showing how minimal the import change is
4. **api_server_with_adapter_fixed.go** - Full adapter usage example

## How It Works

The adapters provide a compatibility layer that:
- Implements the same API surface as the original libraries
- Translates between implementation-specific types and protocol types
- Handles all the conversion logic transparently

This means you can:
1. Keep your existing server logic intact
2. Change only the imports
3. Get all the benefits of the standard SDK

## Benefits

- **Zero Code Changes**: Only imports need to be updated
- **Gradual Migration**: Migrate one server at a time
- **Compatibility**: Works with existing tools and clients
- **Future-Proof**: Use standard SDK features as they become available