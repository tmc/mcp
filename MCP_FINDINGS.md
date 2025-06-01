# MCP Tools Findings

## Summary of Investigation

We investigated two main aspects of the MCP (Model Context Protocol) implementation:

1. **mcpd daemon**: How mcpd is designed to work and what it's used for
2. **echo tool**: Why the echo tool sometimes appears to not respond

## mcpd Findings

`mcpd` is designed as a daemon that acts as a bridge between clients and MCP servers:

- Its primary purpose is to expose MCP servers that communicate via stdin/stdout over network interfaces (Unix sockets or TCP)
- It supports two modes:
  - `once` mode: Creates a single server instance for all client connections (default)
  - `per-connection` mode: Creates a new server instance for each client connection

When testing mcpd, we encountered issues with how it manages server processes. Specifically:

1. In `once` mode, the server started by mcpd exited immediately, leaving no active server for client connections
2. In `per-connection` mode, we encountered the error: `failed to create stdin pipe: exec: StdinPipe after process started`

These issues suggest that mcpd requires specific server implementations that can maintain a persistent connection and properly handle stdin/stdout communication.

## echo Tool Testing

We created a direct test client that interacts with the MCP server running on localhost:7000. Our tests confirmed:

1. **The echo tool works correctly with string inputs**:
   - Successfully returns responses for string messages (e.g., "Hello from direct test!")
   - Correctly formats responses with the text type and prepends "Echo: " to the message
   - Handles empty strings correctly

2. **The echo tool properly validates input types**:
   - Strictly requires the `message` parameter to be a string
   - Returns appropriate validation errors for non-string values:
     ```
     "code": -32603,
     "message": "[
       {
         "code": "invalid_type",
         "expected": "string",
         "received": "number",
         "path": [
           "message"
         ],
         "message": "Expected string, received number"
       }
     ]"
     ```

## Explanation for Observed Behavior

The reason some requests to the echo tool appeared to fail is that:

1. The tool strictly validates input types (must be string)
2. When a non-string value is used (like a number or boolean), the server responds with a validation error
3. The client needs to properly handle these error responses

For example, when using mcpspy, sometimes the output from error responses might not be as obvious as successful responses, making it appear as if there was no response.

## MCP Server Information

The server we tested against identifies as:
- Name: `example-servers/everything`
- Version: `1.0.0`
- Protocol Version: `2025-03-26`

It provides 8 tools:
1. echo: Echoes back the input
2. add: Adds two numbers
3. printEnv: Prints all environment variables
4. longRunningOperation: Demonstrates a long running operation with progress updates
5. sampleLLM: Samples from an LLM using MCP's sampling feature
6. getTinyImage: Returns the MCP_TINY_IMAGE
7. annotatedMessage: Demonstrates how annotations can be used
8. getResourceReference: Returns a resource reference

## Testing Tools Created

During this investigation, we created several testing tools:

1. `cmd/mcpspy/echo_tester.go`: Tests the echo tool with multiple types
2. `cmd/mcp_direct.go`: A comprehensive client for testing MCP servers directly
3. `cmd/mcpd_tester.go`: A test for mcpd functionality (still needs refinement)

These tools can be used for further MCP development and testing.