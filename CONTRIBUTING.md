# Contributing to MCP Go

Thank you for your interest in contributing to the MCP Go implementation! This guide will help you get started.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept responsibility for mistakes

## How to Contribute

### Reporting Issues

Before creating an issue, please:
1. Search existing issues to avoid duplicates
2. Use issue templates when available
3. Include relevant information:
   - Go version (`go version`)
   - OS and architecture
   - Steps to reproduce
   - Expected vs actual behavior
   - Error messages or logs

### Suggesting Features

1. Open a discussion first for major features
2. Explain the use case and benefits
3. Consider implementation complexity
4. Be open to feedback and alternatives

### Submitting Pull Requests

1. **Fork and clone** the repository
2. **Create a branch** from `main` or `next`
3. **Make changes** following our guidelines
4. **Test thoroughly** including new tests
5. **Update documentation** as needed
6. **Submit PR** with clear description

## Development Setup

### Prerequisites

```bash
# Required
go 1.21+
git

# Recommended
make
golangci-lint
gotestsum
```

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/mcp.git
cd mcp

# Add upstream remote
git remote add upstream https://github.com/tmc/mcp.git

# Install dependencies
go mod download

# Build everything
go build ./...

# Run tests
go test ./...
```

## Code Style

### Go Standards

Follow standard Go conventions:
- Run `gofmt -s -w .` before committing
- Run `go vet ./...` to catch issues
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Project Conventions

```go
// Package comments describe the package
package mcp

// Exported types need documentation
// Tool represents an MCP tool with its metadata.
type Tool struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

// Methods should explain what, not how
// RegisterTool adds a new tool to the server's tool registry.
func (s *Server) RegisterTool(tool Tool, handler ToolHandlerFunc) error {
    // Validate inputs
    if tool.Name == "" {
        return fmt.Errorf("tool name cannot be empty")
    }
    
    // Use defer for cleanup
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Early returns for errors
    if _, exists := s.tools[tool.Name]; exists {
        return fmt.Errorf("tool %q already registered", tool.Name)
    }
    
    s.tools[tool.Name] = handler
    return nil
}
```

### Import Organization

```go
import (
    // Standard library
    "context"
    "fmt"
    "time"
    
    // External packages
    "github.com/google/uuid"
    "golang.org/x/sync/errgroup"
    
    // Internal packages
    "github.com/tmc/mcp/internal/jsonrpc2"
    "github.com/tmc/mcp/modelcontextprotocol"
)
```

## Testing

### Writing Tests

```go
func TestServer_RegisterTool(t *testing.T) {
    // Use table-driven tests
    tests := []struct {
        name    string
        tool    Tool
        wantErr bool
    }{
        {
            name: "valid tool",
            tool: Tool{Name: "test", Description: "Test tool"},
            wantErr: false,
        },
        {
            name: "empty name",
            tool: Tool{Name: "", Description: "Test tool"},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Use t.Parallel() when possible
            t.Parallel()
            
            server := NewServer("test", "1.0.0")
            err := server.RegisterTool(tt.tool, nil)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("RegisterTool() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Coverage

- Aim for >70% coverage for new code
- Test error conditions and edge cases
- Include integration tests for complex features
- Add benchmarks for performance-critical code

```bash
# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Commit Messages

Follow Go project style (not conventional commits):

```
package: brief description of change

Longer explanation of the change, why it was needed,
and what it does. Wrap at 72 characters.

Fixes #123
```

Examples:
```
mcp: fix race condition in token validation

Add mutex protection around token validation to prevent
race condition when checking revocation status and metadata
simultaneously. This fixes intermittent auth failures under
high load.

Fixes #456
```

```
cmd/mcp-connect: add test coverage for all transports

Implement comprehensive test suite covering stdio, SSE, and
HTTP transports. Tests include connection lifecycle, error
handling, and concurrent access patterns.
```

## Pull Request Process

### Before Submitting

- [ ] Tests pass: `go test ./...`
- [ ] Code formatted: `gofmt -s -w .`
- [ ] Linted: `go vet ./...`
- [ ] Documentation updated
- [ ] Changelog entry added (for features)
- [ ] Commits are logical and well-described

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Follows code style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No security vulnerabilities introduced
```

### Review Process

1. Automated checks must pass
2. At least one maintainer review required
3. Address feedback constructively
4. Squash commits if requested
5. Maintain backwards compatibility

## Project Structure

```
mcp/
├── cmd/           # Command-line tools
│   └── tool/
│       ├── main.go
│       ├── main_test.go
│       └── README.md
├── internal/      # Internal packages
├── pkg/           # Public packages
├── examples/      # Example implementations
├── docs/          # Documentation
└── exp/           # Experimental features
```

### Adding a New Tool

1. Create directory: `cmd/mcp-newtool/`
2. Implement main.go with standard flags
3. Add comprehensive tests
4. Document in README.md
5. Update root README.md tool list

### Adding a New Feature

1. Discuss in issue/discussion first
2. Start in `exp/` for experimental features
3. Add tests alongside implementation
4. Document public APIs
5. Consider backwards compatibility

## Documentation

### Code Documentation

```go
// Package mcp implements the Model Context Protocol.
//
// The MCP allows AI models to interact with external systems
// through a standardized protocol. This package provides both
// client and server implementations.
package mcp

// Tool represents an MCP tool that can be called by clients.
// Tools must have unique names within a server instance.
type Tool struct {
    // Name is the unique identifier for this tool.
    Name string `json:"name"`
    
    // Description explains what the tool does.
    // This is shown to users in tool listings.
    Description string `json:"description"`
    
    // InputSchema defines the expected parameters
    // as a JSON Schema object.
    InputSchema json.RawMessage `json:"inputSchema"`
}
```

### User Documentation

- Update README.md for user-facing changes
- Add examples for new features
- Include migration guides for breaking changes
- Keep documentation close to code

## Release Process

### Version Numbering

We use [Semantic Versioning](https://semver.org/):
- MAJOR: Breaking API changes
- MINOR: New features, backwards compatible
- PATCH: Bug fixes only

### Release Checklist

1. Update version.go
2. Update CHANGELOG.md
3. Run full test suite
4. Create PR to main branch
5. After merge, tag release
6. Create GitHub release
7. Announce in discussions

## Getting Help

### Resources

- [Documentation](docs/)
- [GitHub Discussions](https://github.com/tmc/mcp/discussions)
- [Issue Tracker](https://github.com/tmc/mcp/issues)

### Communication Channels

- **Questions**: Use GitHub Discussions
- **Bugs**: Open an issue
- **Security**: Email security@example.com
- **Features**: Start a discussion first

## Recognition

Contributors are recognized in:
- [CHANGELOG.md](CHANGELOG.md) for significant contributions
- GitHub's contributor graph
- Release notes for major features

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to MCP Go! Your efforts help make this project better for everyone.