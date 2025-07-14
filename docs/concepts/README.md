# Core Concepts

Understanding the Model Context Protocol (MCP) requires familiarity with several key concepts. This section covers the fundamental building blocks of MCP.

## Overview

MCP is built on these core concepts:

1. **[Protocol Overview](./protocol-overview.md)** - The MCP specification and design principles
2. **[Transports](./transports.md)** - Communication channels (stdio, HTTP, WebSocket)
3. **[JSON-RPC Messages](./jsonrpc.md)** - Request/response message format
4. **[Capabilities](./capabilities.md)** - Tools, Resources, and Prompts
5. **[Type System](./type-system.md)** - Strongly-typed protocol definitions
6. **[Rate Limiting](./rate-limiting.md)** - Request throttling and flow control

## Quick Start

If you're new to MCP, we recommend reading these concepts in order:

1. Start with the [Protocol Overview](./protocol-overview.md) to understand MCP's purpose
2. Learn about [Transports](./transports.md) to see how clients and servers communicate
3. Understand [JSON-RPC Messages](./jsonrpc.md) for the message format
4. Explore [Capabilities](./capabilities.md) to see what MCP can do

## Key Principles

### 1. Transport Agnostic

MCP works over multiple transport layers:
- Standard I/O (stdio)
- HTTP with Server-Sent Events (SSE)
- WebSockets
- Custom transports

### 2. JSON-RPC Based

All communication uses JSON-RPC 2.0:
- Structured request/response format
- Asynchronous notifications
- Error handling

### 3. Capability-Driven

Servers declare their capabilities:
- **Tools**: Functions the server can execute
- **Resources**: Data the server can provide
- **Prompts**: Interactive prompts for users

### 4. Type-Safe

Strong typing throughout:
- Type-safe message definitions
- Validated JSON schemas
- Go generics for compile-time safety

### 5. Extensible

Designed for extension:
- Custom transports
- Additional capabilities
- Protocol versioning

## Architecture

```
┌─────────────┐         ┌─────────────┐
│   Client    │         │   Server    │
├─────────────┤         ├─────────────┤
│   MCP API   │         │   MCP API   │
├─────────────┤         ├─────────────┤
│  Transport  │<------->│  Transport  │
└─────────────┘         └─────────────┘
```

## Common Patterns

### Request-Response

```json
// Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "calculate",
    "arguments": {"a": 1, "b": 2}
  }
}

// Response
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "output": "3"
  }
}
```

### Notifications

```json
{
  "jsonrpc": "2.0",
  "method": "progress",
  "params": {
    "progress": 50,
    "message": "Processing..."
  }
}
```

### Error Handling

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "details": "Missing required field 'name'"
    }
  }
}
```

## Next Steps

- Read the detailed [Protocol Overview](./protocol-overview.md)
- Learn about [Transports](./transports.md)
- Explore [JSON-RPC Messages](./jsonrpc.md)
- Understand [Capabilities](./capabilities.md)

## See Also

- [Getting Started Guide](../getting-started/README.md)
- [API Reference](../api/README.md)
- [Examples](../examples/README.md)