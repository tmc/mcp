# MCP Protocol Conformance Tests

This directory contains a comprehensive suite of tests to verify conformance with the Model Context Protocol (MCP) specification. These tests validate that implementations properly adhere to the protocol requirements for both clients and servers.

## Test Categories

The conformance test suite is organized into the following categories:

1. **Base Messaging** (`01_base_messaging.txt`) - Tests for correct JSON-RPC 2.0 message format and handling
2. **Lifecycle** (`02_lifecycle.txt`) - Tests for initialization, operation, and shutdown phases
3. **Transports** (`03_transports.txt`) - Tests for stdio and HTTP transport mechanisms
4. **Capability Negotiation** (`04_capability_negotiation.txt`) - Tests for client and server capability negotiation
5. **Tools** (`05_tools.txt`) - Tests for tool listing, invocation, and handling
6. **Error Handling** (`06_error_handling.txt`) - Tests for protocol and application error handling
7. **Version Negotiation** (`07_version_negotiation.txt`) - Tests for protocol version compatibility
8. **Security** (`08_security.txt`) - Tests for authentication, authorization, and validation

## Running the Tests

You can run the entire conformance test suite with:

```bash
go test -v ./exp/mcpscripttest -run TestMCPConformance
```

To run a specific test category:

```bash
go test -v ./exp/mcpscripttest -run TestMCPConformanceIndividual/01_base_messaging.txt
```

## Test Coverage

These tests aim to cover all aspects of the MCP specification, including:

- Base protocol message format and semantics
- Proper lifecycle management
- Transport-specific requirements
- Capability discovery and negotiation
- Tool interface contracts
- Error handling and recovery
- Version compatibility
- Security requirements

## Test Structure

Each test file follows a common pattern:

1. Start a test server
2. Initialize the connection
3. Run specific tests for protocol features
4. Verify expected responses and behaviors

## Conditional Tests

The conformance tests support conditional prefixes that allow commands to be skipped based on implementation capabilities. These conditions can be used to make certain tests optional based on what features the implementation supports:

```
# Only run this command if HTTP transport is supported
[http] exec mcp-scripttest-server --http=localhost:8765
```

Commands are executed one at a time, and if a command with a condition fails the condition check, that specific command is skipped but subsequent commands continue to run.

### Syntax

Conditions use the following syntax:

- `[condition]` - Run the command only if the condition is satisfied
- `[!condition]` - Run the command only if the condition is NOT satisfied (negated)
- `[condition1] [condition2]` - Run the command only if ALL conditions are satisfied
- `?` - Make a command optional (continue even if it fails)
- `!` - Expect a command to fail (test fails if command succeeds)

### Standard Conditions

The following conditions are available:

- `stdio` - Check if stdio transport is supported
- `http` - Check if HTTP transport is supported
- `sse` - Check if Server-Sent Events are supported
- `websocket` - Check if WebSocket transport is supported
- `streaming` - Check if streaming is supported
- `tools` - Check if tools capability is supported
- `resources` - Check if resources capability is supported
- `prompts` - Check if prompts capability is supported
- `logging` - Check if logging capability is supported
- `batch` - Check if JSON-RPC batch requests are supported
- `auth` - Check if authentication is supported
- `version <version>` - Check if a specific protocol version is supported
- `progress` - Check if progress notifications are supported
- `extended` - Check if extended tests are enabled
- `env <name> [value]` - Check if an environment variable is set (and optionally equals a value)
- `feature <feature>` - Check if a specific feature is enabled
- `platform <platform>` - Check if running on a specific platform

For convenience, the following platform-specific conditions are also provided:
- `windows` - Check if running on Windows
- `linux` - Check if running on Linux
- `macos`, `darwin` - Check if running on macOS
- `unix` - Check if running on a Unix-like system

### Configuration

These conditions are controlled by environment variables. For example, to disable SSE tests:

```bash
export MCP_DISABLE_SSE=true
```

To enable extended tests:

```bash
export MCP_EXTENDED_TESTS=true
```

## Additional Notes

- The test server used in these tests is `mcp-scripttest-server`, which provides a configurable implementation for testing various aspects of the protocol.
- Many tests require capability flags to be enabled on the server, which are passed as command-line arguments.
- HTTP transport tests require a local server to be started on various ports.
- Some tests may need elevated permissions or have network requirements depending on the implementation.