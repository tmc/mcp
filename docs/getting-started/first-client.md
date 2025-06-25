# Creating Your First MCP Client

This guide shows you how to build a comprehensive MCP client that can connect to servers and utilize their tools, resources, and prompts effectively.

## Project Setup

### 1. Initialize Your Client Project

```bash
mkdir my-mcp-client
cd my-mcp-client
go mod init my-mcp-client
go get github.com/tmc/mcp
```

### 2. Project Structure

```
my-mcp-client/
├── main.go              # Client entry point
├── client/              # Client implementation
│   ├── manager.go       # Connection management
│   ├── tools.go         # Tool interaction
│   ├── resources.go     # Resource access
│   └── prompts.go       # Prompt handling
├── config/              # Configuration
│   └── config.go
└── examples/            # Usage examples
    ├── calculator.go
    ├── file_browser.go
    └── prompt_generator.go
```

## Basic Client Implementation

### main.go

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"

    "github.com/tmc/mcp"
)

func main() {
    var (
        serverCmd   = flag.String("server", "", "MCP server command to run")
        serverURL   = flag.String("url", "", "MCP server URL (for WebSocket/SSE)")
        transport   = flag.String("transport", "stdio", "Transport type: stdio, sse, websocket")
        interactive = flag.Bool("interactive", false, "Run in interactive mode")
    )
    flag.Parse()

    if *serverCmd == "" && *serverURL == "" {
        log.Fatal("Either -server or -url must be specified")
    }

    ctx := context.Background()

    // Create and connect client
    client, err := createClient(*transport, *serverCmd, *serverURL)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Initialize connection
    if err := initializeClient(ctx, client); err != nil {
        log.Fatal("Failed to initialize client:", err)
    }

    if *interactive {
        runInteractiveMode(ctx, client)
    } else {
        runExamples(ctx, client)
    }
}

func createClient(transportType, serverCmd, serverURL string) (*mcp.Client, error) {
    var transport mcp.Transport
    var err error

    switch transportType {
    case "stdio":
        if serverCmd == "" {
            return nil, fmt.Errorf("server command required for stdio transport")
        }
        transport, err = createStdioTransport(serverCmd)
    case "sse":
        if serverURL == "" {
            return nil, fmt.Errorf("server URL required for SSE transport")
        }
        transport = mcp.NewSSEClientTransport(serverURL)
    case "websocket":
        if serverURL == "" {
            return nil, fmt.Errorf("server URL required for WebSocket transport")
        }
        transport = mcp.NewWebSocketClientTransport(serverURL)
    default:
        return nil, fmt.Errorf("unknown transport type: %s", transportType)
    }

    if err != nil {
        return nil, err
    }

    return mcp.NewClient(transport)
}

func createStdioTransport(serverCmd string) (mcp.Transport, error) {
    // Implementation depends on your subprocess management approach
    // This is a simplified version
    cmd := exec.Command("sh", "-c", serverCmd)
    return mcp.NewSubprocessTransport(cmd), nil
}

func initializeClient(ctx context.Context, client *mcp.Client) error {
    initRequest := mcp.InitializeRequest{
        ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
        ClientInfo: mcp.Implementation{
            Name:    "my-mcp-client",
            Version: "1.0.0",
        },
        Capabilities: mcp.ClientCapabilities{},
    }

    result, err := client.Initialize(ctx, initRequest)
    if err != nil {
        return err
    }

    fmt.Printf("✅ Connected to %s v%s\n", 
        result.ServerInfo.Name, 
        result.ServerInfo.Version)

    if result.Instructions != "" {
        fmt.Printf("📋 Server instructions: %s\n", result.Instructions)
    }

    return nil
}
```

## Client Manager Implementation

### client/manager.go

```go
package client

import (
    "context"
    "fmt"
    "sync"

    "github.com/tmc/mcp"
)

type MCPManager struct {
    client       *mcp.Client
    capabilities *mcp.ServerCapabilities
    tools        []mcp.Tool
    resources    []mcp.Resource
    prompts      []mcp.Prompt
    mu           sync.RWMutex
}

func NewMCPManager(client *mcp.Client) *MCPManager {
    return &MCPManager{
        client: client,
    }
}

func (m *MCPManager) Initialize(ctx context.Context) error {
    // Initialize client connection
    initRequest := mcp.InitializeRequest{
        ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
        ClientInfo: mcp.Implementation{
            Name:    "mcp-client-manager",
            Version: "1.0.0",
        },
        Capabilities: mcp.ClientCapabilities{},
    }

    result, err := m.client.Initialize(ctx, initRequest)
    if err != nil {
        return fmt.Errorf("initialization failed: %w", err)
    }

    m.mu.Lock()
    m.capabilities = &result.Capabilities
    m.mu.Unlock()

    // Discover available resources
    if err := m.discoverCapabilities(ctx); err != nil {
        return fmt.Errorf("capability discovery failed: %w", err)
    }

    return nil
}

func (m *MCPManager) discoverCapabilities(ctx context.Context) error {
    // Discover tools
    if m.capabilities.Tools != nil {
        toolsResult, err := m.client.ListTools(ctx, mcp.ListToolsRequest{})
        if err != nil {
            return fmt.Errorf("failed to list tools: %w", err)
        }
        
        m.mu.Lock()
        m.tools = toolsResult.Tools
        m.mu.Unlock()
        
        fmt.Printf("🔧 Discovered %d tools\n", len(toolsResult.Tools))
    }

    // Discover resources
    if m.capabilities.Resources != nil {
        resourcesResult, err := m.client.ListResources(ctx, mcp.ListResourcesRequest{})
        if err != nil {
            return fmt.Errorf("failed to list resources: %w", err)
        }
        
        m.mu.Lock()
        m.resources = resourcesResult.Resources
        m.mu.Unlock()
        
        fmt.Printf("📁 Discovered %d resources\n", len(resourcesResult.Resources))
    }

    // Discover prompts
    if m.capabilities.Prompts != nil {
        promptsResult, err := m.client.ListPrompts(ctx, mcp.ListPromptsRequest{})
        if err != nil {
            return fmt.Errorf("failed to list prompts: %w", err)
        }
        
        m.mu.Lock()
        m.prompts = promptsResult.Prompts
        m.mu.Unlock()
        
        fmt.Printf("💬 Discovered %d prompts\n", len(promptsResult.Prompts))
    }

    return nil
}

func (m *MCPManager) GetCapabilities() *mcp.ServerCapabilities {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.capabilities
}

func (m *MCPManager) GetTools() []mcp.Tool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return append([]mcp.Tool{}, m.tools...)
}

func (m *MCPManager) GetResources() []mcp.Resource {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return append([]mcp.Resource{}, m.resources...)
}

func (m *MCPManager) GetPrompts() []mcp.Prompt {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return append([]mcp.Prompt{}, m.prompts...)
}

func (m *MCPManager) Close() error {
    return m.client.Close()
}
```

## Tool Interaction

### client/tools.go

```go
package client

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/tmc/mcp"
)

type ToolManager struct {
    manager *MCPManager
}

func NewToolManager(manager *MCPManager) *ToolManager {
    return &ToolManager{manager: manager}
}

func (tm *ToolManager) CallTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error) {
    // Validate tool exists
    tools := tm.manager.GetTools()
    var tool *mcp.Tool
    for _, t := range tools {
        if t.Name == name {
            tool = &t
            break
        }
    }

    if tool == nil {
        return nil, fmt.Errorf("tool '%s' not found", name)
    }

    // Validate arguments against schema if available
    if tool.InputSchema != nil {
        if err := tm.validateArguments(arguments, tool.InputSchema); err != nil {
            return nil, fmt.Errorf("argument validation failed: %w", err)
        }
    }

    // Marshal arguments
    argsJSON, err := json.Marshal(arguments)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal arguments: %w", err)
    }

    // Call the tool
    request := mcp.CallToolRequest{
        Name:      name,
        Arguments: argsJSON,
    }

    result, err := tm.manager.client.CallTool(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("tool call failed: %w", err)
    }

    return result, nil
}

func (tm *ToolManager) validateArguments(arguments map[string]any, schema json.RawMessage) error {
    // Basic validation - in a real implementation, use a JSON schema validator
    var schemaMap map[string]any
    if err := json.Unmarshal(schema, &schemaMap); err != nil {
        return err
    }

    properties, ok := schemaMap["properties"].(map[string]any)
    if !ok {
        return nil // No properties to validate
    }

    required, _ := schemaMap["required"].([]any)
    
    // Check required fields
    for _, req := range required {
        reqField, ok := req.(string)
        if !ok {
            continue
        }
        
        if _, exists := arguments[reqField]; !exists {
            return fmt.Errorf("required field '%s' missing", reqField)
        }
    }

    // Basic type checking
    for field, value := range arguments {
        if propSchema, exists := properties[field]; exists {
            if err := tm.validateFieldType(field, value, propSchema); err != nil {
                return err
            }
        }
    }

    return nil
}

func (tm *ToolManager) validateFieldType(field string, value any, schema any) error {
    schemaMap, ok := schema.(map[string]any)
    if !ok {
        return nil
    }

    expectedType, ok := schemaMap["type"].(string)
    if !ok {
        return nil
    }

    switch expectedType {
    case "string":
        if _, ok := value.(string); !ok {
            return fmt.Errorf("field '%s' must be a string", field)
        }
    case "number":
        switch value.(type) {
        case float64, int, int64, float32:
            // Valid number types
        default:
            return fmt.Errorf("field '%s' must be a number", field)
        }
    case "boolean":
        if _, ok := value.(bool); !ok {
            return fmt.Errorf("field '%s' must be a boolean", field)
        }
    case "object":
        if _, ok := value.(map[string]any); !ok {
            return fmt.Errorf("field '%s' must be an object", field)
        }
    case "array":
        if _, ok := value.([]any); !ok {
            return fmt.Errorf("field '%s' must be an array", field)
        }
    }

    return nil
}

func (tm *ToolManager) ListAvailableTools() {
    tools := tm.manager.GetTools()
    fmt.Printf("Available Tools (%d):\n", len(tools))
    for _, tool := range tools {
        fmt.Printf("  🔧 %s: %s\n", tool.Name, tool.Description)
        if tool.InputSchema != nil {
            fmt.Printf("     Input schema available\n")
        }
    }
}
```

## Resource Access

### client/resources.go

```go
package client

import (
    "context"
    "fmt"
    "strings"

    "github.com/tmc/mcp"
)

type ResourceManager struct {
    manager *MCPManager
}

func NewResourceManager(manager *MCPManager) *ResourceManager {
    return &ResourceManager{manager: manager}
}

func (rm *ResourceManager) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
    // Validate resource exists or matches template
    if !rm.isResourceAvailable(uri) {
        return nil, fmt.Errorf("resource '%s' not found or not accessible", uri)
    }

    request := mcp.ReadResourceRequest{
        URI: uri,
    }

    result, err := rm.manager.client.ReadResource(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("failed to read resource: %w", err)
    }

    return result, nil
}

func (rm *ResourceManager) isResourceAvailable(uri string) bool {
    resources := rm.manager.GetResources()
    
    // Check exact match
    for _, resource := range resources {
        if resource.URI == uri {
            return true
        }
    }

    // Check template match (simplified)
    templates, err := rm.getResourceTemplates(context.Background())
    if err != nil {
        return false
    }

    for _, template := range templates {
        if rm.matchesTemplate(uri, template.Template) {
            return true
        }
    }

    return false
}

func (rm *ResourceManager) getResourceTemplates(ctx context.Context) ([]mcp.ResourceTemplate, error) {
    result, err := rm.manager.client.ListResourceTemplates(ctx, mcp.ListResourceTemplatesRequest{})
    if err != nil {
        return nil, err
    }
    return result.Templates, nil
}

func (rm *ResourceManager) matchesTemplate(uri, template string) bool {
    // Simple template matching - replace {param} with wildcard
    pattern := template
    
    // Replace template variables with wildcards
    for strings.Contains(pattern, "{") {
        start := strings.Index(pattern, "{")
        end := strings.Index(pattern[start:], "}")
        if end == -1 {
            break
        }
        end += start
        
        // Replace {param} with *
        pattern = pattern[:start] + "*" + pattern[end+1:]
    }

    // Simple glob matching
    return rm.simpleGlobMatch(pattern, uri)
}

func (rm *ResourceManager) simpleGlobMatch(pattern, text string) bool {
    // Simplified glob matching for demonstration
    if !strings.Contains(pattern, "*") {
        return pattern == text
    }

    parts := strings.Split(pattern, "*")
    if len(parts) == 0 {
        return true
    }

    // Check prefix
    if !strings.HasPrefix(text, parts[0]) {
        return false
    }
    text = text[len(parts[0]):]

    // Check middle parts
    for i := 1; i < len(parts)-1; i++ {
        idx := strings.Index(text, parts[i])
        if idx == -1 {
            return false
        }
        text = text[idx+len(parts[i]):]
    }

    // Check suffix
    if len(parts) > 1 {
        return strings.HasSuffix(text, parts[len(parts)-1])
    }

    return true
}

func (rm *ResourceManager) ListAvailableResources() {
    resources := rm.manager.GetResources()
    fmt.Printf("Available Resources (%d):\n", len(resources))
    for _, resource := range resources {
        fmt.Printf("  📁 %s: %s\n", resource.URI, resource.Description)
        if resource.MimeType != "" {
            fmt.Printf("     Type: %s\n", resource.MimeType)
        }
    }

    // Also list templates
    templates, err := rm.getResourceTemplates(context.Background())
    if err == nil && len(templates) > 0 {
        fmt.Printf("\nResource Templates (%d):\n", len(templates))
        for _, template := range templates {
            fmt.Printf("  📋 %s: %s\n", template.Template, template.Description)
        }
    }
}

func (rm *ResourceManager) BrowseResource(ctx context.Context, uri string) error {
    result, err := rm.ReadResource(ctx, uri)
    if err != nil {
        return err
    }

    fmt.Printf("Resource: %s\n", uri)
    fmt.Printf("Contents (%d items):\n", len(result.Contents))

    for i, content := range result.Contents {
        fmt.Printf("\n--- Content %d ---\n", i+1)
        
        switch c := content.(type) {
        case mcp.TextResourceContents:
            fmt.Printf("Type: Text\n")
            fmt.Printf("URI: %s\n", c.URI)
            if c.MimeType != "" {
                fmt.Printf("MIME Type: %s\n", c.MimeType)
            }
            fmt.Printf("Content:\n%s\n", c.Text)
            
        case mcp.BlobResourceContents:
            fmt.Printf("Type: Binary\n")
            fmt.Printf("URI: %s\n", c.URI)
            if c.MimeType != "" {
                fmt.Printf("MIME Type: %s\n", c.MimeType)
            }
            fmt.Printf("Content: %d bytes (base64 encoded)\n", len(c.Blob))
            
        default:
            fmt.Printf("Type: Unknown\n")
            fmt.Printf("Content: %+v\n", content)
        }
    }

    return nil
}
```

## Usage Examples

### examples/calculator.go

```go
package main

import (
    "context"
    "fmt"
    "log"

    "my-mcp-client/client"
)

func runCalculatorExample(ctx context.Context, manager *client.MCPManager) {
    fmt.Println("\n🧮 Calculator Example")
    fmt.Println("====================")

    toolManager := client.NewToolManager(manager)

    // Test basic arithmetic
    operations := []struct {
        name string
        args map[string]any
    }{
        {"Addition", map[string]any{"operation": "add", "a": 15.5, "b": 7.3}},
        {"Multiplication", map[string]any{"operation": "multiply", "a": 6, "b": 9}},
        {"Square Root", map[string]any{"operation": "sqrt", "a": 64}},
        {"Power", map[string]any{"operation": "power", "a": 2, "b": 8}},
    }

    for _, op := range operations {
        fmt.Printf("\n%s: ", op.name)
        
        result, err := toolManager.CallTool(ctx, "calculator", op.args)
        if err != nil {
            fmt.Printf("❌ Error: %v\n", err)
            continue
        }

        if result.IsError {
            fmt.Printf("❌ Tool Error: %v\n", result.Content)
            continue
        }

        fmt.Printf("✅ ")
        for _, content := range result.Content {
            if contentMap, ok := content.(map[string]any); ok {
                if text, ok := contentMap["text"].(string); ok {
                    fmt.Print(text)
                }
            }
        }
        fmt.Println()
    }
}
```

### examples/file_browser.go

```go
package main

import (
    "context"
    "fmt"

    "my-mcp-client/client"
)

func runFileBrowserExample(ctx context.Context, manager *client.MCPManager) {
    fmt.Println("\n📁 File Browser Example")
    fmt.Println("=======================")

    resourceManager := client.NewResourceManager(manager)

    // List available resources
    resourceManager.ListAvailableResources()

    // Browse specific resources
    resourcesToRead := []string{
        "config://server.json",
        "logs://2024-01-15.log",
    }

    for _, uri := range resourcesToRead {
        fmt.Printf("\n--- Reading %s ---\n", uri)
        if err := resourceManager.BrowseResource(ctx, uri); err != nil {
            fmt.Printf("❌ Error reading %s: %v\n", uri, err)
        }
    }

    // Also demonstrate file operations through tools
    toolManager := client.NewToolManager(manager)
    
    fmt.Println("\n📄 File Operations Example")
    fmt.Println("===========================")

    // List current directory
    result, err := toolManager.CallTool(ctx, "file_ops", map[string]any{
        "action": "list",
        "path":   ".",
    })
    
    if err != nil {
        fmt.Printf("❌ Error listing directory: %v\n", err)
        return
    }

    fmt.Println("Current directory contents:")
    for _, content := range result.Content {
        if contentMap, ok := content.(map[string]any); ok {
            if text, ok := contentMap["text"].(string); ok {
                fmt.Print(text)
            }
        }
    }
}
```

### examples/prompt_generator.go

```go
package main

import (
    "context"
    "fmt"

    "my-mcp-client/client"
)

func runPromptExample(ctx context.Context, manager *client.MCPManager) {
    fmt.Println("\n💬 Prompt Generation Example")
    fmt.Println("============================")

    promptManager := client.NewPromptManager(manager)

    // Generate code review prompt
    sampleCode := `func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}`

    fmt.Println("Generating code review prompt...")
    
    result, err := promptManager.GetPrompt(ctx, "code_review", map[string]any{
        "code":     sampleCode,
        "language": "go",
        "focus":    "performance",
    })

    if err != nil {
        fmt.Printf("❌ Error generating prompt: %v\n", err)
        return
    }

    fmt.Println("✅ Generated prompt:")
    for _, message := range result.Messages {
        fmt.Printf("Role: %s\n", message.Role)
        for _, content := range message.Content {
            if contentMap, ok := content.(map[string]any); ok {
                if text, ok := contentMap["text"].(string); ok {
                    fmt.Printf("Content:\n%s\n", text)
                }
            }
        }
    }

    // Generate documentation prompt
    fmt.Println("\n--- Documentation Prompt ---")
    
    docResult, err := promptManager.GetPrompt(ctx, "documentation", map[string]any{
        "code":  sampleCode,
        "style": "godoc",
    })

    if err != nil {
        fmt.Printf("❌ Error generating documentation prompt: %v\n", err)
        return
    }

    fmt.Println("✅ Generated documentation prompt:")
    for _, message := range docResult.Messages {
        for _, content := range message.Content {
            if contentMap, ok := content.(map[string]any); ok {
                if text, ok := contentMap["text"].(string); ok {
                    fmt.Printf("%s\n", text)
                }
            }
        }
    }
}
```

## Interactive Mode

Add this to your `main.go`:

```go
func runInteractiveMode(ctx context.Context, client *mcp.Client) {
    manager := client.NewMCPManager(client)
    if err := manager.Initialize(ctx); err != nil {
        log.Fatal("Failed to initialize manager:", err)
    }
    defer manager.Close()

    scanner := bufio.NewScanner(os.Stdin)
    toolManager := client.NewToolManager(manager)
    resourceManager := client.NewResourceManager(manager)

    fmt.Println("\n🚀 Interactive MCP Client")
    fmt.Println("Commands: tools, resources, call <tool> <args>, read <uri>, quit")

    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }

        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }

        parts := strings.Fields(line)
        command := parts[0]

        switch command {
        case "quit", "exit":
            return
        case "tools":
            toolManager.ListAvailableTools()
        case "resources":
            resourceManager.ListAvailableResources()
        case "call":
            if len(parts) < 2 {
                fmt.Println("Usage: call <tool> <json-args>")
                continue
            }
            handleToolCall(ctx, toolManager, parts[1:])
        case "read":
            if len(parts) < 2 {
                fmt.Println("Usage: read <uri>")
                continue
            }
            handleResourceRead(ctx, resourceManager, parts[1])
        default:
            fmt.Printf("Unknown command: %s\n", command)
        }
    }
}

func handleToolCall(ctx context.Context, tm *client.ToolManager, args []string) {
    if len(args) < 1 {
        fmt.Println("Tool name required")
        return
    }

    toolName := args[0]
    var arguments map[string]any

    if len(args) > 1 {
        argsJSON := strings.Join(args[1:], " ")
        if err := json.Unmarshal([]byte(argsJSON), &arguments); err != nil {
            fmt.Printf("Invalid JSON arguments: %v\n", err)
            return
        }
    } else {
        arguments = make(map[string]any)
    }

    result, err := tm.CallTool(ctx, toolName, arguments)
    if err != nil {
        fmt.Printf("❌ Error: %v\n", err)
        return
    }

    if result.IsError {
        fmt.Printf("❌ Tool Error: %v\n", result.Content)
        return
    }

    fmt.Printf("✅ Result: %v\n", result.Content)
}

func handleResourceRead(ctx context.Context, rm *client.ResourceManager, uri string) {
    if err := rm.BrowseResource(ctx, uri); err != nil {
        fmt.Printf("❌ Error: %v\n", err)
    }
}
```

## Running Examples

Add this to your `main.go`:

```go
func runExamples(ctx context.Context, client *mcp.Client) {
    manager := client.NewMCPManager(client)
    if err := manager.Initialize(ctx); err != nil {
        log.Fatal("Failed to initialize manager:", err)
    }
    defer manager.Close()

    // Run all examples
    runCalculatorExample(ctx, manager)
    runFileBrowserExample(ctx, manager)
    runPromptExample(ctx, manager)
}
```

## Building and Testing

```bash
# Build the client
go build -o my-mcp-client

# Test with a local server
./my-mcp-client -server="./my-mcp-server"

# Test with SSE server
./my-mcp-client -transport=sse -url="http://localhost:8080"

# Run in interactive mode
./my-mcp-client -server="./my-mcp-server" -interactive
```

## Next Steps

1. **Add Connection Pooling**: Support multiple server connections
2. **Add Caching**: Cache tool/resource/prompt listings
3. **Add Configuration**: Support configuration files
4. **Add Retry Logic**: Handle transient connection failures
5. **Add Logging**: Comprehensive logging and debugging
6. **Add UI**: Build a GUI or web interface

Your MCP client can now interact with any MCP-compliant server and utilize all their capabilities!