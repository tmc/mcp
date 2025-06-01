# MCP Go Servers

This directory contains example MCP server implementations in Go, inspired by the official MCP servers and the awesome-mcp-servers community collection.

## Available Servers

### Core Servers

#### mcp-echo-server
A simple echo server that returns messages with timestamps.

**Tools:**
- `echo`: Echoes back a message with a timestamp

**Usage:**
```bash
cd mcp-echo-server
go run main.go
```

#### mcp-time-server
A time server with timezone conversion capabilities.

**Tools:**
- `get_current_time`: Get current time in any timezone
- `convert_time`: Convert time between timezones

**Usage:**
```bash
cd mcp-time-server
go run main.go
```

#### mcp-fetch-server
A web content fetching server with HTML to text conversion.

**Tools:**
- `fetch`: Fetch content from URLs
- `fetch_text`: Fetch and convert HTML to plain text
- `get_headers`: Get HTTP headers without full content

**Usage:**
```bash
cd mcp-fetch-server
go run main.go
```

#### mcp-git-server
A Git repository management server.

**Tools:**
- Git repository operations and file management

**Usage:**
```bash
cd mcp-git-server
go run main.go
```

#### mcp-memory-server
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

#### mcp-filesystem-server
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

### New Utility Servers

#### mcp-weather-server
Weather information server using OpenWeatherMap API.

**Tools:**
- `get_current_weather`: Get current weather for any location
- `get_weather_forecast`: Get 5-day weather forecast

**Setup:**
```bash
export OPENWEATHER_API_KEY="your_api_key_here"
cd mcp-weather-server
go run main.go
```

#### mcp-calculator-server
Comprehensive mathematical calculation server.

**Tools:**
- `calculate`: Basic arithmetic (add, subtract, multiply, divide)
- `advanced_math`: Advanced functions (power, sqrt, sin, cos, log, etc.)
- `statistics`: Statistical analysis of datasets
- `convert_units`: Unit conversions (temperature, length, weight, volume)

**Usage:**
```bash
cd mcp-calculator-server
go run main.go
```

#### mcp-todo-server
Todo/task management server with persistent storage.

**Tools:**
- `add_todo`: Create new todo items
- `list_todos`: List todos with filtering
- `update_todo`: Update existing todos
- `delete_todo`: Delete todos
- `todo_stats`: Get todo statistics

**Usage:**
```bash
cd mcp-todo-server
go run main.go
```

#### mcp-http-server
HTTP client server for making REST API calls.

**Tools:**
- `http_get`: Make GET requests
- `http_post`: Make POST requests
- `http_put`: Make PUT requests
- `http_delete`: Make DELETE requests
- `http_request`: Make custom HTTP requests

**Usage:**
```bash
cd mcp-http-server
go run main.go
```

#### mcp-system-server
System information and administration server.

**⚠️ Use with caution - includes command execution capabilities**

**Tools:**
- `get_system_info`: Get OS and hardware information
- `list_processes`: List running processes
- `get_disk_usage`: Check disk space
- `get_env_vars`: Access environment variables
- `execute_command`: Execute system commands

**Usage:**
```bash
cd mcp-system-server
go run main.go
```

## Testing Servers

You can test these servers using the existing MCP client tools:

```bash
# Build the server
cd mcp-weather-server
go build -o mcp-weather-server

# Test with mcpcat (if available)
echo '{"method": "tools/list"}' | ./mcp-weather-server | mcpcat

# Or test manually with stdio
echo '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | ./mcp-weather-server
```

## Configuration for Claude Desktop

Add servers to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "weather": {
      "command": "/path/to/mcp-weather-server",
      "env": {
        "OPENWEATHER_API_KEY": "your_api_key_here"
      }
    },
    "calculator": {
      "command": "/path/to/mcp-calculator-server"
    },
    "todo": {
      "command": "/path/to/mcp-todo-server"
    },
    "http": {
      "command": "/path/to/mcp-http-server"
    },
    "system": {
      "command": "/path/to/mcp-system-server"
    }
  }
}
```

## Features

All servers implement:
- Standard MCP protocol compliance
- Stdio transport (for use with Claude and other MCP clients)
- Error handling with proper error responses
- Logging to stderr for debugging
- JSON schema validation for tool parameters
- Comprehensive documentation and examples

## Security Considerations

### High-Security Servers
- **mcp-system-server**: Can execute system commands - use with extreme caution
- **mcp-filesystem-server**: File system access - configure allowed directories carefully

### Medium-Security Servers
- **mcp-http-server**: Network access - may access external APIs
- **mcp-fetch-server**: Web content fetching - validate URLs

### Low-Security Servers
- **mcp-weather-server**: Requires API key - only accesses weather data
- **mcp-calculator-server**: Pure computation - no external access
- **mcp-todo-server**: Local file storage only
- **mcp-time-server**: Time calculations only

## Development Notes

These servers use the `github.com/tmc/mcp` package and demonstrate:
- Tool registration and handling
- Parameter validation with JSON schemas
- Error responses with `isError` flag
- Proper JSON marshaling/unmarshaling
- Security considerations and input validation
- Cross-platform compatibility
- Environment variable configuration
- File-based persistence (where applicable)

## Comparison with Official Servers

These Go implementations provide equivalent or enhanced functionality compared to the official TypeScript/Python servers:

- **mcp-echo-server** ↔ [echo-server](https://github.com/modelcontextprotocol/servers/tree/main/src/echo-server)
- **mcp-time-server** ↔ [time](https://github.com/modelcontextprotocol/servers/tree/main/src/time) 
- **mcp-memory-server** ↔ [memory](https://github.com/modelcontextprotocol/servers/tree/main/src/memory)
- **mcp-filesystem-server** ↔ [filesystem](https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem)
- **mcp-fetch-server** ↔ [fetch](https://github.com/modelcontextprotocol/servers/tree/main/src/fetch)
- **mcp-git-server** ↔ [git](https://github.com/modelcontextprotocol/servers/tree/main/src/git)

The new utility servers (weather, calculator, todo, http, system) are inspired by the community awesome-mcp-servers collection and provide additional practical functionality.

## Contributing

When adding new servers:
1. Follow the existing naming convention (`mcp-*-server`)
2. Include comprehensive README with examples
3. Add proper error handling and validation
4. Consider security implications
5. Add to this main README
6. Test with actual MCP clients