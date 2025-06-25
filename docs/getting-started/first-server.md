# Creating Your First MCP Server

This guide walks you through building a comprehensive MCP server from scratch, covering tools, resources, and prompts.

## Project Setup

### 1. Initialize Your Project

```bash
mkdir my-mcp-server
cd my-mcp-server
go mod init my-mcp-server
go get github.com/tmc/mcp
```

### 2. Project Structure

```
my-mcp-server/
├── main.go          # Server entry point
├── tools/           # Tool implementations
│   ├── calculator.go
│   ├── file_ops.go
│   └── weather.go
├── resources/       # Resource handlers
│   ├── config.go
│   └── logs.go
├── prompts/         # Prompt templates
│   └── templates.go
└── config.json     # Server configuration
```

## Basic Server Implementation

### main.go

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/tmc/mcp"
)

func main() {
    // Create server with metadata
    server := mcp.NewServer(
        "my-mcp-server",
        "1.0.0",
        mcp.WithServerInstructions("A comprehensive MCP server with tools, resources, and prompts"),
    )

    // Register components
    registerTools(server)
    registerResources(server)
    registerPrompts(server)

    // Set up context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown gracefully
    setupGracefulShutdown(cancel)

    // Configure transport
    transport := getTransport()

    // Start server
    log.Printf("Starting %s v%s", "my-mcp-server", "1.0.0")
    if err := server.Serve(ctx, transport); err != nil && err != context.Canceled {
        log.Fatal("Server error:", err)
    }
}

func setupGracefulShutdown(cancel context.CancelFunc) {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        sig := <-sigChan
        log.Printf("Received signal %v, shutting down...", sig)
        cancel()
    }()
}

func getTransport() mcp.Transport {
    // Check environment for transport type
    switch os.Getenv("MCP_TRANSPORT") {
    case "sse":
        return mcp.NewSSETransport(mcp.SSEConfig{
            Port: 8080,
        })
    case "websocket":
        return mcp.NewWebSocketTransport(mcp.WebSocketConfig{
            Port: 8081,
        })
    default:
        // Default to stdio for command-line usage
        return mcp.StdioTransport()
    }
}
```

## Tool Implementation

### tools/calculator.go

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "math"

    "github.com/tmc/mcp"
)

type CalculatorInput struct {
    Operation string  `json:"operation"` // add, subtract, multiply, divide, sqrt, power
    A         float64 `json:"a"`
    B         float64 `json:"b,omitempty"`
}

type CalculatorOutput struct {
    Result float64 `json:"result"`
    Formula string `json:"formula"`
}

func registerCalculatorTool(server *mcp.Server) error {
    tool := mcp.Tool{
        Name:        "calculator",
        Description: "Perform basic mathematical operations",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "operation": {
                    "type": "string",
                    "enum": ["add", "subtract", "multiply", "divide", "sqrt", "power"],
                    "description": "Mathematical operation to perform"
                },
                "a": {
                    "type": "number",
                    "description": "First operand"
                },
                "b": {
                    "type": "number",
                    "description": "Second operand (not required for sqrt)"
                }
            },
            "required": ["operation", "a"]
        }`),
    }

    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var input CalculatorInput
        if err := json.Unmarshal(req.Arguments, &input); err != nil {
            return &mcp.CallToolResult{
                IsError: true,
                Content: []any{
                    map[string]string{
                        "type": "text",
                        "text": fmt.Sprintf("Invalid input: %v", err),
                    },
                },
            }, nil
        }

        result, formula, err := performCalculation(input)
        if err != nil {
            return &mcp.CallToolResult{
                IsError: true,
                Content: []any{
                    map[string]string{
                        "type": "text",
                        "text": err.Error(),
                    },
                },
            }, nil
        }

        output := CalculatorOutput{
            Result:  result,
            Formula: formula,
        }

        outputJSON, _ := json.Marshal(output)
        return &mcp.CallToolResult{
            Content: []any{
                map[string]string{
                    "type": "text",
                    "text": fmt.Sprintf("Result: %g\nFormula: %s", result, formula),
                },
                map[string]string{
                    "type": "text",
                    "text": string(outputJSON),
                },
            },
        }, nil
    }

    return server.RegisterTool(tool, handler)
}

func performCalculation(input CalculatorInput) (float64, string, error) {
    switch input.Operation {
    case "add":
        return input.A + input.B, fmt.Sprintf("%.2f + %.2f", input.A, input.B), nil
    case "subtract":
        return input.A - input.B, fmt.Sprintf("%.2f - %.2f", input.A, input.B), nil
    case "multiply":
        return input.A * input.B, fmt.Sprintf("%.2f × %.2f", input.A, input.B), nil
    case "divide":
        if input.B == 0 {
            return 0, "", fmt.Errorf("division by zero")
        }
        return input.A / input.B, fmt.Sprintf("%.2f ÷ %.2f", input.A, input.B), nil
    case "sqrt":
        if input.A < 0 {
            return 0, "", fmt.Errorf("square root of negative number")
        }
        return math.Sqrt(input.A), fmt.Sprintf("√%.2f", input.A), nil
    case "power":
        return math.Pow(input.A, input.B), fmt.Sprintf("%.2f^%.2f", input.A, input.B), nil
    default:
        return 0, "", fmt.Errorf("unknown operation: %s", input.Operation)
    }
}
```

### tools/file_ops.go

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/tmc/mcp"
)

type FileOperation struct {
    Action string `json:"action"` // read, write, list, exists, delete
    Path   string `json:"path"`
    Content string `json:"content,omitempty"` // for write operations
}

func registerFileOperationsTool(server *mcp.Server) error {
    tool := mcp.Tool{
        Name:        "file_ops",
        Description: "Safe file operations within allowed directories",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "action": {
                    "type": "string",
                    "enum": ["read", "write", "list", "exists", "delete"],
                    "description": "File operation to perform"
                },
                "path": {
                    "type": "string",
                    "description": "File or directory path"
                },
                "content": {
                    "type": "string",
                    "description": "Content to write (for write action)"
                }
            },
            "required": ["action", "path"]
        }`),
    }

    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var input FileOperation
        if err := json.Unmarshal(req.Arguments, &input); err != nil {
            return errorResult("Invalid input: " + err.Error()), nil
        }

        // Security: Only allow operations in safe directories
        if !isPathSafe(input.Path) {
            return errorResult("Path not allowed for security reasons"), nil
        }

        result, err := performFileOperation(input)
        if err != nil {
            return errorResult(err.Error()), nil
        }

        return successResult(result), nil
    }

    return server.RegisterTool(tool, handler)
}

func isPathSafe(path string) bool {
    // Only allow operations in current directory and subdirectories
    absPath, err := filepath.Abs(path)
    if err != nil {
        return false
    }

    wd, err := os.Getwd()
    if err != nil {
        return false
    }

    return strings.HasPrefix(absPath, wd)
}

func performFileOperation(op FileOperation) (string, error) {
    switch op.Action {
    case "read":
        content, err := os.ReadFile(op.Path)
        if err != nil {
            return "", fmt.Errorf("failed to read file: %w", err)
        }
        return string(content), nil

    case "write":
        err := os.WriteFile(op.Path, []byte(op.Content), 0644)
        if err != nil {
            return "", fmt.Errorf("failed to write file: %w", err)
        }
        return fmt.Sprintf("Successfully wrote %d bytes to %s", len(op.Content), op.Path), nil

    case "list":
        entries, err := os.ReadDir(op.Path)
        if err != nil {
            return "", fmt.Errorf("failed to list directory: %w", err)
        }

        var result strings.Builder
        for _, entry := range entries {
            if entry.IsDir() {
                result.WriteString(fmt.Sprintf("📁 %s/\n", entry.Name()))
            } else {
                info, _ := entry.Info()
                result.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", entry.Name(), info.Size()))
            }
        }
        return result.String(), nil

    case "exists":
        _, err := os.Stat(op.Path)
        if os.IsNotExist(err) {
            return fmt.Sprintf("File %s does not exist", op.Path), nil
        } else if err != nil {
            return "", fmt.Errorf("failed to check file: %w", err)
        }
        return fmt.Sprintf("File %s exists", op.Path), nil

    case "delete":
        err := os.Remove(op.Path)
        if err != nil {
            return "", fmt.Errorf("failed to delete file: %w", err)
        }
        return fmt.Sprintf("Successfully deleted %s", op.Path), nil

    default:
        return "", fmt.Errorf("unknown action: %s", op.Action)
    }
}

func errorResult(message string) *mcp.CallToolResult {
    return &mcp.CallToolResult{
        IsError: true,
        Content: []any{
            map[string]string{
                "type": "text",
                "text": message,
            },
        },
    }
}

func successResult(message string) *mcp.CallToolResult {
    return &mcp.CallToolResult{
        Content: []any{
            map[string]string{
                "type": "text",
                "text": message,
            },
        },
    }
}
```

## Resource Implementation

### resources/config.go

```go
package main

import (
    "context"
    "encoding/json"
    "os"

    "github.com/tmc/mcp"
)

func registerConfigResource(server *mcp.Server) error {
    resource := mcp.Resource{
        URI:         "config://server.json",
        Description: "Server configuration file",
        MimeType:    "application/json",
    }

    handler := func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
        // Read configuration file
        configPath := "config.json"
        content, err := os.ReadFile(configPath)
        if err != nil {
            // Return default config if file doesn't exist
            defaultConfig := map[string]any{
                "server": map[string]any{
                    "name":    "my-mcp-server",
                    "version": "1.0.0",
                    "debug":   false,
                },
                "transport": map[string]any{
                    "type": "stdio",
                    "port": 8080,
                },
                "security": map[string]any{
                    "allowedPaths": []string{"."},
                },
            }
            content, _ = json.MarshalIndent(defaultConfig, "", "  ")
        }

        return []mcp.ResourceContents{
            mcp.TextResourceContents{
                URI:      req.URI,
                MimeType: "application/json",
                Text:     string(content),
            },
        }, nil
    }

    return server.RegisterResource(resource, handler)
}
```

### resources/logs.go

```go
package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/tmc/mcp"
)

func registerLogResources(server *mcp.Server) error {
    // Register log template for dynamic log file access
    template := mcp.ResourceTemplate{
        Template:    "logs://{date}.log",
        Description: "Daily log files (format: YYYY-MM-DD)",
    }

    handler := func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
        // Extract date from URI: logs://2024-01-15.log
        uri := req.URI
        if len(uri) < 11 || uri[:7] != "logs://" {
            return nil, fmt.Errorf("invalid log URI format")
        }

        dateStr := uri[7:]
        if len(dateStr) < 10 {
            return nil, fmt.Errorf("invalid date format")
        }

        // Validate date format
        _, err := time.Parse("2006-01-02", dateStr[:10])
        if err != nil {
            return nil, fmt.Errorf("invalid date format: %w", err)
        }

        // Read log file
        logPath := filepath.Join("logs", dateStr[:10]+".log")
        content, err := os.ReadFile(logPath)
        if err != nil {
            if os.IsNotExist(err) {
                content = []byte(fmt.Sprintf("No logs found for %s", dateStr[:10]))
            } else {
                return nil, fmt.Errorf("failed to read log file: %w", err)
            }
        }

        return []mcp.ResourceContents{
            mcp.TextResourceContents{
                URI:      req.URI,
                MimeType: "text/plain",
                Text:     string(content),
            },
        }, nil
    }

    return server.RegisterResourceTemplate(template, handler)
}
```

## Prompt Implementation

### prompts/templates.go

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/tmc/mcp"
)

func registerPrompts(server *mcp.Server) {
    registerCodeReviewPrompt(server)
    registerDocumentationPrompt(server)
}

func registerCodeReviewPrompt(server *mcp.Server) error {
    prompt := mcp.Prompt{
        Name:        "code_review",
        Description: "Generate a code review prompt for the given code",
        Arguments: []mcp.PromptArgument{
            {
                Name:        "code",
                Description: "The code to review",
                Required:    true,
                Schema: map[string]any{
                    "type": "string",
                },
            },
            {
                Name:        "language",
                Description: "Programming language",
                Required:    false,
                Schema: map[string]any{
                    "type": "string",
                    "default": "go",
                },
            },
            {
                Name:        "focus",
                Description: "Areas to focus on (security, performance, readability)",
                Required:    false,
                Schema: map[string]any{
                    "type": "string",
                    "default": "general",
                },
            },
        },
    }

    handler := func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
        code, ok := req.Arguments["code"].(string)
        if !ok || code == "" {
            return nil, fmt.Errorf("code argument is required")
        }

        language := "go"
        if lang, ok := req.Arguments["language"].(string); ok && lang != "" {
            language = lang
        }

        focus := "general"
        if f, ok := req.Arguments["focus"].(string); ok && f != "" {
            focus = f
        }

        // Generate the prompt
        var prompt strings.Builder
        prompt.WriteString("Please review the following ")
        prompt.WriteString(language)
        prompt.WriteString(" code")

        switch focus {
        case "security":
            prompt.WriteString(" with a focus on security vulnerabilities and best practices")
        case "performance":
            prompt.WriteString(" with a focus on performance optimization opportunities")
        case "readability":
            prompt.WriteString(" with a focus on code readability and maintainability")
        default:
            prompt.WriteString(" for overall quality, best practices, and potential improvements")
        }

        prompt.WriteString(":\n\n```")
        prompt.WriteString(language)
        prompt.WriteString("\n")
        prompt.WriteString(code)
        prompt.WriteString("\n```\n\n")
        prompt.WriteString("Please provide:\n")
        prompt.WriteString("1. Overall assessment\n")
        prompt.WriteString("2. Specific issues found\n")
        prompt.WriteString("3. Suggested improvements\n")
        prompt.WriteString("4. Best practices recommendations")

        return &mcp.GetPromptResult{
            Messages: []mcp.PromptMessage{
                {
                    Role: mcp.RoleUser,
                    Content: []any{
                        map[string]string{
                            "type": "text",
                            "text": prompt.String(),
                        },
                    },
                },
            },
        }, nil
    }

    return server.RegisterPrompt(prompt, handler)
}

func registerDocumentationPrompt(server *mcp.Server) error {
    prompt := mcp.Prompt{
        Name:        "documentation",
        Description: "Generate documentation for code or APIs",
        Arguments: []mcp.PromptArgument{
            {
                Name:        "code",
                Description: "Code to document",
                Required:    true,
                Schema: map[string]any{
                    "type": "string",
                },
            },
            {
                Name:        "style",
                Description: "Documentation style (godoc, javadoc, markdown)",
                Required:    false,
                Schema: map[string]any{
                    "type": "string",
                    "default": "godoc",
                },
            },
        },
    }

    handler := func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
        code, ok := req.Arguments["code"].(string)
        if !ok || code == "" {
            return nil, fmt.Errorf("code argument is required")
        }

        style := "godoc"
        if s, ok := req.Arguments["style"].(string); ok && s != "" {
            style = s
        }

        promptText := fmt.Sprintf(`Please generate comprehensive documentation for the following code using %s style:

%s%s%s

Please include:
1. Package/module overview
2. Function/method documentation
3. Parameter descriptions
4. Return value descriptions
5. Usage examples where appropriate
6. Any important notes or warnings

Format the documentation according to %s conventions.`,
            style,
            "```\n",
            code,
            "\n```",
            style)

        return &mcp.GetPromptResult{
            Messages: []mcp.PromptMessage{
                {
                    Role: mcp.RoleUser,
                    Content: []any{
                        map[string]string{
                            "type": "text",
                            "text": promptText,
                        },
                    },
                },
            },
        }, nil
    }

    return server.RegisterPrompt(prompt, handler)
}
```

## Registration Functions

Add these to your `main.go`:

```go
func registerTools(server *mcp.Server) {
    if err := registerCalculatorTool(server); err != nil {
        log.Printf("Failed to register calculator tool: %v", err)
    }

    if err := registerFileOperationsTool(server); err != nil {
        log.Printf("Failed to register file operations tool: %v", err)
    }
}

func registerResources(server *mcp.Server) {
    if err := registerConfigResource(server); err != nil {
        log.Printf("Failed to register config resource: %v", err)
    }

    if err := registerLogResources(server); err != nil {
        log.Printf("Failed to register log resources: %v", err)
    }
}
```

## Testing Your Server

### 1. Build and Run

```bash
go build -o my-mcp-server
./my-mcp-server
```

### 2. Test with curl (SSE transport)

```bash
# Set SSE transport
export MCP_TRANSPORT=sse
./my-mcp-server &

# Test initialization
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}'

# Test calculator tool
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"calculator","arguments":{"operation":"add","a":5,"b":3}}}'
```

### 3. Test with mcp-connect

```bash
go run ./cmd/mcp-connect -cmd="./my-mcp-server"
```

## Next Steps

1. **Add Authentication**: Implement auth for your transport
2. **Add Logging**: Integrate structured logging
3. **Add Configuration**: Support configuration files
4. **Add Middleware**: Implement request/response middleware
5. **Add Metrics**: Monitor server performance
6. **Add Tests**: Write comprehensive tests

Your MCP server is now ready to handle complex interactions with tools, resources, and prompts!