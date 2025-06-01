# mcpscripttest

`mcpscripttest` is a standalone command-line tool for running Model Context Protocol (MCP) conformance tests against an MCP server implementation. The tool validates that MCP implementations correctly adhere to the protocol specification by executing a suite of script-based tests.

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcpscripttest@latest
```

Or build directly from source:

```bash
cd github.com/tmc/mcp
go build -o mcpscripttest ./cmd/mcpscripttest
```

## Usage

```
mcpscripttest [options] -- <server command>
```

The tool requires a server command to be provided after the `--` separator. This is the command that will be used to start the MCP server for testing.

### Options

```
  -run string
        Path to test file or directory (default "testdata/mcp_conformance")
  -v
        Verbose output
  -coverage
        Enable coverage instrumentation
  -debug
        Enable debug shell on test failure
  -help
        Show help message
  -http-port int
        Starting port number for HTTP tests (default 8765)
  -extended
        Run extended tests
```

### Examples

Run all conformance tests against a stdio server:
```bash
mcpscripttest -- go run ./cmd/my-mcp-server
```

Run a specific test:
```bash
mcpscripttest -run 01_base_messaging -- go run ./cmd/my-mcp-server
```

Run tests with coverage:
```bash
mcpscripttest -coverage -- go run ./cmd/my-mcp-server
```

Run tests with verbose output:
```bash
mcpscripttest -v -- go run ./cmd/my-mcp-server
```

Run tests against an HTTP server on a specific port:
```bash
mcpscripttest -http-port 9000 -- go run ./cmd/my-mcp-server --http
```

## How It Works

When you run `mcpscripttest`, the tool:

1. Starts the provided server command
2. Auto-detects the server's capabilities (HTTP, SSE, etc.)
3. Sets environment variables based on those capabilities
4. Runs the specified tests with the conditions applied
5. Reports the results

The server's capabilities are detected by:

- Checking server output for HTTP or SSE indicators
- Testing if the server responds on the specified HTTP port
- Making assumptions about stdio support (always assumed available)

## Conditional Tests

The test scripts use condition prefixes to handle optional parts of the specification:

```
# This command will only run if the HTTP transport is supported
[http] exec mcp-scripttest-server --http=localhost:8765

# This command uses multiple conditions - requires both HTTP and SSE support
[http] [sse] exec mcp-send --http=http://localhost:8765 --sse-listen

# This command runs only if NOT on Windows
[!windows] echo "Running on a non-Windows platform"

# This command runs only if a specific protocol version is supported
[version 2025-03-26] exec special_2025_test
```

The conditions are automatically evaluated based on the detected server capabilities.

## Test Format

The tests are written in the scripttest format, which provides a simple DSL for testing command-line tools. Each test file contains a series of commands, expected outputs, and assertions.