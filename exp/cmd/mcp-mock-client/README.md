# MCP Mock Client

The MCP Mock Client is a tool for testing MCP servers by sending pre-defined requests and validating responses.

## Features

- Replay recorded MCP requests from a file
- Execute scenario-based testing with advanced response validation
- Support for sophisticated pattern matching and expectations
- Dry-run mode for verifying requests without sending them
- Verbose output for debugging

## Usage

```
mcp-mock-client [flags] recording|scenario
```

### Flags

- `-n`: Print requests instead of sending them (dry-run mode)
- `-scenario`: Run a scenario file instead of a recording
- `-validate`: Validate responses against expectations defined in the scenario
- `-timeout duration`: Timeout for the entire scenario (default 30s)
- `-step-delay duration`: Delay between scenario steps
- `-v`: Enable verbose output

## Recording Format

A recording file contains MCP requests and responses in the following format:

```
mcp-in {"jsonrpc":"2.0","method":"initialize","params":{},"id":1}
mcp-out {"jsonrpc":"2.0","result":{"capabilities":{}},"id":1}
mcp-in {"jsonrpc":"2.0","method":"listTools","params":{},"id":2}
mcp-out {"jsonrpc":"2.0","result":{"tools":[]},"id":2}
```

Only the lines with `mcp-in` prefix are sent by the client.

## Scenario Format

Scenarios are defined in JSON format and provide a structured way to test MCP servers with advanced pattern matching and response validation.

Example scenario file:

```json
{
  "name": "Basic MCP Server Test",
  "description": "Tests basic interaction with an MCP server",
  "steps": [
    {
      "name": "Initialize",
      "description": "Initialize the connection",
      "request": {
        "jsonrpc": "2.0",
        "method": "initialize",
        "params": {
          "clientInfo": {
            "name": "test-client",
            "version": "0.1.0"
          }
        },
        "id": 1
      },
      "expectations": [
        {
          "type": "response",
          "pattern": {
            "jsonrpc": "2.0",
            "result": {
              "serverInfo": {
                "name": "{{string}}",
                "version": "{{string}}"
              },
              "capabilities": "{{object}}"
            },
            "id": 1
          },
          "timeout": 5000000000,
          "fail_fast": true
        }
      ]
    }
  ]
}
```

### Pattern Matching

The scenario-based testing supports advanced pattern matching for validating responses:

- Simple equality matching: `{"id": 1}`
- Type matching: `{"name": "{{string}}"`, `{"age": "{{number}}"`, `{"data": "{{object}}"`
- Regex matching: `{"version": "/^[0-9]+\\.[0-9]+\\.[0-9]+$/"}`
- Wildcards: `{"value": "{{any}}"` or `{"value": "{{*}}"}`
- Partial object matching: `{"data": {"{{partial}}": true, "required_field": "value"}}`
- Array item template: `{"items": [{"{{items}}": true, "name": "{{string}}"}]}`

### Expectation Types

- `response`: Expects a successful response
- `error`: Expects an error response
- `notification`: Expects a notification (method call with no ID)

## Examples

### Replay a recording

```shell
mcp-mock-client recording.mcp | mcp-server
```

### Dry run a recording

```shell
mcp-mock-client -n recording.mcp
```

### Run a scenario

```shell
mcp-mock-client -scenario scenario.json | mcp-server
```

### Run a scenario with response validation

```shell
mcp-mock-client -scenario -validate scenario.json | mcp-server
```

### Run a scenario with verbose output

```shell
mcp-mock-client -scenario -v scenario.json | mcp-server
```

## Integration with Test Frameworks

The mcp-mock-client can be used with testing frameworks for automated validation:

```go
func TestMCPServer(t *testing.T) {
    cmd := exec.Command("mcp-mock-client", "-scenario", "-validate", "scenario.json")
    cmd.Stdin = getServerOutputPipe()
    cmd.Stdout = getServerInputPipe()
    
    err := cmd.Run()
    if err != nil {
        t.Fatalf("Scenario validation failed: %v", err)
    }
}
```