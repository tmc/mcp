# mcp-capabilities

`mcp-capabilities` is a command-line tool that connects to an MCP server and reports its capabilities.

## Overview

Model Context Protocol (MCP) servers can support various features including tools, prompts, resources, and experimental capabilities. This tool connects to a server, queries its capabilities, and provides a comprehensive report of what features are supported.

## Usage

```bash
mcp-capabilities [options]
```

## Options

- `--config <file>`: Path to an MCP server configuration file
- `--server <command>`: Command to start the MCP server
- `--timeout <duration>`: Timeout for server operations (default: 30s)
- `--output <file>`: Output file for capability report (if not specified, prints to stdout)
- `--json`: Output in JSON format instead of human-readable text
- `--tools=<bool>`: Check if server supports tools (default: true)
- `--prompts=<bool>`: Check if server supports prompts (default: true)
- `--resources=<bool>`: Check if server supports resources (default: true)
- `--experimental=<bool>`: Check if server has experimental capabilities (default: true)

Note: You must specify either `--config` or `--server`, but not both.

## Examples

### Check capabilities of a running server

```bash
mcp-capabilities --server "./path/to/server"
```

### Check capabilities using a configuration file

```bash
mcp-capabilities --config "./path/to/config.json"
```

### Save capabilities to a file in JSON format

```bash
mcp-capabilities --server "./path/to/server" --output capabilities.json --json
```

### Check only specific capabilities

```bash
mcp-capabilities --server "./path/to/server" --tools=true --prompts=false --resources=false
```

## Output Format

### Human-readable output (default)

```
Server: example-server (version 1.0.0)
Protocol Version: 2024-11-05

Capabilities:
- Tools: true
  Available tools:
  - echo
  - calculator
- Prompts: false
- Resources: true

Experimental Capabilities:
{
  "sampling": true,
  "streaming": false
}

Server Instructions:
This server provides a calculator and echo tool.
```

### JSON output (with --json flag)

```json
{
  "serverInfo": {
    "name": "example-server",
    "version": "1.0.0"
  },
  "protocol": "2024-11-05",
  "tools": true,
  "toolsList": [
    "echo",
    "calculator"
  ],
  "prompts": false,
  "resources": true,
  "experimental": {
    "sampling": true,
    "streaming": false
  },
  "instructions": "This server provides a calculator and echo tool."
}
```

## Use Cases

- **Server Verification**: Verify that a server supports the required capabilities before using it
- **Discovery**: Explore what features are available on an MCP server
- **Debugging**: Diagnose server connectivity and capability issues
- **Documentation**: Generate capability reports for server documentation
- **Integration Testing**: Check server capability support in automated tests

## Notes

- The tool attempts to determine capabilities by calling the corresponding MCP methods
- If a method call fails, the tool assumes that the capability is not supported
- Some servers may return errors for unsupported methods rather than a proper "not supported" response
- The JSON format provides a machine-readable output that can be parsed by other tools