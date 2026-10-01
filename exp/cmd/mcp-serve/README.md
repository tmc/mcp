# mcp-serve

A utility for managing MCP server instances in tests.

## Overview

`mcp-serve` provides a clean and consistent way to start, stop, and check the status of MCP server processes during testing. It handles:

- Starting server processes
- Managing process lifecycle
- Redirecting stdout/stderr to files
- Cleaning up when tests are done
- Handling signals properly
- Cross-platform compatibility for Linux, macOS, and Windows

## Usage

### Starting a server

```bash
# Basic usage
mcp-serve -- <command> [args...]

# Example
mcp-serve -- mcp-scripttest-server --stdio
```

### Checking server status

```bash
mcp-serve --status
```

### Stopping a running server

```bash
mcp-serve --stop
```

### Sending data to a running server

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | mcp-serve --send
```

### Specifying a workspace

```bash
mcp-serve --workspace=/tmp/my-test-workspace -- <command> [args...]
```

## In Testing Scripts

When used in mcpscripttest scripts, commands would look like:

```
# Start a server
exec mcp-serve -- $server_command

# Send a request
setstdin {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}
exec mcp-serve --send

# Check for expected response in stdout
stdout '"id":1'
stdout '"jsonrpc":"2.0"'

# Stop the server when done
exec mcp-serve --stop
```

## Advanced Options

- `--timeout=<duration>`: Set the timeout for graceful shutdown (default: 10s)
- `--v`: Enable verbose logging
- `--workspace=<path>`: Specify a directory to store server data (PID file, logs)

## Environment Variables

`mcp-serve` sets the following environment variables when a server is running:

- `MCP_SERVER_PID`: The PID of the running server process
- `MCP_ENDPOINT`: The base path for named pipes (.mcp-server)
- `MCP_SERVER_WORKSPACE`: The workspace directory path
- `MCP_SERVER_ADDR`: The server address if running with --http option

## Files and Pipes

In the workspace directory, `mcp-serve` creates the following files:

- `.mcp-server.pid`: Contains the PID of the running server
- `.mcp-server.stdout`: Captures stdout from the server process
- `.mcp-server.stderr`: Captures stderr from the server process
- `.mcp-server.in`: Named pipe for sending input to the server
- `.mcp-server.out`: Named pipe for receiving output from the server

## Cross-Platform Support

`mcp-serve` attempts to use the most efficient IPC mechanism available on each platform:

- On Linux, it can use the /proc filesystem to directly communicate with processes
- On all Unix systems, it uses named pipes (FIFOs) when available
- On Windows and other platforms, it falls back to regular files with special handling

## Example Workflow

```bash
# Start an MCP server
mcp-serve -- go run ./examples/server/main.go

# Send an initialize request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | mcp-serve --send

# Check if server is running
mcp-serve --status

# Stop the server
mcp-serve --stop
```