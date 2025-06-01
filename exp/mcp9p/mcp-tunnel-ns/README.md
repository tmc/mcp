# mcp-tunnel

A tunneling service for exposing local Model Context Protocol (MCP) servers to the cloud through secure WebSocket connections.

## Overview

mcp-tunnel consists of two components:

1. **Server**: A Cloud Run deployable service that provides public HTTP/SSE endpoints and manages WebSocket tunnels
2. **Client**: A local CLI tool that connects to the server and forwards requests to local MCP servers

## Architecture

```
Internet -> Cloud Run (mcp-tunnel server) <--WebSocket--> Local Machine (mcp-tunnel client) -> Local MCP Server
```

## Features

- Support for all MCP transport types (stdio, SSE, streamableHttp)
- Secure WebSocket tunneling with token authentication
- Automatic reconnection and keep-alive
- Public URLs with unique tunnel IDs
- Minimal latency overhead

## Installation

### Server Deployment (Cloud Run)

1. Clone the repository:
```bash
git clone https://github.com/tmc/mcp
cd mcp/cmd/mcp-tunnel/server
```

2. Deploy to Cloud Run:
```bash
gcloud builds submit --config cloudbuild.yaml
```

Or manually:
```bash
# Build and push the image
docker build -t gcr.io/YOUR_PROJECT/mcp-tunnel -f Dockerfile ../../..
docker push gcr.io/YOUR_PROJECT/mcp-tunnel

# Deploy to Cloud Run
gcloud run deploy mcp-tunnel \
  --image gcr.io/YOUR_PROJECT/mcp-tunnel \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

### Client Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-tunnel/client@latest
```

## Usage

### Basic Usage (stdio transport)

```bash
# Expose a local MCP server running on stdio
mcp-tunnel -- npx @modelcontextprotocol/server-everything stdio

# Output:
# Creating tunnel...
# Tunnel established!
# Public URL: https://mcp-tunnel-xxxxx.run.app/tunnels/abc123
# Local command: npx @modelcontextprotocol/server-everything stdio
# Transport: stdio
```

### HTTP Transport

```bash
# For servers already running on HTTP
mcp-tunnel -transport=http -- localhost:3000

# Or
mcp-tunnel -transport=http -- http://localhost:3000
```

### SSE Transport

```bash
# For SSE-enabled servers
mcp-tunnel -transport=sse -- localhost:3000/sse
```

### Custom Server URL

```bash
# Connect to a self-hosted tunnel server
mcp-tunnel -server=https://my-tunnel-server.com -- command
```

### Verbose Mode

```bash
# Show additional debugging information
mcp-tunnel -v -- command
```

## API

### Server Endpoints

- `POST /tunnels` - Create a new tunnel
- `GET /tunnels/:id` - Proxy requests to tunnel (HTTP/SSE)
- `GET /tunnels/:id/ws` - WebSocket connection endpoint

### Message Protocol

The WebSocket protocol uses JSON messages:

```json
// Request from server to client
{
  "type": "request",
  "id": "unique-id",
  "payload": {...}  // Original MCP request
}

// Response from client to server
{
  "type": "response",
  "id": "unique-id",
  "payload": {...},  // MCP response
  "error": "..."     // Optional error message
}

// Keep-alive
{
  "type": "ping"
}
{
  "type": "pong"
}
```

## Security

- Token-based authentication for WebSocket connections
- HTTPS/WSS encryption for all communications
- Unique tunnel IDs prevent unauthorized access
- Automatic tunnel expiration after inactivity

## Development

### Running Locally

1. Start the server:
```bash
cd cmd/mcp-tunnel/server
go run main.go
```

2. Start the client:
```bash
cd cmd/mcp-tunnel/client
go run main.go -- npx @modelcontextprotocol/server-everything stdio
```

### Testing

```bash
# Test with a simple echo server
mcp-tunnel -- go run examples/echo_example.go

# Test with different transports
mcp-tunnel -transport=sse -- npx @modelcontextprotocol/server-everything sse
mcp-tunnel -transport=http -- npx @modelcontextprotocol/server-everything streamableHttp
```

## Troubleshooting

### Connection Issues

- Ensure the server is deployed and accessible
- Check network connectivity and firewall rules
- Verify the correct transport type is specified

### Authentication Errors

- Token mismatch or expiration
- Check server logs for detailed error messages

### Performance

- Use appropriate buffer sizes for large responses
- Consider connection pooling for HTTP transports
- Monitor Cloud Run metrics for scaling issues

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - See LICENSE file for details