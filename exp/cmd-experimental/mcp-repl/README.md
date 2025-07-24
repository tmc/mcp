# mcp-repl

Interactive REPL for MCP (Model Context Protocol) servers with comprehensive developer experience features.

## Features

- **Multi-server support**: Connect to multiple MCP servers simultaneously
- **Auto-completion**: Tab completion for commands, server names, tools, resources, and prompts
- **Session management**: Save and restore sessions with server connections and variables
- **Command history**: Persistent command history with search capabilities
- **Script execution**: Execute command scripts for automation
- **Variable management**: Set and use variables across commands
- **Alias support**: Create command aliases for frequent operations
- **Color output**: Configurable colored output for better readability
- **Transport support**: Supports stdio, HTTP, and SSE transports

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-repl@latest
```

## Usage

### Basic Usage

```bash
# Start interactive REPL
mcp-repl

# Connect to a server on startup
mcp-repl -server "go run ./examples/servers/mcp-time-server"

# Execute a script
mcp-repl -script commands.txt

# Run in non-interactive mode
mcp-repl -script commands.txt -interactive=false
```

### Configuration

The REPL uses a configuration file at `~/.mcp-repl-config.json`:

```json
{
  "history_file": "~/.mcp-repl-history",
  "auto_complete": true,
  "color_output": true,
  "default_timeout": "30s",
  "servers": {
    "timeserver": {
      "command": ["go", "run", "./examples/servers/mcp-time-server"],
      "transport": "stdio",
      "description": "Time server example",
      "auto_connect": true
    },
    "httpserver": {
      "transport": "http",
      "url": "http://localhost:8080",
      "description": "HTTP server",
      "auto_connect": false
    }
  },
  "aliases": {
    "t": "tools",
    "r": "resources",
    "p": "prompts"
  }
}
```

## Commands

### Connection Management

- `connect <name> <command...>` - Connect to a server
- `disconnect <name>` - Disconnect from a server
- `servers` - List connected servers
- `use <name>` - Switch to a server
- `ping [name]` - Ping a server
- `info [name]` - Show server information

### Server Operations

- `tools` - List available tools
- `resources` - List available resources
- `prompts` - List available prompts
- `call <tool> [args...]` - Call a tool
- `read <resource>` - Read a resource
- `prompt <name> [args...]` - Get a prompt

### Session Management

- `save <file>` - Save session to file
- `load <file>` - Load session from file
- `script <file>` - Execute script file
- `history` - Show command history

### Variables and Configuration

- `set <var> <value>` - Set a variable
- `get <var>` - Get a variable
- `config` - Show configuration
- `alias <name> <command>` - Create command alias

### Utility

- `clear` - Clear screen
- `help [command]` - Show help
- `exit` - Exit REPL

## Examples

### Connecting to Servers

```bash
# Connect via stdio
mcp> connect timeserver go run ./examples/servers/mcp-time-server

# Connect via HTTP
mcp> connect httpserver --transport http --url http://localhost:8080

# Switch to a server
mcp> use timeserver
[timeserver] mcp> tools
```

### Tool Operations

```bash
# List tools
[timeserver] mcp> tools
Available tools:
  get_time - Get current time
  format_time - Format time string

# Call a tool
[timeserver] mcp> call get_time
Tool call result:
Current time: 2024-01-15T10:30:00Z

# Call tool with arguments
[timeserver] mcp> call format_time timestamp=1642248600 format="2006-01-02"
Tool call result:
Formatted time: 2024-01-15
```

### Resource Operations

```bash
# List resources
[server] mcp> resources
Available resources:
  file://config.json - Configuration file
  file://logs/app.log - Application log

# Read a resource
[server] mcp> read file://config.json
Resource contents:
URI: file://config.json
MIME type: application/json
Content: {"setting": "value"}
```

### Session Management

```bash
# Save current session
mcp> save my-session.json
Session saved to: my-session.json

# Load session
mcp> load my-session.json
Session loaded from: my-session.json
```

### Script Execution

Create a script file `commands.txt`:

```
# Connect to time server
connect timeserver go run ./examples/servers/mcp-time-server
use timeserver

# Get current time
call get_time

# Save session
save session.json
```

Execute the script:

```bash
mcp-repl -script commands.txt
```

### Variables and Aliases

```bash
# Set variables
mcp> set server_port 8080
mcp> set debug true

# Use variables (in scripts)
mcp> get server_port
server_port = 8080

# Create aliases
mcp> alias t tools
mcp> alias lt "tools | grep list"

# Use aliases
mcp> t
Available tools:
  get_time - Get current time
```

## Auto-completion

The REPL provides comprehensive auto-completion:

- **Commands**: Tab completion for all available commands
- **Server names**: Complete server names for connection commands
- **Tool names**: Complete tool names for call command
- **Resource names**: Complete resource URIs for read command
- **Prompt names**: Complete prompt names for prompt command

## Configuration Options

### Command Line Flags

- `-config <file>` - Configuration file path (default: `~/.mcp-repl-config.json`)
- `-history <file>` - History file path (default: `~/.mcp-repl-history`)
- `-no-color` - Disable colored output
- `-debug` - Enable debug mode
- `-server <cmd>` - Server command to connect to on startup
- `-url <url>` - Server URL for HTTP/SSE transport
- `-transport <type>` - Transport type (stdio, http, sse)
- `-interactive` - Run in interactive mode (default: true)
- `-script <file>` - Script file to execute
- `-version` - Show version information

### Configuration File Options

- `history_file` - Path to history file
- `auto_complete` - Enable auto-completion
- `color_output` - Enable colored output
- `default_timeout` - Default timeout for operations
- `servers` - Pre-configured server definitions
- `aliases` - Command aliases

## Error Handling

The REPL provides comprehensive error handling:

- Connection errors are reported with context
- Tool call errors include server response details
- Script execution errors show line numbers
- Timeout errors are handled gracefully

## Multi-server Support

The REPL can maintain connections to multiple servers simultaneously:

```bash
# Connect to multiple servers
mcp> connect server1 go run ./server1
mcp> connect server2 go run ./server2

# Switch between servers
mcp> use server1
[server1] mcp> tools

mcp> use server2
[server2] mcp> tools

# Ping specific server
mcp> ping server1
Ping successful: 1.2ms
```

## Security

- All server connections are isolated
- No automatic execution of untrusted scripts
- Configuration files use safe permissions
- Input validation for all commands

## Development

### Building

```bash
cd cmd/mcp-repl
go build
```

### Testing

```bash
go test
```

### Dependencies

- `github.com/chzyer/readline` - Readline functionality
- `github.com/tmc/mcp` - MCP protocol implementation

## License

Part of the MCP Go implementation project.