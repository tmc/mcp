# MCP Spec Coverage for Tools

## Features Our Reflection/Generation Tools Must Cover

### 1. Output Schema Support (Draft Spec)
```go
// mcp-goast should detect and generate output schemas
func DetectOutputSchema(fn *types.Func) *ToolSchema {
    // Analyze function return type
    // Generate JSON Schema for structured output
    // Handle both structured and unstructured formats
}

// Generate dual-format results for compatibility
func GenerateToolResult(tool *Tool) string {
    if tool.OutputSchema != nil {
        return `
        // Structured result with compatibility content
        return draft.NewStructuredToolResult(
            structuredData,
            compatibilityContent,
            nil, // isError
            nil, // meta
        )
        `
    } else {
        return `
        // Traditional unstructured result
        return &CallToolResult{
            Content: []Content{...},
        }
        `
    }
}
```

### 2. Tool Annotations Detection
```go
// Detect annotations from comments or tags
type AnnotationDetector struct{}

func (d *AnnotationDetector) Detect(fn *types.Func) *ToolAnnotations {
    // Check for comment directives:
    // +mcp:readonly
    // +mcp:destructive
    // +mcp:idempotent
    // +mcp:openworld
    
    // Or struct tags on handler receiver:
    // `mcp:"readonly,idempotent"`
}
```

### 3. Comprehensive Content Type Support
```go
// Enhanced output detection for all content types
func DetectContentType(ret types.Type) ContentType {
    switch {
    case isImageType(ret):
        return ContentTypeImage
    case isAudioType(ret):
        return ContentTypeAudio
    case isResourceType(ret):
        return ContentTypeResource
    default:
        return ContentTypeText
    }
}

// Generate proper content based on type
func GenerateContent(value interface{}, contentType ContentType) Content {
    switch contentType {
    case ContentTypeImage:
        return ImageContent{
            Type:     "image",
            Data:     base64.Encode(value),
            MimeType: detectMimeType(value),
        }
    // ... other types
    }
}
```

### 4. Error Handling Patterns
```go
// Detect error patterns in tool implementations
func GenerateErrorHandling(tool *Tool) string {
    return `
    if err != nil {
        errorContent := []Content{
            TextContent{
                Type: "text",
                Text: fmt.Sprintf("Error: %v", err),
            },
        }
        
        return &CallToolResult{
            Content: errorContent,
            IsError: &true,
        }
    }
    `
}
```

### 5. Pagination Support
```go
// Generate paginated list handlers
func GeneratePaginatedTool(tool *Tool) string {
    return `
    func (s *Server) List{{ .Name }}(params ListParams) (*ListResult, error) {
        results, nextCursor := s.paginate(params.Cursor, params.Limit)
        
        return &ListResult{
            Items:      results,
            NextCursor: nextCursor,
        }, nil
    }
    `
}
```

### 6. Progress Token Support
```go
// Generate progress-aware handlers
func GenerateProgressHandler(tool *Tool) string {
    return `
    func (s *Server) {{ .Name }}(ctx context.Context, params Params) (*Result, error) {
        if params.Meta != nil && params.Meta.ProgressToken != nil {
            s.startProgress(params.Meta.ProgressToken)
            defer s.completeProgress(params.Meta.ProgressToken)
        }
        
        // Tool implementation with progress updates
        for i := 0; i < total; i++ {
            s.updateProgress(params.Meta.ProgressToken, i, total)
            // ... work ...
        }
    }
    `
}
```

### 7. Dynamic Tool Registration
```go
// Support dynamic tool lists
type DynamicToolServer struct {
    tools map[string]*Tool
    mu    sync.RWMutex
}

func (s *DynamicToolServer) RegisterTool(tool *Tool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.tools[tool.Name] = tool
    
    // Send tool list changed notification
    s.notify(&ToolListChangedNotificationParams{})
}
```

### 8. Schema Validation
```go
// Validate inputs against schema
func GenerateInputValidation(tool *Tool) string {
    return `
    func validate{{ .Name }}Input(params map[string]any) error {
        schema := {{ .InputSchema }}
        return validateAgainstSchema(params, schema)
    }
    `
}
```

## Testing Coverage

### 1. Round-trip Testing
```go
func TestToolRoundTrip(t *testing.T) {
    // Original function
    originalFunc := getTimeFunc()
    
    // Generate tool definition
    tool := reflectToTool(originalFunc)
    
    // Generate handler from tool
    handler := generateHandler(tool)
    
    // Call handler
    result := handler(params)
    
    // Verify result matches original
    assert.Equal(t, originalFunc(), result)
}
```

### 2. Schema Compliance Testing
```go
func TestSchemaCompliance(t *testing.T) {
    tool := generateTool()
    
    // Test with valid input
    validResult := callTool(tool, validInput)
    assert.NoError(t, validateSchema(validResult, tool.OutputSchema))
    
    // Test with invalid input
    _, err := callTool(tool, invalidInput)
    assert.Error(t, err)
}
```

### 3. Content Type Testing
```go
func TestContentTypes(t *testing.T) {
    tests := []struct {
        name        string
        tool        *Tool
        wantContent ContentType
    }{
        {"text tool", textTool, ContentTypeText},
        {"image tool", imageTool, ContentTypeImage},
        {"audio tool", audioTool, ContentTypeAudio},
        {"resource tool", resourceTool, ContentTypeResource},
    }
    
    for _, tt := range tests {
        result := callTool(tt.tool, defaultParams)
        assert.Equal(t, tt.wantContent, result.Content[0].Type)
    }
}
```

## Integration Checklist

- [ ] Output schema detection from return types
- [ ] Structured content generation for draft spec
- [ ] Compatibility content for backwards compatibility
- [ ] Tool annotation detection from code/comments
- [ ] All content types (text, image, audio, resource)
- [ ] Error handling with isError flag
- [ ] Pagination support with cursors
- [ ] Progress token pass-through
- [ ] Dynamic tool registration
- [ ] Schema validation for inputs/outputs
- [ ] Metadata support in all messages
- [ ] Tool list change notifications

This ensures our reflection and generation tools fully cover the MCP specification for tools and tool calls, including both the stable and draft specifications.