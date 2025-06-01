# MCP Command-Line Tools Architecture

## Overview

The MCP project includes a comprehensive suite of command-line tools for development, debugging, monitoring, and testing Model Context Protocol implementations.

## Tool Categories

### 1. Core Protocol Tools

#### Client/Server Implementation
- **`mcp-serve`**: Start an MCP server from any command
  - Wraps existing programs to expose them as MCP servers
  - Supports stdio transport by default
  
- **`mcp-connect`**: Connect to MCP servers
  - Client implementation for testing servers
  - Supports multiple transport types
  
- **`mcp-send`**: Send individual MCP messages
  - Useful for testing specific protocol interactions
  - One-shot message sending
  
- **`mcp-probe`**: Probe server capabilities
  - Tests server initialization and tool discovery
  - Validates protocol compliance

### 2. Monitoring & Debugging

#### Traffic Analysis
- **`mcpspy`**: Primary traffic monitoring tool
  - Logs all MCP messages with timestamps
  - Supports JSON pretty-printing
  - Creates `.mcp` trace files
  
- **`mcp-proxy`**: Intercept and modify traffic
  - Man-in-the-middle proxy for debugging
  - Can modify messages in flight
  
- **`mcp-debug`**: Interactive debugging
  - Step through protocol interactions
  - Inspect message contents

### 3. Trace Processing

#### Analysis Tools
- **`mcpcat`**: Colorize trace files
  - Human-readable display of `.mcp` files
  - Syntax highlighting for different message types
  
- **`mcpdiff`**: Compare trace files
  - Diff two `.mcp` files
  - Identify protocol differences
  - (Note: `-compare` flag for shadow traces not implemented)
  
- **`mcp-sort`**: Sort by timestamp
  - Reorder trace events chronologically
  - Handle out-of-order messages
  
- **`mcptrace-to-otel`**: OpenTelemetry conversion
  - Export traces to observability platforms
  - Integration with distributed tracing

### 4. Testing & Validation

#### Replay & Shadow Testing
- **`mcp-replay`**: Replay recorded sessions
  - Reproduce protocol interactions
  - Regression testing
  - Mock server/client modes
  
- **`mcp-shadow`**: Shadow server testing
  - Run primary and shadow servers in parallel
  - Compare responses for compatibility
  - A/B testing for implementations

### 5. Utility Tools

#### Protocol Testing
- **`test-*`** utilities:
  - `test-id`: Message ID handling
  - `test-jsonrpc`: JSON-RPC compliance
  - `test-minimal`: Minimum protocol support
  - `test-response`: Response validation
  - `json-test`: JSON parsing/generation
  - `raw-test`: Raw protocol handling
  - `simple-test`: Basic functionality

#### Inspection
- **`inspect-id`**: Analyze message IDs
  - Debug ID generation issues
  - Trace request/response pairs

## Data Flow Architecture

```
┌─────────────────┐
│  Implementation │
├─────────────────┤
│ • mcp-serve     │
│ • mcp-connect   │
│ • mcp-send      │
│ • mcp-probe     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Monitoring    │
├─────────────────┤
│ • mcpspy        │ ──────► .mcp trace files
│ • mcp-proxy     │
│ • mcp-debug     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Analysis     │
├─────────────────┤
│ • mcpcat        │ ◄────── .mcp trace files
│ • mcpdiff       │
│ • mcp-sort      │
│ • mcptrace-to-otel │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Testing      │
├─────────────────┤
│ • mcp-replay    │ ◄────── .mcp trace files
│ • mcp-shadow    │
│ • test-*        │
└─────────────────┘
```

## Common Workflows

### 1. Development Workflow
```bash
# Start a server
mcp-serve -- python my_server.py

# Test with probe
mcp-probe localhost:8080

# Send specific messages
mcp-send '{"method":"initialize","params":{}}' 

# Monitor traffic
mcpspy -- mcp-connect localhost:8080
```

### 2. Debugging Workflow
```bash
# Capture trace
mcpspy -f trace.mcp -- mcp-serve -- python server.py

# View colorized
mcpcat trace.mcp

# Compare with reference
mcpdiff reference.mcp trace.mcp

# Replay for debugging
mcp-replay trace.mcp
```

### 3. Testing Workflow
```bash
# A/B test implementations
mcp-shadow --primary "python server_v1.py" --shadow "python server_v2.py"

# Regression test
mcp-replay regression-tests/*.mcp

# Sort and analyze
mcp-sort unsorted.mcp > sorted.mcp
mcpcat sorted.mcp
```

## File Format

### MCP Trace Format (.mcp)
- Line-based format
- Each line: `direction JSON # timestamp [metadata]`
- Directions: `mcp-send`, `mcp-recv`, `mcp-send-shadow`, `mcp-recv-shadow`
- Metadata: `spanid`, `parentid`, `baggage`

Example:
```
mcp-recv {"method":"initialize","id":1} # 1234567890.123 spanid=abc123
mcp-send {"result":{},"id":1} # 1234567890.456 spanid=def456
```

## Integration Points

1. **mcpscripttest**: Test framework integration
   - Most tools available as commands
   - `mcp-probe` added to default tools
   - Scripttest format for testing

2. **Transport Support**:
   - stdio (default)
   - HTTP/SSE
   - TCP sockets
   - Unix domain sockets

3. **Observability**:
   - OpenTelemetry via `mcptrace-to-otel`
   - Structured logging
   - Trace correlation with span IDs

## Known Limitations

1. **mcpdiff**: Missing `-compare` flag for shadow trace comparison
2. **Test Coverage**: Some tools lack comprehensive tests
3. **Tool Discovery**: Not all tools are in default mcpscripttest list
4. **Documentation**: Some tools need better usage documentation

## Future Enhancements

1. **Unified Configuration**: Common config format for all tools
2. **Plugin System**: Extensible analysis and processing
3. **Better Integration**: Tighter coupling between related tools
4. **Performance Tools**: Latency and throughput analysis
5. **Security Tools**: Protocol security validation