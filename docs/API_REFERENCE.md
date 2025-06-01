# MCP Go SDK API Reference

## Overview

The Model Context Protocol (MCP) Go SDK provides a complete, type-safe implementation of the MCP specification for building both clients and servers. This reference covers all public APIs, types, and usage patterns.

## Table of Contents

- [Getting Started](#getting-started)
- [Core Types](#core-types)
- [Client API](#client-api) 
- [Server API](#server-api)
- [Transport Layer](#transport-layer)
- [Error Handling](#error-handling)
- [Advanced Usage](#advanced-usage)

## Getting Started

### Installation

```bash
go get github.com/tmc/mcp
```

### Basic Server Example

```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/tmc/mcp"
)

func main() {
    // Create a new MCP server
    server := mcp.NewServer("my-server", "1.0.0")
    
    // Register a tool
    tool := mcp.Tool{
        Name:        "echo",
        Description: "Echoes the input message",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "message": map[string]interface{}{
                    "type":        "string",
                    "description": "Message to echo",
                },
            },
            "required": []string{"message"},
        },
    }
    
    handler := func(ctx context.Context, req mcp.CallToolRequest) (mcp.CallToolResult, error) {
        message := req.Arguments["message"].(string)
        return mcp.CallToolResult{
            Content: []interface{}{
                map[string]interface{}{
                    "type": "text",
                    "text": fmt.Sprintf("Echo: %s", message),
                },
            },
        }, nil
    }
    
    server.RegisterTool(tool, handler)
    
    // Serve over stdio
    transport := &mcp.ReadWriteCloserTransport{os.Stdin}
    if err := server.Serve(context.Background(), transport); err != nil {
        log.Fatal(err)
    }
}
```

### Basic Client Example

```go
package main

import (
    "context"
    "log"
    "os/exec"
    
    "github.com/tmc/mcp"
)

func main() {
    // Launch server process
    cmd := exec.Command("./my-mcp-server")
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()
    
    // Create transport
    transport := &mcp.ProcessTransport{
        Stdin:  stdin,
        Stdout: stdout,
    }
    
    // Create client
    client, err := mcp.NewClient(transport)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Initialize connection
    err = client.Initialize(context.Background(), mcp.InitializeRequest{
        ProtocolVersion: mcp.PROTOCOL_VERSION,
        Capabilities:    mcp.ClientCapabilities{},
        ClientInfo: &mcp.Implementation{
            Name:    "my-client",
            Version: "1.0.0",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Call a tool
    result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
        Name: "echo",
        Arguments: map[string]interface{}{
            "message": "Hello, MCP!",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Tool result: %+v\n", result)
}
```

## Core Types

### Method

Protocol method constants for MCP operations.

```go
type Method string

const (
    MethodInitialize                Method = "initialize"
    MethodInitialized               Method = "initialized"
    MethodPing                      Method = "ping"
    MethodToolsList                 Method = "tools/list"
    MethodToolsCall                 Method = "tools/call"
    MethodPromptsList               Method = "prompts/list"
    MethodPromptsGet                Method = "prompts/get"
    MethodResourcesList             Method = "resources/list"
    MethodResourcesRead             Method = "resources/read"
    MethodResourceTemplatesList     Method = "resources/templates/list"
    MethodNotificationCancelled     Method = "notifications/cancelled"
)
```

### Implementation

Represents implementation information for clients and servers.

```go
type Implementation struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}
```

## Client API

### Client

The main client type for MCP communication.

```go
type Client struct {
    // Internal fields...
}
```

#### NewClient

Creates a new MCP client with the specified transport.

```go
func NewClient(transport Transport, opts ...ClientOption) (*Client, error)
```

**Parameters:**
- `transport`: The transport layer for communication
- `opts`: Optional configuration options

**Returns:**
- `*Client`: The created client instance
- `error`: Any initialization error

**Example:**
```go
transport := &mcp.ReadWriteCloserTransport{conn}
client, err := mcp.NewClient(transport)
```

#### Initialize

Initializes the MCP session with the server.

```go
func (c *Client) Initialize(ctx context.Context, req InitializeRequest) error
```

**Parameters:**
- `ctx`: Context for the operation
- `req`: Initialization request parameters

**Returns:**
- `error`: Any initialization error

**Example:**
```go
err := client.Initialize(ctx, mcp.InitializeRequest{
    ProtocolVersion: mcp.PROTOCOL_VERSION,
    Capabilities:    mcp.ClientCapabilities{},
    ClientInfo: &mcp.Implementation{
        Name:    "my-client",
        Version: "1.0.0",
    },
})
```

#### CallTool

Calls a tool on the server.

```go
func (c *Client) CallTool(ctx context.Context, req CallToolRequest) (CallToolResult, error)
```

**Parameters:**
- `ctx`: Context for the operation
- `req`: Tool call request

**Returns:**
- `CallToolResult`: The tool execution result
- `error`: Any execution error

**Example:**
```go
result, err := client.CallTool(ctx, mcp.CallToolRequest{
    Name: "calculator",
    Arguments: map[string]interface{}{
        "operation": "add",
        "a": 5,
        "b": 3,
    },
})
```

#### ListTools

Lists available tools from the server.

```go
func (c *Client) ListTools(ctx context.Context, req ListToolsRequest) (ListToolsResult, error)
```

#### ReadResource

Reads a resource from the server.

```go
func (c *Client) ReadResource(ctx context.Context, req ReadResourceRequest) (ReadResourceResult, error)
```

#### ListResources

Lists available resources from the server.

```go
func (c *Client) ListResources(ctx context.Context, req ListResourcesRequest) (ListResourcesResult, error)
```

#### GetPrompt

Gets a prompt from the server.

```go
func (c *Client) GetPrompt(ctx context.Context, req GetPromptRequest) (GetPromptResult, error)
```

#### ListPrompts

Lists available prompts from the server.

```go
func (c *Client) ListPrompts(ctx context.Context, req ListPromptsRequest) (ListPromptsResult, error)
```

#### Close

Closes the client connection.

```go
func (c *Client) Close() error
```

### Client Options

#### WithClientNotificationHandler

Sets a notification handler for the client.

```go
func WithClientNotificationHandler(handler NotificationHandler) ClientOption
```

**Example:**
```go
handler := func(notification mcp.JSONRPCNotification) {
    log.Printf("Received notification: %s", notification.Method)
}
client, err := mcp.NewClient(transport, mcp.WithClientNotificationHandler(handler))
```

## Server API

### Server

The main server type for MCP communication.

```go
type Server struct {
    // Internal fields...
}
```

#### NewServer

Creates a new MCP server.

```go
func NewServer(name, version string, opts ...ServerOption) *Server
```

**Parameters:**
- `name`: Server name
- `version`: Server version
- `opts`: Optional configuration options

**Returns:**
- `*Server`: The created server instance

**Example:**
```go
server := mcp.NewServer("my-server", "1.0.0")
```

#### RegisterTool

Registers a tool with the server.

```go
func (s *Server) RegisterTool(tool Tool, handler ToolHandlerFunc) error
```

**Parameters:**
- `tool`: Tool definition
- `handler`: Function to handle tool calls

**Returns:**
- `error`: Registration error if any

**Example:**
```go
tool := mcp.Tool{
    Name:        "greet",
    Description: "Greets a person",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "name": map[string]interface{}{
                "type": "string",
            },
        },
    },
}

handler := func(ctx context.Context, req mcp.CallToolRequest) (mcp.CallToolResult, error) {
    name := req.Arguments["name"].(string)
    return mcp.CallToolResult{
        Content: []interface{}{
            map[string]interface{}{
                "type": "text",
                "text": fmt.Sprintf("Hello, %s!", name),
            },
        },
    }, nil
}

err := server.RegisterTool(tool, handler)
```

#### RegisterResource

Registers a resource with the server.

```go
func (s *Server) RegisterResource(resource Resource, handler ReadResourceHandlerFunc) error
```

#### RegisterResourceTemplate

Registers a resource template with the server.

```go
func (s *Server) RegisterResourceTemplate(template ResourceTemplate, handler ResourceTemplateHandlerFunc) error
```

#### RegisterPrompt

Registers a prompt with the server.

```go
func (s *Server) RegisterPrompt(prompt Prompt, handler GetPromptHandlerFunc) error
```

#### Serve

Starts serving MCP requests over the specified transport.

```go
func (s *Server) Serve(ctx context.Context, transport Transport) error
```

**Parameters:**
- `ctx`: Context for the server operation
- `transport`: Transport layer for communication

**Returns:**
- `error`: Any serving error

**Example:**
```go
transport := &mcp.ReadWriteCloserTransport{os.Stdin}
err := server.Serve(context.Background(), transport)
```

### Handler Function Types

#### ToolHandlerFunc

Handler function for tool calls.

```go
type ToolHandlerFunc func(ctx context.Context, req CallToolRequest) (CallToolResult, error)
```

#### ReadResourceHandlerFunc

Handler function for resource reads.

```go
type ReadResourceHandlerFunc func(ctx context.Context, req ReadResourceRequest) (ReadResourceResult, error)
```

#### ResourceTemplateHandlerFunc

Handler function for resource template operations.

```go
type ResourceTemplateHandlerFunc func(ctx context.Context, req ResourceTemplateRequest) (ResourceTemplateResult, error)
```

#### GetPromptHandlerFunc

Handler function for prompt requests.

```go
type GetPromptHandlerFunc func(ctx context.Context, req GetPromptRequest) (GetPromptResult, error)
```

### Server Options

#### WithServerName

Sets a custom server name.

```go
func WithServerName(name string) ServerOption
```

#### WithServerVersion

Sets a custom server version.

```go
func WithServerVersion(version string) ServerOption
```

#### WithServerInstructions

Sets custom server instructions.

```go
func WithServerInstructions(instructions string) ServerOption
```

## Transport Layer

### Transport Interface

Base interface for MCP transport implementations.

```go
type Transport interface {
    // Implementation specific
}
```

### ReadWriteCloserTransport

Transport implementation for io.ReadWriteCloser.

```go
type ReadWriteCloserTransport struct {
    io.ReadWriteCloser
}
```

**Example:**
```go
// Stdio transport
transport := &mcp.ReadWriteCloserTransport{os.Stdin}

// Network transport
conn, err := net.Dial("tcp", "localhost:8080")
transport := &mcp.ReadWriteCloserTransport{conn}
```

### ProcessTransport

Transport implementation for process communication.

```go
type ProcessTransport struct {
    Stdin  io.WriteCloser
    Stdout io.ReadCloser
}
```

**Example:**
```go
cmd := exec.Command("./server")
stdin, _ := cmd.StdinPipe()
stdout, _ := cmd.StdoutPipe()

transport := &mcp.ProcessTransport{
    Stdin:  stdin,
    Stdout: stdout,
}
```

## Error Handling

### Common Error Types

The SDK uses standard Go error handling patterns. Common error scenarios include:

- **Connection errors**: Network or process communication failures
- **Protocol errors**: Invalid JSON-RPC messages or MCP protocol violations
- **Tool errors**: Errors returned by tool implementations
- **Validation errors**: Invalid parameters or missing required fields

### Error Response Handling

```go
result, err := client.CallTool(ctx, request)
if err != nil {
    // Handle different error types
    switch {
    case strings.Contains(err.Error(), "tool not found"):
        // Handle tool not found
    case strings.Contains(err.Error(), "invalid arguments"):
        // Handle invalid arguments
    default:
        // Handle other errors
    }
}
```

## Advanced Usage

### Custom Transports

Implement custom transport layers by satisfying the transport interface:

```go
type MyCustomTransport struct {
    // Custom fields
}

func (t *MyCustomTransport) Read(p []byte) (n int, err error) {
    // Custom read implementation
}

func (t *MyCustomTransport) Write(p []byte) (n int, err error) {
    // Custom write implementation
}

func (t *MyCustomTransport) Close() error {
    // Custom close implementation
}
```

### Middleware and Interceptors

Add custom logic around tool calls:

```go
type LoggingHandler struct {
    inner mcp.ToolHandlerFunc
}

func (h LoggingHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (mcp.CallToolResult, error) {
    log.Printf("Calling tool: %s with args: %v", req.Name, req.Arguments)
    
    result, err := h.inner(ctx, req)
    
    if err != nil {
        log.Printf("Tool %s failed: %v", req.Name, err)
    } else {
        log.Printf("Tool %s succeeded", req.Name)
    }
    
    return result, err
}

// Wrap tool handler
wrappedHandler := LoggingHandler{inner: originalHandler}
server.RegisterTool(tool, wrappedHandler.Handle)
```

### Context and Cancellation

Use contexts for timeout and cancellation:

```go
// Set timeout for tool call
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := client.CallTool(ctx, request)
```

### Concurrent Operations

The SDK is designed to be thread-safe for concurrent operations:

```go
var wg sync.WaitGroup

for _, request := range requests {
    wg.Add(1)
    go func(req mcp.CallToolRequest) {
        defer wg.Done()
        result, err := client.CallTool(ctx, req)
        // Process result
    }(request)
}

wg.Wait()
```

## Best Practices

### Resource Management

Always close clients and connections:

```go
client, err := mcp.NewClient(transport)
if err != nil {
    return err
}
defer client.Close() // Ensure cleanup
```

### Error Handling

Implement comprehensive error handling:

```go
result, err := client.CallTool(ctx, request)
if err != nil {
    // Log error details
    log.Printf("Tool call failed: %v", err)
    
    // Decide on retry strategy
    if isRetryableError(err) {
        // Implement retry logic
    }
    
    return nil, fmt.Errorf("tool execution failed: %w", err)
}
```

### Schema Validation

Define clear schemas for tools:

```go
tool := mcp.Tool{
    Name:        "process_data",
    Description: "Processes data according to specified rules",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "data": map[string]interface{}{
                "type":        "array",
                "items":       map[string]interface{}{"type": "string"},
                "description": "Array of data items to process",
            },
            "rules": map[string]interface{}{
                "type":        "object",
                "description": "Processing rules",
                "properties": map[string]interface{}{
                    "filter": map[string]interface{}{"type": "string"},
                    "sort":   map[string]interface{}{"type": "boolean"},
                },
            },
        },
        "required": []string{"data"},
    },
}
```

## See Also

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Examples Repository](../examples/)
- [Integration Guide](INTEGRATION_GUIDE.md)
- [Testing Guide](TESTING_GUIDE.md)