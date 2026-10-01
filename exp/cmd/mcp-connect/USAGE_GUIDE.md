# MCP-Connect Usage Guide

## Overview

`mcp-connect` is a unified client that works with all MCP transport types:
- **stdio**: Direct process communication
- **sse**: Server-Sent Events over HTTP
- **http**: Streamable HTTP transport

## Quick Start

```bash
# Build the tool
go build -o mcp-connect main.go

# Basic usage (stdio)
./mcp-connect

# Use different transport
./mcp-connect -transport=sse -url=http://localhost:3001
./mcp-connect -transport=http -url=http://localhost:3001
```

## Transport-Specific Examples

### STDIO Transport

Default transport, runs MCP server as subprocess:

```bash
# Interactive mode
./mcp-connect
# Type: {"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}

# Single request
./mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'

# Custom command
./mcp-connect -cmd="node my-mcp-server.js"
```

### SSE Transport

For servers using Server-Sent Events:

```bash
# Start server (in another terminal)
npx @modelcontextprotocol/server-everything sse

# Connect and send request
./mcp-connect -transport=sse -url=http://localhost:3001 \
  -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

### HTTP Transport

For streamable HTTP servers:

```bash
# Start server (in another terminal)
npx @modelcontextprotocol/server-everything streamableHttp

# Connect and send request
./mcp-connect -transport=http -url=http://localhost:3001 \
  -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

## Script Mode

Process multiple requests from a file:

```bash
# Create request file
cat > requests.txt << EOF
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"message":"Hello"}}}
EOF

# Run script
./mcp-connect -script=requests.txt
```

## Common Tasks

### Initialize and List Tools
```bash
# Create initialization script
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mcp-connect","version":"1.0.0"}}}' > init.json
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' >> init.json

# Run with any transport
./mcp-connect -script=init.json                          # stdio
./mcp-connect -transport=sse -script=init.json          # sse
./mcp-connect -transport=http -script=init.json         # http
```

### Call a Tool
```bash
# Echo tool example
./mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"message":"Hello MCP"}}}'
```

### Interactive Exploration
```bash
# Start interactive session
./mcp-connect

# Initialize
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}

# List available methods
{"jsonrpc":"2.0","id":2,"method":"prompts/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":4,"method":"resources/list","params":{}}
```

## Command Line Options

| Option | Description | Example |
|--------|-------------|---------|
| `-transport` | Transport type (stdio, sse, http) | `-transport=sse` |
| `-url` | Server URL for HTTP transports | `-url=http://localhost:3001` |
| `-cmd` | Custom command for stdio | `-cmd="node server.js"` |
| `-request` | Single JSON-RPC request | `-request='{"jsonrpc":"2.0",...}'` |
| `-script` | File with multiple requests | `-script=requests.txt` |
| `-v` | Verbose output | `-v` |

## Tips

1. **Default behavior**: Without options, uses stdio with `npx @modelcontextprotocol/server-everything stdio`
2. **Error handling**: Use `-v` flag for detailed error messages
3. **Port conflicts**: SSE and HTTP servers use port 3001 by default
4. **Script format**: One JSON request per line, empty lines ignored
5. **Interactive mode**: Best for exploration and testing

## Troubleshooting

### Connection Refused
```bash
# Make sure server is running
npx @modelcontextprotocol/server-everything sse
```

### Invalid JSON
```bash
# Use single quotes to wrap JSON
./mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"test","params":{}}'
```

### Port Already in Use
```bash
# Kill existing processes
pkill -f "@modelcontextprotocol/server-everything"
```

## Integration Examples

### Shell Script
```bash
#!/bin/bash
# test-mcp.sh
RESPONSE=$(./mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}')
echo "$RESPONSE" | jq '.result.tools[].name'
```

### Python Integration
```python
import subprocess
import json

def call_mcp(method, params=None):
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params or {}
    }
    
    cmd = ["./mcp-connect", "-request", json.dumps(request)]
    result = subprocess.run(cmd, capture_output=True, text=True)
    
    return json.loads(result.stdout)

# Usage
response = call_mcp("tools/list")
print(response)
```

## Summary

`mcp-connect` provides a unified interface for all MCP transports, making it easy to:
- Test MCP servers regardless of transport
- Switch between transports without changing code
- Script complex interactions
- Integrate MCP into existing workflows

Whether you're developing an MCP server or building a client application, `mcp-connect` simplifies the process of working with the Model Context Protocol.