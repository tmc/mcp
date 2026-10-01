# mcp-attach

`mcp-attach` is a tool that "attaches" a shell session to a specific mcpd instance.

## Overview

When working with multiple Model Context Protocol (MCP) servers managed by `mcpd` daemons, it's helpful to have a way to isolate which server a particular shell session is interacting with. `mcp-attach` starts a new shell with the appropriate environment variables set to target a specific `mcpd` instance.

This allows MCP client tools to automatically discover and communicate with the correct MCP server without needing explicit socket paths on each command.

## Usage

```
mcp-attach [flags] <service-name|pid|socket-path>
```

### Target Types

`mcp-attach` accepts three types of targets:

1. **Service Name**: A named service (e.g., `myweather`, `database`)
   - Looks for service at `~/.srv/mcp/<service-name>`
   - Falls back to `~/.mcpd/svc.<service-name>.sock`

2. **PID**: The process ID of an mcpd instance (e.g., `12345`)
   - Looks for socket at `~/.mcpd/sock.<pid>` or variants

3. **Socket Path**: Direct path to a Unix domain socket (e.g., `/tmp/mcpd.sock`)
   - Must be a valid Unix domain socket

### Special Names

- `current`: Attaches to the most recently started interactive `mcpd` (using the `~/.mcpd/current` symlink)

## Examples

```bash
# Attach to a service named "myweather"
mcp-attach myweather

# Attach to an mcpd with PID 12345
mcp-attach 12345

# Attach directly to a specific socket
mcp-attach /tmp/my-custom-socket.sock

# Attach to the most recently started mcpd
mcp-attach current

# Use a specific shell
mcp-attach -shell /bin/zsh myweather

# Set a different environment variable
mcp-attach -socket-env MY_MCP_SOCKET weatherservice
```

## Flags

- `-shell string`: Shell to execute (default: `$SHELL` or `/bin/sh`)
- `-socket-env string`: Environment variable to set (default: `MCP_SOCKET_PATH`)
- `-root-dir string`: Root directory for socket discovery (default: `~/.mcpd` or `$XDG_RUNTIME_DIR/mcpd`)
- `-srv-dir string`: Root directory for service discovery (default: `~/.srv/mcp`)
- `-v`: Verbose mode

## Integration with mcpd

`mcp-attach` is designed to work seamlessly with `mcpd`, which manages MCP-compliant server commands. When `mcpd` starts, it:

1. Creates a Unix domain socket
2. Optionally creates a service file in `~/.srv/mcp/<service-name>` containing the socket path
3. May create a `~/.mcpd/current` symlink pointing to its socket

`mcp-attach` uses these conventions to locate and connect to the appropriate `mcpd` instance.

## Environment

The shell started by `mcp-attach` inherits all environment variables from the parent shell, with the addition of:

- `MCP_SOCKET_PATH` (or the value specified by `-socket-env`): Set to the path of the target mcpd's socket

This ensures that MCP client tools automatically use the correct mcpd instance.