# MCP Go Implementation Architecture Overview

## System Architecture

The MCP Go implementation follows a layered architecture that provides clear separation of concerns and extensibility:

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
│  ┌─────────────────┐           ┌─────────────────────────┐   │
│  │  MCP Client     │           │     MCP Server          │   │
│  │                 │           │                         │   │
│  │ - Initialize    │           │ - RegisterTool          │   │
│  │ - ListTools     │           │ - RegisterResource      │   │
│  │ - CallTool      │           │ - RegisterPrompt        │   │
│  │ - ListResources │           │ - Serve                 │   │
│  │ - ReadResource  │           │                         │   │
│  └─────────────────┘           └─────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                     Protocol Layer                          │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                JSON-RPC 2.0 Handler                    │ │
│  │                                                         │ │
│  │ - Request/Response correlation                          │ │
│  │ - Notification handling                                 │ │
│  │ - Error propagation                                     │ │
│  │ - Cancellation support                                  │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Transport Layer                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │    Stdio     │  │     SSE      │  │    WebSocket     │   │
│  │  Transport   │  │  Transport   │  │    Transport     │   │
│  │              │  │              │  │                  │   │
│  │ - stdin/out  │  │ - HTTP/SSE   │  │ - WS protocol   │   │
│  │ - Process    │  │ - EventSource│  │ - Full duplex   │   │
│  │   spawning   │  │ - Streaming  │  │ - Real-time     │   │
│  └──────────────┘  └──────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Client (`client.go`)

The MCP Client provides a high-level interface for connecting to and interacting with MCP servers:

**Key Responsibilities:**
- Protocol handshake and capability negotiation
- Request-response handling with automatic correlation
- Context-aware cancellation with server notification
- Connection lifecycle management

**Key Features:**
- Automatic JSON-RPC message handling
- Built-in cancellation support via `notifications/cancelled`
- Thread-safe operation with proper synchronization
- Extensible notification handling

### 2. Server (`server.go`)

The MCP Server provides a framework for building MCP-compliant servers:

**Key Responsibilities:**
- Tool, resource, and prompt registration
- Request dispatching and handler management  
- Capability advertisement
- Connection handling for various transports

**Key Features:**
- Type-safe handler registration
- Automatic JSON-RPC protocol compliance
- Built-in notification dispatching
- Extensible transport support

### 3. Transport Layer (`transport.go`, `transport_*.go`)

The transport layer abstracts connection mechanisms:

**Available Transports:**
- **Stdio Transport**: Process communication via stdin/stdout
- **SSE Transport**: HTTP Server-Sent Events for web integration
- **WebSocket Transport**: Full-duplex WebSocket connections

**Design Principles:**
- Transport-agnostic protocol implementation
- Pluggable transport architecture
- Automatic connection management
- Proper error handling and cleanup

### 4. Type System (`types.go`, `modelcontextprotocol/`)

Comprehensive type definitions ensuring protocol compliance:

**Core Types:**
- Protocol messages (requests, responses, notifications)
- Content types (text, image, blob)
- Capability definitions
- Error representations

**Features:**
- JSON serialization/deserialization
- Type-safe interfaces
- Polymorphic content handling
- Extensible design for future protocol versions

## Data Flow

### Client Request Flow

```
Client Application
       │
       ▼
┌─────────────────┐
│ Client.CallTool │
└─────────────────┘
       │
       ▼
┌─────────────────┐     ┌──────────────────┐
│   call() method │────▶│ JSON-RPC Request │
└─────────────────┘     └──────────────────┘
       │                         │
       ▼                         ▼
┌─────────────────┐     ┌──────────────────┐
│ Context Monitor │     │ Transport Layer  │
│ (cancellation)  │     │ (send request)   │
└─────────────────┘     └──────────────────┘
       │                         │
       ▼                         ▼
┌─────────────────┐     ┌──────────────────┐
│ Cancel Notify   │     │ Server Handler   │
│ (if cancelled)  │     │                  │
└─────────────────┘     └──────────────────┘
```

### Server Request Processing

```
Transport Layer
       │
       ▼
┌─────────────────────┐
│ handleRequest()     │
└─────────────────────┘
       │
       ▼
┌─────────────────────┐
│ Method Dispatch     │
│ (handlers map)      │
└─────────────────────┘
       │
       ▼
┌─────────────────────┐     ┌──────────────────┐
│ Parameter Parsing   │────▶│ Handler Function │
│ (JSON unmarshal)    │     │ (user code)      │
└─────────────────────┘     └──────────────────┘
       │                             │
       ▼                             ▼
┌─────────────────────┐     ┌──────────────────┐
│ Response Creation   │◀────│ Handler Result   │
│ (JSON marshal)      │     │                  │
└─────────────────────┘     └──────────────────┘
       │
       ▼
┌─────────────────────┐
│ Transport Response  │
└─────────────────────┘
```

## Concurrency Model

### Thread Safety

- **Client**: Thread-safe operations with RWMutex protection
- **Server**: Concurrent request handling with proper synchronization
- **Transport**: Individual transport implementations handle concurrency
- **Handlers**: User-provided handlers must be thread-safe

### Context Handling

- All operations accept `context.Context` for cancellation and timeouts
- Client automatically sends cancellation notifications
- Server handlers can check context for early termination
- Proper cleanup on context cancellation

## Error Handling Strategy

### Error Types

1. **Protocol Errors**: JSON-RPC standard errors with proper codes
2. **Transport Errors**: Connection-level failures and timeouts
3. **Application Errors**: Tool/resource handler errors
4. **System Errors**: Internal implementation errors

### Error Propagation

```
Handler Error
     │
     ▼
┌─────────────────┐
│ CallToolResult  │
│ IsError: true   │
└─────────────────┘
     │
     ▼
┌─────────────────┐
│ JSON-RPC        │
│ Success Response│
│ (error in data) │
└─────────────────┘
```

Application errors are wrapped in successful JSON-RPC responses with the `IsError` flag set, following MCP protocol conventions.

## Extension Points

### Custom Transports

Implement the `Transport` interface:

```go
type Transport interface {
    jsonrpc2.Dialer
    Close() error
}
```

### Custom Handlers

Register handlers for protocol methods:

```go
server.RegisterTool(tool, handler)
server.RegisterResource(resource, handler)
server.RegisterPrompt(prompt, handler)
```

### Custom Content Types

Implement the `Content` interface for new content types:

```go
type Content interface {
    content()
}
```

## Performance Considerations

### Memory Management

- Streaming JSON parsing for large messages
- Connection pooling for multiple client scenarios
- Proper resource cleanup and connection closing

### Optimization Features

- Transport-level message batching
- Async notification handling
- Context-aware request prioritization
- Automatic resource deallocation

## Security Model

### Transport Security

- TLS support for WebSocket and HTTP transports
- Process isolation for stdio transport
- Input validation at protocol layer

### Application Security

- JSON schema validation for tool inputs
- Resource access control through handler logic
- Proper error message sanitization
- Context-based request authorization

This architecture provides a robust, extensible foundation for MCP implementations while maintaining protocol compliance and performance.