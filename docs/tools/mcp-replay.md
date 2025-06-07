# mcp-replay

Record and replay MCP (Model Context Protocol) sessions with timing preservation and mock capabilities.

## Overview

`mcp-replay` is a versatile tool for recording, replaying, and mocking MCP communications. It can:
- Replay recorded MCP sessions with original timing
- Act as a mock server responding to requests
- Act as a mock client sending requests
- Control playback speed and filtering

## Usage

```bash
mcp-replay [options] [trace-file]
```

## Options

### Input/Output Options
- `-f <file>` - Recording file to replay (alternative to positional argument)
- `-o <file>` - Output file (default: stdout)
- `-trace <file>` - Write transmitted messages to trace file

### Playback Control
- `-speed <n>` - Replay speed multiplier (1.0 = original speed)
- `-strip` - Strip timestamps entirely
- `-rel` - Use relative timestamps from start
- `-now` - Use current time for timestamps

### Message Filtering
- `-sends` - Only replay send messages
- `-recvs` - Only replay receive messages
- `-json` - Output only JSON content (no prefix/timestamp)

### Mock Modes
- `-mock-server` - Act as mock server, respond to matching requests
- `-mock-client` - Act as mock client, send requests and expect responses
- `-auto-respond` - Auto-send all server responses in sequence
- `-compare-shadow` - Use shadow responses instead of primary responses (mock-server mode)
- `-timeout <duration>` - Timeout for mock operations (default: 5s)

### Other Options
- `-v` - Verbose output
- `-q` - Quiet mode: suppress log messages
- `-preserve-order` - Preserve message order (default: true)

## Examples

### Basic Replay

Replay a recorded session:
```bash
mcp-replay trace.mcp
```

Replay at double speed:
```bash
mcp-replay -speed 2.0 trace.mcp
```

### Mock Server

Create a mock server from a trace:
```bash
mcp-replay -mock-server server-trace.mcp
```

With automatic responses:
```bash
mcp-replay -mock-server -auto-respond trace.mcp
```

### Mock Client

Create a mock client:
```bash
mcp-replay -mock-client client-requests.mcp | ./server
```

### Filter Messages

Only replay sent messages:
```bash
mcp-replay -sends trace.mcp
```

Only replay received messages:
```bash
mcp-replay -recvs trace.mcp
```

### Timing Control

Strip all timestamps:
```bash
mcp-replay -strip trace.mcp
```

Use relative timestamps:
```bash
mcp-replay -rel trace.mcp
```

## Mock Server Mode

In mock server mode, `mcp-replay`:

1. Reads recorded server responses from the trace file
2. Listens on stdin for matching requests
3. Responds with the appropriate recorded response
4. Maintains session state and request ordering

### Shadow Response Mode

When using `-compare-shadow` flag in mock server mode, `mcp-replay` uses shadow server responses (marked as `mcp-send-shadow` in the trace) instead of primary responses. This is useful for testing alternative implementations:

```bash
# Use shadow responses in mock server
mcp-replay -mock-server -compare-shadow trace-with-shadow.mcp
```

This requires trace files created by `mcp-shadow -compare` that contain both primary and shadow responses.

### Example Mock Server Session

```bash
# Start mock server
mcp-replay -mock-server -v reference-server.mcp

# In another terminal, send requests
echo '{"jsonrpc":"2.0","method":"initialize","id":1}' | nc localhost 8080
```

### Auto-Response Mode

With `-auto-respond`, the mock server automatically sends all responses in sequence without waiting for matching requests:

```bash
mcp-replay -mock-server -auto-respond trace.mcp
```

## Mock Client Mode

In mock client mode, `mcp-replay`:

1. Reads recorded client requests from the trace file
2. Sends requests to stdout (piped to server)
3. Reads responses from stdin
4. Optionally validates responses against recorded ones

### Example Mock Client Session

```bash
# Use mock client to test a server
mcp-replay -mock-client test-requests.mcp | ./my-server | tee responses.log
```

## Trace File Format

Trace files use the MCP trace format:
```
mcp-send {"jsonrpc":"2.0","method":"test","id":1} # 1234567890.123
mcp-recv {"jsonrpc":"2.0","result":"ok","id":1} # 1234567890.456
```

## Advanced Usage

### Pipeline Testing

Test server with recorded client behavior:
```bash
mcp-replay -mock-client client.mcp | ./server | mcp-replay -mock-server server.mcp
```

### Speed Testing

Test server under accelerated load:
```bash
mcp-replay -speed 10.0 -mock-client stress-test.mcp | ./server
```

### Response Timing Analysis

Extract timing information:
```bash
mcp-replay -rel trace.mcp | grep "mcp-recv" | awk '{print $NF}'
```

### Continuous Integration

Use in CI pipelines:
```bash
#!/bin/bash
# Test server against reference behavior
mcp-replay -mock-client ci-test.mcp | ./server > actual.log
mcp-replay -recvs ci-test.mcp > expected.log
diff expected.log actual.log
```

## Best Practices

1. **Record comprehensive traces** during development for testing
2. **Use mock server mode** for client development
3. **Use mock client mode** for server testing
4. **Preserve timing** when testing performance-sensitive code
5. **Strip timestamps** when comparing behavior only

## Error Handling

- Timeout errors in mock mode indicate missing responses
- Malformed JSON errors suggest corrupted trace files
- Use `-v` for detailed error diagnostics

## Integration Examples

### With mcp-spy

Record sessions for replay:
```bash
# Record
mcp-spy -f session.mcp -- ./server

# Replay
mcp-replay session.mcp
```

### With mcp-shadow

Test with shadow server responses:
```bash
# Create trace with shadow responses
mcp-shadow -primary "./server-v1" -shadow "./server-v2" -compare -o shadow.mcp

# Replay using shadow responses
mcp-replay -mock-server -shadow shadow.mcp
```

### With mcpdiff

Compare replayed outputs:
```bash
mcp-replay trace1.mcp > output1.log
mcp-replay trace2.mcp > output2.log
mcpdiff output1.log output2.log
```

### With mcp-sort

Sort before replay:
```bash
mcp-sort -timestamp trace.mcp | mcp-replay
```

## Troubleshooting

### Mock server not responding

Check request format matches trace:
```bash
mcp-replay -v -mock-server trace.mcp
```

### Timing issues

Adjust speed or disable timing:
```bash
mcp-replay -speed 0 trace.mcp  # Instant playback
mcp-replay -strip trace.mcp     # No timestamps
```

### Memory usage with large traces

Stream processing for large files:
```bash
tail -f large-trace.mcp | mcp-replay
```

## Performance Considerations

- Large trace files are streamed, not loaded entirely
- Mock server mode maintains minimal state
- Auto-respond mode may consume more memory

## See Also

- [mcp-spy](./mcp-spy.md) - Record MCP traces
- [mcp-shadow](../../cmd/mcp-shadow/README.md) - Create traces with shadow responses
- [mcpdiff](./mcpdiff.md) - Compare traces
- [Testing Guide](../testing/README.md) - Testing strategies