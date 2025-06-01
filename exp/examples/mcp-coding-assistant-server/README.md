# MCP Coding Assistant Server

A flexible and ergonomic Model Context Protocol (MCP) server implementation for AI coding assistants.

## Features

- Complete implementation of the MCP protocol for coding assistants
- Supports all standard tool types:
  - Task - Launch a new task that can run other tools
  - Bash - Execute shell commands with rich output formatting
  - Batch - Run multiple tools in parallel for improved performance
  - Glob - Fast file pattern matching for any codebase size
  - Grep - Content search using regular expressions
  - LS - List files and directories
  - Read - Read file contents with line numbers
  - Edit - Make precise edits to files
  - MultiEdit - Apply multiple edits to a file in one operation
  - Write - Create or overwrite files
  - NotebookRead/Edit - Jupyter notebook integration
  - WebFetch - Fetch and process content from the web
  - TodoRead/Write - Manage todo lists
  - WebSearch - Search the web and use results
- Modular design for easy extension with custom tools
- Configurable server options
- Support for both HTTP and stdio transports

## Getting Started

### Prerequisites

- Go 1.18 or higher

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/mcp-coding-assistant-server
cd mcp-coding-assistant-server

# Build the project
go build -o mcp-coding-assistant-server .
```

### Running the Server

```bash
# Run with default options (HTTP on port 8080)
./mcp-coding-assistant-server

# Run with custom port
./mcp-coding-assistant-server -port 9000

# Run in stdio mode (for integration with other tools)
./mcp-coding-assistant-server -stdio
```

## API Usage

The server implements the standard Model Context Protocol interface. Here's an example of a basic request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "Bash",
    "arguments": {
      "command": "ls -la"
    }
  }
}
```

## Extending with Custom Tools

To extend the server with your own custom tools, simply:

1. Define your tool's input structs
2. Implement a handler function with the signature:
   ```go
   func YourToolHandler(ctx context.Context, input YourToolInput) (interface{}, error)
   ```
3. Register it with the server in main():
   ```go
   srv.RegisterTool("YourTool", YourToolHandler)
   ```

## Environment Variables

- `MCP_DEBUG`: Set to `true` to enable debug logging
- `MCP_LOG_FILE`: Path to log file (defaults to stderr)

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.