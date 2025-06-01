# SDK2 - The Ultimate Stdlib-Idiomatic MCP API

SDK2 is a completely reimagined Go API for the Model Context Protocol (MCP) that follows Go standard library patterns religiously. It's designed to feel like a natural extension of Go's standard library rather than a foreign protocol implementation.

**This API demonstrates how to build protocol libraries that leverage existing Go knowledge and feel familiar to any Go developer.**

## Quick Start

### Server Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/tmc/mcp/exp/sdk2"
    "github.com/tmc/mcp/exp/sdk2/transport"
)

type EchoTool struct{}

func (e *EchoTool) Handle(ctx context.Context, arguments map[string]any) (*sdk2.ToolResult, error) {
    message := arguments["message"].(string)
    return &sdk2.ToolResult{
        Content: []sdk2.Content{
            sdk2.TextContent{Text: "Echo: " + message},
        },
    }, nil
}

func (e *EchoTool) Description() string {
    return "Echoes back the input message"
}

func (e *EchoTool) Schema() json.RawMessage {
    schema := map[string]any{
        "type": "object",
        "properties": map[string]any{
            "message": map[string]any{
                "type": "string",
                "description": "Message to echo",
            },
        },
        "required": []string{"message"},
    }
    bytes, _ := json.Marshal(schema)
    return json.RawMessage(bytes)
}

func main() {
    server := sdk2.NewServer("echo-server", "1.0.0")
    server.AddTool("echo", &EchoTool{})
    
    transport := transport.NewStdio()
    
    if err := server.Serve(context.Background(), transport); err != nil {
        log.Fatal(err)
    }
}
```

### Client Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/tmc/mcp/exp/sdk2"
    "github.com/tmc/mcp/exp/sdk2/transport"
)

func main() {
    // Connect to server (process, network, etc.)
    transport := transport.NewStdio()
    
    client := sdk2.NewClient(
        transport,
        sdk2.WithTimeout(10*time.Second),
        sdk2.WithRetries(3, time.Second),
    )
    defer client.Close()

    ctx := context.Background()

    // List available tools
    tools, err := client.ListTools(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for _, tool := range tools {
        fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
    }

    // Call a tool
    result, err := client.CallTool(ctx, "echo", map[string]any{
        "message": "Hello, MCP!",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %+v\n", result)
}
```

## Key Features

- **Stdlib Idioms**: Follows patterns from `net/http`, `database/sql`, etc.
- **Type Safety**: Strong typing with compile-time guarantees
- **Context Support**: First-class context support for cancellation and timeouts
- **Extensible**: Easy to add custom transports and handlers
- **Testable**: Mock transports and comprehensive test coverage

## Architecture

```
┌─────────────────┬─────────────────┐
│     Client      │     Server      │
├─────────────────┼─────────────────┤
│   Transport     │   Transport     │
└─────────────────┴─────────────────┘
```

### Interfaces

- `Client` - High-level client operations
- `Server` - Server with handler registration
- `Transport` - Pluggable communication layer
- `ToolHandler` - Function-like tool interface
- `ResourceHandler` - Resource content provider
- `PromptHandler` - Prompt template processor

### Transports

- **Stdio**: For subprocess communication
- **ReadWriteCloser**: Generic wrapper for any `io.ReadWriteCloser`
- **Extensible**: Easy to add WebSocket, TCP, HTTP, etc.

## Testing

Run the test suite:

```bash
go test ./...
```

Run integration tests:

```bash
go test -v ./... -run Integration
```

## Examples

The `examples/` directory contains:

- `calculator/` - Complete calculator server implementation
- `client/` - Client demonstrating tool usage

Run them:

```bash
# Terminal 1: Start calculator server
go run ./examples/calculator

# Terminal 2: Run client
go run ./examples/client
```

## Comparison with Main SDK

| Feature | Main SDK | SDK2 |
|---------|----------|------|
| API Style | Procedural | Interface-based |
| Type Safety | Mixed | Strong |
| Error Handling | Basic | Wrapped with context |
| Testing | Limited | Comprehensive |
| Extensibility | Moderate | High |
| Stdlib Idioms | Some | Throughout |

## Status

🚧 **Experimental** - Ready for experimentation, not production use.

## Contributing

This is an experimental package for exploring API design. Feedback and suggestions welcome!