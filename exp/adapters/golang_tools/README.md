# Golang Tools Adapter

This adapter allows golang.org/x/tools/internal/mcp servers to work with the standard MCP SDK. It provides seamless integration between the two different implementation patterns.

## Features

- Converts between golang-tools types and standard MCP protocol types
- Supports tools and prompts (resources if supported)
- Handles proper error handling and type conversions
- Integrates with golang-tools Server type

## Usage

```go
package main

import (
    "context"
    "github.com/tmc/mcp/adapters"
    "github.com/tmc/mcp/server"
)

func main() {
    // Create a standard MCP server
    mcpServer := server.New(server.Options{
        Name:         "my-server",
        Version:      "1.0.0",
        Instructions: "Server instructions",
    })
    
    // Register tools and prompts with the MCP server
    // ...
    
    // Create the golang-tools adapter
    adapter := adapters.DefaultRegistry.Get("golang-tools")
    
    // Initialize the adapter with the server
    ctx := context.Background()
    if err := adapter.Initialize(ctx, mcpServer); err != nil {
        panic(err)
    }
    
    // Use the adapter to handle requests
    result, err := adapter.HandleRequest(ctx, "tools/list", nil)
    if err != nil {
        panic(err)
    }
    
    // Process the result...
}
```

## Implementation Details

The adapter works by:

1. Creating a golang-tools server instance internally
2. Converting SDK server tools and prompts to golang-tools format
3. Translating between the two protocol formats for all requests and responses
4. Maintaining compatibility with both implementations

## Type Conversions

The adapter handles conversion between:

- `protocol.Content` ↔ `mcpProtocol.Content`
- `protocol.Tool` ↔ `mcpProtocol.Tool`
- `protocol.Prompt` ↔ `mcpProtocol.Prompt`
- Request/response parameters between the two formats

## Testing

Run the tests with:

```bash
go test ./mcp/adapters/golang_tools
```

The test suite includes:
- Initialization tests
- Request handling tests
- Content conversion tests
- Integration tests