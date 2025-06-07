# Transports

MCP supports multiple transport mechanisms for client-server communication. Each transport has different characteristics suitable for various deployment scenarios.

## Supported Transports

### 1. Standard I/O (stdio)

The default transport using stdin/stdout pipes.

**Characteristics:**
- Synchronous, bidirectional
- Process-based isolation
- No network overhead
- Simple to implement

**Use Cases:**
- Local command-line tools
- Subprocess communication
- Development and testing

**Example:**
```go
// Server
transport := mcp.NewStdioTransport()
server := mcp.NewServer(transport)

// Client  
transport := mcp.NewStdioTransport()
client := mcp.NewClient(transport)
```

### 2. HTTP with Server-Sent Events (SSE)

HTTP transport with SSE for server-to-client streaming.

**Characteristics:**
- Request-response for client-to-server
- Streaming for server-to-client
- Firewall-friendly
- Supports proxies

**Use Cases:**
- Web applications
- Cross-network communication
- Cloud deployments

**Example:**
```go
// Server
transport := mcp.NewHTTPTransport(":8080")
server := mcp.NewServer(transport)

// Client
transport := mcp.NewHTTPTransport("http://localhost:8080")
client := mcp.NewClient(transport)
```

### 3. WebSockets

Full-duplex, bidirectional communication over a single connection.

**Characteristics:**
- Bidirectional streaming
- Low latency
- Persistent connection
- Real-time updates

**Use Cases:**
- Real-time applications
- Interactive sessions
- High-frequency updates

**Example:**
```go
// Server
transport := mcp.NewWebSocketTransport(":8080")
server := mcp.NewServer(transport)

// Client
transport := mcp.NewWebSocketTransport("ws://localhost:8080")
client := mcp.NewClient(transport)
```

### 4. Command Transport

Execute commands as transport mechanism.

**Characteristics:**
- Flexible command execution
- Process management
- Custom arguments
- Environment configuration

**Use Cases:**
- Dynamic server startup
- Complex initialization
- Container orchestration

**Example:**
```go
transport := mcp.NewCommandTransport("node", "server.js", "--mcp")
client := mcp.NewClient(transport)
```

## Transport Interface

All transports implement the same interface:

```go
type Transport interface {
    // Send a message
    Send(ctx context.Context, msg json.RawMessage) error
    
    // Receive a message
    Receive(ctx context.Context) (json.RawMessage, error)
    
    // Close the transport
    Close() error
}
```

## Transport Selection

Choose transports based on:

### Deployment Environment

| Environment | Recommended Transport |
|------------|---------------------|
| Local CLI | stdio |
| Web App | HTTP/SSE |
| Microservices | HTTP or WebSocket |
| Real-time | WebSocket |

### Performance Requirements

| Requirement | Best Transport |
|------------|---------------|
| Low latency | WebSocket |
| High throughput | stdio |
| Firewall-friendly | HTTP |
| Streaming | SSE or WebSocket |

### Security Needs

| Need | Transport Feature |
|------|------------------|
| Authentication | HTTP headers |
| Encryption | TLS/HTTPS |
| Network isolation | stdio |
| Access control | HTTP middleware |

## Configuration Examples

### Stdio Transport

```go
// Basic stdio
transport := &StdioTransport{
    Stdin:  os.Stdin,
    Stdout: os.Stdout,
}

// With custom pipes
transport := &StdioTransport{
    Stdin:  customIn,
    Stdout: customOut,
}
```

### HTTP Transport

```go
// Server with TLS
transport := &HTTPTransport{
    Addr:      ":443",
    TLSConfig: tlsConfig,
    Handler:   mux,
}

// Client with authentication
transport := &HTTPTransport{
    URL: "https://api.example.com",
    Headers: map[string]string{
        "Authorization": "Bearer token",
    },
}
```

### WebSocket Transport

```go
// Server with options
transport := &WebSocketTransport{
    Addr: ":8080",
    Upgrader: websocket.Upgrader{
        CheckOrigin: checkOrigin,
    },
}

// Client with reconnection
transport := &WebSocketTransport{
    URL:            "wss://api.example.com",
    ReconnectDelay: 5 * time.Second,
}
```

## Custom Transports

Implement custom transports for special requirements:

```go
type CustomTransport struct {
    // Custom fields
}

func (t *CustomTransport) Send(ctx context.Context, msg json.RawMessage) error {
    // Custom send logic
    return nil
}

func (t *CustomTransport) Receive(ctx context.Context) (json.RawMessage, error) {
    // Custom receive logic
    return nil, nil
}

func (t *CustomTransport) Close() error {
    // Cleanup
    return nil
}
```

## Transport Adapters

Convert between transport types:

```go
// HTTP to stdio adapter
adapter := &TransportAdapter{
    From: httpTransport,
    To:   stdioTransport,
}
```

## Error Handling

Common transport errors:

```go
// Connection errors
var ErrConnectionLost = errors.New("connection lost")
var ErrTimeout = errors.New("operation timeout")

// Protocol errors
var ErrInvalidMessage = errors.New("invalid message format")
var ErrUnsupportedVersion = errors.New("unsupported protocol version")
```

## Testing Transports

Test utilities for transports:

```go
// Mock transport for testing
transport := &MockTransport{
    SendFunc: func(ctx context.Context, msg json.RawMessage) error {
        // Test logic
        return nil
    },
}

// Transport test harness
harness := &TransportTestHarness{
    Transport: transport,
    Timeout:   5 * time.Second,
}
```

## Performance Considerations

### Buffering

```go
// Buffered transport
transport := &BufferedTransport{
    Transport:  baseTransport,
    BufferSize: 1024,
}
```

### Connection Pooling

```go
// Pooled HTTP transport
transport := &PooledHTTPTransport{
    MaxConns:     100,
    IdleTimeout:  30 * time.Second,
}
```

### Compression

```go
// Compressed transport
transport := &CompressedTransport{
    Transport:   baseTransport,
    Compression: gzip.DefaultCompression,
}
```

## Best Practices

1. **Choose the right transport** for your use case
2. **Handle errors gracefully** with retries and fallbacks
3. **Monitor transport health** with metrics
4. **Implement timeouts** for all operations
5. **Use connection pooling** for HTTP transports
6. **Enable compression** for large payloads
7. **Secure transports** with TLS when needed

## Next Steps

- Learn about [JSON-RPC Messages](./jsonrpc.md)
- Explore [Capabilities](./capabilities.md)
- See [Transport Examples](../examples/transports.md)

## See Also

- [Protocol Overview](./protocol-overview.md)
- [API Reference](../api/transport.md)
- [Custom Transport Guide](../advanced/custom-transports.md)