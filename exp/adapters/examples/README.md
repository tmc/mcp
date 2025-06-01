# MCP Adapter Test Servers

This directory contains test servers demonstrating how to use the MCP adapters to bridge between different MCP implementation libraries and the standard `github.com/tmc/mcp` SDK.

## Mark3Labs Test Server

The Mark3Labs test server demonstrates how to use the Mark3Labs adapter to integrate Mark3Labs-style MCP servers with the SDK.

### Building

```bash
cd mark3labs_test_server
go build
```

### Running

```bash
./mark3labs_test_server --stdio
```

### Features

The test server demonstrates:
- Tool registration using Mark3Labs API
- Resource registration
- Prompt registration
- Adapter initialization and usage

## Golang-Tools Test Server

The Golang-Tools test server demonstrates how to use the Golang-Tools adapter to bridge golang-tools-internal-mcp implementations with the SDK.

### Building

```bash
cd golang_tools_test_server
go build
```

### Running

```bash
./golang_tools_test_server --stdio
```

### Features

The test server demonstrates:
- Creating an SDK server with tools and prompts
- Using the adapter to wrap the SDK server
- Handler conversion between SDK and golang-tools patterns

## Testing

Both test servers can be tested using the MCP mock client:

```bash
# Test mark3labs server
mcp-mock-client -scenario ../../../cmd/mcp-mock-client/testdata/scenarios/basic_scenario.json -- ./mark3labs_test_server --stdio

# Test golang-tools server  
mcp-mock-client -scenario ../../../cmd/mcp-mock-client/testdata/scenarios/basic_scenario.json -- ./golang_tools_test_server --stdio
```

## Architecture

The adapters work by:

1. **Mark3Labs Adapter**: 
   - Accepts Mark3Labs tools, resources, and prompts
   - Converts between Mark3Labs types and SDK protocol types
   - Handles request routing and response conversion

2. **Golang-Tools Adapter**:
   - Initializes from an SDK server
   - Extracts tools and prompts from the SDK server
   - Wraps handlers to convert between SDK and golang-tools formats
   - Provides compatibility layer for golang-tools MCP implementations

Both adapters implement the `adapters.Adapter` interface and can be used with `server.WithAdapter()` when creating an SDK server.