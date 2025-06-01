# Experimental MCP Tools and Features

This directory contains experimental features, tools, and libraries that extend the core MCP functionality. These components are under active development and may undergo significant changes.

## Overview

The experimental (`exp/`) directory serves as an incubation area for:
- Advanced tooling and utilities
- Code generation capabilities
- Testing infrastructure enhancements
- Protocol adapters and integrations
- Performance analysis tools

## Directory Structure

### Core Libraries and Packages

#### [`adapters/`](adapters/)
Protocol adapters for different MCP implementations (golang_tools, mark3labs).

#### [`sourcegen/`](sourcegen/)
Core code generation library for converting MCP specs to Go code.

#### [`generictypes/`](generictypes/)
Generic type utilities including builders, optional types, and collections.

### Command-Line Tools

#### Code Generation Tools
- [`cmd/mcp2go/`](cmd/mcp2go/) - Convert MCP tool descriptions to Go code
- [`cmd/cmd2mcpserver/`](cmd/cmd2mcpserver/) - Generate MCP servers from CLI tools
- [`cmd/ctx-go-src/`](cmd/ctx-go-src/) - Extract Go package sources in txtar format
- [`cmd/jsonrpc2gostruct/`](cmd/jsonrpc2gostruct/) - Convert JSON-RPC to Go structs
- [`cmd/mcptrace2gostruct/`](cmd/mcptrace2gostruct/) - Convert MCP traces to Go structs
- [`cmd/schema2go/`](cmd/schema2go/) - Convert JSON schemas to Go types

#### Testing and Validation
- [`cmd/mcpscripttest/`](cmd/mcpscripttest/) - Script-based testing framework
- [`cmd/mcp-mock-client/`](cmd/mcp-mock-client/) - Mock client for testing
- [`cmd/mcp-test/`](cmd/mcp-test/) - MCP specification compliance testing
- [`cmd/mcp-verify/`](cmd/mcp-verify/) - Verify MCP implementations

#### Server Management
- [`cmd/mcpd/`](cmd/mcpd/) - MCP daemon with streaming support
- [`cmd/mcp-baseline/`](cmd/mcp-baseline/) - Baseline server implementation
- [`cmd/mcp-capabilities/`](cmd/mcp-capabilities/) - Server capability detection
- [`cmd/mcp-config/`](cmd/mcp-config/) - Configuration management

#### Development Tools
- [`cmd/mcp-spy/`](cmd/mcp-spy/) - Monitor MCP traffic (experimental version)
- [`cmd/mcp-attach/`](cmd/mcp-attach/) - Attach to running MCP sessions
- [`cmd/mcp-vet/`](cmd/mcp-vet/) - Static analysis for MCP code
- [`cmd/mcpspec-lsp/`](cmd/mcpspec-lsp/) - Language Server Protocol integration

#### Coverage and Analysis
- [`cmd/coverage-viz/`](cmd/coverage-viz/) - Visualize test coverage
- [`covtest/`](covtest/) - Coverage testing utilities
- [`cov2codecov/`](cov2codecov/) - Convert to Codecov format
- [`covdiff/`](covdiff/) - Compare coverage reports

### Testing Infrastructure

#### [`mcpscripttest/`](mcpscripttest/)
Advanced script-based testing framework with:
- Coverage integration
- Fuzzing support
- Call graph analysis
- Test dependency tracking

### Documentation

#### [`docs/`](docs/)
Technical documentation for experimental features:
- Tool architecture and design
- Implementation roadmaps
- Integration guides
- Advanced usage examples

## Getting Started

### Building Experimental Tools

```bash
# Build all experimental tools
cd exp
go build ./cmd/...

# Build specific tool
go build ./cmd/mcp2go
```

### Using Code Generation Tools

```bash
# Convert MCP tool to Go code
mcp2go tool.json

# Generate MCP server from CLI
cmd2mcpserver /path/to/binary

# Extract Go source code
ctx-go-src github.com/user/package
```

### Testing Infrastructure

```bash
# Run scripttest framework
mcpscripttest -- ./server

# Use mock client
mcp-mock-client -scenario test.json -- ./server

# Run with coverage
mcpscripttest -coverage -- ./server
```

## Key Features

### Code Generation
- Automatic Go code generation from MCP specs
- CLI tool to MCP server conversion
- JSON schema to Go type conversion
- Protocol adapter generation

### Enhanced Testing
- Script-based testing with txtar format
- Coverage collection and visualization
- Fuzzing integration
- Call graph analysis

### Server Management
- Daemon for managing MCP servers
- Streaming support (SSE, WebSocket)
- Configuration management
- Process lifecycle control

## Development Status

⚠️ **Experimental Notice**: These tools are under active development and may:
- Have breaking API changes
- Contain bugs or incomplete features
- Be reorganized or removed
- Not have full documentation

Use in production at your own risk.

## Contributing

When contributing to experimental features:
1. Update relevant documentation
2. Add comprehensive tests
3. Follow existing patterns
4. Consider promoting stable features to main repository

## Documentation

- [`CLAUDE.md`](CLAUDE.md) - Guidelines for Claude Code
- [`docs/`](docs/) - Technical documentation
- Individual tool READMEs in their directories

## Future Plans

- Promote stable tools to main repository
- Enhanced code generation capabilities
- Improved testing infrastructure
- Better integration with core MCP
- Performance optimization tools

For more details, see the [experimental documentation](docs/).