# mcp-shadow

The `mcp-shadow` tool forwards MCP traffic to a primary server while also shadowing requests to a secondary server. This is useful for:

- Testing new server implementations without affecting production traffic
- Comparing responses between different server versions
- Load testing shadow servers with real traffic patterns
- A/B testing different server behaviors

## Features

- Forwards traffic to both primary and shadow servers concurrently
- Records both responses in mcptrace format with OpenTelemetry trace context support
- Comments out shadow server responses with `# ` prefix (legacy mode)
- Supports traffic splitting/sampling modes
- Links shadow responses to original requests using span IDs
- Preserves trace context for distributed tracing
- Compare mode: outputs enhanced mcptrace format with automatic response correlation

## Usage

```bash
# Basic usage: forward to primary and shadow servers
mcp-shadow -primary "mcp-server-v1" -shadow "mcp-server-v2" -o trace.mcp

# With trace context generation
mcp-shadow -primary "mcp-server-v1" -shadow "mcp-server-v2" -trace -o trace.mcp

# With custom baggage
mcp-shadow -primary "mcp-server-v1" -shadow "mcp-server-v2" -trace -baggage "env=test,version=2.0" -o trace.mcp

# With random sampling (50% to shadow)
mcp-shadow -primary "mcp-server-v1" -shadow "mcp-server-v2" -split-mode random -split-percent 50 -o trace.mcp

# Compare mode - enhanced mcptrace format for comparison
mcp-shadow -primary "mcp-server-v1" -shadow "mcp-server-v2" -compare -trace -o compare.mcp
```

## Options

- `-primary`: Command to execute for the primary server (required)
- `-shadow`: Command to execute for the shadow server (required)
- `-o`: Output file for mcptrace recording
- `-v`: Verbose mode
- `-q`: Quiet mode
- `-trace`: Generate OpenTelemetry trace context
- `-baggage`: Trace-level baggage (key=value,key=value)
- `-timeout`: Timeout for shadow server responses (default: 5s)
- `-split-mode`: Traffic splitting mode: shadow, random, round-robin (default: shadow)
- `-split-percent`: Percentage of traffic to shadow (0-100, default: 100)
- `-compare`: Output enhanced mcptrace format with automatic response correlation

## Output Format

The tool produces an extended mcptrace format that includes both primary and shadow responses.

### Legacy Format (default)
Shadow responses are commented out with `# ` prefix:

```
# mcptrace:v1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01 baggage=shadow=true
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000 spanid=aaaaaaaaaaaaaaa1
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550 spanid=aaaaaaaaaaaaaaa2
# mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true,"shadow":true}}} # 1683000000.560 spanid=bbbbbbbbbbbbbbb2 linksto=aaaaaaaaaaaaaaa2 baggage=shadow=true
```

### Compare Format (-compare mode)
Shadow responses use distinct direction (`send-shadow`):

```
# mcptrace:v1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01 baggage=shadow=true compare=true
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000 spanid=aaaaaaaaaaaaaaa1
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550 spanid=aaaaaaaaaaaaaaa2
mcp-send-shadow {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true,"shadow":true}}} # 1683000000.560 spanid=bbbbbbbbbbbbbbb2 linksto=aaaaaaaaaaaaaaa2 baggage=shadow=true,compare=true
```

Note:
- In legacy mode: Shadow responses are prefixed with `# ` to comment them out
- In compare mode: All responses are preserved with correlation metadata
- Shadow responses include `linksto` referencing the request span
- Shadow responses have `shadow=true` in their baggage
- Compare mode adds `compare=true` to the mcptrace header

## Pipeline Usage

```bash
# Use with mcp-replay to compare responses
mcp-shadow -primary "server-v1" -shadow "server-v2" -o shadow.mcp < requests.mcp
mcp-replay -compare-mode shadow.mcp

# Export to OpenTelemetry for visualization
mcp-shadow -primary "server-v1" -shadow "server-v2" -trace -o shadow.mcp < requests.mcp
mcp-replay -export-otel shadow.mcp
```

## Examples

### Testing a New Server Version

```bash
# Record traffic with shadow testing
echo '{"jsonrpc":"2.0","method":"initialize","id":1}' | mcp-shadow \
  -primary "mcp-server --version 1.0" \
  -shadow "mcp-server --version 2.0" \
  -trace -o comparison.mcp

# Analyze differences
mcp-replay -filter "baggage.shadow=true" comparison.mcp > shadow-responses.mcp
mcp-replay -filter "baggage.shadow!=true" comparison.mcp > primary-responses.mcp
mcpdiff primary-responses.mcp shadow-responses.mcp
```

### Load Testing with Sampling

```bash
# Send 10% of traffic to shadow server for load testing
mcp-shadow \
  -primary "mcp-server --prod" \
  -shadow "mcp-server --test" \
  -split-mode random \
  -split-percent 10 \
  -trace -baggage "test=load,percent=10" \
  -o load-test.mcp
```

### A/B Testing Features

```bash
# Test new feature with 50/50 split
mcp-shadow \
  -primary "mcp-server --feature-off" \
  -shadow "mcp-server --feature-on" \
  -split-mode random \
  -split-percent 50 \
  -trace -baggage "test=ab,feature=new-algorithm" \
  -o ab-test.mcp

# Analyze results
mcp-replay -export-otel ab-test.mcp
```

## See Also

- `mcpspy`: Records MCP interactions
- `mcp-replay`: Replays MCP recordings
- `mcpdiff`: Compares MCP trace files
- `mcp-tsnorm`: Normalizes timestamps in trace files
