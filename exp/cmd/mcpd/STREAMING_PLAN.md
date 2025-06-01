# Streaming Support for mcpd

This document outlines the plan for adding streaming capabilities to mcpd, including Server-Sent Events (SSE), HTTP streaming, and WebSocket support.

## Goals

1. Add Server-Sent Events (SSE) support
2. Implement streamable HTTP endpoints
3. Prepare the architecture for WebSocket support
4. Ensure backward compatibility with existing functionality

## Architecture Changes

### 1. Transport Layer Enhancements

The current transport layer needs to be extended to support different types of streaming:

```go
// Transport interface with streaming support
type Transport interface {
    // Existing methods
    Listen() error
    Close() error
    
    // New streaming methods
    EnableSSE() error
    EnableWebSockets() error
    GetHTTPHandler() http.Handler
}
```

### 2. New Connection Types

Add support for different connection types:

```go
// ConnectionType represents different ways clients can connect to mcpd
type ConnectionType string

const (
    ConnectionTypeUnix     ConnectionType = "unix"
    ConnectionTypeTCP      ConnectionType = "tcp"
    ConnectionTypeSSE      ConnectionType = "sse"
    ConnectionTypeWebSocket ConnectionType = "websocket"
    ConnectionTypeHTTPStream ConnectionType = "httpstream"
)

// Connection represents a client connection with its type
type Connection struct {
    Type ConnectionType
    ID   string
    // Other connection details...
}
```

### 3. HTTP Server Integration

Add an HTTP server to mcpd that can handle various types of streaming requests:

```go
// HTTPServer manages HTTP routes for streaming
type HTTPServer struct {
    router      *http.ServeMux
    sseClients  map[string]*SSEClient
    wsClients   map[string]*WebSocketClient
    httpStreams map[string]*HTTPStreamClient
}

// Initialize server routes
func (s *HTTPServer) Init() {
    s.router.HandleFunc("/sse", s.handleSSE)
    s.router.HandleFunc("/stream", s.handleHTTPStream)
    s.router.HandleFunc("/ws", s.handleWebSocket)
}
```

## Implementation Plan

### Phase 1: Server-Sent Events (SSE)

1. Create a new `SSEHandler` type that implements SSE protocol
2. Add SSE route to HTTP server
3. Add client tracking and message forwarding
4. Implement event filtering and channel management

### Phase 2: HTTP Streaming

1. Create a `StreamHandler` for long-lived HTTP responses
2. Implement chunked transfer encoding for responses
3. Add content-type negotiation
4. Handle connection timeouts and client disconnects

### Phase 3: WebSocket Preparation

1. Create interfaces for WebSocket support
2. Add route handling for WebSocket upgrade requests
3. Design message framing for WebSocket protocol
4. Prepare connection lifecycle management

### Phase 4: Integration with Daemon

1. Extend daemon configuration to enable/disable streaming features
2. Implement session management for streaming connections
3. Add command-line flags for streaming options
4. Update documentation with streaming usage details

## Command-Line Options

Add the following options to mcpd:

```
-http string
    HTTP address to listen on for streaming endpoints (e.g., :8081)
-enable-sse
    Enable Server-Sent Events endpoint
-enable-ws
    Enable WebSocket support (experimental)
-stream-timeout duration
    Timeout for streaming connections (default 1h)
```

## Client Usage Examples

### Server-Sent Events

```bash
# Start mcpd with SSE support
mcpd -socket /tmp/mcp.sock -http :8081 -enable-sse -- go run ./mcp-server

# Connect and receive events
curl -N http://localhost:8081/sse
```

### HTTP Streaming

```bash
# Start mcpd with HTTP streaming
mcpd -socket /tmp/mcp.sock -http :8081 -- go run ./mcp-server

# Connect and receive streamed responses
curl -N http://localhost:8081/stream
```

### WebSocket (Future)

```bash
# Start mcpd with WebSocket support
mcpd -socket /tmp/mcp.sock -http :8081 -enable-ws -- go run ./mcp-server

# Connect via WebSocket (using wscat tool)
wscat -c ws://localhost:8081/ws
```

## Message Flow

1. Client connects to streaming endpoint
2. mcpd establishes connection to MCP server
3. Client sends requests via the streaming connection
4. mcpd forwards requests to MCP server
5. MCP server responses are sent back through the streaming connection

## Security Considerations

1. Cross-Origin Resource Sharing (CORS) configuration
2. Authentication for streaming endpoints
3. Rate limiting for streaming connections
4. Secure WebSocket connections (wss://)

## Next Steps

1. Implement SSE support in transport package
2. Add HTTP server with streaming routes
3. Update daemon to use new transport features
4. Add configuration and command-line options
5. Create examples and tests for streaming functionality