# mcp-spy

Monitor and record MCP (Model Context Protocol) interactions between clients and servers.

## Overview

`mcp-spy` acts as a transparent proxy that sits between an MCP client and server, logging all communications while passing them through unchanged. It's essential for debugging, testing, and understanding MCP protocol interactions.

## Usage

```bash
mcp-spy [options] -- command [args...]
```

## Options

- `-f <file>` - Output recording file (MCP trace format)
- `-v` - Verbose mode: print interactions to stderr
- `-vv` - Very verbose mode: print raw stdin/stdout/stderr
- `-a` - Append to existing file instead of overwriting
- `-no-stderr` - Do not copy stderr from command
- `-pretty` - Pretty-print JSON output
- `-pipe` - Explicitly enable pipe mode
- `-indent <n>` - Indentation level for output
- `-indent-char <s>` - Character(s) to use for indentation
- `-pass-through` - Pass JSON through unmodified (no mcp- prefix)
- `-auto-indent` - Automatically determine indentation level
- `-unbuffered` - Force unbuffered output
- `-q` - Quiet mode: suppress log messages
- `-no-echo` - Don't echo received messages to stdout

## Examples

### Basic Monitoring

Monitor an MCP server and print interactions:
```bash
mcp-spy -v -- go run ./examples/servers/mcp-echo-server
```

### Record to File

Save all interactions to a trace file:
```bash
mcp-spy -f server-trace.mcp -- node my-mcp-server.js
```

### Pretty JSON Output

Monitor with formatted JSON:
```bash
mcp-spy -v -pretty -- ./mcp-server
```

### Pipeline Monitoring

Use in a pipeline with auto-indentation:
```bash
cat requests.json | mcp-spy -auto-indent -- ./server | mcp-spy -auto-indent -- ./client
```

### Debugging Mode

Maximum verbosity for debugging:
```bash
mcp-spy -vv -pretty -f debug.mcp -- ./problematic-server
```

## How It Works

1. `mcp-spy` starts the specified command as a subprocess
2. It creates pipes for stdin, stdout, and optionally stderr
3. All data passing through is logged with timestamps
4. Messages are formatted with `mcp-send` or `mcp-recv` prefixes
5. The subprocess runs normally, unaware of the monitoring

## Output Format

### Trace File Format

Each line in the trace file follows this format:
```
mcp-send {"jsonrpc":"2.0","method":"test","id":1} # 1234567890.123
mcp-recv {"jsonrpc":"2.0","result":"ok","id":1} # 1234567890.456
```

### Verbose Output

With `-v`, interactions are printed to stderr:
```
[2024-01-15 10:30:00.123] --> {"jsonrpc":"2.0","method":"initialize","id":1}
[2024-01-15 10:30:00.456] <-- {"jsonrpc":"2.0","result":{...},"id":1}
```

## Common Use Cases

### Development

Monitor server during development:
```bash
mcp-spy -v -pretty -- go run ./cmd/server/main.go
```

### Testing

Record interactions for testing:
```bash
mcp-spy -f test-case.mcp -- ./server < test-inputs.json
```

### Debugging

Debug protocol issues:
```bash
mcp-spy -vv -f debug.mcp -no-stderr -- ./failing-server
```

### CI/CD

Use in automated testing:
```bash
mcp-spy -q -f ci-trace.mcp -- make test-server
```

## Advanced Features

### Pipeline Support

`mcp-spy` can be chained in pipelines:
```bash
echo '{"jsonrpc":"2.0","method":"test","id":1}' | 
  mcp-spy -v -- ./server | 
  mcp-spy -v -- ./client
```

### Auto-indentation

Automatically indent nested spy instances:
```bash
mcp-spy -auto-indent -- mcp-spy -auto-indent -- ./server
```

### Pass-through Mode

Pass JSON without modification:
```bash
mcp-spy -pass-through -- ./server
```

## Error Handling

- Returns the exit code of the monitored process
- Logs errors to stderr unless `-q` is specified
- Handles SIGINT/SIGTERM gracefully

## Best Practices

1. **Always use `-v` during development** for immediate feedback
2. **Record traces with `-f`** for later analysis
3. **Use `-pretty`** when reading JSON manually
4. **Add `-no-stderr`** when stderr contains non-MCP output
5. **Use `-q`** in production or CI environments

## Integration Examples

### With mcp-replay

Record and replay sessions:
```bash
# Record
mcp-spy -f session.mcp -- ./server

# Replay
mcp-replay -mock-server session.mcp
```

### With mcpdiff

Compare server behaviors:
```bash
mcp-spy -f server1.mcp -- ./server1
mcp-spy -f server2.mcp -- ./server2
mcpdiff server1.mcp server2.mcp
```

### With mcp-connect

Monitor client connections:
```bash
mcp-connect -cmd="mcp-spy -v -- ./server"
```

## Troubleshooting

### No output appearing

Check if the server is actually producing output:
```bash
mcp-spy -vv -- ./server
```

### Buffering issues

Force unbuffered output:
```bash
mcp-spy -unbuffered -- ./server
```

### Permission errors

Ensure the trace file directory is writable:
```bash
mcp-spy -f /tmp/trace.mcp -- ./server
```

## See Also

- [mcp-replay](./mcp-replay.md) - Replay recorded sessions
- [mcpdiff](./mcpdiff.md) - Compare trace files
- [MCP Testing Guide](../testing/README.md)