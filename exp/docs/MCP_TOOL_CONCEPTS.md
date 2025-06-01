# Complete MCP Tool Concepts Coverage

## 1. User Interaction Model

### Key Concepts to Implement:

```go
// Tool invocation with human-in-the-loop
type ToolInvocation struct {
    Tool      *Tool
    Arguments map[string]any
    RequiresConfirmation bool  // Based on annotations
    ConfirmationMessage  string
    ApprovalCallback     func() error
}

// UI requirements for tool exposure
type ToolUIMetadata struct {
    VisualIndicator  string   // Icon or badge for active tools
    ConfirmationUI   UIConfig // Confirmation dialog settings
    ToolExposureList []string // Which tools are exposed to AI
}
```

### Implementation Requirements:

1. **Human-in-the-Loop (HITL)**
```go
// Generate confirmation handlers
func GenerateConfirmationHandler(tool *Tool) string {
    if tool.Annotations != nil && tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint {
        return `
        // Require confirmation for destructive operations
        if !confirmWithUser("This action may destroy data. Continue?") {
            return nil, errors.New("operation cancelled by user")
        }
        `
    }
    return ""
}
```

2. **Visual Indicators**
```go
// Generate UI metadata for tools
type ToolVisualMetadata struct {
    InvocationIndicator string
    StatusBadge        string
    ConfirmationLevel  string // "none", "warning", "critical"
}
```

## 2. Capabilities Declaration

```go
// Server capability detection and generation
func GenerateCapabilities(tools []*Tool, supportsDynamicRegistration bool) ServerCapabilities {
    return ServerCapabilities{
        Tools: &ToolsServerCapability{
            ListChanged: &supportsDynamicRegistration,
        },
    }
}

// Client must check capabilities
func (c *Client) VerifyToolSupport() error {
    if c.serverCapabilities.Tools == nil {
        return errors.New("server does not support tools")
    }
    return nil
}
```

## 3. Protocol Messages Implementation

### List Tools with Pagination
```go
// Generate paginated list handler
func GenerateListToolsHandler() string {
    return `
    func (s *Server) HandleListTools(params ListToolsRequestParams) (*ListToolsResult, error) {
        // Extract cursor for pagination
        var startIndex int
        if params.Cursor != nil {
            startIndex = s.decodeCursor(*params.Cursor)
        }
        
        // Paginate tools
        pageSize := 20
        endIndex := min(startIndex + pageSize, len(s.tools))
        
        tools := s.tools[startIndex:endIndex]
        
        var nextCursor *Cursor
        if endIndex < len(s.tools) {
            cursor := s.encodeCursor(endIndex)
            nextCursor = &cursor
        }
        
        return &ListToolsResult{
            Tools:      tools,
            NextCursor: nextCursor,
        }, nil
    }
    `
}
```

### Tool Invocation
```go
// Generate secure tool call handler
func GenerateToolCallHandler(tool *Tool) string {
    return `
    func (s *Server) Handle{{ .Name }}Call(params CallToolRequestParams) (*CallToolResult, error) {
        // Validate tool name
        if params.Name != "{{ .Name }}" {
            return nil, fmt.Errorf("unknown tool: %s", params.Name)
        }
        
        // Validate inputs against schema
        if err := s.validateInputs(params.Arguments, {{ .Name }}Schema); err != nil {
            return nil, NewInvalidParamsError(err)
        }
        
        // Rate limiting
        if err := s.rateLimiter.Check(params.Name); err != nil {
            return nil, NewRateLimitError(err)
        }
        
        // Access control
        if !s.hasPermission(ctx, params.Name) {
            return nil, NewAccessDeniedError()
        }
        
        // Execute tool with timeout
        ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
        
        result, err := s.execute{{ .Name }}(ctx, params.Arguments)
        if err != nil {
            // Tool execution error
            return &CallToolResult{
                Content: []Content{
                    TextContent{
                        Type: "text",
                        Text: fmt.Sprintf("Error: %v", err),
                    },
                },
                IsError: &true,
            }, nil
        }
        
        // Sanitize output
        sanitized := s.sanitizeOutput(result)
        
        return &CallToolResult{
            Content: sanitized,
            IsError: &false,
        }, nil
    }
    `
}
```

### List Changed Notification
```go
// Dynamic tool registration with notifications
type DynamicToolRegistry struct {
    tools    map[string]*Tool
    mu       sync.RWMutex
    notifier NotificationSender
}

func (r *DynamicToolRegistry) RegisterTool(tool *Tool) {
    r.mu.Lock()
    r.tools[tool.Name] = tool
    r.mu.Unlock()
    
    // Send notification if capability is declared
    if r.hasListChangedCapability() {
        r.notifier.Send(&ToolListChangedNotificationParams{
            Meta: map[string]any{
                "timestamp": time.Now().Unix(),
            },
        })
    }
}
```

## 4. Data Types with Full Support

### Tool Annotations for Trust & Safety
```go
// Enhanced tool with security annotations
type SecureTool struct {
    Tool
    TrustLevel     string   // "trusted", "untrusted", "verified"
    SecurityScopes []string // Required permissions
    AuditRequired  bool     // Log all invocations
}

// Validate trust level
func (s *Server) ValidateTrust(tool *Tool) error {
    if tool.Annotations == nil {
        return nil // No annotations to validate
    }
    
    // Only trusted servers can provide trusted annotations
    if !s.isTrustedServer() {
        tool.Annotations = nil // Strip untrusted annotations
    }
    
    return nil
}
```

### Content Type Handlers
```go
// Generate content type specific handlers
func GenerateContentHandlers() map[string]ContentHandler {
    return map[string]ContentHandler{
        "text": func(data any) (Content, error) {
            return TextContent{
                Type: "text",
                Text: fmt.Sprint(data),
            }, nil
        },
        "image": func(data any) (Content, error) {
            bytes, ok := data.([]byte)
            if !ok {
                return nil, errors.New("invalid image data")
            }
            return ImageContent{
                Type:     "image",
                Data:     base64.StdEncoding.EncodeToString(bytes),
                MimeType: detectMimeType(bytes),
            }, nil
        },
        "audio": func(data any) (Content, error) {
            // Similar to image
        },
        "resource": func(data any) (Content, error) {
            res, ok := data.(ResourceData)
            if !ok {
                return nil, errors.New("invalid resource data")
            }
            return EmbeddedResource{
                Type: "resource",
                Resource: TextResourceContents{
                    BaseResourceContents: BaseResourceContents{
                        URI:      res.URI,
                        MimeType: &res.MimeType,
                    },
                    Text: res.Text,
                },
            }, nil
        },
    }
}
```

## 5. Error Handling Implementation

### Protocol vs Tool Errors
```go
// Error handler generator
func GenerateErrorHandling() string {
    return `
    // Protocol error for unknown tool
    if _, exists := s.tools[toolName]; !exists {
        return JSONRPCResponse{
            ID: req.ID,
            Error: &ErrorObject{
                Code:    -32602,
                Message: fmt.Sprintf("Unknown tool: %s", toolName),
            },
        }
    }
    
    // Tool execution error
    result, err := tool.Execute(args)
    if err != nil {
        return JSONRPCResponse{
            ID: req.ID,
            Result: CallToolResult{
                Content: []Content{
                    TextContent{
                        Type: "text",
                        Text: err.Error(),
                    },
                },
                IsError: &true,
            },
        }
    }
    `
}
```

## 6. Security Implementation

### Server-Side Security
```go
// Security middleware generator
func GenerateSecurityMiddleware(tool *Tool) string {
    return `
    // Input validation
    func validate{{ .Name }}Inputs(args map[string]any) error {
        // Check required fields
        for _, required := range {{ .Name }}Schema.Required {
            if _, ok := args[required]; !ok {
                return fmt.Errorf("missing required field: %s", required)
            }
        }
        
        // Validate types and ranges
        return validateSchema(args, {{ .Name }}Schema)
    }
    
    // Access control
    func check{{ .Name }}Access(ctx context.Context) error {
        user := getUserFromContext(ctx)
        if !user.HasPermission("tools.{{ .Name }}") {
            return ErrAccessDenied
        }
        return nil
    }
    
    // Rate limiting
    func rate{{ .Name }}Limit(clientID string) error {
        key := fmt.Sprintf("tool:%s:client:%s", "{{ .Name }}", clientID)
        if !rateLimiter.Allow(key) {
            return ErrRateLimitExceeded
        }
        return nil
    }
    
    // Output sanitization
    func sanitize{{ .Name }}Output(result any) any {
        // Remove sensitive data
        cleaned := removeSensitiveFields(result)
        
        // Escape potentially dangerous content
        return escapeOutput(cleaned)
    }
    `
}
```

### Client-Side Security
```go
// Client security implementation
type SecureToolClient struct {
    *Client
    confirmationUI ConfirmationUI
    logger         AuditLogger
}

func (c *SecureToolClient) CallTool(name string, args map[string]any) (*CallToolResult, error) {
    // Show inputs to user before sending
    if err := c.confirmationUI.ShowInputs(name, args); err != nil {
        return nil, err
    }
    
    // Get user confirmation for sensitive operations
    tool := c.getToolDefinition(name)
    if tool.RequiresConfirmation() {
        if !c.confirmationUI.Confirm(fmt.Sprintf("Execute %s?", name)) {
            return nil, ErrUserCancelled
        }
    }
    
    // Set timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Log invocation
    c.logger.LogToolCall(name, args)
    
    // Call tool
    result, err := c.Client.CallTool(ctx, name, args)
    
    // Log result
    c.logger.LogToolResult(name, result, err)
    
    // Validate result before passing to LLM
    if err := c.validateResult(result); err != nil {
        return nil, err
    }
    
    return result, nil
}
```

## 7. Complete Feature Checklist

### User Interaction Model
- [ ] Model-controlled tool discovery and invocation
- [ ] Human-in-the-loop confirmation system
- [ ] Clear UI indicators for tool exposure
- [ ] Visual indicators for tool invocation
- [ ] Confirmation prompts for operations

### Capabilities
- [ ] Tools capability declaration
- [ ] ListChanged notification support
- [ ] Dynamic tool registration

### Protocol Messages
- [ ] tools/list with pagination
- [ ] tools/call with full parameter support
- [ ] notifications/tools/list_changed
- [ ] Proper error responses

### Data Types
- [ ] Complete Tool definition with annotations
- [ ] All content types (text, image, audio, resource)
- [ ] Error indication in results
- [ ] Metadata support

### Security
- [ ] Input validation against schemas
- [ ] Access control implementation
- [ ] Rate limiting
- [ ] Output sanitization
- [ ] Audit logging
- [ ] Timeout handling
- [ ] User confirmation for sensitive operations
- [ ] Trust level validation for annotations

### Trust & Safety
- [ ] Annotation validation based on server trust
- [ ] Input display before execution
- [ ] Destructive operation warnings
- [ ] Comprehensive error handling
- [ ] Secure defaults

This comprehensive coverage ensures all MCP tool concepts are properly implemented in our reflection and generation tools.