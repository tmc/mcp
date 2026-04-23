# MCP Go Quickstart Guide

Get up and running with MCP Go in 5 minutes! This guide will walk you through creating your first MCP server and client.

## Prerequisites

- Go 1.21 or higher
- Basic understanding of Go programming
- Terminal/command line access

## Installation

```bash
go mod init your-mcp-project
go get github.com/tmc/mcp
```

## Quick Example: Echo Server

Let's create a simple MCP server that echoes back whatever you send it.

### Step 1: Create the Server

Create `main.go`:

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/tmc/mcp"
)

func main() {
    // Create a new MCP server
    server := mcp.NewServer("echo-server", "1.0.0")

    // Register an echo tool
    echoTool := mcp.Tool{
        Name:        "echo",
        Description: "Echo back the input text",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "text": {
                    "type": "string",
                    "description": "Text to echo back"
                }
            },
            "required": ["text"]
        }`),
    }

    server.RegisterTool(echoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Parse the input
        var input struct {
            Text string `json:"text"`
        }
        if err := json.Unmarshal(req.Arguments, &input); err != nil {
            return &mcp.CallToolResult{
                IsError: true,
                Content: []any{
                    map[string]string{
                        "type": "text",
                        "text": "Invalid input: " + err.Error(),
                    },
                },
            }, nil
        }

        // Return the echoed text
        return &mcp.CallToolResult{
            Content: []any{
                map[string]string{
                    "type": "text",
                    "text": "Echo: " + input.Text,
                },
            },
        }, nil
    })

    // Set up graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Listen for interrupt signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigChan
        log.Println("Shutting down...")
        cancel()
    }()

    // Start the server (uses stdio by default)
    log.Println("Starting echo server...")
    if err := server.Serve(ctx, nil); err != nil && err != context.Canceled {
        log.Fatal("Server error:", err)
    }
}
```

### Step 2: Build and Test

```bash
# Build the server
go build -o echo-server

# Test with mcp-connect (if available)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}' | ./echo-server

# Or test with the included tools
go run ./exp/cmd/mcp-connect -cmd="./echo-server"
```

## Quick Example: Using a Client

Create `client.go` to connect to your echo server:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os/exec"

    "github.com/tmc/mcp"
)

func main() {
    // Start the echo server as a subprocess
    cmd := exec.Command("./echo-server")
    
    // Create a transport that communicates with the subprocess
    transport := mcp.NewSubprocessTransport(cmd)
    
    // Create an MCP client
    client, err := mcp.NewClient(transport)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Initialize the connection
    ctx := context.Background()
    initRequest := mcp.InitializeRequest{
        ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
        ClientInfo: mcp.Implementation{
            Name:    "echo-client",
            Version: "1.0.0",
        },
        Capabilities: mcp.ClientCapabilities{},
    }

    initResult, err := client.Initialize(ctx, initRequest)
    if err != nil {
        log.Fatal("Failed to initialize:", err)
    }

    fmt.Printf("Connected to: %s v%s\n", 
        initResult.ServerInfo.Name, 
        initResult.ServerInfo.Version)

    // List available tools
    toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
    if err != nil {
        log.Fatal("Failed to list tools:", err)
    }

    fmt.Printf("Available tools: %d\n", len(toolsResult.Tools))
    for _, tool := range toolsResult.Tools {
        fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
    }

    // Call the echo tool
    echoArgs := map[string]string{
        "text": "Hello, MCP World!",
    }
    argsJSON, _ := json.Marshal(echoArgs)

    toolResult, err := client.CallTool(ctx, mcp.CallToolRequest{
        Name:      "echo",
        Arguments: argsJSON,
    })
    if err != nil {
        log.Fatal("Tool call failed:", err)
    }

    if toolResult.IsError {
        fmt.Println("Tool returned error:", toolResult.Content)
    } else {
        fmt.Println("Tool result:", toolResult.Content)
    }
}
```

## Testing Your Setup

### Option 1: Using Built-in Tools

If you have the MCP Go tools installed:

```bash
# Test server with mcp-connect
go run ./exp/cmd/mcp-connect -cmd="./echo-server"

# Test server with mcp-probe
(cd cmd/mcp-probe && GOWORK=off go run . ../../echo-server)
```

### Option 2: Manual Testing

Create a simple test script `test.sh`:

```bash
#!/bin/bash
# Send initialization request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}' | ./echo-server

# Send tool list request  
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./echo-server

# Send tool call request
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello MCP"}}}' | ./echo-server
```

## Common Patterns

### 1. Error Handling

```go
server.RegisterTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // For application errors, return them in the result
    return &mcp.CallToolResult{
        IsError: true,
        Content: []any{
            map[string]string{
                "type": "text", 
                "text": "Application error message",
            },
        },
    }, nil
    
    // For system errors, return Go errors
    // return nil, fmt.Errorf("system error: %w", err)
})
```

### 2. Multiple Content Types

```go
return &mcp.CallToolResult{
    Content: []any{
        map[string]string{
            "type": "text",
            "text": "Here's your result:",
        },
        map[string]any{
            "type": "text",
            "text": string(jsonData),
        },
    },
}, nil
```

### 3. Resource Registration

```go
// Register a simple resource
resource := mcp.Resource{
    URI:         "file:///etc/hosts",
    Description: "System hosts file",
    MimeType:    "text/plain",
}

server.RegisterResource(resource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    content, err := os.ReadFile("/etc/hosts")
    if err != nil {
        return nil, err
    }
    
    return []mcp.ResourceContents{
        mcp.TextResourceContents{
            URI:      req.URI,
            MimeType: "text/plain",
            Text:     string(content),
        },
    }, nil
})
```

## Next Steps

1. **Read the Architecture Guide**: Understanding the core concepts
2. **Try Different Transports**: Experiment with SSE and WebSocket
3. **Build Real Tools**: Create tools that interact with your systems
4. **Add Resources**: Expose data sources through the resource API
5. **Explore Examples**: Check out the examples in the repository

## Common Issues

### Server Won't Start
- Ensure Go version is 1.21+
- Check for port conflicts (SSE/WebSocket transports)
- Verify JSON-RPC message formatting

### Client Connection Issues
- Check subprocess execution permissions
- Verify transport configuration
- Ensure proper initialization sequence

### Tool Errors
- Validate JSON schema matches your input
- Check argument parsing logic
- Ensure proper error handling in tools

## Resources

- [Full Documentation](../README.md)
- [Architecture Overview](../architecture/overview.md)
- [Transport Guide](../architecture/transport.md)
- [Examples Repository](../../examples/)
- [Testing Guide](../testing/README.md)

You're now ready to build powerful MCP applications! The echo server demonstrates the core concepts, and you can extend it with real functionality for your use cases.
