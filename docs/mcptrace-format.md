# MCP Trace Collection Format

A standardized format for organizing collections of MCP trace files.

## Directory Structure

```
session-2024-01-15T10-30-45-abc123def/
├── manifest.json                    # Session metadata and file index
├── stdio.mcp                       # Main stdio transport trace
├── sse-session-abc123def.mcp        # SSE transport trace (if applicable)
├── websocket-conn-456ghi789.mcp     # WebSocket transport trace (if applicable)
├── shadow-stdio.mcp                 # Shadow trace for stdio (if recording)
├── logs/                           # Associated log files
│   ├── client.log
│   ├── server.log
│   └── proxy.log
└── metadata/                       # Additional session data
    ├── client-info.json
    ├── server-info.json
    └── timing.json
```

## Naming Convention

### Session Directory
```
session-{ISO8601-timestamp}-{session-id}
```
- `{ISO8601-timestamp}`: UTC timestamp when session started (YYYY-MM-DDTHH-mm-ss)
- `{session-id}`: Short unique identifier (8-12 hex chars)

**Examples:**
- `session-2024-01-15T10-30-45-abc123def456`
- `session-2024-12-18T15-22-13-f4a2b8c1e9d7`

### Trace Files

#### Transport-based naming:
- `stdio.mcp` - Standard I/O transport
- `sse-session-{sse-session-id}.mcp` - Server-Sent Events transport
- `websocket-conn-{connection-id}.mcp` - WebSocket transport
- `http-{request-sequence}.mcp` - HTTP request/response pairs

#### Shadow recordings:
- `shadow-{transport}.mcp` - Shadow trace for comparison

#### Multi-client scenarios:
- `stdio-client-{client-id}.mcp`
- `sse-session-{sse-session-id}-client-{client-id}.mcp`

## Manifest Format

`manifest.json` contains session metadata:

```json
{
  "version": "1.0",
  "session": {
    "id": "abc123def456",
    "created": "2024-01-15T10:30:45.123Z",
    "duration": 125.456,
    "status": "completed"
  },
  "client": {
    "name": "claude-desktop",
    "version": "0.5.2",
    "transport": "stdio"
  },
  "server": {
    "name": "mcp-filesystem-server", 
    "version": "1.2.0",
    "command": ["node", "dist/index.js", "/home/user/workspace"]
  },
  "transports": [
    {
      "type": "stdio",
      "file": "stdio.mcp",
      "primary": true,
      "entries": 245,
      "size": 125840
    },
    {
      "type": "sse",
      "file": "sse-session-abc123def.mcp", 
      "sessionId": "abc123def",
      "url": "http://localhost:8080/sse",
      "entries": 89,
      "size": 45120
    }
  ],
  "shadows": [
    {
      "original": "stdio.mcp",
      "shadow": "shadow-stdio.mcp",
      "purpose": "validation"
    }
  ],
  "tags": ["development", "filesystem", "debugging"],
  "notes": "Testing filesystem operations with large directory listing"
}
```

## File Format Extensions

### Enhanced Trace Header
```
# mcptrace:v1
# session-id: abc123def456
# transport: sse
# sse-session-id: def456ghi789
# client: claude-desktop/0.5.2
# server: mcp-filesystem-server/1.2.0
# created: 2024-01-15T10:30:45.123Z
```

### Cross-reference Events
For multi-transport sessions, traces can reference each other:
```
# cross-ref: sse-session-abc123def.mcp:line-42
mcp-recv {"jsonrpc":"2.0","id":1,"method":"initialize","params":{}} # 1734567890.123
```

## Discovery and Tooling

### Directory Recognition
Tools should recognize trace collections by:
1. Presence of `manifest.json`
2. Directory name matching `session-*` pattern
3. At least one `.mcp` file

### Collection Tools
- `mcptrace-collect` - Create collection from individual traces
- `mcptrace-viewer` - Enhanced to handle collections
- `mcptrace-export` - Export collection to various formats
- `mcptrace-analyze` - Cross-transport analysis

## Use Cases

### Development Debugging
```
session-2024-01-15T10-30-45-abc123def/
├── manifest.json
├── stdio.mcp                 # Main interaction
└── logs/
    └── server.log            # Server-side logs
```

### Multi-transport Testing
```
session-2024-01-15T14-22-10-def456ghi/
├── manifest.json
├── stdio.mcp                 # Initial handshake
├── sse-session-abc123.mcp    # Streaming operations
├── websocket-conn-456.mcp    # Real-time updates
└── shadow-stdio.mcp          # Validation trace
```

### Load Testing
```
session-2024-01-15T16-45-30-ghi789jkl/
├── manifest.json
├── stdio-client-001.mcp      # Client 1
├── stdio-client-002.mcp      # Client 2
├── stdio-client-003.mcp      # Client 3
└── metadata/
    └── performance.json      # Timing analysis
```

## Benefits

1. **Organization**: Clear structure for complex debugging scenarios
2. **Correlation**: Easy to correlate events across transports
3. **Tooling**: Standardized format enables powerful analysis tools
4. **Sharing**: Collections can be easily shared and reproduced
5. **Automation**: CI/CD can generate collections for test analysis
6. **Historical**: Archived sessions for debugging regressions