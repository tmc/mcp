# mcp-serve

Serve MCP protocols over HTTP and Server-Sent Events (SSE) transports.

## Overview

`mcp-serve` wraps stdio-based MCP servers to expose them over HTTP transports, enabling:
- HTTP/SSE endpoints for stdio servers
- Multiple transport options
- Request/response handling
- Connection management

## Usage

```bash
mcp-serve [options] -- command [args...]
```

## Options

### Transport Options
- `-http <addr>` - HTTP listen address (default: `:3001`)
- `-sse` - Enable Server-Sent Events
- `-streamable` - Enable streamable HTTP (chunked responses)

### Logging Options
- `-v` - Verbose output
- `-debug` - Debug mode with detailed logging
- `-log <file>` - Log to file instead of stderr

### Server Options
- `-timeout <duration>` - Request timeout (default: 30s)
- `-max-connections <n>` - Maximum concurrent connections

## Examples

### Basic SSE Server

Expose stdio server over SSE:
```bash
# Serve MCP server over SSE
mcp-serve -sse -- npx @modelcontextprotocol/server-everything stdio

# Custom port
mcp-serve -sse -http :8080 -- node server.js
```

### HTTP Server

Serve over standard HTTP:
```bash
# Basic HTTP server
mcp-serve -- ./mcp-server

# With verbose logging
mcp-serve -v -debug -- go run ./server
```

### Production Deployment

Production-ready configuration:
```bash
# Production settings
mcp-serve \
  -http :443 \
  -sse \
  -timeout 60s \
  -max-connections 1000 \
  -log /var/log/mcp-serve.log \
  -- \
  ./production-server
```

## Transport Modes

### SSE Mode

Server-Sent Events for real-time communication:
```bash
# Enable SSE
mcp-serve -sse -- ./server

# Client connection
curl -N http://localhost:3001/sse
```

### HTTP Mode

Standard HTTP request/response:
```bash
# Basic HTTP
mcp-serve -- ./server

# Client request
curl -X POST http://localhost:3001 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

### Streamable HTTP

Chunked transfer encoding:
```bash
# Enable streaming
mcp-serve -streamable -- ./server
```

## API Endpoints

### `/` - HTTP POST

Standard JSON-RPC endpoint:
```bash
curl -X POST http://localhost:3001 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"test"}'
```

### `/sse` - Server-Sent Events

SSE connection endpoint:
```bash
# Connect to SSE stream
curl -N http://localhost:3001/sse

# With EventSource API
const source = new EventSource('http://localhost:3001/sse');
source.onmessage = (event) => console.log(event.data);
```

### `/health` - Health Check

Server health status:
```bash
curl http://localhost:3001/health
```

## Use Cases

### 1. Web Application Integration

Expose MCP server to web apps:
```javascript
// JavaScript client
const response = await fetch('http://localhost:3001', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    jsonrpc: '2.0',
    id: 1,
    method: 'tools/call',
    params: { name: 'calculator', arguments: { a: 1, b: 2 } }
  })
});
```

### 2. Cross-Network Access

Make local servers accessible:
```bash
# Expose local server
mcp-serve -http :8080 -sse -- ./local-server

# Access from remote
curl -X POST http://server.example.com:8080
```

### 3. Testing and Development

Test HTTP transports:
```bash
# Development server
mcp-serve -v -debug -sse -- npm run dev

# Test with mcp-connect
mcp-connect -transport=sse -url=http://localhost:3001
```

### 4. Production Deployment

Deploy with reverse proxy:
```nginx
# Nginx configuration
location /mcp {
  proxy_pass http://localhost:3001;
  proxy_http_version 1.1;
  proxy_set_header Connection "";
  
  # For SSE
  proxy_set_header Cache-Control no-cache;
  proxy_set_header X-Accel-Buffering no;
}
```

## Configuration

### Environment Variables

- `MCP_SERVE_PORT` - Default HTTP port
- `MCP_SERVE_TIMEOUT` - Default timeout
- `MCP_SERVE_LOG` - Default log file

### TLS/SSL

For HTTPS support, use a reverse proxy:
```bash
# Behind nginx/caddy
mcp-serve -http :3001 -- ./server

# Or with built-in TLS (future feature)
mcp-serve -https :443 -cert cert.pem -key key.pem -- ./server
```

## Error Handling

### Connection Errors

Handle client disconnections:
```bash
# With automatic cleanup
mcp-serve -sse -max-idle 5m -- ./server
```

### Timeout Handling

Configure appropriate timeouts:
```bash
# Long-running operations
mcp-serve -timeout 5m -- ./slow-server

# Quick responses
mcp-serve -timeout 5s -- ./fast-server
```

## Monitoring

### Logs

Monitor server activity:
```bash
# Tail logs
tail -f /var/log/mcp-serve.log

# With structured logging
mcp-serve -log-format json -- ./server
```

### Metrics

Track performance:
```bash
# With metrics endpoint (future)
mcp-serve -metrics :9090 -- ./server

# Prometheus scraping
curl http://localhost:9090/metrics
```

## Best Practices

1. **Use SSE** for real-time applications
2. **Set appropriate timeouts** based on operation duration
3. **Log errors** for debugging
4. **Monitor connections** in production
5. **Use reverse proxy** for TLS and load balancing

## Troubleshooting

### Server Won't Start

Check port availability:
```bash
# Check if port is in use
lsof -i :3001

# Use different port
mcp-serve -http :8080 -- ./server
```

### Connection Drops

Enable keep-alive:
```bash
# With connection management
mcp-serve -sse -keep-alive 30s -- ./server
```

### Performance Issues

Optimize settings:
```bash
# Increase limits
mcp-serve \
  -max-connections 5000 \
  -buffer-size 64KB \
  -- ./server
```

## Integration Examples

### With mcp-proxy

Add monitoring layer:
```bash
# Serve through proxy
mcp-serve -sse -- mcp-proxy -v -t -- ./server
```

### With Docker

Containerized deployment:
```dockerfile
FROM node:18
WORKDIR /app
COPY . .
EXPOSE 3001
CMD ["mcp-serve", "-sse", "--", "node", "server.js"]
```

### In Kubernetes

Deploy as service:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: mcp-server
spec:
  ports:
  - port: 80
    targetPort: 3001
  selector:
    app: mcp-server
```

## See Also

- [mcp-connect](./mcp-connect.md) - Client for HTTP/SSE
- [mcp-proxy](./mcp-proxy.md) - Traffic monitoring
- [Transport Guide](../concepts/transports.md) - Transport details