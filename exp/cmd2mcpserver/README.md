# cmd2mcpserver

A tool that converts Go command-line applications into Model Context Protocol (MCP) servers.

## Overview

`cmd2mcpserver` takes an existing Go binary and generates a complete MCP server that wraps the command, exposing its functionality as an MCP tool. It can analyze Go source code to automatically extract flag definitions and generate appropriate parameter schemas.

## Features

- Converts any Go CLI tool into an MCP server
- Automatically extracts flag definitions from source code
- Generates proper JSON schemas for tool parameters
- Handles different flag types (string, int, bool, float64)
- Creates a complete Go module with proper dependencies

## Installation

```bash
go install github.com/tmc/mcp/exp/cmd/cmd2mcpserver@latest
```

## Usage

Basic usage:
```bash
cmd2mcpserver ./mybinary
```

With options:
```bash
cmd2mcpserver -output ./myserver -module github.com/user/myserver -source ./src ./mybinary
```

### Flags

- `-output`: Output directory for the generated MCP server (default: `./{binary}-mcp-server`)
- `-module`: Go module name for the generated server (default: `github.com/generated/{binary}-mcp-server`)
- `-server`: Server struct name (default: derived from binary name)
- `-tool`: Tool name for MCP (default: binary name)
- `-desc`: Tool description
- `-source`: Source directory to analyze for flags (optional)
- `-dry-run`: Output generated source as txtar to stdout (no files created)
- `-v`: Verbose output

## Example

Let's say you have a simple CLI tool called `mytool`:

```go
package main

import "flag"

func main() {
    verbose := flag.Bool("verbose", false, "Enable verbose output")
    count := flag.Int("count", 1, "Number of iterations")
    flag.Parse()
    // ... tool logic ...
}
```

Convert it to an MCP server:

```bash
cmd2mcpserver -source ./mytool-src ./mytool
```

This generates an MCP server that:
1. Exposes `mytool` as an MCP tool
2. Automatically creates parameter schemas for `verbose` and `count` flags
3. Handles parameter validation and command execution
4. Returns structured output

## Generated Server Structure

The generated server includes:
- `go.mod`: Module definition with MCP dependencies
- `main.go`: Complete MCP server implementation
- `bin/`: Directory containing the wrapped binary

## How It Works

1. **Flag Extraction**: If source code is provided, the tool analyzes it to find `flag.XXX()` calls
2. **Schema Generation**: Creates JSON schemas for each flag type
3. **Server Generation**: Creates an MCP server that wraps the binary
4. **Binary Integration**: Copies the binary and sets up proper execution

## Supported Flag Types

- `flag.String()` / `flag.StringVar()` → `string` schema
- `flag.Int()` / `flag.IntVar()` → `integer` schema  
- `flag.Bool()` / `flag.BoolVar()` → `boolean` schema
- `flag.Float64()` / `flag.Float64Var()` → `number` schema

## Dry Run Mode

Preview what will be generated without creating files:

```bash
cmd2mcpserver -dry-run ./mytool
```

This outputs the generated files in txtar format, which can be saved to a file:

```bash
cmd2mcpserver -dry-run ./mytool > mytool-server.txtar
```

## Development

To test the tool:

```bash
# Build the demo
go build -o demo-cli ./demo/demo.go

# Convert it to an MCP server
go run ./cmd/cmd2mcpserver -source ./demo ./demo-cli

# Preview with dry-run
go run ./cmd/cmd2mcpserver -dry-run -source ./demo ./demo-cli

# Run the generated server
cd demo-cli-mcp-server
go run .
```

## License

Part of the MCP project.