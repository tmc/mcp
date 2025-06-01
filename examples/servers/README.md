# MCP Go Servers

This directory contains example MCP server implementations in Go, inspired by the official MCP servers in the [modelcontextprotocol/servers](https://github.com/modelcontextprotocol/servers) repository.

## Available Servers

### mcp-echo-server
A simple echo server that returns messages with timestamps.

**Tools:**
- `echo`: Echoes back a message with a timestamp

**Usage:**
```bash
cd mcp-echo-server
go run main.go
```

### mcp-time-server-enhanced
An enhanced time server with timezone conversion capabilities.

**Tools:**
- `get_current_time`: Get current time in any timezone
- `convert_time`: Convert time between multiple timezones  
- `list_timezones`: List common timezones by region

**Usage:**
```bash
cd mcp-time-server-enhanced
go run main.go
```

### mcp-memory-server
A knowledge graph-based memory server for storing entities, relations, and observations.

**Tools:**
- `create_entities`: Create new entities in the knowledge graph
- `create_relations`: Create relations between entities
- `add_observations`: Add observations to existing entities
- `search_memory`: Search entities and relations

**Usage:**
```bash
cd mcp-memory-server
# Optional: set custom memory file location
export MEMORY_FILE_PATH=/path/to/memory.json
go run main.go
```

### mcp-filesystem-server
A secure filesystem server with directory access controls.

**Tools:**
- `read_file`: Read file contents
- `write_file`: Write content to a file
- `list_directory`: List directory contents
- `get_file_info`: Get file/directory information

**Usage:**
```bash
cd mcp-filesystem-server
# Allow access to specific directories
go run main.go /path/to/allowed/dir1 /path/to/allowed/dir2
```

## Testing Servers

You can test these servers using the existing MCP client tools:

```bash
# Build the server
cd mcp-echo-server
go build -o mcp-echo-server

# Test with mcpcat (if available)
echo '{"method": "tools/list"}' | ./mcp-echo-server | mcpcat

# Or test manually with stdio
echo '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | ./mcp-echo-server
```

## Features

All servers implement:
- Standard MCP protocol compliance
- Stdio transport (for use with Claude and other MCP clients)
- Error handling with proper error responses
- Logging to stderr for debugging
- JSON schema validation for tool parameters

## Development Notes

These servers use the `github.com/tmc/mcp` package and demonstrate:
- Tool registration and handling
- Parameter validation
- Error responses with `isError` flag
- Proper JSON marshaling/unmarshaling
- Security considerations (especially for filesystem access)

## Comparison with Official Servers

These Go implementations provide equivalent functionality to the official TypeScript/Python servers:

- **mcp-echo-server** ↔ [echo-server](https://github.com/modelcontextprotocol/servers/tree/main/src/echo-server)
- **mcp-time-server-enhanced** ↔ [time](https://github.com/modelcontextprotocol/servers/tree/main/src/time) 
- **mcp-memory-server** ↔ [memory](https://github.com/modelcontextprotocol/servers/tree/main/src/memory)
- **mcp-filesystem-server** ↔ [filesystem](https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem)

The Go versions often include additional features and more comprehensive error handling.