# mcp-config

`mcp-config` is a command-line tool for managing MCP server configurations.

## Overview

This tool allows you to create, edit, validate, and format MCP server configuration files. It supports templates for common server types and provides a consistent format for defining MCP servers.

## Usage

```
mcp-config [options]
```

### Options

- `--create <file>`: Create a new MCP server configuration file
- `--edit <file>`: Edit an existing MCP server configuration file
- `--validate <file>`: Validate an MCP server configuration file
- `--format <file>`: Format an MCP server configuration file (pretty-print JSON)
- `--template <type>`: Use a template to create a configuration (basic, filesystem, calculator)
- `--name <name>`: Server name for new configurations (default: "mcp-server")
- `--version <version>`: Server version for new configurations (default: "1.0.0")
- `--command <command>`: Server command for new configurations
- `--transport <type>`: Transport type (stdio, http, sse) (default: "stdio")

## Examples

### Create a new configuration using a template

```bash
# Create a new calculator server configuration
mcp-config --create calculator.json --template calculator

# Create a new filesystem server configuration with custom name and version
mcp-config --create fs-server.json --template filesystem --name "fs-server" --version "2.0.0"
```

### Edit an existing configuration

```bash
# Update the server command in an existing configuration
mcp-config --edit server.json --command "./new-server-binary"

# Update the transport type in an existing configuration
mcp-config --edit server.json --transport http
```

### Validate a configuration

```bash
# Check if a configuration file is valid
mcp-config --validate server.json
```

### Format a configuration

```bash
# Pretty-print a configuration file
mcp-config --format server.json
```

## Configuration Format

MCP server configurations are JSON files with the following structure:

```json
{
  "name": "server-name",
  "version": "1.0.0",
  "description": "Optional description",
  "command": "./server-binary",
  "transport": "stdio",
  "environment": {
    "ENV_VAR1": "value1",
    "ENV_VAR2": "value2"
  },
  "instructions": "Optional server instructions",
  "tools": [
    {
      "name": "toolName",
      "description": "Tool description",
      "inputSchema": {
        "type": "object",
        "properties": {
          "param": {
            "type": "string",
            "description": "Parameter description"
          }
        },
        "required": ["param"]
      }
    }
  ],
  "prompts": [
    {
      "name": "promptName",
      "description": "Prompt description",
      "arguments": {
        "arg1": "value1",
        "arg2": "value2"
      }
    }
  ]
}
```

## Available Templates

### Basic

A simple echo server with a single tool that echoes back input.

### Filesystem

A filesystem server with tools for listing files, reading files, and writing files.

### Calculator

A calculator server with tools for basic arithmetic operations (add, subtract, multiply, divide).