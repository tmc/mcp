# mcp-proxy

A transparent proxy for monitoring and debugging MCP communications across all transport types.

## Overview

`mcp-proxy` sits between MCP clients and servers, logging all requests and responses while passing them through transparently. It supports:

- **stdio**: Process-based communication
- **tcp**: Raw TCP socket connections (similar to socat)
- **http**: HTTP and SSE transports

## Features

- Transparent proxying of all MCP messages
- Request/response logging with timestamps
- Verbose mode for full message details
- Support for all MCP transport types
- TCP socket server mode with per-connection process spawning
- Optional mcpspy integration for enhanced logging
- Zero modification to client or server code

## Installation

```bash
go build -o mcp-proxy main.go
```

## Usage

### STDIO Proxy

Monitor communication between a client and stdio server:

```bash
# Basic usage
mcp-proxy -- npx @modelcontextprotocol/server-everything stdio

# With timestamps and verbose output
mcp-proxy -v -t -- npx @modelcontextprotocol/server-everything stdio

# With mcpspy wrapping
mcp-proxy -spy -spy-v -- node server.js
```

### TCP Socket Server

Listen on a TCP port and spawn server processes for each connection:

```bash
# Basic TCP server (like socat TCP-LISTEN)
mcp-proxy -transport=tcp -listen=:7000 -- npx @modelcontextprotocol/server-everything stdio

# With mcpspy integration (equivalent to socat with mcpspy)
mcp-proxy -transport=tcp -listen=:7000 -spy -spy-v -spy-vv -- node server.js

# Full example replacing socat command:
# Original: socat TCP-LISTEN:7000,fork,reuseaddr EXEC:"mcpspy -v -vv -- ${SPYCMD}"
# New:      mcp-proxy -transport=tcp -listen=:7000 -spy -spy-v -spy-vv -- ${SPYCMD}
```

### HTTP Proxy

Monitor HTTP-based transports (SSE, streamableHttp):

```bash
# Start the actual server
npx @modelcontextprotocol/server-everything sse

# Start proxy (listening on :8080, forwarding to :3001)
mcp-proxy -transport=http -listen=:8080 -target=http://localhost:3001

# Connect client to proxy instead of server
mcp-connect -transport=sse -url=http://localhost:8080
```

## Examples

### TCP Server with Enhanced Logging

```bash
# Start TCP proxy with full logging
mcp-proxy -transport=tcp -listen=:7000 -v -t -spy -spy-vv -spy-f trace.mcp -- node server.js

# Connect with netcat
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | nc localhost 7000

# Or connect with mcp-connect
mcp-connect -transport=tcp -url=localhost:7000
```

### Development Workflow

```bash
# Monitor stdio server with mcp-connect
mcp-connect -cmd="mcp-proxy -v -t -- npx @modelcontextprotocol/server-everything stdio"

# Debug HTTP transport
mcp-proxy -transport=http -v -listen=:8080 -target=http://localhost:3001
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `-transport` | Transport type (stdio, tcp, http) | stdio |
| `-v` | Verbose output (full JSON) | false |
| `-t` | Include timestamps | false |
| `-listen` | TCP/HTTP proxy listen address | :8080 |
| `-target` | HTTP proxy target URL | http://localhost:3001 |
| `-spy` | Wrap command with mcpspy | false |
| `-spy-v` | Pass -v to mcpspy | false |
| `-spy-vv` | Pass -vv to mcpspy | false |
| `-spy-pretty` | Pass -pretty to mcpspy | false |
| `-spy-f` | Pass -f to mcpspy for recording | "" |

## Output Format

### Normal Mode
```
→ REQUEST: method=initialize id=1
← RESPONSE: id=1 success
→ REQUEST: method=tools/list id=2
← RESPONSE: id=2 success
```

### Verbose Mode
```
→ REQUEST: {
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
← RESPONSE: {
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {}
  }
}
```

### With Timestamps
```
[15:04:05.123] → REQUEST: method=initialize id=1
[15:04:05.234] ← RESPONSE: id=1 success
```

### TCP Mode with Connections
```
ℹ INFO: TCP proxy listening on :7000
ℹ INFO: New connection from 127.0.0.1:52341
[15:04:05.123] → REQUEST: method=initialize id=1
[15:04:05.234] ← RESPONSE: id=1 success
ℹ INFO: Connection from 127.0.0.1:52341 closed
```

## Use Cases

1. **Debugging**: See exactly what messages are being exchanged
2. **Development**: Monitor your MCP implementation
3. **Testing**: Verify correct protocol usage
4. **Learning**: Understand MCP message flow
5. **Network Services**: Expose MCP servers over TCP sockets
6. **Production Monitoring**: Log all MCP traffic with mcpspy

## TCP Mode Features

The TCP transport provides functionality similar to `socat`:

- Listen on specified port (like `TCP-LISTEN:7000`)
- Fork new process for each connection (like `fork`)
- Reuse address automatically (like `reuseaddr`)
- Execute command for each connection (like `EXEC`)
- Optional mcpspy wrapping for enhanced logging

## Comparison with Other Tools

| Feature | socat | mcpspy | mcp-proxy |
|---------|-------|--------|-----------|
| TCP Listen | ✓ | ✗ | ✓ |
| Process Forking | ✓ | ✗ | ✓ |
| MCP Awareness | ✗ | ✓ | ✓ |
| mcpspy Integration | Manual | N/A | Built-in |
| Multiple Transports | ✗ | stdio only | All |
| JSON Parsing | ✗ | ✓ | ✓ |

## Integration

Use with other MCP tools:

```bash
# With mcp-connect via TCP
mcp-proxy -transport=tcp -listen=:7000 -- node server.js &
mcp-connect -transport=tcp -url=localhost:7000

# In testing scripts
mcp-proxy -spy -spy-f test.mcp -- npm test

# For development with live monitoring
mcp-proxy -v -t -spy -spy-vv -- go run ./my-mcp-server
```

## Testing

```bash
# Run test suite
./test_tcp.sh

# Test specific connection
./test_client.sh localhost 7000
```