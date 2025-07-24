# MCP Code Generation Tools

This directory contains the comprehensive code generation and scaffolding tools for the MCP ecosystem, implementing C1-C3 from the MCP Go roadmap.

## Tools Overview

### 1. mcp-gen - Multi-language Code Generator
The core code generation tool that creates type-safe clients, servers, and boilerplate code from MCP server definitions.

### 2. mcp-scaffold - Project Scaffolding Tool
Complete project scaffolding with templates, best practices, CI/CD configuration, and dependency management.

### 3. mcp-migrate - Migration and Upgrade Assistant
Migration tool for protocol versions and framework changes with interactive wizards.

## Features

### mcp-gen Capabilities

- **Multi-language Support**: Go, TypeScript, Python, Rust, Java
- **Type-safe Generation**: Automatic type-safe client/server generation
- **Template Engine**: Robust template system with language-specific functions
- **Schema Analysis**: JSON schema to native type conversion
- **Test Generation**: Comprehensive test suite generation
- **Documentation**: Automatic API documentation generation
- **Plugin Architecture**: Extensible plugin system

### mcp-scaffold Capabilities

- **Project Templates**: Basic, Advanced, Enterprise templates
- **Multi-language Projects**: Support for all major languages
- **CI/CD Integration**: GitHub Actions, GitLab CI, Jenkins
- **Best Practices**: Linting, formatting, testing setup
- **Dependency Management**: Automatic dependency configuration
- **Docker Support**: Container configuration for deployment

### mcp-migrate Capabilities

- **Protocol Migration**: Upgrade between MCP protocol versions
- **Code Transformation**: Automated code transformation
- **Migration Planning**: Interactive migration planning
- **Rollback Support**: Safe rollback mechanisms
- **Compatibility Analysis**: Breaking change detection
- **Interactive Wizards**: User-friendly migration process

## Quick Start

### Install Tools

```bash
# Install all tools
go install github.com/tmc/mcp/exp/cmd/mcp-gen@latest
go install github.com/tmc/mcp/exp/cmd/mcp-scaffold@latest
go install github.com/tmc/mcp/exp/cmd/mcp-migrate@latest
```

### Generate Client from Server

```bash
# Generate Go client from running server
mcp-gen client -lang go -output ./client -package github.com/user/client ./server

# Generate TypeScript client from schema
mcp-gen client -lang typescript -output ./ts-client schema.json
```

### Scaffold New Project

```bash
# Create advanced Go server project
mcp-scaffold server -lang go -template advanced -author myname my-server

# Create TypeScript client project
mcp-scaffold client -lang typescript -template basic my-client
```

### Migrate Protocol Version

```bash
# Analyze project for migration
mcp-migrate analyze -lang go -path ./my-project

# Upgrade from v1.0 to v2.0
mcp-migrate upgrade -from 1.0 -to 2.0 -lang go -backup
```

## Examples

### Generate Complete Go Client

```bash
# From time server schema
mcp-gen client -lang go -output ./time-client -package github.com/user/time-client examples/time-server-schema.json

# Generated files:
# - client.go (type-safe client)
# - types.go (input/output types)
# - client_test.go (comprehensive tests)
# - README.md (documentation)
```

### Generate Server Stub

```bash
# Generate server implementation
mcp-gen server -lang go -output ./time-server -package timeserver examples/time-server-schema.json

# Generated files:
# - server.go (server implementation)
# - types.go (type definitions)
# - main.go (entry point)
# - Dockerfile (containerization)
# - Makefile (build automation)
```

### Scaffold Enterprise Project

```bash
# Create full enterprise project
mcp-scaffold server -lang go -template enterprise -ci github -license MIT my-enterprise-server

# Generated structure:
# my-enterprise-server/
# ├── cmd/
# │   └── server/
# │       └── main.go
# ├── internal/
# │   ├── handlers/
# │   ├── middleware/
# │   └── config/
# ├── pkg/
# │   └── api/
# ├── tests/
# ├── docs/
# ├── .github/
# │   └── workflows/
# ├── Dockerfile
# ├── docker-compose.yml
# ├── Makefile
# ├── go.mod
# └── README.md
```

## Configuration

### mcp-gen Configuration

Create `mcp-gen.json`:

```json
{
  "language": "go",
  "output": "./generated",
  "package": "github.com/user/project",
  "type_safe": true,
  "middleware": true,
  "documentation": true,
  "tests": true,
  "examples": true,
  "go": {
    "module_path": "github.com/user/project",
    "go_version": "1.21",
    "use_generics": true,
    "use_contexts": true,
    "use_middleware": true
  },
  "templates": {
    "directory": "./custom-templates",
    "custom_vars": {
      "author": "Your Name",
      "license": "MIT"
    }
  }
}
```

### mcp-scaffold Configuration

Templates and CI/CD integration:

```bash
# Configure project template
mcp-scaffold init -lang go -template enterprise -ci github -license Apache-2.0

# Available templates:
# - basic: Minimal project structure
# - advanced: Testing, docs, examples
# - enterprise: Full enterprise features
```

### mcp-migrate Configuration

Migration configuration:

```json
{
  "from": "1.0",
  "to": "2.0",
  "language": "go",
  "backup": true,
  "transformations": {
    "api_changes": true,
    "type_updates": true,
    "dependency_updates": true
  },
  "validation": {
    "compile_check": true,
    "test_execution": true,
    "lint_check": true
  }
}
```

## Advanced Usage

### Custom Templates

Create custom templates for specific use cases:

```bash
# Create template directory
mkdir -p ./custom-templates/go/client

# Create custom client template
cat > ./custom-templates/go/client/client.tmpl << 'EOF'
package {{.Package}}

import (
    "context"
    "github.com/tmc/mcp"
)

// {{.Client.Name}} - Custom client implementation
type {{.Client.Name}} struct {
    client *mcp.Client
    config *Config
}

// Custom client methods...
EOF

# Use custom template
mcp-gen client -lang go -config ./mcp-gen.json -template custom ./schema.json
```

### Plugin Development

Create custom plugins:

```bash
# Generate plugin boilerplate
mcp-gen plugin -lang go -output ./plugins my-plugin

# Implement plugin interface
# Register with mcp-gen
```

### Batch Processing

Process multiple schemas:

```bash
# Generate clients for multiple services
for schema in schemas/*.json; do
    mcp-gen client -lang go -output "./clients/$(basename $schema .json)" "$schema"
done
```

## Integration Examples

### CI/CD Pipeline

```yaml
# .github/workflows/generate.yml
name: Generate MCP Code
on:
  push:
    paths: ['schemas/**']

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install mcp-gen
      run: go install github.com/tmc/mcp/exp/cmd/mcp-gen@latest
    
    - name: Generate clients
      run: |
        for schema in schemas/*.json; do
          mcp-gen client -lang go -output "./clients/$(basename $schema .json)" "$schema"
        done
    
    - name: Generate tests
      run: |
        for client in clients/*/; do
          mcp-gen tests -lang go -output "$client/tests" "$client"
        done
    
    - name: Commit generated code
      run: |
        git config --local user.email "action@github.com"
        git config --local user.name "GitHub Action"
        git add .
        git commit -m "Generated code from schemas" || exit 0
        git push
```

### Docker Integration

```dockerfile
# Dockerfile for code generation
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go install github.com/tmc/mcp/exp/cmd/mcp-gen@latest

# Generation stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /go/bin/mcp-gen .
COPY schemas/ ./schemas/

# Generate all clients
RUN for schema in schemas/*.json; do \
      ./mcp-gen client -lang go -output "./clients/$(basename $schema .json)" "$schema"; \
    done

# Runtime stage
FROM scratch
COPY --from=builder /root/clients/ /clients/
```

## Performance

### Benchmarks

```bash
# Run performance benchmarks
go test -bench=. -benchmem ./...

# Example results:
# BenchmarkCodeGeneration-8    100    10.5ms/op    2.1MB/alloc
# BenchmarkSchemaAnalysis-8    500     2.1ms/op    0.5MB/alloc
# BenchmarkTemplateExecution-8 1000    1.2ms/op    0.3MB/alloc
```

### Optimization Tips

1. **Use caching**: Enable template caching for repeated generations
2. **Parallel processing**: Generate multiple files in parallel
3. **Incremental generation**: Only regenerate changed files
4. **Template optimization**: Optimize templates for performance

## Troubleshooting

### Common Issues

1. **Template not found**: Check template paths and naming
2. **Invalid schema**: Validate JSON schema format
3. **Permission errors**: Ensure write permissions for output directory
4. **Language not supported**: Verify language is in supported list

### Debug Mode

```bash
# Enable verbose logging
mcp-gen client -lang go -verbose -output ./debug schema.json

# Dry run to preview changes
mcp-gen client -lang go -dry-run -output ./preview schema.json
```

### Getting Help

```bash
# Get help for specific commands
mcp-gen help client
mcp-scaffold help server
mcp-migrate help upgrade

# Show version information
mcp-gen version
mcp-scaffold version
mcp-migrate version
```

## Contributing

### Development Setup

```bash
# Clone repository
git clone https://github.com/tmc/mcp
cd mcp/exp/cmd/mcp-gen

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Build tools
go build -o bin/mcp-gen .
go build -o bin/mcp-scaffold ../mcp-scaffold
go build -o bin/mcp-migrate ../mcp-migrate
```

### Adding Language Support

1. Create language-specific analyzer in `internal/codegen/`
2. Add templates in `internal/templates/templates/{language}/`
3. Update configuration in `internal/config/`
4. Add tests in `*_test.go` files
5. Update documentation

### Template Development

1. Study existing templates in `internal/templates/templates/`
2. Use template functions from `internal/templates/engine.go`
3. Test templates with various schemas
4. Document template variables and functions

## License

MIT License - see LICENSE file for details.

## Support

- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Documentation**: [MCP Documentation](https://github.com/tmc/mcp)
- **Examples**: `examples/` directory

---

**Generated with mcp-gen v0.1.0**