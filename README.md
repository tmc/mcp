# mcp

A collection of programs, utilities, and examples for assembling production-ready Go MCP programs.

## Overview

Model Context Protocol (MCP) is a standardized protocol for AI models to interact with external systems. This implementation provides:

- Complete client and server implementations
- Multiple transport mechanisms (stdio, HTTP, WebSocket)
- Type-safe API with Go generics
- Management of tools, resources, and prompts
- Comprehensive testing and validation utilities
- **Build Status**: ✅ All packages build successfully
- **Test Status**: ✅ 22/23 packages passing tests

## Project Structure

```
mcp/
├── client.go, server.go, transport.go  # Core implementation
├── cmd/                                # Command-line tools
│   ├── mcp-connect/                    # Connection utilities
│   ├── mcp-debug/                      # Debug tools
│   ├── mcp-fake/                       # Mock servers
│   ├── mcp-probe/                      # Server probing
│   ├── mcp-proxy/                      # Traffic proxying
│   ├── mcp-replay/                     # Session recording/playback
│   ├── mcp-send/                       # Message sending
│   ├── mcp-serve/                      # Server hosting
│   ├── mcp-shadow/                     # Shadow testing
│   ├── mcp-sort/                       # Trace sorting
│   ├── mcpcat/                         # Trace viewing
│   ├── mcpdiff/                        # Trace comparison
│   ├── mcpspy/                         # Traffic monitoring
│   └── mcptrace-to-otel/               # OpenTelemetry conversion
├── examples/                           # Example implementations
│   └── servers/                        # 25+ server examples
├── exp/                                # Experimental features
│   ├── mcpscripttest/                  # Testing framework
│   ├── adapters/                       # System adapters
│   └── cmd/                            # Experimental tools
├── modelcontextprotocol/               # Core protocol types
├── internal/                           # Internal implementation details
├── testing/                            # Test utilities
└── traces/                             # Example trace files
```

## Installation

To build the MCP tools:

```bash
go build ./...
```

## Command-line Tools

### Core Tools

- **mcp-connect**: Connect to MCP servers with various transports (stdio, HTTP, SSE)
- **mcp-debug**: Debug MCP server interactions and protocol flows
- **mcp-fake**: Create fake/mock MCP servers for testing
- **mcp-probe**: Probe and test MCP server capabilities and responses
- **mcp-proxy**: Proxy MCP traffic for debugging and analysis
- **mcp-replay**: Record and replay MCP sessions for testing
- **mcp-send**: Send individual MCP requests to servers
- **mcp-serve**: Serve MCP servers with various transport options
- **mcp-shadow**: Create shadow testing with dual server comparison
- **mcp-sort**: Sort and organize MCP trace files

### Utility Tools

- **mcpcat**: Display and format MCP trace files
- **mcpdiff**: Compare MCP traces and sessions
- **mcpspy**: Monitor and spy on MCP traffic
- **mcptrace-to-otel**: Convert MCP traces to OpenTelemetry format

### Example Usage

```bash
# Connect to a server via stdio
mcp-connect --stdio ./path/to/server

# Debug server interactions
mcp-debug --server ./server --verbose

# Proxy MCP traffic for analysis
mcp-proxy --listen :8080 --target stdio://./server

# Record and replay sessions
mcp-replay --record session.mcp --server ./server
mcp-replay --playback session.mcp
```

## Example Servers

The repository includes several example MCP servers:

- **Filesystem Server**: File system access
- **Calculator Server**: Arithmetic operations
- **Time Server**: Time-related information
- **SQLite Server**: Database access
- **Echo Server**: Simple echo service

## Development

### Build Commands

- Build all: `go build ./...`
- Test all: `go test ./...`
- Test single package: `go test github.com/tmc/mcp/[package]`
- Test single test: `go test -run TestName github.com/tmc/mcp/[package]`
- Format code: `gofmt -s -w .`
- Lint: `go vet ./...`

### Code Style Guidelines

- Format code with `gofmt -s -w .` before committing
- Imports: standard library first, then external packages, then local packages
- Error handling: check errors with descriptive context using `fmt.Errorf("doing X: %w", err)`
- Context: pass `context.Context` as first parameter for I/O functions
- Variable naming: camelCase for variables, PascalCase for exported identifiers
- Function size: keep functions focused on a single responsibility, typically under 50 lines
- Comments: document the "why" not just the "what" in comments
- Package design: small, focused packages with clear responsibilities
- Testing: use table-driven tests with descriptive names

## Future Plans

### Google's MCP Implementation Integration

We plan to integrate Google's MCP implementation (`golang/tools/internal/mcp`) as our core implementation:

1. Importing Google's implementation code
2. Creating adaptation layers to use standard jsonrpc2 packages
3. Extending with our own additional features and tools
4. Maintaining compatibility with existing tools

This approach will:
- Clearly separate core protocol from utilities and extensions
- Support experimental features without disrupting stable components
- Follow Go project best practices for organization
- Facilitate easy updates when Google's implementation changes

### Tool Enhancements

1. **mcp-capabilities**:
   - Check for support of specific tools by name
   - Deep inspection of tool functionality
   - Compare capabilities between different servers
   - Visualize capabilities as a compatibility matrix

2. **mcp-config**:
   - Schema validation for configuration files
   - Template customization options
   - Support for more transport types
   - Tool and prompt management commands

3. **mcp-jsonschema**:
   - Generate code from schemas (TypeScript interfaces, Go structs)
   - Merge schemas from multiple servers
   - Compare schemas between different server versions
   - Output in different formats

4. **Integration**:
   - Registry for sharing and discovering MCP server configurations
   - Web interface for editing configurations and viewing schemas
   - Integration with development tools and IDEs

## Documentation

Detailed documentation can be found in the `docs/` directory:

- `docs/development/`: Development guidelines
- `docs/integration/`: Integration guides
- `docs/servers/`: Server implementation guidelines

## License

This project is licensed under the MIT License - see the LICENSE file for details.
