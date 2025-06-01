# MCP Adapters

The adapters package provides a standardized way to integrate various MCP server implementations with the standard MCP SDK. These adapters handle the translation between different implementation patterns and the SDK server interface.

## Overview

MCP has multiple server implementations across different repositories, each with their own patterns and approaches. This package provides adapters that:

- Wrap existing server implementations
- Translate between different API patterns
- Provide a consistent interface for the SDK
- Enable reuse of existing server code

## Available Adapters

### mark3labs

Adapter for servers from the `mark3labs-mcp-go` repository. This implementation follows specific patterns that differ from the standard SDK interface.

```go
import "github.com/tmc/mcp/adapters/mark3labs"

// Create and use the adapter
adapter := mark3labs.NewAdapter()
```

### golang_tools

Adapter for servers from the `golang-tools-internal-mcp` repository. This implementation has its own patterns and conventions that need translation.

```go
import "github.com/tmc/mcp/adapters/golang_tools"

// Create and use the adapter
adapter := golang_tools.NewAdapter()
```

## Usage

To use an adapter:

1. Import the specific adapter package
2. Create a new adapter instance
3. Initialize it with your server
4. Use it to handle requests

```go
import (
    "context"
    "github.com/tmc/mcp/adapters"
    "github.com/tmc/mcp/server"
)

// Get an adapter from the registry
adapter, ok := adapters.DefaultRegistry.Get("mark3labs")
if !ok {
    panic("adapter not found")
}

// Initialize with a server
server := server.New(serverOptions)
err := adapter.Initialize(ctx, server)
if err != nil {
    panic(err)
}

// Use the adapter to handle requests
result, err := adapter.HandleRequest(ctx, method, params)
```

## Creating New Adapters

To create a new adapter:

1. Implement the `Adapter` interface
2. Register it with the default registry
3. Handle the translation between patterns

```go
type MyAdapter struct {
    server server.Server
}

func (a *MyAdapter) Initialize(ctx context.Context, server server.Server) error {
    a.server = server
    return nil
}

func (a *MyAdapter) HandleRequest(ctx context.Context, method string, params any) (any, error) {
    // Implement request handling
    return nil, nil
}

func (a *MyAdapter) GetCapabilities() protocol.ServerCapabilities {
    return protocol.ServerCapabilities{}
}

func init() {
    adapters.DefaultRegistry.Register("myadapter", func() adapters.Adapter {
        return &MyAdapter{}
    })
}
```

## Architecture

The adapter pattern follows these principles:

1. **Interface Consistency**: All adapters implement the same `Adapter` interface
2. **Registration**: Adapters register themselves for discovery
3. **Translation Layer**: Each adapter handles its own pattern translation
4. **Minimal Dependencies**: Adapters only depend on what they need

## Contributing

When adding new adapters:

1. Study the target implementation's patterns
2. Create appropriate translation logic
3. Test with the target servers
4. Document any limitations or special considerations
5. Add examples showing usage
