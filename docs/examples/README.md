# MCP Examples

This section contains practical examples of MCP implementations, from simple servers to complex integrations.

## Example Categories

### 📦 [Server Examples](./servers.md)
Complete MCP server implementations:
- Echo Server - Simple request/response
- Time Server - Current time and timezone info
- Calculator Server - Mathematical operations
- File System Server - File access and manipulation
- SQLite Server - Database operations
- Everything Server - Comprehensive example with all features

### 🔌 [Client Examples](./clients.md)
MCP client implementations:
- Basic Client - Simple request/response
- Interactive Client - User interaction
- Batch Client - Bulk operations
- Streaming Client - Real-time updates

### 🎯 [Common Patterns](./patterns.md)
Frequently used patterns:
- Authentication & Authorization
- Error Handling
- Rate Limiting
- Caching
- Logging & Monitoring

### 🚀 [Transport Examples](./transports.md)
Transport-specific implementations:
- stdio Transport
- HTTP/SSE Transport
- WebSocket Transport
- Custom Transport

## Quick Start Examples

### Minimal Server

```go
package main

import (
    "github.com/tmc/mcp"
)

func main() {
    server := mcp.NewServer()
    
    // Add a simple tool
    server.AddTool("greet", mcp.ToolFunc(func(args map[string]any) (any, error) {
        name := args["name"].(string)
        return map[string]string{
            "message": "Hello, " + name + "!",
        }, nil
    }))
    
    // Start server on stdio
    server.Serve(mcp.NewStdioTransport())
}
```

### Minimal Client

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/mcp"
)

func main() {
    // Create client
    client := mcp.NewClient(mcp.NewStdioTransport())
    
    // Initialize
    if err := client.Initialize(context.Background()); err != nil {
        panic(err)
    }
    
    // Call tool
    result, err := client.CallTool(context.Background(), "greet", map[string]any{
        "name": "World",
    })
    
    fmt.Println(result)
}
```

## Complete Examples

### 1. Echo Server

A simple server that echoes back messages:

```go
// See examples/servers/mcp-echo-server/main.go
server := mcp.NewServer()

server.AddTool("echo", mcp.ToolFunc(func(args map[string]any) (any, error) {
    return args["message"], nil
}))
```

### 2. Calculator Server

Mathematical operations server:

```go
// See examples/servers/mcp-calculator-server/main.go
server.AddTool("calculate", &CalculatorTool{})

type CalculatorTool struct{}

func (c *CalculatorTool) Execute(args map[string]any) (any, error) {
    op := args["operation"].(string)
    a := args["a"].(float64)
    b := args["b"].(float64)
    
    switch op {
    case "add":
        return a + b, nil
    case "subtract":
        return a - b, nil
    // ... more operations
    }
}
```

### 3. File System Server

File access and manipulation:

```go
// See examples/servers/mcp-filesystem-server/main.go
server.AddResource("files", &FileSystemResource{
    Root: "/allowed/path",
})

server.AddTool("readFile", mcp.ToolFunc(func(args map[string]any) (any, error) {
    path := args["path"].(string)
    content, err := os.ReadFile(path)
    return string(content), err
}))
```

## Running Examples

### Using Go Run

```bash
# Run echo server
go run ./examples/servers/mcp-echo-server/main.go

# Run with mcp-spy for debugging
mcp-spy -v -- go run ./examples/servers/mcp-echo-server/main.go
```

### Building and Running

```bash
# Build server
go build -o echo-server ./examples/servers/mcp-echo-server

# Run server
./echo-server

# Connect with client
mcp-connect -cmd="./echo-server"
```

### Testing Examples

```bash
# Test with mock client
mcp-replay -mock-client test.mcp | ./echo-server

# Test with scripttest
cd examples/servers/mcp-echo-server
go test ./...
```

## Example Patterns

### Tool Registration

```go
// Function-based tool
server.AddTool("simple", mcp.ToolFunc(func(args map[string]any) (any, error) {
    return "result", nil
}))

// Struct-based tool
server.AddTool("complex", &ComplexTool{
    Config: config,
})

// Generic tool with type safety
server.AddGenericTool("typed", func(args struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}) (string, error) {
    return fmt.Sprintf("%s is %d years old", args.Name, args.Age), nil
})
```

### Resource Handling

```go
// Simple resource
server.AddResource("config", mcp.ResourceFunc(func() (any, error) {
    return loadConfig(), nil
}))

// Dynamic resource
server.AddResource("data/*", &DataResource{
    Pattern: "data/*",
    Handler: handleDataRequest,
})
```

### Error Handling

```go
server.AddTool("safe", mcp.ToolFunc(func(args map[string]any) (any, error) {
    value, ok := args["required"]
    if !ok {
        return nil, mcp.NewError(mcp.ErrorInvalidParams, "missing required field")
    }
    
    result, err := process(value)
    if err != nil {
        return nil, mcp.WrapError(err, "processing failed")
    }
    
    return result, nil
}))
```

## Best Practices from Examples

1. **Start Simple** - Begin with echo server, add complexity gradually
2. **Use Type Safety** - Leverage Go's type system with generics
3. **Handle Errors** - Always return appropriate MCP errors
4. **Document Tools** - Provide clear descriptions and schemas
5. **Test Thoroughly** - Use the testing tools provided
6. **Monitor Traffic** - Use mcp-spy during development

## Example Structure

Each example typically includes:

```
examples/servers/mcp-{name}-server/
├── main.go           # Server implementation
├── README.md         # Documentation
├── go.mod            # Dependencies
├── handlers.go       # Tool/resource handlers
├── types.go          # Type definitions
└── testdata/         # Test files
    ├── test.txt      # Scripttest files
    └── golden/       # Golden test files
```

## Next Steps

- Browse [Server Examples](./servers.md)
- Explore [Client Examples](./clients.md)
- Learn [Common Patterns](./patterns.md)
- Try [Transport Examples](./transports.md)

## Contributing Examples

To add new examples:

1. Create directory: `examples/{type}/mcp-{name}-{type}/`
2. Implement example following patterns
3. Add README with clear documentation
4. Include tests using scripttest
5. Update this index

## See Also

- [Getting Started](../getting-started/README.md)
- [Core Concepts](../concepts/README.md)
- [Testing Guide](../testing/README.md)
- [API Reference](../api/README.md)