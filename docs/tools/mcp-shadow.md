# mcp-shadow

Shadow MCP traffic to test alternative server implementations and compare responses.

## Overview

`mcp-shadow` runs a primary MCP server alongside one or more shadow servers, forwarding all requests to all servers and optionally comparing their responses. This is useful for:
- Testing new server implementations against production servers
- Comparing behavior between different versions
- Load testing shadow implementations
- Safe production testing without affecting clients

## Usage

```bash
mcp-shadow [options] --primary <cmd> --shadow <cmd> [--shadow <cmd>...]
```

## Options

### Server Configuration
- `--primary <cmd>` - Primary server command (required)
- `--shadow <cmd>` - Shadow server command (can be specified multiple times)
- `--timeout <duration>` - Timeout for server responses (default: 30s)

### Comparison Options
- `--compare` - Compare shadow responses with primary responses
- `--ignore-timestamps` - Ignore timestamps in comparison
- `--ignore-ids` - Ignore JSON-RPC IDs in comparison
- `--semantic` - Use semantic JSON comparison

### Output Options
- `-o <file>` - Output file for trace (default: stdout)
- `-v` - Verbose output
- `--diff-only` - Only show differences, not matching responses
- `--format <type>` - Output format: `trace`, `json`, `diff` (default: trace)

### Transport Options
- `--transport <type>` - Transport type: `stdio`, `tcp`, `http` (default: stdio)
- `--listen <addr>` - Listen address for tcp/http (default: :8080)

## Examples

### Basic Shadow Testing

Compare two server implementations:
```bash
mcp-shadow --primary "./server-v1" --shadow "./server-v2" --compare
```

### Multiple Shadow Servers

Test against multiple alternatives:
```bash
mcp-shadow \
  --primary "npm run server:prod" \
  --shadow "./server-experimental" \
  --shadow "./server-development" \
  --compare
```

### Production Shadowing

Shadow production traffic to test server:
```bash
mcp-shadow \
  --primary "node production-server.js" \
  --shadow "node test-server.js" \
  --transport tcp \
  --listen :7000 \
  -o shadow-$(date +%Y%m%d).mcp
```

### Detailed Comparison

Compare with semantic JSON matching:
```bash
mcp-shadow \
  --primary "./server-stable" \
  --shadow "./server-canary" \
  --compare \
  --semantic \
  --ignore-timestamps \
  --ignore-ids \
  --diff-only
```

## Output Formats

### Trace Format (default)

Standard MCP trace with shadow responses:
```
mcp-send {"jsonrpc":"2.0","method":"test","id":1} # 1234567890.123
mcp-recv {"jsonrpc":"2.0","result":"ok","id":1} # 1234567890.456
mcp-recv-shadow {"jsonrpc":"2.0","result":"ok","id":1} # 1234567890.457
```

### JSON Format

Structured comparison output:
```json
{
  "request": {"jsonrpc":"2.0","method":"test","id":1},
  "primary": {"jsonrpc":"2.0","result":"ok","id":1},
  "shadow": {"jsonrpc":"2.0","result":"ok","id":1},
  "match": true,
  "latency": {
    "primary": 10,
    "shadow": 11
  }
}
```

### Diff Format

Shows only differences:
```diff
Request: {"jsonrpc":"2.0","method":"calculate","id":2}
- Primary: {"jsonrpc":"2.0","result":{"value":42},"id":2}
+ Shadow:  {"jsonrpc":"2.0","result":{"value":43},"id":2}
```

## Shadow Response Format

When shadow servers produce different responses, they're recorded as:
```
mcp-recv {"jsonrpc":"2.0","result":"primary","id":1} # 1234567890.123
mcp-recv-shadow {"jsonrpc":"2.0","result":"shadow","id":1} # 1234567890.124
```

These shadow responses can then be used by `mcp-replay` with the `-shadow` flag.

## Use Cases

### 1. A/B Testing

Test new implementations before deployment:
```bash
mcp-shadow \
  --primary "./server-v1.0" \
  --shadow "./server-v2.0-rc1" \
  --compare \
  -o "ab-test-$(date +%Y%m%d).mcp"
```

### 2. Performance Testing

Compare latency between implementations:
```bash
mcp-shadow \
  --primary "./server-optimized" \
  --shadow "./server-baseline" \
  --compare \
  --format json | jq '.latency'
```

### 3. Regression Testing

Ensure new version behaves identically:
```bash
#!/bin/bash
mcp-shadow \
  --primary "./server-stable" \
  --shadow "./server-dev" \
  --compare \
  --semantic \
  --diff-only > differences.log

if [ -s differences.log ]; then
  echo "Regression detected!"
  exit 1
fi
```

### 4. Canary Testing

Test experimental features in production:
```bash
mcp-shadow \
  --primary "node server.js --production" \
  --shadow "node server.js --experimental" \
  --transport tcp \
  --listen :8080 \
  --compare \
  -o canary.mcp
```

## Integration Examples

### With mcp-replay

Use shadow responses for testing:
```bash
# Create trace with shadow responses
mcp-shadow --primary "./v1" --shadow "./v2" --compare -o trace.mcp

# Replay using shadow responses
mcp-replay -mock-server -shadow trace.mcp
```

### With mcpdiff

Compare shadow traces:
```bash
# Create two shadow traces
mcp-shadow --primary "./baseline" --shadow "./test1" -o trace1.mcp
mcp-shadow --primary "./baseline" --shadow "./test2" -o trace2.mcp

# Compare the shadow responses
mcpdiff -compare trace1.mcp trace2.mcp
```

### With mcp-proxy

Shadow through TCP proxy:
```bash
# Start proxy with shadowing
mcp-proxy -transport tcp -listen :7000 -- \
  mcp-shadow --primary "./server" --shadow "./test-server"
```

## Performance Considerations

- Shadow servers run in parallel, not affecting primary latency
- Comparison is done asynchronously after responses
- Large payloads may increase memory usage
- Use `--timeout` to prevent hanging on slow shadows

## Best Practices

1. **Start Simple**: Begin with one shadow server
2. **Monitor Resources**: Shadow servers consume additional CPU/memory
3. **Use Semantic Comparison**: For meaningful difference detection
4. **Log Everything**: Save traces for later analysis
5. **Test Incrementally**: Add shadows one at a time

## Troubleshooting

### Shadow server not starting

Check command and permissions:
```bash
mcp-shadow -v --primary "echo test" --shadow "./server"
```

### Comparison showing false differences

Use semantic comparison options:
```bash
mcp-shadow --compare --semantic --ignore-timestamps --ignore-ids
```

### High memory usage

Limit response buffering:
```bash
mcp-shadow --max-response-size 1MB
```

## Security Considerations

- Shadow servers receive all production data
- Ensure shadow servers have appropriate access controls
- Consider data privacy when logging traces
- Use separate environments for sensitive data

## See Also

- [mcp-replay](./mcp-replay.md) - Replay traces with shadow responses
- [mcpdiff](./mcpdiff.md) - Compare trace files
- [mcp-proxy](./mcp-proxy.md) - TCP proxy for shadowing
- [Testing Guide](../testing/README.md) - Shadow testing strategies