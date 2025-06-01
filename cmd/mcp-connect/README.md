# mcp-connect

A unified client for all MCP transport types (stdio, SSE, streamableHttp).

## Overview

`mcp-connect` provides a single interface to communicate with MCP servers regardless of their transport mechanism. It supports:

- **stdio**: Process-based communication (default)
- **sse**: Server-Sent Events over HTTP
- **http**: Streamable HTTP transport

## Installation

```bash
cd /path/to/mcp/cmd/mcp-connect
go build -o mcp-connect main.go
```

## Usage

### Basic Usage

```bash
# Default stdio transport with server-everything
mcp-connect

# Specify transport type
mcp-connect -transport=sse -url=http://localhost:3001

# Custom stdio command
mcp-connect -transport=stdio -cmd="node my-server.js"
```

### Examples

#### STDIO Transport
```bash
# Interactive mode
mcp-connect

# Single request
mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'

# Script mode
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' > requests.txt
mcp-connect -script=requests.txt
```

#### SSE Transport
```bash
# Connect to SSE server
mcp-connect -transport=sse -url=http://localhost:3001

# With specific request
mcp-connect -transport=sse -url=http://localhost:3001 \
  -request='{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

#### HTTP Transport
```bash
# Connect to streamableHttp server
mcp-connect -transport=http -url=http://localhost:3001

# With verbose output
mcp-connect -transport=http -url=http://localhost:3001 -v \
  -request='{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### Script Mode

Create a file with multiple requests (one per line):

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"TestClient","version":"1.0.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"message":"Hello"}}}
```

Run the script:
```bash
mcp-connect -script=test-requests.txt
```

## Options

- `-transport`: Transport type (stdio, sse, http)
- `-url`: Server URL for HTTP-based transports
- `-cmd`: Command to run for stdio transport
- `-request`: Single JSON-RPC request to send
- `-script`: File containing multiple requests
- `-timeout`: Request timeout (default: 10s)
- `-v`: Verbose output

## Transport Details

### STDIO
- Default transport
- Runs command in subprocess
- Communicates via stdin/stdout
- Best for CLI tools

### SSE
- Connects to `/sse` endpoint
- Gets session-specific message endpoint
- Maintains persistent connection
- Best for web apps

### HTTP (Streamable)
- Posts to `/mcp` endpoint
- Requires Accept headers
- Handles streaming responses
- Best for HTTP APIs

## Examples

### Initialize and List Tools
```bash
# Create script
cat > test.txt << EOF
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mcp-connect","version":"1.0.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
EOF

# Run with different transports
mcp-connect -script=test.txt                              # stdio
mcp-connect -transport=sse -script=test.txt               # sse
mcp-connect -transport=http -script=test.txt              # http
```

### Interactive Session
```bash
# Start interactive session
mcp-connect

# Type requests manually
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
```

## Error Handling

The tool will report errors for:
- Connection failures
- Invalid JSON
- Transport-specific issues
- Timeout errors

Use `-v` flag for detailed error information.

## Future Enhancements

- [ ] Transport auto-detection based on URL
- [ ] Support for batch requests
- [ ] Configuration file support
- [ ] Response filtering/formatting
- [ ] Shell-style command parsing