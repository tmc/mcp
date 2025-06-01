# mcp-goast Use Cases for MCP Development

## Real Examples from the MCP Codebase

### 1. Automatic Handler Generation

**Current Pain Point**: Manually writing boilerplate for each new handler
```go
// Currently we write this manually:
func (s *Server) HandleToolCall(ctx context.Context, req *CallToolRequest) (*CallToolResponse, error) {
    // Validate request
    if req.Name == "" {
        return nil, fmt.Errorf("tool name required")
    }
    
    // Find tool
    tool, ok := s.tools[req.Name]
    if !ok {
        return nil, fmt.Errorf("tool not found: %s", req.Name)
    }
    
    // Execute tool
    result, err := tool.Call(ctx, req.Arguments)
    if err != nil {
        return nil, err
    }
    
    return &CallToolResponse{
        Content: result,
    }, nil
}
```

**With mcp-goast**:
```bash
# Analyze existing handlers to learn patterns
mcp-goast analyze-patterns ./server.go

# Generate new handler from tool definition
mcp-goast gen-handler weather-tool.json --pattern=CallTool

# Output: Complete handler with error handling, validation, and logging
```

### 2. Interface Implementation Discovery

**Current Pain Point**: Finding all transport implementations
```bash
# Find all implementations of Transport interface
mcp-goast impl Transport ./...

# Output:
# - StdioTransport in transport_stdio.go:45
# - SSETransport in transport_sse.go:78  
# - WebSocketTransport in transport_ws.go:23
```

### 3. Tool Definition Extraction

**Current Pain Point**: Keeping tool definitions in sync with code
```go
// Handler in code
func (s *Server) HandleGetTime(ctx context.Context, req *GetTimeRequest) (*GetTimeResponse, error) {
    return &GetTimeResponse{
        Time: time.Now().Format(time.RFC3339),
    }, nil
}
```

**With mcp-goast**:
```bash
# Extract tool definition from handler
mcp-goast extract-tool ./server.go:HandleGetTime

# Output: MCP tool definition JSON
{
    "name": "get_time",
    "description": "Returns the current time",
    "inputSchema": {
        "type": "object",
        "properties": {}
    },
    "outputSchema": {
        "type": "object",
        "properties": {
            "time": {"type": "string", "format": "date-time"}
        }
    }
}
```

### 4. Test Generation

**Current Pain Point**: Writing comprehensive tests for handlers
```bash
# Generate test cases from handler
mcp-goast gen-tests ./server.go:HandleToolCall

# Output: Test file with:
# - Happy path tests
# - Error cases (missing name, tool not found)
# - Edge cases (nil arguments, context cancellation)
# - Benchmarks
```

### 5. Schema Validation

**Current Pain Point**: Ensuring Go types match JSON schemas
```bash
# Validate types against schemas
mcp-goast validate-schemas ./types.go ./schemas/

# Output:
# ✓ CallToolRequest matches call_tool_request.json
# ✗ CallToolResponse missing field 'isError' from schema
# ✓ Tool matches tool.json
```

### 6. Dependency Analysis

**Current Pain Point**: Understanding impact of changes
```bash
# What depends on Transport interface?
mcp-goast deps Transport

# Output:
# Direct dependencies:
# - Server uses Transport (server.go:34)
# - Client uses Transport (client.go:45)
# 
# Implementations:
# - StdioTransport (transport_stdio.go:45)
# - SSETransport (transport_sse.go:78)
#
# Will affect 15 test files if changed
```

### 7. Migration Assistance

**Current Pain Point**: Updating code for protocol changes
```bash
# Migrate from v1 to v2 protocol
mcp-goast migrate --from=v1 --to=v2 ./...

# Changes:
# - Add 'protocol' field to all responses
# - Update error format to new structure
# - Add backward compatibility layer
# - Generate migration tests
```

### 8. Documentation Generation

**Current Pain Point**: Keeping docs in sync with code
```bash
# Generate documentation from code
mcp-goast gen-docs ./server.go

# Output: Markdown documentation with:
# - All public methods
# - Parameter descriptions from comments
# - Return types and error conditions
# - Example usage from tests
```

## Specific MCP Codebase Benefits

1. **Transport Layer Evolution**
   - Easily add new transport types
   - Ensure consistent interface implementation
   - Generate transport-specific tests

2. **Protocol Version Management**
   - Track breaking changes
   - Generate compatibility layers
   - Migrate existing code automatically

3. **Tool Registry Enhancement**
   - Extract tools from existing codebases
   - Validate tool definitions
   - Generate handler stubs

4. **Error Handling Standardization**
   - Analyze current error patterns
   - Generate consistent error handling
   - Create error type mappings

5. **Performance Optimization**
   - Identify allocation patterns
   - Generate benchmarks for hot paths
   - Suggest optimization opportunities

## Integration with Existing Tools

1. **With mcpscripttest**:
   ```bash
   # Generate script tests from handlers
   mcp-goast gen-scripttest ./server.go
   ```

2. **With mcp2go**:
   ```bash
   # Round-trip validation
   mcp-goast extract-tool ./handler.go | mcp2go - | mcp-goast validate -
   ```

3. **With mcpd**:
   ```bash
   # Generate server wrapper
   mcp-goast gen-server ./handlers/ | mcpd -
   ```

## ROI Calculation

Based on current MCP development patterns:

1. **Handler Development**: 
   - Current: 30-45 minutes per handler
   - With mcp-goast: 5 minutes
   - **Savings**: 85%

2. **Test Writing**:
   - Current: 1-2 hours per handler
   - With mcp-goast: 15 minutes
   - **Savings**: 80%

3. **Documentation**:
   - Current: Often skipped or outdated
   - With mcp-goast: Always current, auto-generated
   - **Quality**: 100% improvement

4. **Refactoring**:
   - Current: 2-3 days for protocol updates
   - With mcp-goast: 2-3 hours
   - **Savings**: 90%

Total estimated productivity gain: **3-5x for MCP development**