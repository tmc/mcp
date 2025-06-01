# mcpd - MCP Server Daemon

`mcpd` manages and provides network access to MCP-compliant server commands that communicate over stdin/stdout. It defaults to using Unix domain sockets for client connections and logs all interactions to an MCP trace file.

## Overview

`mcpd` acts as a bridge between clients and MCP servers that communicate via stdin/stdout. It:

1. Starts an MCP server command
2. Listens for client connections on a Unix socket (default) or TCP port
3. Passes client requests to the server's stdin
4. Returns server responses from stdout back to clients
5. Logs all traffic to an MCP trace file for recording and analysis
6. Optionally handles interactive prompts directly via TTY

## Usage

```
mcpd [flags] -- <server_command> [server_args...]
```

The `--` separator is mandatory to distinguish between `mcpd` flags and the server command with its arguments.

## Example Usage

Start a simple time server and expose it via a Unix socket:

```sh
mcpd -log-file time-server.mcp -- go run ./examples/servers/mcp-time-server
```

Start an echo server on TCP port 8080:

```sh
mcpd -tcp :8080 -log-file echo-server.mcp -- go run ./examples/servers/mcp-echo-server
```

Use the per-connection mode to create a new server instance for each client:

```sh
mcpd -mode per-connection -log-file per-conn.mcp -- go run ./examples/servers/mcp-echo-server
```

## Main Features

- **Network Bridging**: Exposes stdio-based MCP servers over Unix domain sockets or TCP
- **Trace Logging**: Records all interactions to an MCP trace file (.mcp format)
- **Server Management**:
  - `once` mode (default): One server instance handles all client connections
  - `per-connection` mode: Starts a new server instance for each client connection
- **Auto-Discovery**: Outputs the listening endpoint for easy connection by clients
- **Process Management**: Optional PID file management for both mcpd and server processes

## Flags

```
-log-file string
    MCP trace file path (.mcp format)
-mode string
    Server lifecycle mode: once (default), per-connection (default "once")
-pid-file string
    Path to write PID file
-server-log string
    Log file for server stdout/stderr (default: stderr)
-server-pid-file string
    Path to write server process PID file
-socket string
    Unix domain socket path (default: auto-generated)
-tcp string
    TCP address to listen on (e.g., :8080)
-timeout duration
    Auto-terminate after specified duration (for testing)
-v    Enable verbose logging
-i    Run in interactive mode, handling prompts via TTY
-no-tty-prompt
    Disable TTY prompting in interactive mode
```

## Primary Use Case

`mcpd` is particularly useful for:

1. Exposing stdio-based MCP servers over network interfaces
2. Testing MCP servers in isolated environments (e.g., scripttest)
3. Creating reliable trace recordings of MCP interactions
4. Converting local processes into network services
5. Providing interactive shell-like sessions with MCP servers

## Trace Files

All client-server interactions are logged to the specified trace file in `.mcp` format, which can be analyzed or replayed using other MCP tools like `mcp-replay`.

## Interactive Mode

In interactive mode (enabled with the `-i` flag), mcpd can handle interactive prompts from the server directly via the TTY:

1. When a server sends a special `interactive/promptUser` request, mcpd displays the prompt to the user
2. The user inputs a response via the terminal
3. mcpd collects the input and sends it back to the server as an `interactive/userInput` request
4. The server receives the user's input and continues processing

This creates a seamless experience similar to `script -r`, where an interactive session can be recorded and replayed.

### Interactive Example

```sh
$ mcpd -i -socket /tmp/interactive.sock -- ./my_interactive_server
SERVICE_SOCKET=/tmp/interactive.sock
```

In another terminal:
```sh
$ echo '{"jsonrpc":"2.0","id":"1","method":"prompt","params":{}}' | nc -U /tmp/interactive.sock
```

The first terminal will display a prompt, collect input, and send it back to the server.

## Integration with mcp-attach

For non-interactive usage or when multiple shells need to interact with the same server, the `mcp-attach` tool can be used to "attach" a shell session to a specific mcpd instance:

```sh
mcp-attach 12345  # Attach to mcpd with PID 12345
```