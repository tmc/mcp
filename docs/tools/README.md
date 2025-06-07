# MCP Command Line Tools

The MCP project provides a comprehensive suite of command-line tools for working with the Model Context Protocol. These tools help with debugging, testing, monitoring, and managing MCP communications.

## Core Tools

### Traffic Monitoring & Analysis

- **[mcp-spy](./mcp-spy.md)** - Monitor and record MCP protocol traffic
  ```bash
  mcp-spy -v -- go run ./examples/servers/mcp-echo-server
  ```

- **[mcpdiff](./mcpdiff.md)** - Compare MCP trace files and highlight differences
  ```bash
  mcpdiff trace1.mcp trace2.mcp
  ```

### Session Management

- **[mcp-replay](./mcp-replay.md)** - Record and replay MCP sessions
  ```bash
  mcp-replay -mock-server trace.mcp
  ```

- **[mcp-sort](./mcp-sort.md)** - Sort trace files by timestamp or strip timestamps
  ```bash
  mcp-sort -timestamp trace.mcp > sorted.mcp
  ```

### Client & Server Tools

- **[mcp-connect](./mcp-connect.md)** - Universal MCP client for all transport types
  ```bash
  mcp-connect -transport=sse -url=http://localhost:3001
  ```

- **[mcp-serve](./mcp-serve.md)** - Serve MCP protocols over various transports
  ```bash
  mcp-serve -http=:8080 -- node server.js
  ```

### Network Tools

- **[mcp-proxy](./mcp-proxy.md)** - Proxy MCP connections between transports
  ```bash
  mcp-proxy -v -t -- npx @modelcontextprotocol/server-everything stdio
  ```

- **[mcp-shadow](./mcp-shadow.md)** - Shadow MCP traffic for testing and monitoring
  ```bash
  mcp-shadow -input=prod.mcp -output=shadow.mcp
  ```

## Tool Categories

### Development Tools
Tools for building and testing MCP implementations:
- `mcp-spy` - Debug protocol interactions
- `mcp-replay` - Test with recorded sessions
- `mcpdiff` - Compare protocol behavior

### Operations Tools
Tools for running MCP in production:
- `mcp-serve` - Production server deployment
- `mcp-proxy` - Traffic routing and monitoring
- `mcp-connect` - Client connectivity

### Analysis Tools
Tools for understanding MCP traffic:
- `mcpdiff` - Trace comparison
- `mcp-sort` - Trace organization
- `mcp-shadow` - Traffic analysis

## Quick Start Examples

### Monitor a Server
```bash
# Start a server with monitoring
mcp-spy -v -f trace.mcp -- go run ./examples/servers/mcp-time-server
```

### Compare Two Implementations
```bash
# Record traces from two different servers
mcp-spy -f server1.mcp -- ./server1
mcp-spy -f server2.mcp -- ./server2

# Compare the traces
mcpdiff server1.mcp server2.mcp
```

### Test with Mock Data
```bash
# Create a mock server from a trace
mcp-replay -mock-server reference.mcp

# Create a mock client
mcp-replay -mock-client test-requests.mcp
```

## Tool Workflows

### Development Workflow
```bash
# 1. Start server with monitoring
mcp-spy -v -pretty -- go run ./server/main.go

# 2. Connect and test
mcp-connect -script=test-requests.json

# 3. Analyze results
mcpdiff expected.mcp actual.mcp
```

### Testing Workflow
```bash
# 1. Record reference behavior
mcp-spy -f reference.mcp -- ./known-good-server

# 2. Test new implementation
mcp-replay -mock-client reference.mcp | ./new-server

# 3. Compare results
mcpdiff reference.mcp new-server.mcp
```

### Debugging Workflow
```bash
# 1. Capture failing interaction
mcp-spy -vv -f debug.mcp -- ./failing-server

# 2. Replay specific requests
mcp-replay -sends debug.mcp | ./server

# 3. Analyze responses
mcp-replay -recvs debug.mcp
```

## Installation

See the [Installation Guide](../getting-started/installation.md) for detailed instructions.

## Common Options

Most MCP tools share common options:

- `-v` - Verbose output
- `-f <file>` - Output file
- `-h` - Help and usage information
- `-version` - Tool version

## Environment Variables

Some tools respect these environment variables:

- `MCP_TRACE_DIR` - Default directory for trace files
- `MCP_VERBOSE` - Enable verbose output by default
- `MCP_TIMEOUT` - Default timeout for operations

## See Also

- [Getting Started Guide](../getting-started/README.md)
- [Testing Guide](../testing/README.md)
- [Examples](../examples/README.md)