# jsonrpc2gostruct

A command-line utility to convert JSON-RPC 2.0 messages and JSON Schema definitions to Go struct definitions.

> **Note:** This tool may be renamed to `jsonschema2gostruct` in the future as it primarily deals with JSON Schema conversion rather than just JSON-RPC messages.

## Features

- Convert JSON-RPC requests to Go structs
- Generate structs from JSON schema definitions
- Batch process multiple schema files
- Infer types from JSON data
- Create idiomatic Go code with proper naming conventions
- Add appropriate JSON tags with omitempty for optional fields

## Installation

```bash
go install github.com/tmc/mcp/exp/cmd/jsonrpc2gostruct@latest
```

## Usage

### Process a JSON-RPC Request

```bash
# From a file
jsonrpc2gostruct -package mcptools examples/call_tool_request.json

# From stdin
cat examples/call_tool_request.json | jsonrpc2gostruct -package mcptools
```

### Generate and Save to a File

```bash
jsonrpc2gostruct -package mcptools -out calculator_request.go examples/call_tool_request.json
```

### Process Multiple Files at Once

```bash
jsonrpc2gostruct -batch -dir examples -pattern "*.json" -package mcptypes -out generated_types.go
```

## Command Line Options

| Flag | Description |
|------|-------------|
| `-package` | Package name for the generated Go code (default: "main") |
| `-struct` | Name of the struct to generate (default: "JSONRPCRequest") |
| `-out` | Output file (default: stdout) |
| `-batch` | Process multiple schema files in batch mode |
| `-dir` | Directory containing schema files for batch mode |
| `-pattern` | File pattern for batch mode (default: "*.json") |

## Examples

### Generate struct from a JSON-RPC request

Input:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "calculator",
    "arguments": {
      "operation": "add",
      "a": 5,
      "b": 3
    }
  }
}
```

Output:
```go
package mcptools

// ToolsCallRequest is a request for the tools/call method
type ToolsCallRequest struct {
    // The arguments to pass to the tool
    Arguments map[string]interface{} `json:"arguments,omitempty"`
    // The name of the tool to call
    Name string `json:"name"`
}
```

### Generate structs from a JSON Schema file

Input (`calculator.json`):
```json
{
  "type": "object",
  "description": "Calculator tool input schema",
  "properties": {
    "operation": {
      "type": "string",
      "description": "The operation to perform",
      "enum": ["add", "subtract", "multiply", "divide"]
    },
    "a": {
      "type": "number",
      "description": "First operand"
    },
    "b": {
      "type": "number",
      "description": "Second operand"
    }
  },
  "required": ["operation", "a", "b"]
}
```

Output:
```go
package main

// Schema - Calculator tool input schema
type Schema struct {
	// First operand
	A float64 `json:"a"`
	// The operation to perform
	Operation string `json:"operation"`
	// Second operand
	B float64 `json:"b"`
}
```

### Generate structs from JSON-RPC tools response

Input (from `list_tools` response):
```json
{
  "jsonrpc": "2.0",
  "result": {
    "tools": [
      {
        "name": "calculator",
        "description": "A simple calculator tool",
        "inputSchema": {
          "type": "object",
          "properties": {
            "operation": {
              "type": "string",
              "description": "The operation to perform"
            },
            "a": {
              "type": "number",
              "description": "First operand"
            },
            "b": {
              "type": "number",
              "description": "Second operand"
            }
          },
          "required": ["operation", "a", "b"]
        }
      }
    ]
  }
}
```

Output:
```go
package main

// CalculatorInput - A simple calculator tool
type CalculatorInput struct {
	// First operand
	A float64 `json:"a"`
	// The operation to perform
	Operation string `json:"operation"`
	// Second operand
	B float64 `json:"b"`
}
```

### Generate structs from Claude Code Tools JSON format

The tool supports the Claude Code Tools JSON format that might be found in files like `cc-tools.json`:

```bash
jsonrpc2gostruct -package claudetools /tmp/cc-tools.json
```

This format typically contains tool definitions with JSON Schema that defines the input parameters for each tool.

## Integration with MCP

This tool helps maintain type alignment between different MCP implementations by:

1. Ensuring consistent struct naming conventions
2. Creating proper JSON field tags matching the protocol specification
3. Providing a single source of truth for schema definitions
4. Simplifying the process of updating type definitions as the protocol evolves