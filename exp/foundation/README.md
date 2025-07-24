# MCP Foundation Infrastructure

This directory contains the foundational infrastructure for the MCP command toolkit expansion. These components provide shared libraries and frameworks that all MCP tools depend on.

## Overview

The foundation infrastructure is designed to provide consistent, reusable components following the Russ Cox coding style and Go best practices. All components are production-ready with comprehensive test coverage (>80%) and maintain backward compatibility.

## Components

### 1. Configuration Management (`config/`)

The configuration management system provides unified configuration loading, validation, and management across all MCP tools.

**Key Features:**
- Hierarchical configuration with tool-specific sections
- Multiple configuration sources (files, environment variables, defaults)
- Thread-safe configuration access with change notifications
- Automatic default value application with struct tags
- Configuration validation with custom validators

**Usage:**
```go
import "github.com/tmc/mcp/internal/foundation/config"

// Create configuration manager
manager, err := config.NewManager(
    config.WithConfigFile("config.yaml"),
    config.WithEnvPrefix("MCP_"),
    config.WithValidator("global", validateGlobal),
)

// Load configuration
ctx := context.Background()
if err := manager.Load(ctx); err != nil {
    log.Fatal(err)
}

// Access configuration
cfg := manager.Get()
fmt.Println("Log level:", cfg.Global.LogLevel)

// Get tool-specific config
var toolConfig MyToolConfig
if err := manager.GetTool("my-tool", &toolConfig); err != nil {
    log.Fatal(err)
}
```

**Configuration Format:**
```yaml
global:
  log_level: info
  output_format: json
  no_color: false
  transport:
    default_type: stdio
    timeout: 30s
  performance:
    max_concurrency: 10
    buffer_size: 8192
  security:
    enable_auth: false
    require_https: true

tools:
  my-tool:
    enabled: true
    config:
      host: localhost
      port: 8080
```

### 2. Transport Abstraction (`transport/`)

The enhanced transport abstraction layer v2 provides unified interface for all transport types with plugin support.

**Key Features:**
- Unified transport interface for stdio, HTTP, WebSocket, TCP, Unix sockets
- Plugin architecture for extensible transport types
- Connection pooling and lifecycle management
- Health checks and automatic retry logic
- Middleware support for cross-cutting concerns

**Usage:**
```go
import "github.com/tmc/mcp/internal/foundation/transport"

// Create transport manager
manager, err := transport.NewManager(transport.ManagerConfig{
    DefaultType:   "stdio",
    EnablePlugins: true,
})

// Create transport from configuration
config := transport.Config{
    Type:       "http",
    Parameters: map[string]interface{}{
        "url": "http://localhost:8080/mcp",
    },
    Timeout: 30 * time.Second,
}

transport, err := manager.Create(config)
if err != nil {
    log.Fatal(err)
}

// Use transport
ctx := context.Background()
conn, err := transport.Dial(ctx)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()
```

**Transport Types:**
- `stdio`: Standard input/output transport
- `http`: HTTP/HTTPS transport with POST requests
- `websocket`: WebSocket transport for real-time communication
- `tcp`: TCP socket transport
- `unix`: Unix domain socket transport

### 3. Output Formatting (`output/`)

The unified output formatting library supports JSON, YAML, Table, and CSV formats with color support.

**Key Features:**
- Multiple output formats with consistent interface
- Automatic data structure conversion to tables
- Color support with terminal detection
- Configurable formatting options (alignment, sorting, etc.)
- Reflection-based field extraction from structs

**Usage:**
```go
import "github.com/tmc/mcp/internal/foundation/output"

// Create formatter
config := output.Config{
    Format: output.FormatTable,
    Color:  true,
    Pretty: true,
    Table: output.TableConfig{
        Headers: true,
        Borders: true,
        SortBy:  "name",
    },
}

formatter, err := output.NewFormatter(config)
if err != nil {
    log.Fatal(err)
}

// Format data
data := []Person{
    {Name: "Alice", Age: 30, City: "New York"},
    {Name: "Bob", Age: 25, City: "San Francisco"},
}

if err := formatter.Format(data); err != nil {
    log.Fatal(err)
}
```

**Output Formats:**
- `json`: JSON format with pretty printing
- `yaml`: YAML format with proper indentation
- `table`: ASCII table with headers, borders, and sorting
- `csv`: CSV format with configurable separators
- `text`: Plain text format with templates

### 4. Error Handling (`errors/`)

The common error handling framework provides standardized error codes and messaging.

**Key Features:**
- Structured errors with error codes and categories
- Error wrapping with context preservation
- Retry information and temporary error classification
- Stack trace capture for debugging
- Error handler registry with hooks

**Usage:**
```go
import "github.com/tmc/mcp/internal/foundation/errors"

// Create structured error
err := errors.New(errors.CodeInvalidArgument, "invalid configuration")
err = err.WithDetail("field", "timeout")
err = err.WithContext("operation", "load_config")
err = err.WithRetry(true, 5*time.Second, 3)

// Wrap existing error
if err != nil {
    return errors.Wrap(err, errors.CodeConfiguration, "failed to load config")
}

// Error handling
if errors.Is(err, errors.CodeTimeout) {
    // Handle timeout
}

if errors.IsRetryable(err) {
    // Retry operation
}

// Register error handler
errors.RegisterHandler(errors.CodeTimeout, func(err *errors.Error) error {
    log.Printf("Timeout error: %v", err)
    return nil
})
```

**Error Codes:**
- General: `unknown`, `invalid_argument`, `not_found`, `permission_denied`
- MCP-specific: `protocol`, `transport`, `tool`, `resource`, `prompt`
- Tool-specific: `validation`, `conversion`, `formatting`, `parsing`

### 5. Plugin Architecture (`plugin/`)

The plugin architecture enables extensible tool functionality with lifecycle management.

**Key Features:**
- Plugin interface with lifecycle methods
- Dependency resolution and validation
- Plugin registry with hot-reload support
- Hook system for extensibility
- Built-in and dynamic plugin loading

**Usage:**
```go
import "github.com/tmc/mcp/internal/foundation/plugin"

// Implement plugin
type MyPlugin struct {
    *plugin.BasePlugin
}

func (p *MyPlugin) Initialize(ctx context.Context, config plugin.Config) error {
    // Initialize plugin
    return nil
}

func (p *MyPlugin) Start(ctx context.Context) error {
    // Start plugin
    return nil
}

// Register plugin
manager := plugin.DefaultManager()
registry := manager.GetRegistry()

pluginInstance := &MyPlugin{
    BasePlugin: plugin.NewBasePlugin("my-plugin", "1.0.0", "My plugin"),
}

config := plugin.Config{
    Enabled: true,
    Config:  map[string]interface{}{},
}

if err := registry.Register(pluginInstance, config); err != nil {
    log.Fatal(err)
}

// Start plugin
if err := registry.Start("my-plugin"); err != nil {
    log.Fatal(err)
}
```

**Plugin Lifecycle:**
1. `Initialize`: Plugin initialization with configuration
2. `Start`: Plugin startup
3. `Stop`: Plugin shutdown
4. `Cleanup`: Resource cleanup

## Design Principles

### 1. Consistency
- All components follow the same API patterns
- Consistent error handling and configuration
- Unified logging and debugging interfaces

### 2. Performance
- Thread-safe operations with minimal locking
- Connection pooling and resource reuse
- Efficient data structures and algorithms

### 3. Extensibility
- Plugin architecture for custom functionality
- Middleware support for cross-cutting concerns
- Hook system for customization

### 4. Reliability
- Comprehensive error handling with retry logic
- Health checks and monitoring
- Graceful degradation and fallback mechanisms

### 5. Testability
- Comprehensive test coverage (>80%)
- Mocking support for external dependencies
- Benchmarking for performance validation

## Integration Guide

### Adding a New Tool

1. **Configuration**: Define tool-specific configuration structure
2. **Transport**: Choose appropriate transport type or create custom
3. **Output**: Use unified output formatting for consistency
4. **Error Handling**: Use structured errors with appropriate codes
5. **Plugin Support**: Implement plugin hooks if extensibility is needed

### Example Tool Integration

```go
package main

import (
    "context"
    "log"
    
    "github.com/tmc/mcp/internal/foundation/config"
    "github.com/tmc/mcp/internal/foundation/transport"
    "github.com/tmc/mcp/internal/foundation/output"
    "github.com/tmc/mcp/internal/foundation/errors"
)

type MyTool struct {
    config    *config.Manager
    transport transport.Transport
    formatter *output.Formatter
}

func NewMyTool() (*MyTool, error) {
    // Load configuration
    configMgr, err := config.NewManager(
        config.WithConfigFile("config.yaml"),
        config.WithEnvPrefix("MYTOOL_"),
    )
    if err != nil {
        return nil, errors.Wrap(err, errors.CodeConfiguration, "failed to create config manager")
    }
    
    if err := configMgr.Load(context.Background()); err != nil {
        return nil, errors.Wrap(err, errors.CodeConfiguration, "failed to load configuration")
    }
    
    // Create transport
    transportMgr, err := transport.NewManager(transport.ManagerConfig{
        DefaultType: "stdio",
    })
    if err != nil {
        return nil, errors.Wrap(err, errors.CodeTransport, "failed to create transport manager")
    }
    
    transportCfg := transport.Config{
        Type: configMgr.Get().Global.Transport.DefaultType,
    }
    
    transport, err := transportMgr.Create(transportCfg)
    if err != nil {
        return nil, errors.Wrap(err, errors.CodeTransport, "failed to create transport")
    }
    
    // Create formatter
    outputCfg := output.Config{
        Format: output.Format(configMgr.Get().Global.OutputFormat),
        Color:  !configMgr.Get().Global.NoColor,
        Pretty: true,
    }
    
    formatter, err := output.NewFormatter(outputCfg)
    if err != nil {
        return nil, errors.Wrap(err, errors.CodeFormatting, "failed to create formatter")
    }
    
    return &MyTool{
        config:    configMgr,
        transport: transport,
        formatter: formatter,
    }, nil
}

func (t *MyTool) Run(ctx context.Context) error {
    // Tool implementation using foundation components
    return nil
}
```

## Testing

All foundation components include comprehensive test suites:

```bash
# Run all tests
go test ./internal/foundation/...

# Run tests with coverage
go test -cover ./internal/foundation/...

# Run benchmarks
go test -bench=. ./internal/foundation/...
```

## Contributing

When adding new foundation components:

1. Follow the Russ Cox coding style
2. Include comprehensive tests (>80% coverage)
3. Add benchmarks for performance-critical code
4. Update documentation and examples
5. Ensure thread-safety and error handling
6. Maintain backward compatibility

## License

This code is part of the MCP Go implementation and follows the same licensing terms as the parent project.