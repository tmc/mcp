# cmd2mcpserver Demo

This demo shows how to convert a Go CLI tool into an MCP server.

## Step 1: Create a CLI Tool

First, let's look at our demo CLI tool:

```go
// demo/demo.go
package main

import (
    "flag"
    "fmt"
)

func main() {
    verbose := flag.Bool("verbose", false, "Enable verbose output")
    format := flag.String("format", "text", "Output format")
    count := flag.Int("count", 1, "Number of repetitions")
    flag.Parse()
    
    // ... tool logic ...
}
```

## Step 2: Convert to MCP Server

Run `cmd2mcpserver`:

```bash
cmd2mcpserver -source ./demo -output ./myserver ./demo-cli
```

## Step 3: Generated MCP Server

The tool generates a complete MCP server:

```go
// myserver/main.go
type DemoCliTool struct {
    binaryPath string
}

func (t *DemoCliTool) InputSchema() any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "verbose": map[string]any{
                "type": "boolean",
                "description": "Enable verbose output",
                "default": false,
            },
            "format": map[string]any{
                "type": "string", 
                "description": "Output format",
                "default": "text",
            },
            "count": map[string]any{
                "type": "integer",
                "description": "Number of repetitions",
                "default": 1,
            },
        },
    }
}

func (t *DemoCliTool) Execute(ctx context.Context, params any) (any, error) {
    // Converts params to CLI flags and executes the binary
}
```

## Step 4: Run the Server

```bash
cd myserver
go run .
```

Now your CLI tool is available as an MCP tool!

## Features

✅ Automatic flag extraction from source code  
✅ JSON schema generation for parameters  
✅ Type-safe parameter handling  
✅ Error handling and exit codes  
✅ Complete Go module with dependencies  

## Use Cases

- Expose legacy CLI tools via MCP
- Create MCP wrappers for system utilities
- Build tool integrations without modifying source
- Rapid prototyping of MCP tools