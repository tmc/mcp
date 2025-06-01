# mcp-trace-codegen

Real-time Go code generation from MCP trace data. This tool analyzes MCP trace files and generates corresponding Go code as it processes each line, making it perfect for demos and understanding MCP protocol interactions.

## Features

- Real-time code generation as trace data is processed
- Automatic detection of servers, clients, tools, and handlers
- Generation of complete MCP server/client implementations
- Visual progress indicators
- Support for both file input and streaming

## Installation

```bash
go install github.com/tmc/mcp/exp/cmd/mcp-trace-codegen@latest
```

## Usage

### Basic Usage

Process a trace file and generate code:

```bash
mcp-trace-codegen < trace.mcp > generated.go
```

### Real-time Display

Watch code generation in real-time:

```bash
mcp-trace-codegen -realtime < trace.mcp
```

### Streaming Mode

Pipe trace data for live code generation:

```bash
tail -f trace.mcp | mcp-trace-codegen -realtime -clear
```

### Options

- `-realtime`: Enable real-time display mode
- `-progress`: Show progress indicators (default: true)
- `-clear`: Clear screen on updates (default: true)
- `-package`: Package name for generated code (default: "generated")
- `-output`: Output file (default: stdout)

## Demo

Run the included demo script:

```bash
./demo.sh
```

This will:
1. Create a sample MCP trace
2. Show real-time code generation
3. Simulate streaming trace data

## Example Output

As the tool processes trace data, it generates Go code like:

```go
package weather_server

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/tmc/mcp"
    "github.com/tmc/mcp/modelcontextprotocol"
)

// MCPServer implements the MCP protocol
type MCPServer struct {
    *mcp.Server
    tools map[string]*Tool
}

// NewMCPServer creates a new MCP server
func NewMCPServer() *MCPServer {
    server := &MCPServer{
        Server: mcp.NewServer(),
        tools: make(map[string]*Tool),
    }
    
    server.SetInfo("demo-server", "1.0")
    
    // Configure capabilities
    server.EnableTools()
    
    // Register tools
    server.RegisterTool("get_weather")
    
    return server
}

// getWeather implements the get_weather tool
// Get current weather for a location
func (s *MCPServer) handleGetWeather(ctx context.Context, params json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
    // Parse input parameters
    var input struct {
        Location string `json:"location"`
        Units    string `json:"units"`
    }
    if err := json.Unmarshal(params, &input); err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }
    
    // Validate required fields
    if input.Location == "" {
        return nil, fmt.Errorf("location is required")
    }
    
    // TODO: Implement tool logic
    // Examples from trace:
    // Example 1: {
    //     "location": "San Francisco",
    //     "units": "celsius"
    // }
    
    return &modelcontextprotocol.CallToolResult{
        Content: []modelcontextprotocol.Content{
            {
                Type: "text",
                Text: "Tool executed successfully",
            },
        },
    }, nil
}

func main() {
    // Create and run server
    server := NewMCPServer()
    
    // Use stdio transport by default
    transport := mcp.NewStdioTransport()
    
    if err := server.Serve(transport); err != nil {
        fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
        os.Exit(1)
    }
}
```

## How It Works

1. **Trace Parsing**: Reads MCP trace lines in format:
   ```
   timestamp direction method payload
   ```

2. **State Analysis**: Builds understanding of:
   - Server/client roles
   - Available tools
   - Message patterns
   - Capabilities

3. **Code Generation**: Produces:
   - Server/client types
   - Tool implementations
   - Handler functions
   - Main function

4. **Real-time Updates**: Continuously updates generated code as new trace data arrives

## Use Cases

- **Demo Tool**: Show how MCP traces map to Go code
- **Learning Aid**: Understand MCP protocol through code generation
- **Rapid Prototyping**: Generate initial implementation from traces
- **Debugging**: Visualize protocol interactions as code

## Integration

This tool is part of the MCP Go implementation suite and works well with:
- `mcpspy`: Capture live MCP traffic
- `mcpdiff`: Compare trace files
- `mcp2go`: Generate code from MCP definitions

## Contributing

Contributions welcome! Areas for improvement:
- Better type inference from trace data
- Support for more MCP features
- Improved code formatting
- Template customization