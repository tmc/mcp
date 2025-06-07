# Conditional Testing with MCPScripttest

This document explains how to use conditional directives in mcpscripttest to create flexible, adaptable tests for different MCP implementations.

## Overview

MCPScripttest supports a conditional execution system that allows test commands to be skipped when specific features or capabilities are not supported by an implementation. This makes it possible to:

1. Create a single set of comprehensive conformance tests covering all features of the specification
2. Let implementers run the same tests without modifying them, even if they don't support all features
3. Allow for implementation-specific extensions and features

## Condition Syntax

Conditions are specified as prefixes to commands in scripttest files using square brackets:

```
# Basic condition - run only if stdio transport is supported
[stdio] exec mcp-scripttest-server --stdio

# Negated condition - run only if NOT on Windows
[!windows] echo "Running on a non-Windows platform"

# Multiple conditions - run only if ALL are satisfied
[http] [sse] exec mcp-send --http=localhost:8765 --sse-listen

# Condition with arguments
[version 2025-03-26] echo "Using protocol version 2025-03-26"
```

## Condition Types

### Standard Conditions

MCPScripttest provides the following built-in conditions:

#### Transport Conditions
- `stdio` - Check if stdio transport is supported
- `http` - Check if HTTP transport is supported
- `sse` - Check if Server-Sent Events are supported
- `websocket` - Check if WebSocket transport is supported
- `streaming` - Check if streaming is supported

#### Capability Conditions
- `tools` - Check if tools capability is supported
- `resources` - Check if resources capability is supported
- `prompts` - Check if prompts capability is supported
- `logging` - Check if logging capability is supported
- `batch` - Check if JSON-RPC batch requests are supported
- `auth` - Check if authentication is supported
- `progress` - Check if progress notifications are supported

#### Environment Conditions
- `version <version>` - Check if a specific protocol version is supported
- `env <name> [value]` - Check if an environment variable is set (and optionally equals a value)
- `feature <feature>` - Check if a specific feature is enabled
- `extended` - Check if extended tests are enabled

#### Platform Conditions
- `windows` - Check if running on Windows
- `linux` - Check if running on Linux
- `macos`, `darwin` - Check if running on macOS
- `unix` - Check if running on a Unix-like system
- `platform <platform>` - Generic platform check

### Custom Conditions

You can define your own custom conditions for implementation-specific features by adding them to the `CustomConditions` map in `MCPScripttestOptions`:

```go
options := mcpscripttest.DefaultOptions()
options.CustomConditions = map[string]script.Cond{
    "my_feature": script.Condition(func() error {
        if os.Getenv("MY_FEATURE_ENABLED") != "true" {
            return fmt.Errorf("my feature is not enabled")
        }
        return nil
    }),
    "custom_setting": script.TestCondition(
        script.CondUsage{
            Summary: "check if a custom setting is enabled",
            Args:    "setting_name",
        },
        func(s *script.State, args ...string) error {
            if len(args) != 1 {
                return script.ErrUsage
            }
            setting := args[0]
            if os.Getenv("SETTING_"+setting) != "true" {
                return fmt.Errorf("setting %s is not enabled", setting)
            }
            return nil
        },
    ),
}
```

## Other Command Prefixes

In addition to conditions, scripttest also supports these command prefixes:

- `?` - Makes a command optional; the test continues even if the command fails
- `!` - Expects a command to fail; the test fails if the command succeeds

## Controlling Conditions via Environment Variables

You can control condition evaluation by setting environment variables:

```bash
# Disable specific features
export MCP_DISABLE_HTTP=true
export MCP_DISABLE_SSE=true

# Enable specific features
export MCP_EXTENDED_TESTS=true
export MCP_ENABLE_WEBSOCKET=true

# Define supported protocol versions
export MCP_SUPPORTED_VERSIONS="2025-03-26,draft"
```

These environment variables can be set directly in your environment or by using command-line flags with the `mcpscripttest` tool:

```bash
mcpscripttest -all -disable-http -disable-sse -extended -versions="2025-03-26,draft"
```

## Complete Example

Here's a complete example that demonstrates various condition types:

```
# Transport conditions
[stdio] exec mcp-scripttest-server --stdio
[http] exec mcp-scripttest-server --http=localhost:8765

# Capability conditions
[tools] exec mcp-send --method=tools/list
[resources] exec mcp-send --method=resources/list

# Version-specific tests
[version 2024-11-05] echo "Testing 2024-11-05 features"
[version 2025-03-26] echo "Testing 2025-03-26 features"

# Platform-specific tests
[windows] echo "Testing on Windows"
[unix] echo "Testing on Unix-like systems"

# Multiple conditions
[http] [sse] [version 2025-03-26] echo "Testing HTTP+SSE with version 2025-03-26"

# Custom conditions (if defined)
[my_feature] echo "My custom feature is enabled"
[custom_setting advanced_mode] echo "Advanced mode is enabled"
```

## Best Practices

1. **Use conditions liberally:** Add conditions to any test that depends on optional features.

2. **Group related commands:** Keep commands with the same conditions together and use the same condition on each line.

3. **Use multiple conditions when needed:** If a test depends on multiple features, include all the necessary conditions.

4. **Document custom conditions:** If you add custom conditions, document them clearly for other test authors.

5. **Test in different configurations:** Try running tests with various feature configurations to verify conditions work correctly.

6. **Default to inclusive tests:** Write tests that cover all features, then use conditions to make parts optional, rather than creating separate test files for different feature sets.

## See Also

- [MCP Conformance Tests](../exp/mcpscripttest/testdata/mcp_conformance/README.md)
- [Custom Condition Examples](../exp/mcpscripttest/examples/custom_condition_test.go)
- [MCPScripttest Documentation](../exp/mcpscripttest/README.md)