# mcp-connect

Universal MCP client for all transport types (stdio, SSE, HTTP).

## Overview

`mcp-connect` is a versatile MCP client that can connect to servers using any transport type. It provides:
- Interactive and batch modes
- Support for all MCP transports
- Script execution capabilities
- Direct command sending

## Usage

```bash
mcp-connect [options]
```

## Options

### Transport Options
- `-transport <type>` - Transport type: `stdio`, `sse`, `http` (default: stdio)
- `-url <url>` - Server URL for HTTP/SSE transports
- `-cmd <command>` - Command to run for stdio transport

### Operation Modes
- `-script <file>` - Execute requests from script file
- `-request <json>` - Send single request and exit
- `-interactive` - Force interactive mode (default for TTY)

### Output Options
- `-json` - Output raw JSON responses
- `-pretty` - Pretty-print JSON (default: true)
- `-quiet` - Suppress informational messages

## Examples

### Stdio Transport

Connect to stdio server:
```bash
# Interactive mode
mcp-connect -cmd="npx @modelcontextprotocol/server-everything stdio"

# With specific command
mcp-connect -cmd="go run ./server/main.go"
```

### SSE Transport

Connect to SSE server:
```bash
# Connect to SSE endpoint
mcp-connect -transport=sse -url=http://localhost:3001

# With authentication
mcp-connect -transport=sse -url=https://api.example.com -header="Authorization: Bearer TOKEN"
```

### HTTP Transport

Connect to HTTP server:
```bash
# Connect to HTTP endpoint
mcp-connect -transport=http -url=http://localhost:3001
```

### Script Mode

Execute requests from file:
```bash
# Run script
mcp-connect -script=test-requests.json

# Script with specific transport
mcp-connect -transport=sse -url=http://localhost:3001 -script=integration-test.json
```

## Script File Format

Script files contain JSON requests, one per line:
```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/execute","params":{"name":"calculator","arguments":{"operation":"add","a":1,"b":2}}}
```

## Interactive Mode

In interactive mode, you can type commands:
```
> initialize
> tools/list
> tools/execute calculator {"operation":"add","a":1,"b":2}
> exit
```

## Single Request Mode

Send one request and exit:
```bash
# Send initialize request
mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'

# Execute tool
mcp-connect -cmd="./server" -request='{"jsonrpc":"2.0","id":1,"method":"tools/execute","params":{"name":"echo","arguments":{"message":"test"}}}'
```

## Use Cases

### 1. Development Testing

Test server during development:
```bash
# Quick interactive testing
mcp-connect -cmd="go run ./server"

# Test specific endpoints
mcp-connect -transport=sse -url=http://localhost:3001
```

### 2. Integration Testing

Run automated tests:
```bash
#!/bin/bash
# Run integration tests
mcp-connect -script=integration-tests.json > results.log
if grep -q "error" results.log; then
  echo "Tests failed"
  exit 1
fi
```

### 3. Production Monitoring

Check server health:
```bash
# Health check
mcp-connect -url=https://api.example.com -request='{"jsonrpc":"2.0","id":1,"method":"health"}'
```

### 4. Debugging

Debug protocol issues:
```bash
# With mcp-proxy for logging
mcp-connect -cmd="mcp-proxy -v -- ./server"

# Direct connection with verbose output
mcp-connect -v -transport=sse -url=http://localhost:3001
```

## Environment Variables

- `MCP_TIMEOUT` - Request timeout (default: 30s)
- `MCP_HEADERS` - Additional headers for HTTP transports
- `MCP_DEBUG` - Enable debug logging

## Error Handling

- Automatic reconnection for transient failures
- Timeout handling for hung requests
- Clear error messages for protocol violations

## Integration Examples

### With mcp-proxy

Connect through proxy:
```bash
# Start proxy
mcp-proxy -transport=tcp -listen=:7000 -- ./server

# Connect to proxy
mcp-connect -transport=tcp -url=localhost:7000
```

### With mcp-spy

Monitor connection:
```bash
# Connect with monitoring
mcp-connect -cmd="mcp-spy -v -- ./server"
```

### In Scripts

```bash
#!/bin/bash
# Automated testing script

# Start server
./server &
SERVER_PID=$!

# Wait for startup
sleep 2

# Run tests
mcp-connect -script=tests.json

# Cleanup
kill $SERVER_PID
```

## Best Practices

1. **Use scripts** for repeatable tests
2. **Set timeouts** for production use
3. **Log responses** for debugging
4. **Check exit codes** in automation
5. **Use environment variables** for configuration

## Troubleshooting

### Connection Failed

Check server is running:
```bash
# For stdio
ps aux | grep server

# For HTTP/SSE
curl http://localhost:3001/health
```

### Timeout Errors

Increase timeout:
```bash
MCP_TIMEOUT=60s mcp-connect -cmd="./slow-server"
```

### Invalid JSON

Validate request format:
```bash
# Test JSON parsing
echo '{"jsonrpc":"2.0","id":1,"method":"test"}' | jq .
```

## See Also

- [mcp-serve](./mcp-serve.md) - Server management
- [mcp-proxy](./mcp-proxy.md) - Connection monitoring
- [Transport Guide](../concepts/transports.md) - Transport details