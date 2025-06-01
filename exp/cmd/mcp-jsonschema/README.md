# mcp-jsonschema

`mcp-jsonschema` is a command-line tool that extracts JSON schemas from MCP servers or configuration files. It helps developers understand the input requirements for MCP tools and can be used to generate documentation, client-side validation, or type definitions.

## Overview

Model Context Protocol (MCP) servers expose tools that have specific input requirements defined using JSON Schema. This tool connects to an MCP server, queries its available tools, and extracts the input schemas for those tools. The schemas can then be used for documentation, code generation, validation, or other purposes.

## Usage

```bash
mcp-jsonschema [options]
```

## Options

- `--config <file>`: Path to an MCP server configuration file
- `--server <command>`: Command to start the MCP server
- `--timeout <duration>`: Timeout for server operations (default: 30s)
- `--output <file>`: Output file for JSON schema (if not specified, prints to stdout)
- `--pretty`: Pretty-print the JSON output (default: true)

Note: You must specify either `--config` or `--server`, but not both.

## Examples

### Extract schemas from a running server

Start a server and extract its schemas:

```bash
mcp-jsonschema --server "./path/to/server"
```

### Extract schemas from a configuration file

Use a configuration file to start a server and extract schemas:

```bash
mcp-jsonschema --config "./path/to/config.json"
```

### Save schemas to a file

Output the schemas to a file instead of stdout:

```bash
mcp-jsonschema --server "./path/to/server" --output schemas.json
```

### Adjust timeout for slow servers

Increase the timeout for servers that take longer to start or respond:

```bash
mcp-jsonschema --server "./path/to/server" --timeout 60s
```

### Use with other MCP tools

Use with `mcp-config` to create and extract schemas in one workflow:

```bash
# First create a configuration
mcp-config --create calculator.json --template calculator

# Then extract schemas from it
mcp-jsonschema --config calculator.json --output calculator-schemas.json
```

## How It Works

1. The tool connects to the specified MCP server or starts a new server process based on a configuration file
2. It initializes the MCP protocol and requests a list of available tools using the `listTools` method
3. For each tool, it extracts the input schema from the tool definition
4. The schemas are collected and output in JSON format, with tool names as keys

## Output Format

The output is a JSON object where each key is a tool name and each value is the JSON Schema for that tool's input:

```json
{
  "toolName1": {
    "type": "object",
    "properties": {
      "param1": {
        "type": "string",
        "description": "Description of parameter 1"
      },
      "param2": {
        "type": "number",
        "description": "Description of parameter 2"
      }
    },
    "required": ["param1"]
  },
  "toolName2": {
    "type": "object",
    "properties": {
      "option": {
        "type": "boolean",
        "description": "Description of the option"
      }
    }
  }
}
```

## Use Cases

- **Documentation Generation**: Create documentation for MCP servers and their tools
- **Type Definition Generation**: Generate TypeScript interfaces, Go structs, or other type definitions
- **Client-Side Validation**: Use schemas to validate input before sending requests to the server
- **UI Generation**: Automatically generate forms or other UI elements based on the schema
- **Testing**: Create test cases that cover the input requirements of each tool

## Notes

- Most tools will include an input schema, but some might not. Tools without schemas will not be included in the output.
- JSON Schema is a standard format (see [json-schema.org](https://json-schema.org/)) for describing the structure of JSON data.
- This tool requires the MCP server to implement the `listTools` method according to the MCP specification.
- The tool follows the MCP protocol for initialization and communication.