# Streaming Features in mcpd

This document shows examples of how to use the new streaming features in mcpd.

## Server-Sent Events (SSE)

Server-Sent Events provide a mechanism for servers to push updates to web clients over HTTP. They are well-suited for one-way communication from server to client.

### Starting mcpd with SSE support

```bash
# Start mcpd with SSE support and connect it to an MCP echo server
mcpd -socket /tmp/mcp.sock -http :8081 -enable-sse -- go run ./examples/servers/mcp-echo-server
```

### Connecting to the SSE endpoint

Using curl (in a separate terminal):

```bash
curl -N http://localhost:8081/sse
```

You'll receive events in this format:

```
event: connected
data: {"client_id":"sse-1621234567890"}

event: message
data: {"method":"notification","params":{"message":"Server update"},"jsonrpc":"2.0"}
```

### Using from JavaScript

```javascript
const evtSource = new EventSource("http://localhost:8081/sse");

evtSource.addEventListener("connected", (event) => {
  const data = JSON.parse(event.data);
  console.log("Connected with client ID:", data.client_id);
});

evtSource.addEventListener("message", (event) => {
  const data = JSON.parse(event.data);
  console.log("Received message:", data);
});

evtSource.onerror = (err) => {
  console.error("EventSource error:", err);
};
```

## HTTP Streaming

HTTP streaming allows long-lived HTTP connections where the server can continue sending data in chunks.

### Starting mcpd with HTTP streaming

```bash
# Start mcpd with HTTP streaming and connect it to an MCP server
mcpd -socket /tmp/mcp.sock -http :8081 -enable-stream -- go run ./examples/servers/mcp-echo-server
```

### Connecting to the streaming endpoint

```bash
curl -N http://localhost:8081/stream
```

You'll receive line-delimited JSON responses:

```
{"client_id":"stream-1621234567890","status":"connected"}
{"jsonrpc":"2.0","method":"notification","params":{"message":"Update 1"}}
{"jsonrpc":"2.0","method":"notification","params":{"message":"Update 2"}}
```

## WebSocket Support (Experimental)

WebSocket support provides bidirectional communication over a single, long-lived connection.

### Starting mcpd with WebSocket support

```bash
# Start mcpd with WebSocket support and connect it to an MCP server
mcpd -socket /tmp/mcp.sock -http :8081 -enable-ws -- go run ./examples/servers/mcp-echo-server
```

### Connecting with a WebSocket client

Using the `wscat` tool:

```bash
wscat -c ws://localhost:8081/ws
```

## Combining Multiple Streaming Options

You can enable multiple streaming options simultaneously:

```bash
# Enable all streaming options
mcpd -socket /tmp/mcp.sock -http :8081 -enable-sse -enable-stream -enable-ws -- go run ./examples/servers/mcp-echo-server
```

## Setting a Custom Timeout

By default, streaming connections timeout after 1 hour. You can customize this:

```bash
# Set a 30-minute timeout for streaming connections
mcpd -socket /tmp/mcp.sock -http :8081 -enable-sse -stream-timeout 30m -- go run ./examples/servers/mcp-echo-server
```

## Using with Different Server Modes

The streaming options work with both server modes:

```bash
# Use with 'once' mode (default) - one server for all connections
mcpd -mode once -socket /tmp/mcp.sock -http :8081 -enable-sse -- go run ./examples/servers/mcp-echo-server

# Use with 'per-connection' mode - new server for each connection
mcpd -mode per-connection -socket /tmp/mcp.sock -http :8081 -enable-sse -- go run ./examples/servers/mcp-echo-server
```