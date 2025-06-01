# MCP Trace to Go Code: Visual Example

## Input: MCP Trace

```
2024-01-15T10:00:00 -> initialize {"method":"initialize","params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"demo-client","version":"1.0"}}}
2024-01-15T10:00:01 <- initialize {"result":{"protocolVersion":"1.0","capabilities":{"tools":{"listChanged":true}},"serverInfo":{"name":"demo-server","version":"1.0"}}}
2024-01-15T10:00:02 -> tools/list {"method":"tools/list","params":{}}
2024-01-15T10:00:03 <- tools/list {"result":{"tools":[{"name":"get_weather","description":"Get current weather for a location","inputSchema":{"type":"object","properties":{"location":{"type":"string","description":"City name"},"units":{"type":"string","enum":["celsius","fahrenheit"]}},"required":["location"]}}]}}
2024-01-15T10:00:04 -> tools/call {"method":"tools/call","params":{"name":"get_weather","arguments":{"location":"San Francisco","units":"celsius"}}}
2024-01-15T10:00:05 <- tools/call {"result":{"content":[{"type":"text","text":"Current weather in San Francisco: 18°C, partly cloudy"}]}}
```

## Real-Time Processing

As each line is processed, the tool builds understanding:

### After Line 1 (Client Initialize):
```
Discovered: Client (demo-client v1.0)
```

### After Line 2 (Server Response):
```
Discovered: Server (demo-server v1.0)
Capabilities: Tools with listChanged
```

### After Line 4 (Tool List):
```
Discovered Tool: get_weather
- Takes: location (required), units (optional)
- Purpose: Get current weather for a location
```

### After Line 6 (Tool Call):
```
Tool Usage Example: get_weather
- Input: {"location": "San Francisco", "units": "celsius"}
- Output: Text content with weather info
```

## Output: Generated Go Code

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

// MCPClient implements the MCP client
type MCPClient struct {
    *mcp.Client
}

// NewMCPClient creates a new MCP client
func NewMCPClient() *MCPClient {
    return &MCPClient{
        Client: mcp.NewClient(),
    }
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

## Key Features Demonstrated

1. **Progressive Understanding**: The tool builds its understanding incrementally as it processes each trace line

2. **Automatic Detection**:
   - Server/client roles from message direction
   - Available tools from list responses
   - Parameter schemas from tool definitions
   - Usage examples from actual calls

3. **Smart Code Generation**:
   - Proper type definitions
   - Input validation
   - Example comments from real usage
   - Complete working skeleton

4. **Real-Time Updates**: The generated code updates live as new trace data arrives, perfect for demos and debugging

## Use Cases

- **Demo**: Show how MCP protocol maps to Go implementation
- **Learning**: Understand MCP by seeing trace→code transformation
- **Debugging**: Visualize what's happening in MCP communication
- **Prototyping**: Quickly generate initial implementation from traces