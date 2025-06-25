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

## Real-World Examples

### File System Server

A practical file system MCP server:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    
    "github.com/tmc/mcp"
)

func createFileSystemServer() *mcp.Server {
    server := mcp.NewServer("filesystem-server", "1.0.0",
        mcp.WithServerInstructions("Provides safe file system access"))
    
    // Register file operations tool
    registerFileOperations(server)
    
    // Register file content resources
    registerFileResources(server)
    
    return server
}

func registerFileOperations(server *mcp.Server) {
    tool := mcp.Tool{
        Name:        "file_operations",
        Description: "Perform file system operations",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "operation": {
                    "type": "string",
                    "enum": ["read", "write", "list", "exists", "mkdir"],
                    "description": "Operation to perform"
                },
                "path": {
                    "type": "string",
                    "description": "File or directory path"
                },
                "content": {
                    "type": "string",
                    "description": "Content to write (for write operation)"
                }
            },
            "required": ["operation", "path"]
        }`),
    }
    
    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var params struct {
            Operation string `json:"operation"`
            Path      string `json:"path"`
            Content   string `json:"content,omitempty"`
        }
        
        if err := json.Unmarshal(req.Arguments, &params); err != nil {
            return &mcp.CallToolResult{
                IsError: true,
                Content: []any{map[string]string{
                    "type": "text",
                    "text": fmt.Sprintf("Invalid arguments: %v", err),
                }},
            }, nil
        }
        
        // Security check: only allow operations in current directory
        if !isPathSafe(params.Path) {
            return &mcp.CallToolResult{
                IsError: true,
                Content: []any{map[string]string{
                    "type": "text",
                    "text": "Path not allowed for security reasons",
                }},
            }, nil
        }
        
        switch params.Operation {
        case "read":
            content, err := os.ReadFile(params.Path)
            if err != nil {
                return &mcp.CallToolResult{
                    IsError: true,
                    Content: []any{map[string]string{
                        "type": "text",
                        "text": fmt.Sprintf("Failed to read file: %v", err),
                    }},
                }, nil
            }
            return &mcp.CallToolResult{
                Content: []any{map[string]string{
                    "type": "text",
                    "text": string(content),
                }},
            }, nil
            
        case "list":
            entries, err := os.ReadDir(params.Path)
            if err != nil {
                return &mcp.CallToolResult{
                    IsError: true,
                    Content: []any{map[string]string{
                        "type": "text",
                        "text": fmt.Sprintf("Failed to list directory: %v", err),
                    }},
                }, nil
            }
            
            var result strings.Builder
            for _, entry := range entries {
                if entry.IsDir() {
                    result.WriteString(fmt.Sprintf("📁 %s/\n", entry.Name()))
                } else {
                    info, _ := entry.Info()
                    result.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", entry.Name(), info.Size()))
                }
            }
            
            return &mcp.CallToolResult{
                Content: []any{map[string]string{
                    "type": "text",
                    "text": result.String(),
                }},
            }, nil
            
        default:
            return &mcp.CallToolResult{
                IsError: true,
                Content: []any{map[string]string{
                    "type": "text",
                    "text": fmt.Sprintf("Unknown operation: %s", params.Operation),
                }},
            }, nil
        }
    }
    
    server.RegisterTool(tool, handler)
}

func isPathSafe(path string) bool {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return false
    }
    wd, err := os.Getwd()
    if err != nil {
        return false
    }
    return strings.HasPrefix(absPath, wd)
}
```

### API Integration Client

A client that integrates with multiple MCP servers:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "sync"
    
    "github.com/tmc/mcp"
)

type MCPOrchestrator struct {
    clients map[string]*mcp.Client
    mu      sync.RWMutex
}

func NewMCPOrchestrator() *MCPOrchestrator {
    return &MCPOrchestrator{
        clients: make(map[string]*mcp.Client),
    }
}

func (o *MCPOrchestrator) AddServer(name string, transport mcp.Transport) error {
    client, err := mcp.NewClient(transport, mcp.WithNotificationHandler(
        func(notification mcp.JSONRPCNotification) {
            log.Printf("[%s] Notification: %s", name, notification.Method)
        },
    ))
    if err != nil {
        return fmt.Errorf("failed to create client for %s: %w", name, err)
    }
    
    // Initialize the client
    ctx := context.Background()
    _, err = client.Initialize(ctx, mcp.InitializeRequest{
        ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
        ClientInfo: mcp.Implementation{
            Name:    "orchestrator-client",
            Version: "1.0.0",
        },
        Capabilities: mcp.ClientCapabilities{},
    })
    if err != nil {
        client.Close()
        return fmt.Errorf("failed to initialize client for %s: %w", name, err)
    }
    
    o.mu.Lock()
    o.clients[name] = client
    o.mu.Unlock()
    
    return nil
}

func (o *MCPOrchestrator) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
    o.mu.RLock()
    client, exists := o.clients[serverName]
    o.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("server %s not found", serverName)
    }
    
    argsJSON, err := json.Marshal(args)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal arguments: %w", err)
    }
    
    return client.CallTool(ctx, mcp.CallToolRequest{
        Name:      toolName,
        Arguments: argsJSON,
    })
}

// Example usage
func main() {
    orchestrator := NewMCPOrchestrator()
    
    // Add filesystem server
    orchestrator.AddServer("filesystem", createStdioTransport("./fs-server"))
    
    // Add calculator server  
    orchestrator.AddServer("calculator", createStdioTransport("./calc-server"))
    
    ctx := context.Background()
    
    // Use filesystem server
    result, err := orchestrator.CallTool(ctx, "filesystem", "file_operations", map[string]any{
        "operation": "list",
        "path":      ".",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Directory listing: %+v\n", result.Content)
    
    // Use calculator server
    result, err = orchestrator.CallTool(ctx, "calculator", "calculate", map[string]any{
        "operation": "add",
        "a":         10,
        "b":         5,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Calculation result: %+v\n", result.Content)
}
```

### Database Integration Server

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    
    "github.com/tmc/mcp"
    _ "github.com/lib/pq" // PostgreSQL driver
)

type DatabaseServer struct {
    server *mcp.Server
    db     *sql.DB
}

func NewDatabaseServer(dbURL string) (*DatabaseServer, error) {
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return nil, err
    }
    
    server := mcp.NewServer("database-server", "1.0.0")
    
    ds := &DatabaseServer{
        server: server,
        db:     db,
    }
    
    ds.registerTools()
    ds.registerResources()
    
    return ds, nil
}

func (ds *DatabaseServer) registerTools() {
    // Register query tool
    queryTool := mcp.Tool{
        Name:        "sql_query",
        Description: "Execute SQL query (SELECT only)",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "SQL SELECT query to execute"
                },
                "limit": {
                    "type": "integer",
                    "default": 100,
                    "description": "Maximum number of rows to return"
                }
            },
            "required": ["query"]
        }`),
    }
    
    ds.server.RegisterTool(queryTool, ds.handleQuery)
    
    // Register schema inspection tool
    schemaTool := mcp.Tool{
        Name:        "describe_table",
        Description: "Get table schema information",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "table_name": {
                    "type": "string",
                    "description": "Name of the table to describe"
                }
            },
            "required": ["table_name"]
        }`),
    }
    
    ds.server.RegisterTool(schemaTool, ds.handleDescribeTable)
}

func (ds *DatabaseServer) handleQuery(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    var params struct {
        Query string `json:"query"`
        Limit int    `json:"limit"`
    }
    params.Limit = 100 // default
    
    if err := json.Unmarshal(req.Arguments, &params); err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []any{map[string]string{
                "type": "text",
                "text": fmt.Sprintf("Invalid arguments: %v", err),
            }},
        }, nil
    }
    
    // Security: only allow SELECT queries
    if !isSelectQuery(params.Query) {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []any{map[string]string{
                "type": "text",
                "text": "Only SELECT queries are allowed",
            }},
        }, nil
    }
    
    rows, err := ds.db.QueryContext(ctx, params.Query)
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []any{map[string]string{
                "type": "text",
                "text": fmt.Sprintf("Query failed: %v", err),
            }},
        }, nil
    }
    defer rows.Close()
    
    // Get column names
    columns, err := rows.Columns()
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []any{map[string]string{
                "type": "text",
                "text": fmt.Sprintf("Failed to get columns: %v", err),
            }},
        }, nil
    }
    
    // Collect results
    var results []map[string]any
    for rows.Next() && len(results) < params.Limit {
        values := make([]any, len(columns))
        valuePtrs := make([]any, len(columns))
        for i := range values {
            valuePtrs[i] = &values[i]
        }
        
        if err := rows.Scan(valuePtrs...); err != nil {
            continue
        }
        
        row := make(map[string]any)
        for i, col := range columns {
            row[col] = values[i]
        }
        results = append(results, row)
    }
    
    // Format results as JSON
    resultJSON, err := json.MarshalIndent(results, "", "  ")
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []any{map[string]string{
                "type": "text",
                "text": fmt.Sprintf("Failed to format results: %v", err),
            }},
        }, nil
    }
    
    return &mcp.CallToolResult{
        Content: []any{
            map[string]string{
                "type": "text",
                "text": fmt.Sprintf("Query returned %d rows:\n%s", len(results), string(resultJSON)),
            },
        },
    }, nil
}

func isSelectQuery(query string) bool {
    // Simple check - in production, use a proper SQL parser
    query = strings.TrimSpace(strings.ToUpper(query))
    return strings.HasPrefix(query, "SELECT")
}
```

## Integration Patterns

### Middleware Pattern

```go
type MiddlewareFunc func(mcp.ToolHandlerFunc) mcp.ToolHandlerFunc

// Logging middleware
func LoggingMiddleware(logger *log.Logger) MiddlewareFunc {
    return func(next mcp.ToolHandlerFunc) mcp.ToolHandlerFunc {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            start := time.Now()
            logger.Printf("[%s] Starting tool call: %s", req.Name, string(req.Arguments))
            
            result, err := next(ctx, req)
            
            duration := time.Since(start)
            if err != nil {
                logger.Printf("[%s] Tool call failed after %v: %v", req.Name, duration, err)
            } else {
                logger.Printf("[%s] Tool call completed after %v", req.Name, duration)
            }
            
            return result, err
        }
    }
}

// Authentication middleware
func AuthMiddleware(validateToken func(string) bool) MiddlewareFunc {
    return func(next mcp.ToolHandlerFunc) mcp.ToolHandlerFunc {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            // Extract token from context or arguments
            token, ok := ctx.Value("auth_token").(string)
            if !ok || !validateToken(token) {
                return &mcp.CallToolResult{
                    IsError: true,
                    Content: []any{map[string]string{
                        "type": "text",
                        "text": "Authentication required",
                    }},
                }, nil
            }
            
            return next(ctx, req)
        }
    }
}

// Apply middleware to tools
func applyMiddleware(handler mcp.ToolHandlerFunc, middlewares ...MiddlewareFunc) mcp.ToolHandlerFunc {
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }
    return handler
}

// Usage
server := mcp.NewServer("my-server", "1.0.0")
logger := log.New(os.Stdout, "[MCP] ", log.LstdFlags)

enhancedHandler := applyMiddleware(
    originalHandler,
    LoggingMiddleware(logger),
    AuthMiddleware(validateJWT),
)

server.RegisterTool(tool, enhancedHandler)
```

### Connection Pooling

```go
type ClientPool struct {
    clients chan *mcp.Client
    factory func() (*mcp.Client, error)
    maxSize int
    mu      sync.Mutex
}

func NewClientPool(maxSize int, factory func() (*mcp.Client, error)) *ClientPool {
    return &ClientPool{
        clients: make(chan *mcp.Client, maxSize),
        factory: factory,
        maxSize: maxSize,
    }
}

func (p *ClientPool) Get(ctx context.Context) (*mcp.Client, error) {
    select {
    case client := <-p.clients:
        // Validate client is still connected
        if err := client.Ping(ctx); err != nil {
            client.Close()
            return p.createNew()
        }
        return client, nil
    default:
        return p.createNew()
    }
}

func (p *ClientPool) Put(client *mcp.Client) {
    select {
    case p.clients <- client:
        // Successfully returned to pool
    default:
        // Pool is full, close the client
        client.Close()
    }
}

func (p *ClientPool) createNew() (*mcp.Client, error) {
    return p.factory()
}

// Usage with retry logic
func CallToolWithRetry(pool *ClientPool, ctx context.Context, req mcp.CallToolRequest, maxRetries int) (*mcp.CallToolResult, error) {
    var lastErr error
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        client, err := pool.Get(ctx)
        if err != nil {
            lastErr = err
            continue
        }
        
        result, err := client.CallTool(ctx, req)
        if err == nil {
            pool.Put(client)
            return result, nil
        }
        
        // Don't return bad clients to pool
        client.Close()
        lastErr = err
        
        // Exponential backoff
        if attempt < maxRetries {
            delay := time.Duration(1<<uint(attempt)) * time.Second
            select {
            case <-time.After(delay):
                continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
    
    return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
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
    InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "data": {
                "type": "array",
                "items": {"type": "string"},
                "description": "Array of data items to process"
            },
            "rules": {
                "type": "object",
                "description": "Processing rules",
                "properties": {
                    "filter": {"type": "string"},
                    "sort": {"type": "boolean"}
                }
            }
        },
        "required": ["data"]
    }`),
}
```

## Performance Optimization

### Batch Operations

```go
// Batch multiple tool calls for efficiency
type BatchRequest struct {
    Requests []mcp.CallToolRequest `json:"requests"`
}

type BatchResult struct {
    Results []mcp.CallToolResult `json:"results"`
    Errors  []string             `json:"errors"`
}

func HandleBatchTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    var batchReq BatchRequest
    if err := json.Unmarshal(req.Arguments, &batchReq); err != nil {
        return nil, err
    }
    
    var results []mcp.CallToolResult
    var errors []string
    
    // Process requests in parallel
    sem := make(chan struct{}, 10) // Limit concurrency
    var wg sync.WaitGroup
    var mu sync.Mutex
    
    for i, request := range batchReq.Requests {
        wg.Add(1)
        go func(idx int, req mcp.CallToolRequest) {
            defer wg.Done()
            
            sem <- struct{}{} // Acquire semaphore
            defer func() { <-sem }() // Release semaphore
            
            result, err := processSingleRequest(ctx, req)
            
            mu.Lock()
            if err != nil {
                errors = append(errors, fmt.Sprintf("Request %d: %v", idx, err))
            } else {
                results = append(results, *result)
            }
            mu.Unlock()
        }(i, request)
    }
    
    wg.Wait()
    
    batchResult := BatchResult{
        Results: results,
        Errors:  errors,
    }
    
    resultJSON, _ := json.Marshal(batchResult)
    return &mcp.CallToolResult{
        Content: []any{map[string]string{
            "type": "text",
            "text": string(resultJSON),
        }},
    }, nil
}
```

### Caching

```go
type CachedToolHandler struct {
    handler mcp.ToolHandlerFunc
    cache   map[string]*mcp.CallToolResult
    mu      sync.RWMutex
    ttl     time.Duration
    lastAccess map[string]time.Time
}

func NewCachedToolHandler(handler mcp.ToolHandlerFunc, ttl time.Duration) *CachedToolHandler {
    c := &CachedToolHandler{
        handler:    handler,
        cache:      make(map[string]*mcp.CallToolResult),
        ttl:        ttl,
        lastAccess: make(map[string]time.Time),
    }
    
    // Start cleanup goroutine
    go c.cleanup()
    
    return c
}

func (c *CachedToolHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    key := c.getCacheKey(req)
    
    // Check cache first
    c.mu.RLock()
    if result, exists := c.cache[key]; exists {
        if time.Since(c.lastAccess[key]) < c.ttl {
            c.mu.RUnlock()
            return result, nil
        }
    }
    c.mu.RUnlock()
    
    // Cache miss or expired, call actual handler
    result, err := c.handler(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Store in cache
    c.mu.Lock()
    c.cache[key] = result
    c.lastAccess[key] = time.Now()
    c.mu.Unlock()
    
    return result, nil
}

func (c *CachedToolHandler) getCacheKey(req mcp.CallToolRequest) string {
    return fmt.Sprintf("%s:%s", req.Name, string(req.Arguments))
}

func (c *CachedToolHandler) cleanup() {
    ticker := time.NewTicker(c.ttl)
    defer ticker.Stop()
    
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, lastAccess := range c.lastAccess {
            if now.Sub(lastAccess) > c.ttl {
                delete(c.cache, key)
                delete(c.lastAccess, key)
            }
        }
        c.mu.Unlock()
    }
}
```

## Testing Patterns

### Mock Server for Testing

```go
type MockMCPServer struct {
    tools     map[string]func(mcp.CallToolRequest) (*mcp.CallToolResult, error)
    resources map[string][]mcp.ResourceContents
}

func NewMockMCPServer() *MockMCPServer {
    return &MockMCPServer{
        tools:     make(map[string]func(mcp.CallToolRequest) (*mcp.CallToolResult, error)),
        resources: make(map[string][]mcp.ResourceContents),
    }
}

func (m *MockMCPServer) RegisterMockTool(name string, handler func(mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
    m.tools[name] = handler
}

func (m *MockMCPServer) RegisterMockResource(uri string, contents []mcp.ResourceContents) {
    m.resources[uri] = contents
}

// Test example
func TestCalculatorTool(t *testing.T) {
    mock := NewMockMCPServer()
    
    mock.RegisterMockTool("calculator", func(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var params struct {
            Operation string  `json:"operation"`
            A         float64 `json:"a"`
            B         float64 `json:"b"`
        }
        
        json.Unmarshal(req.Arguments, &params)
        
        var result float64
        switch params.Operation {
        case "add":
            result = params.A + params.B
        case "subtract":
            result = params.A - params.B
        default:
            return &mcp.CallToolResult{
                IsError: true,
                Content: []any{map[string]string{
                    "type": "text",
                    "text": "Unknown operation",
                }},
            }, nil
        }
        
        return &mcp.CallToolResult{
            Content: []any{map[string]any{
                "type": "text",
                "text": fmt.Sprintf("Result: %g", result),
            }},
        }, nil
    })
    
    // Test addition
    result, err := mock.CallTool(mcp.CallToolRequest{
        Name: "calculator",
        Arguments: json.RawMessage(`{"operation":"add","a":5,"b":3}`),
    })
    
    assert.NoError(t, err)
    assert.False(t, result.IsError)
    assert.Contains(t, result.Content[0].(map[string]any)["text"], "8")
}
```

## See Also

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Architecture Overview](architecture/overview.md)
- [Getting Started Guide](getting-started/quickstart.md)
- [Examples Repository](../examples/)
- [Testing Guide](../testing/README.md)