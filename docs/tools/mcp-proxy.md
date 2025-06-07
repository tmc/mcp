# mcp-proxy

Transparent proxy for MCP communications with support for multiple transport types and optional mcpspy integration.

## Overview

`mcp-proxy` acts as a transparent proxy between MCP clients and servers, supporting:
- Standard I/O (stdio) transport
- TCP socket transport 
- HTTP/SSE transport
- Optional `mcpspy` wrapping for enhanced logging
- Real-time request/response monitoring

## Usage

```bash
mcp-proxy [options] -- command [args...]
```

## Options

### General Options
- `-transport <type>` - Transport type: `stdio`, `tcp`, `http` (default: `stdio`)
- `-v` - Verbose output
- `-t` - Include timestamps
- `-listen <addr>` - Listen address for TCP/HTTP proxy (default: `:8080`)
- `-target <url>` - Target URL for HTTP proxy

### mcpspy Integration Options
- `-spy` - Wrap server command with mcpspy
- `-spy-v` - Pass `-v` flag to mcpspy
- `-spy-vv` - Pass `-vv` flag to mcpspy  
- `-spy-pretty` - Pass `-pretty` flag to mcpspy
- `-spy-f <file>` - Pass `-f` flag to mcpspy for recording

## Transport Modes

### 1. Stdio Transport

Standard input/output proxying:

```bash
# Basic stdio proxy
mcp-proxy -transport stdio -- node server.js

# With verbose logging and timestamps
mcp-proxy -transport stdio -v -t -- go run ./server

# With mcpspy integration
mcp-proxy -transport stdio -spy -spy-v -- npx @modelcontextprotocol/server-everything stdio
```

### 2. TCP Transport

Listen on TCP socket and spawn server on each connection:

```bash
# Listen on port 7000
mcp-proxy -transport tcp -listen :7000 -- npx @modelcontextprotocol/server-everything stdio

# With mcpspy wrapping
mcp-proxy -transport tcp -listen :7000 -spy -spy-vv -- node server.js

# Record traces with mcpspy
mcp-proxy -transport tcp -listen :7000 -spy -spy-f traces.mcp -- ./server
```

### 3. HTTP Transport

Reverse proxy for HTTP/SSE servers:

```bash
# Proxy HTTP server
mcp-proxy -transport http -listen :8080 -target http://localhost:3001

# With logging
mcp-proxy -transport http -v -t -listen :8080 -target http://localhost:3001
```

## Examples

### TCP Proxy with mcpspy (socat replacement)

The TCP transport mode enables the workflow requested:

```bash
# Original socat command:
export SPYCMD="npx @modelcontextprotocol/server-everything stdio"
socat TCP-LISTEN:7000,fork,reuseaddr EXEC:"mcpspy -v -vv -- ${SPYCMD}"

# Equivalent with mcp-proxy:
mcp-proxy -transport tcp -listen :7000 -spy -spy-v -spy-vv -- npx @modelcontextprotocol/server-everything stdio
```

### Multiple Connections

TCP mode handles multiple simultaneous connections:

```bash
# Start TCP proxy
mcp-proxy -transport tcp -listen :7000 -spy -spy-pretty -- ./server

# Connect multiple clients
nc localhost 7000  # Client 1
nc localhost 7000  # Client 2
nc localhost 7000  # Client 3
```

Each connection spawns a new server instance.

### Recording Sessions

Record all sessions to trace files:

```bash
# Record with timestamps
mcp-proxy -transport tcp -listen :7000 -spy -spy-f "trace-$(date +%Y%m%d-%H%M%S).mcp" -- ./server
```

### Development Workflow

Monitor and debug during development:

```bash
# Maximum verbosity for debugging
mcp-proxy -transport tcp -listen :7000 -v -t -spy -spy-vv -spy-pretty -- go run ./server/main.go
```

## Output Format

### Default Output

```
ℹ INFO: TCP proxy listening on :7000
ℹ INFO: New connection from 127.0.0.1:52341
→ REQUEST: method=initialize id=1
← RESPONSE: id=1 success
```

### Verbose Output

With `-v` flag:
```
[10:30:45.123] → REQUEST: {
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
[10:30:45.456] ← RESPONSE: {
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "1.0"
  }
}
```

## Integration Examples

### With mcp-connect

```bash
# Start proxy server
mcp-proxy -transport tcp -listen :7000 -- ./server &

# Connect with mcp-connect
mcp-connect -transport tcp -url localhost:7000
```

### With Docker

```bash
# In Dockerfile
EXPOSE 7000
CMD ["mcp-proxy", "-transport", "tcp", "-listen", ":7000", "--", "node", "server.js"]
```

### In Scripts

```bash
#!/bin/bash
# start-mcp-server.sh

# Start proxy with logging
mcp-proxy \
  -transport tcp \
  -listen :${MCP_PORT:-7000} \
  -spy \
  -spy-f "logs/mcp-$(date +%Y%m%d).log" \
  -- \
  node server.js
```

## Advanced Features

### Graceful Shutdown

The TCP proxy handles SIGINT/SIGTERM:
```bash
# Start proxy
mcp-proxy -transport tcp -listen :7000 -- ./server

# Graceful shutdown
kill -TERM $PID
```

### Connection Logging

Each connection is logged with client address:
```
ℹ INFO: New connection from 192.168.1.100:45678
ℹ INFO: Connection from 192.168.1.100:45678 closed
```

### Error Handling

Detailed error reporting:
```
✗ ERROR: accept error: use of closed network connection
✗ ERROR: command exited with error: exit status 1
```

## Security Considerations

1. **Bind Address**: Use specific interfaces instead of `:7000`
   ```bash
   mcp-proxy -transport tcp -listen 127.0.0.1:7000 -- ./server
   ```

2. **Firewall**: Ensure ports are properly firewalled

3. **Command Injection**: The proxy executes commands directly, so validate inputs

## Performance Tips

1. **Buffering**: The proxy uses buffered I/O for efficiency

2. **Concurrent Connections**: Each TCP connection runs in a separate goroutine

3. **Resource Limits**: Set system limits for maximum connections
   ```bash
   ulimit -n 10000  # Increase file descriptor limit
   ```

## Troubleshooting

### Port Already in Use

```bash
# Check what's using the port
lsof -i :7000

# Use a different port
mcp-proxy -transport tcp -listen :7001 -- ./server
```

### Connection Refused

```bash
# Ensure server is running
mcp-proxy -v -transport tcp -listen :7000 -- ./server

# Check firewall rules
sudo iptables -L
```

### mcpspy Not Found

```bash
# Ensure mcpspy is in PATH
which mcpspy

# Or specify full path
PATH=/usr/local/bin:$PATH mcp-proxy -spy -- ./server
```

## Comparison with socat

| Feature | socat | mcp-proxy |
|---------|-------|-----------|
| TCP Listen | ✓ | ✓ |
| Fork on Accept | ✓ | ✓ (goroutines) |
| MCP Logging | ✗ | ✓ |
| mcpspy Integration | Manual | Built-in |
| JSON-RPC Parsing | ✗ | ✓ |
| Multiple Transports | ✗ | ✓ |

## See Also

- [mcp-spy](./mcp-spy.md) - Traffic monitoring
- [mcp-connect](./mcp-connect.md) - MCP client
- [TCP Socket Guide](../advanced/tcp-sockets.md)
- [Security Guide](../advanced/security.md)