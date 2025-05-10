# mcp

A collection of programs, utilities, and examples for assembling production-ready Go MCP programs.

## Overview

Model Context Protocol (MCP) is a standardized protocol for AI models to interact with external systems. This implementation provides:

- Complete client and server implementations
- Multiple transport mechanisms (stdio, HTTP, WebSocket)
- Type-safe API with Go generics
- Management of tools, resources, and prompts
- Comprehensive testing and validation utilities

## Project Structure

```
mcp/
├── client.go, server.go, transport.go  # Core implementation
├── cmd/                                # Command-line tools
│   ├── mcp-capabilities/               # Server capability detection
│   ├── mcp-config/                     # Configuration management
│   ├── mcp-jsonschema/                 # Schema extraction
│   ├── mcp-mock-client/                # Test client
│   ├── mcp-recv/, mcp-send/            # Message utilities 
│   ├── mcp-replay/                     # Session recording/playback
│   ├── mcp-spy/                        # Traffic monitoring
│   ├── mcp-start/, mcp-test/           # Server utilities
│   ├── mcp-verify/                     # Compliance verification
│   └── mcpd/                           # MCP daemon (in development)
├── adapters/                           # System adapters
├── examples/                           # Example implementations
│   ├── servers/                        # Server examples
│   └── hosts/                          # Host applications
├── schema/                             # Schema functionality
├── mcptrace/                           # Tracing utilities
├── testing/                            # Test utilities
├── exp/                                # Experimental features
│   └── tools/                          # Experimental tools
└── internal/                           # Internal implementation details
```

## Installation

To build the MCP tools:

```bash
go build ./...
```

## Command-line Tools

### mcp-capabilities

Checks MCP server capabilities including support for tools, prompts, resources, and experimental features.

```bash
# Check server capabilities
mcp-capabilities --server "./path/to/server"

# Output in JSON format
mcp-capabilities --server "./path/to/server" --output capabilities.json --json
```

### mcp-config

Manages MCP server configurations. Creates, edits, validates, and formats configuration files.

```bash
# Create a configuration from template
mcp-config --create calculator.json --template calculator

# Validate a configuration
mcp-config --validate server.json
```

### mcp-jsonschema

Extracts JSON schemas from MCP servers or configuration files.

```bash
# Extract schemas from a server
mcp-jsonschema --server "./path/to/server"

# Save schemas to a file
mcp-jsonschema --server "./path/to/server" --output schemas.json
```

### Additional Tools

- **mcp-spy**: Monitors traffic between clients and servers
- **mcp-mock-client**: Creates mock clients for testing
- **mcp-replay**: Records and replays MCP sessions
- **mcp-start**: Starts servers from configuration files
- **mcp-test**: Tests servers against specifications
- **mcp-verify**: Verifies compliance with the MCP specification

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
