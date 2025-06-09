# mcpscripttestext - MCP Script Test Extensions

This package contains the "fancy" commands and conditions that were extracted from the core `mcpscripttest` package to keep the core minimal and focused on essential functionality.

## Overview

The core `mcpscripttest` package now provides minimal functionality with just:
- Basic script testing framework
- Coverage integration  
- Tool installation
- Essential `setstdin` command

All advanced commands and conditions have been moved to extension packages in `mcpscripttestext`:

## Extension Packages

### `bashext` - Bash Command Extensions
- Advanced bash command execution with coverage support
- Bash script tracing and coverage collection
- Environment variable handling

**Commands:** `bash`

### `mcptools` - MCP Tool Command Extensions  
- All MCP tool commands (mcpspy, mcpdiff, etc.)
- Tool execution and management
- Input/output handling

**Commands:** `mcp-replay`, `mcp-spy`, `mcp-start`, `mcp-test`, `mcp-verify`, `mcp-send`, `mcp-recv`, `mcp-serve`, `mcp-scripttest-server`, `mcpspy`, `mcpdiff`, `mcpcat`, `mcp-sort`, `mcp-shadow`, `mcp-probe`, `setstdin`

### `serverext` - MCP Server Management Extensions
- MCP server lifecycle management
- Server communication and control
- Environment configuration

**Commands:** `mcp-server-start`, `mcp-server-send`, `mcp-server-stop`, `mcp-server-output`, `setenv`

**Conditions:** `mcp_server_running`, `stdio`, `http`, `sse`, `http_session`, `multi_connection`, `test_server_delay`, `test_server_cancel`, `test_server_validate_stdout`, `server_provided`, `server_arg`

### `conditionsext` - Advanced Protocol Conditions
- MCP protocol capability detection
- Transport condition checking
- Advanced protocol validation

**Conditions:** `stdio`, `http`, `sse`, `tools`, `resources`, `prompts`, `logging`, `sampling`, `tools_list_changed`, `resources_subscribe`, `resources_list_changed`, `prompts_list_changed`, `client_sampling`, `client_roots`, `protocol_version`, `server_name`, `server_version`, `client_name`, `client_version`, `test_coverage`, `test_debug`, `test_timeout`

## Usage

### Using All Extensions
```go
import (
    "github.com/tmc/mcp/testing/mcpscripttest"
    "github.com/tmc/mcp/exp/mcpscripttestext"
)

func TestWithExtensions(t *testing.T) {
    opts := mcpscripttest.DefaultOptions()
    
    // Add all extension commands and conditions
    for name, cmd := range mcpscripttestext.DefaultCommands() {
        opts.CustomCommands[name] = cmd
    }
    for name, cond := range mcpscripttestext.DefaultConditions() {
        opts.CustomConditions[name] = cond
    }
    
    mcpscripttest.TestWithOptions(t, "testdata/*.txt", opts)
}
```

### Using Specific Extensions
```go
import (
    "github.com/tmc/mcp/testing/mcpscripttest"
    "github.com/tmc/mcp/exp/mcpscripttestext/bashext"
    "github.com/tmc/mcp/exp/mcpscripttestext/mcptools"
)

func TestWithBashAndTools(t *testing.T) {
    opts := mcpscripttest.DefaultOptions()
    
    // Add only bash and MCP tool commands
    for name, cmd := range bashext.DefaultCommands() {
        opts.CustomCommands[name] = cmd
    }
    for name, cmd := range mcptools.DefaultCommands() {
        opts.CustomCommands[name] = cmd
    }
    
    mcpscripttest.TestWithOptions(t, "testdata/*.txt", opts)
}
```

### Using Minimal Mode
```go
import "github.com/tmc/mcp/testing/mcpscripttest"

func TestMinimal(t *testing.T) {
    // Just use the minimal mode - no extensions needed
    mcpscripttest.TestMinimal(t, "testdata/*.txt")
}
```

## Migration Guide

### From Old mcpscripttest
If you were using the old full-featured mcpscripttest, your tests should continue to work unchanged since the main `Test()` function still includes all commands by default.

### To Minimal Mode
If you want to use the new minimal mode for simpler tests:

```go
// Old:
mcpscripttest.Test(t, "testdata/*.txt")

// New minimal:
mcpscripttest.TestMinimal(t, "testdata/*.txt")
```

### Custom Extensions
If you need only specific functionality:

```go
// Add only the commands you need
opts := mcpscripttest.DefaultMinimalOptions()
opts.CustomCommands["bash"] = bashext.DefaultCommands()["bash"]
mcpscripttest.TestMinimal(t, "testdata/*.txt", opts)
```

## Benefits

1. **Reduced Dependencies**: Core mcpscripttest has minimal dependencies
2. **Modular Design**: Use only the extensions you need
3. **Better Testing**: Easier to test core functionality independently
4. **Flexibility**: Mix and match extensions as needed
5. **Maintenance**: Easier to maintain and extend individual components

## Future Work

- [ ] Move actual command implementations from internal package to extensions
- [ ] Complete TODO placeholders with real implementations
- [ ] Add more granular extension packages
- [ ] Provide convenience functions for common extension combinations