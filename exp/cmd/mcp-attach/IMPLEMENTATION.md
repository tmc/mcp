# mcp-attach Implementation Details

`mcp-attach` is a specialized tool designed to create shell sessions that are "attached" to specific MCP daemon instances (mcpd).

## Key Functionality

1. **Target Resolution**: `mcp-attach` can locate mcpd sockets via multiple methods:
   - Direct socket path (e.g., `/tmp/mcpd.sock`)
   - Process ID (e.g., `12345`, resolves to `~/.mcpd/sock.12345`)
   - Service name (e.g., `myweather`, resolves via `~/.srv/mcp/myweather`)
   - The special name `current` (uses the `~/.mcpd/current` symlink)

2. **Environment Setup**: It starts a new shell with environment variables set to target the specific mcpd:
   - `MCP_SOCKET_PATH`: Path to the mcpd Unix domain socket
   - This allows MCP client tools like `mcp-rpc` to automatically discover and use the correct mcpd

3. **Socket Discovery**: It implements conventions for where sockets and service files are stored:
   - `~/.mcpd/` (default) or `$XDG_RUNTIME_DIR/mcpd/` for socket files
   - `~/.srv/mcp/` for service name discovery

## Discovery Convention

The socket discovery implements these conventions:

1. **PID-based Discovery**:
   - Socket path: `~/.mcpd/sock.<PID>`, `~/.mcpd/<PID>.sock`, or `~/.mcpd/mcpd.<PID>.sock`

2. **Service Name Discovery**:
   - Looks for service file: `~/.srv/mcp/<service-name>`
   - This file contains the path to the actual socket

3. **"Current" Instance Discovery**:
   - Uses symlink: `~/.mcpd/current` → points to the socket of the most recently started interactive mcpd

## Plan 9 Inspiration

This design draws inspiration from Plan 9's approach to services and namespaces:

1. **Services as Files**: Unix domain sockets are used as the primary IPC mechanism
2. **Contextual Environment**: Shell sessions inherit a specific MCP context through environment variables
3. **Hierarchical Discovery**: Service files provide a simple lookup mechanism

## Integration with mcpd

For the full system to work, `mcpd` should:

1. Create sockets with predictable names (`~/.mcpd/sock.<PID>`)
2. Optionally register services (`~/.srv/mcp/<service-name>`)
3. Update the "current" symlink when appropriate
4. Report socket paths on startup

## Usage Examples

```bash
# Attach to mcpd by PID
mcp-attach 12345

# Attach to a named service
mcp-attach weather-service

# Attach to the most recently started mcpd
mcp-attach current

# Attach directly to a specific socket
mcp-attach /path/to/custom/socket.sock

# Use a specific shell
mcp-attach -shell /bin/zsh 12345
```

## Future Enhancements

Potential future improvements:

1. **Service Discovery Protocol**: mcpd could implement a discovery protocol that allows clients to query for available services
2. **Interactive Selection**: When multiple mcpd instances are running, offer an interactive menu to select one
3. **Nested mcpd Support**: Enhance support for the "graph-like" structure described in notes.md, where one mcpd can route to others