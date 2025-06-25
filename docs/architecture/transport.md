# Transport Layer Architecture

## Overview

The MCP Go implementation uses a pluggable transport architecture that abstracts the underlying communication mechanism from the protocol layer. This design enables MCP to work over various connection types while maintaining protocol consistency.

## Transport Interface

All transports implement the core `Transport` interface:

```go
type Transport interface {
    jsonrpc2.Dialer  // Provides Dial(ctx, network, address) method
    Close() error    // Cleanup resources
}
```

The transport acts as a bridge between the JSON-RPC layer and the actual communication medium.

## Available Transports

### 1. Stdio Transport

**Use Case**: Process-to-process communication, command-line tools, shell integration

**Implementation**: `StdioTransport()`

```go
transport := mcp.StdioTransport()
server.Serve(ctx, transport)
```

**Architecture**:
```
┌─────────────────┐    stdin     ┌─────────────────┐
│   MCP Client    │◄─────────────│   MCP Server    │
│   (Parent)      │              │   (Child)      │
│                 │    stdout    │                 │
│                 │─────────────►│                 │
└─────────────────┘              └─────────────────┘
```

**Features**:
- Direct process communication
- No network overhead
- Automatic process lifecycle management
- Platform-agnostic (Unix/Windows)

**Internal Structure**:
```go
type ReadWriteCloserTransport struct {
    ReadWriteCloser io.ReadWriteCloser
}

// StdioTransport creates stdin/stdout transport
func StdioTransport() Transport {
    return &ReadWriteCloserTransport{
        ReadWriteCloser: struct {
            io.Reader
            io.Writer  
            io.Closer
        }{
            Reader: os.Stdin,
            Writer: os.Stdout,
            Closer: io.NopCloser(nil),
        },
    }
}
```

### 2. Server-Sent Events (SSE) Transport

**Use Case**: Web applications, browser integration, real-time updates

**Implementation**: `transport_sse.go`

```go
transport := mcp.NewSSETransport(config)
server.Serve(ctx, transport)
```

**Architecture**:
```
┌─────────────────┐   HTTP/SSE   ┌─────────────────┐
│   Web Client    │◄─────────────│   MCP Server    │
│   (Browser)     │              │   (HTTP Server) │
│                 │   POST/PUT   │                 │
│                 │─────────────►│                 │
└─────────────────┘              └─────────────────┘
```

**Protocol Flow**:
1. Client establishes SSE connection (`GET /events`)
2. Server sends responses via SSE stream
3. Client sends requests via HTTP POST/PUT
4. Bidirectional communication achieved through dual channels

**Features**:
- HTTP-based, firewall-friendly
- Built-in reconnection handling
- CORS support for web applications
- Event streaming for real-time updates

**Internal Structure**:
```go
type SSETransport struct {
    config     SSEConfig
    httpServer *http.Server
    clients    map[string]*SSEClient
    mu         sync.RWMutex
}

type SSEClient struct {
    events   chan []byte
    requests chan *http.Request
    done     chan struct{}
}
```

### 3. WebSocket Transport

**Use Case**: Real-time applications, low-latency communication, full-duplex scenarios

**Implementation**: `transport_websocket.go`

```go
transport := mcp.NewWebSocketTransport(url)
client, _ := mcp.NewClient(transport)
```

**Architecture**:
```
┌─────────────────┐   WebSocket  ┌─────────────────┐
│   MCP Client    │◄────────────►│   MCP Server    │
│                 │ Full Duplex  │                 │
│                 │              │                 │
└─────────────────┘              └─────────────────┘
```

**Features**:
- True full-duplex communication
- Low protocol overhead
- Native browser support
- Real-time bidirectional messaging

**Internal Structure**:
```go
type WebSocketTransport struct {
    url    string
    conn   *websocket.Conn
    done   chan struct{}
    config WebSocketConfig
}
```

## Transport Patterns

### Connection Lifecycle

```
┌─────────────────┐
│   Initialize    │
│   Transport     │
└─────────────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│   Establish     │────▶│   Handshake      │
│   Connection    │     │   (if needed)    │
└─────────────────┘     └──────────────────┘
         │                        │
         ▼                        ▼
┌─────────────────┐     ┌──────────────────┐
│   Message       │◄────│   Ready State    │
│   Exchange      │     │                  │
└─────────────────┘     └──────────────────┘
         │                        │
         ▼                        ▼
┌─────────────────┐     ┌──────────────────┐
│   Cleanup       │◄────│   Termination    │
│   Resources     │     │   Signal         │
└─────────────────┘     └──────────────────┘
```

### Message Flow Patterns

#### Request-Response Pattern
```go
// Client side
result, err := client.CallTool(ctx, request)

// Transport layer handles:
// 1. Serialize request to JSON-RPC
// 2. Send via transport
// 3. Await response
// 4. Deserialize and return
```

#### Notification Pattern  
```go
// Server side - send notification
server.dispatch.NotifyListChanged(ctx, MethodToolListChanged)

// Transport layer handles:
// 1. Serialize notification
// 2. Send via transport (no response expected)
// 3. Client handler receives notification
```

## Transport-Specific Features

### Stdio Transport Features

**Flushing Strategy**:
```go
type flushingReadWriteCloser struct {
    io.ReadWriteCloser
    logger *slog.Logger
}

func (f *flushingReadWriteCloser) Write(p []byte) (n int, err error) {
    n, err = f.ReadWriteCloser.Write(p)
    if err != nil {
        return n, err
    }
    
    // Try multiple flushing strategies
    if flusher, ok := f.ReadWriteCloser.(interface{ Flush() error }); ok {
        flusher.Flush()
    } else if syncer, ok := f.ReadWriteCloser.(interface{ Sync() error }); ok {
        syncer.Sync()
    }
    
    return n, nil
}
```

**Process Management**:
- Automatic cleanup on process termination
- Signal handling for graceful shutdown
- Stdin/stdout redirection handling

### SSE Transport Features

**Event Streaming**:
```http
GET /events HTTP/1.1
Accept: text/event-stream
Cache-Control: no-cache

HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache

data: {"jsonrpc":"2.0","id":1,"result":{"tools":[]}}

data: {"jsonrpc":"2.0","method":"notifications/tools/list_changed"}
```

**Request Handling**:
```http
POST /request HTTP/1.1
Content-Type: application/json

{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
```

**CORS Support**:
```go
func (t *SSETransport) setupCORS(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
```

### WebSocket Transport Features

**Connection Upgrading**:
```go
func (t *WebSocketTransport) handleUpgrade(w http.ResponseWriter, r *http.Request) {
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true // Configure appropriately for production
        },
    }
    
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    
    t.conn = conn
}
```

**Message Types**:
```go
// Text messages for JSON-RPC
conn.WriteMessage(websocket.TextMessage, jsonData)

// Binary messages for large payloads (future)
conn.WriteMessage(websocket.BinaryMessage, binaryData)

// Close messages for clean shutdown
conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1000, ""))
```

## Error Handling

### Transport-Level Errors

```go
var (
    ErrTransportClosed    = errors.New("mcp: transport closed")
    ErrConnectionFailed   = errors.New("mcp: connection failed")
    ErrInvalidMessage     = errors.New("mcp: invalid message format")
    ErrTransportTimeout   = errors.New("mcp: transport timeout")
)
```

### Error Recovery Strategies

**Connection Recovery**:
```go
func (t *Transport) handleError(err error) {
    if isRecoverable(err) {
        t.attemptReconnect()
    } else {
        t.close()
    }
}
```

**Message Recovery**:
- Invalid JSON: Log and continue
- Protocol errors: Send JSON-RPC error response  
- Transport errors: Attempt reconnection

## Performance Considerations

### Buffering Strategies

**Stdio Transport**: 
- Line-buffered output for immediate delivery
- Automatic flushing to prevent deadlocks

**SSE Transport**:
- Event buffering for burst traffic
- Keep-alive messages for connection maintenance

**WebSocket Transport**:
- Message queue for high throughput
- Compression support for large payloads

### Connection Pooling

```go
type TransportPool struct {
    transports chan Transport
    factory    func() Transport
    maxSize    int
}

func (p *TransportPool) Get() Transport {
    select {
    case t := <-p.transports:
        return t
    default:
        return p.factory()
    }
}
```

## Security Considerations

### Transport Security

**Stdio**: Process isolation provides security boundary
**SSE**: Use HTTPS in production, implement authentication  
**WebSocket**: WSS (WebSocket Secure) for encrypted communication

### Authentication Patterns

```go
type AuthenticatedTransport struct {
    Transport
    authenticator Authenticator
}

func (t *AuthenticatedTransport) Dial(ctx context.Context, network, address string) (io.ReadWriteCloser, error) {
    conn, err := t.Transport.Dial(ctx, network, address)
    if err != nil {
        return nil, err
    }
    
    if err := t.authenticator.Authenticate(conn); err != nil {
        conn.Close()
        return nil, err
    }
    
    return conn, nil
}
```

## Testing Transports

### Mock Transport

```go
type MockTransport struct {
    requests  [][]byte
    responses [][]byte
    index     int
}

func (m *MockTransport) Dial(ctx context.Context, network, address string) (io.ReadWriteCloser, error) {
    return &mockConn{transport: m}, nil
}
```

### Transport Test Patterns

```go
func TestTransportReliability(t *testing.T) {
    transport := createTestTransport()
    defer transport.Close()
    
    // Test connection establishment
    // Test message delivery
    // Test error recovery
    // Test cleanup
}
```

This transport architecture provides flexibility, reliability, and performance while maintaining a clean separation between protocol logic and communication mechanisms.