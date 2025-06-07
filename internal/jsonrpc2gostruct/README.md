# JSON-RPC 2.0 to Go Struct Converter

This utility converts JSON-RPC 2.0 schemas, requests, and responses to Go struct definitions, making it easy to maintain type alignment between implementations.

## Features

- Convert JSON-RPC requests to Go structs
- Generate structs from JSON schema definitions
- Batch process multiple schema files
- Infer types from JSON data
- Create idiomatic Go code with proper naming conventions
- Add appropriate JSON tags with omitempty for optional fields

## Command-Line Usage

```bash
# Generate struct from a JSON-RPC request
go run cmd/jsonrpc2gostruct/main.go -package mcptools examples/call_tool_request.json

# Generate struct to a file
go run cmd/jsonrpc2gostruct/main.go -package mcptools -out calculator_request.go examples/call_tool_request.json

# Process multiple files at once
go run cmd/jsonrpc2gostruct/main.go -batch -dir examples -pattern "*.json" -package mcptypes -out generated_types.go
```

## Programmatic Usage

```go
import "github.com/tmc/mcp/internal/jsonrpc2gostruct"

// Convert a JSON-RPC request to a Go struct
func generateFromRequest() {
    jsonrpcRequest := []byte(`{
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "calculator",
            "arguments": {
                "operation": "add",
                "a": 5,
                "b": 3
            }
        }
    }`)
    
    goCode, err := jsonrpc2gostruct.ParseJSONRPCRequestToStruct(
        jsonrpcRequest, 
        "mypackage", 
        "ToolsCall"
    )
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    fmt.Println(goCode)
}

// Generate structs from multiple schemas
func generateMultipleStructs() {
    schemas := map[string][]byte{
        "CallToolRequest": []byte(`{
            "type": "object",
            "properties": {
                "name": {"type": "string"},
                "arguments": {"type": "object"}
            },
            "required": ["name"]
        }`),
        "ReadResourceRequest": []byte(`{
            "type": "object",
            "properties": {
                "uri": {"type": "string"}
            },
            "required": ["uri"]
        }`),
    }
    
    goCode, err := jsonrpc2gostruct.GenerateMultipleStructs(schemas, "mcptypes")
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    
    fmt.Println(goCode)
}
```

## Benefits for MCP Implementation

- Ensures type consistency between implementations
- Simplifies maintaining compatibility with the JSON-RPC specification
- Reduces manual effort in creating and updating type definitions
- Provides a single source of truth for schema definitions
- Makes it easy to track changes between protocol versions

## Integration with Other Tools

This utility can be used in combination with other code generation tools like:

- Swagger/OpenAPI generators
- Protocol buffer compilers
- JSON Schema validators

By using this tool, you can ensure that your Go implementation of the Model Context Protocol stays in sync with the official specification and other language implementations.