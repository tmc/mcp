# Enhanced Reflection and Generation for MCP Spec Coverage

## 🔍 Reflection Enhancements

### 1. **reflect-mcp**: Complete MCP Type Mapping
```go
// exp/reflect/mcp/types.go
package mcp

// Enhanced type reflection for complete MCP coverage
type TypeReflector struct {
    resolver *types.Package
    cache    map[types.Type]*MCPType
}

type MCPType struct {
    GoType      types.Type
    Schema      *jsonschema.Schema
    Validations []Validation
    Examples    []Example
    
    // MCP-specific metadata
    ContentType string          // text, image, resource, audio
    MIME        string          // for content types
    Encoding    string          // base64, utf-8, etc.
    Constraints []Constraint    // MCP constraints
}

// Core reflection methods
func (r *TypeReflector) ReflectStruct(t *types.Struct) (*MCPType, error)
func (r *TypeReflector) ReflectInterface(t *types.Interface) (*MCPType, error)
func (r *TypeReflector) ReflectFunc(t *types.Func) (*MCPTool, error)
```

### 2. **reflect-content**: Content Type Detection
```go
// exp/reflect/content/detector.go
package content

type ContentDetector struct {
    mimeTypes map[string]string
}

// Detect content type from Go type and tags
func (d *ContentDetector) Detect(t types.Type) (*ContentType, error) {
    // Check struct tags for hints
    if tag := getTag(t, "mcp"); tag != "" {
        return d.ParseTag(tag)
    }
    
    // Infer from type name and structure
    switch {
    case isImageType(t):
        return &ContentType{Type: "image", MIME: detectImageMIME(t)}, nil
    case isAudioType(t):
        return &ContentType{Type: "audio", MIME: detectAudioMIME(t)}, nil
    case isResourceType(t):
        return &ContentType{Type: "resource", URI: extractURI(t)}, nil
    default:
        return &ContentType{Type: "text", MIME: "text/plain"}, nil
    }
}
```

### 3. **reflect-capability**: Capability Detection
```go
// exp/reflect/capability/detector.go
package capability

type CapabilityDetector struct {
    spec *mcp.Specification
}

// Detect which MCP capabilities a type/package supports
func (d *CapabilityDetector) Detect(pkg *types.Package) (*Capabilities, error) {
    caps := &Capabilities{}
    
    // Check for tool support
    if hasToolHandlers(pkg) {
        caps.Tools = true
        caps.ToolCapabilities = detectToolCapabilities(pkg)
    }
    
    // Check for resource support
    if hasResourceHandlers(pkg) {
        caps.Resources = true
        caps.ResourceCapabilities = detectResourceCapabilities(pkg)
    }
    
    // Check for sampling support
    if hasSamplingMethods(pkg) {
        caps.Sampling = true
        caps.SamplingModels = detectSamplingModels(pkg)
    }
    
    // Check for notification support
    if hasNotificationHandlers(pkg) {
        caps.Notifications = true
        caps.NotificationTypes = detectNotificationTypes(pkg)
    }
    
    return caps, nil
}
```

### 4. **reflect-errors**: Error Type Mapping
```go
// exp/reflect/errors/mapper.go
package errors

type ErrorMapper struct {
    errorTypes map[types.Type]*MCPError
}

// Map Go errors to MCP error types
func (m *ErrorMapper) MapError(err error) *mcp.Error {
    t := reflect.TypeOf(err)
    
    // Check for standard MCP error types
    switch {
    case implements(t, "InvalidRequest"):
        return &mcp.Error{Code: -32600, Message: err.Error()}
    case implements(t, "MethodNotFound"):
        return &mcp.Error{Code: -32601, Message: err.Error()}
    case implements(t, "InvalidParams"):
        return &mcp.Error{Code: -32602, Message: err.Error()}
    case implements(t, "InternalError"):
        return &mcp.Error{Code: -32603, Message: err.Error()}
    default:
        // Custom error mapping
        return m.mapCustomError(err)
    }
}
```

## 🏗️ Generation Enhancements

### 5. **gen-schema**: Advanced Schema Generation
```go
// exp/gen/schema/generator.go
package schema

type SchemaGenerator struct {
    options SchemaOptions
}

type SchemaOptions struct {
    IncludeExamples    bool
    IncludeValidation  bool
    StrictNullHandling bool
    GenerateRefs       bool
}

// Generate complete JSON Schema from Go type
func (g *SchemaGenerator) Generate(t types.Type) (*jsonschema.Schema, error) {
    schema := &jsonschema.Schema{
        Type: mapGoTypeToJSON(t),
    }
    
    // Add MCP-specific extensions
    if isContentType(t) {
        schema.Extensions = map[string]interface{}{
            "x-mcp-content-type": detectContentType(t),
            "x-mcp-encoding":     detectEncoding(t),
        }
    }
    
    // Handle nullable types properly
    if g.options.StrictNullHandling {
        schema.AnyOf = handleNullable(t)
    }
    
    // Add validation rules
    if g.options.IncludeValidation {
        schema.Validations = generateValidations(t)
    }
    
    return schema, nil
}
```

### 6. **gen-handlers**: Handler Generation with Full Spec
```go
// exp/gen/handlers/generator.go
package handlers

type HandlerGenerator struct {
    spec     *mcp.Specification
    template string
}

// Generate handler that covers all MCP features
func (g *HandlerGenerator) GenerateHandler(tool *mcp.Tool) (string, error) {
    return g.executeTemplate(handlerTemplate, map[string]interface{}{
        "tool":         tool,
        "includeAsync": g.spec.SupportsAsync(),
        "includeTrace": g.spec.SupportsTracing(),
        "errorTypes":   g.spec.ErrorTypes,
    })
}

const handlerTemplate = `
func (s *Server) Handle{{.tool.Name}}(ctx context.Context, req *{{.tool.Name}}Request) (*{{.tool.Name}}Response, error) {
    {{if .includeTrace}}
    span := trace.Start(ctx, "handle.{{.tool.Name}}")
    defer span.End()
    {{end}}
    
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, mcp.NewInvalidParamsError(err)
    }
    
    {{if .includeAsync}}
    // Check if async requested
    if req.Async {
        return s.handleAsync(ctx, req)
    }
    {{end}}
    
    // Implementation
    result, err := s.process{{.tool.Name}}(ctx, req)
    if err != nil {
        return nil, mapError(err)
    }
    
    return &{{.tool.Name}}Response{
        Content: []mcp.Content{
            {
                Type: result.ContentType(),
                {{if eq result.ContentType "text"}}
                Text: result.Text(),
                {{else if eq result.ContentType "image"}}
                Data: result.ImageData(),
                MIMEType: result.MIMEType(),
                {{else if eq result.ContentType "resource"}}
                Resource: result.Resource(),
                {{end}}
            },
        },
    }, nil
}
`
```

### 7. **gen-transport**: Transport Adapter Generation
```go
// exp/gen/transport/generator.go
package transport

type TransportGenerator struct {
    transports []string // stdio, sse, websocket
}

// Generate transport adapters for all supported types
func (g *TransportGenerator) Generate(server types.Type) (map[string]string, error) {
    adapters := make(map[string]string)
    
    for _, transport := range g.transports {
        switch transport {
        case "stdio":
            adapters["stdio"] = g.generateStdioAdapter(server)
        case "sse":
            adapters["sse"] = g.generateSSEAdapter(server)
        case "websocket":
            adapters["websocket"] = g.generateWebSocketAdapter(server)
        }
    }
    
    return adapters, nil
}

// Generate stdio adapter with proper framing
func (g *TransportGenerator) generateStdioAdapter(server types.Type) string {
    return `
type StdioTransport struct {
    server *{{.ServerType}}
    reader *bufio.Reader
    writer *bufio.Writer
}

func (t *StdioTransport) Run() error {
    t.reader = bufio.NewReader(os.Stdin)
    t.writer = bufio.NewWriter(os.Stdout)
    
    for {
        // Read content length header
        header, err := t.reader.ReadString('\n')
        if err != nil {
            return err
        }
        
        length, err := parseContentLength(header)
        if err != nil {
            continue
        }
        
        // Read message body
        body := make([]byte, length)
        if _, err := io.ReadFull(t.reader, body); err != nil {
            return err
        }
        
        // Process message
        response, err := t.server.HandleMessage(body)
        if err != nil {
            response = errorResponse(err)
        }
        
        // Write response
        t.writeResponse(response)
    }
}
`
}
```

### 8. **gen-client**: Type-Safe Client Generation
```go
// exp/gen/client/generator.go
package client

type ClientGenerator struct {
    spec    *mcp.Specification
    options ClientOptions
}

// Generate type-safe client with all MCP features
func (g *ClientGenerator) Generate() (string, error) {
    return g.executeTemplate(clientTemplate, map[string]interface{}{
        "tools":         g.spec.Tools,
        "resources":     g.spec.Resources,
        "notifications": g.spec.Notifications,
        "async":         g.options.IncludeAsync,
        "retry":         g.options.IncludeRetry,
    })
}

const clientTemplate = `
type Client struct {
    transport mcp.Transport
    {{if .async}}
    pending map[string]chan *Response
    {{end}}
    {{if .retry}}
    retryPolicy RetryPolicy
    {{end}}
}

{{range .tools}}
func (c *Client) {{.Name}}(ctx context.Context, params *{{.Name}}Params) (*{{.Name}}Result, error) {
    req := &mcp.Request{
        Method: "tool/call",
        Params: mcp.CallToolParams{
            Name:      "{{.Name}}",
            Arguments: params,
        },
    }
    
    {{if $.retry}}
    resp, err := c.sendWithRetry(ctx, req)
    {{else}}
    resp, err := c.send(ctx, req)
    {{end}}
    
    if err != nil {
        return nil, err
    }
    
    var result {{.Name}}Result
    if err := json.Unmarshal(resp.Result, &result); err != nil {
        return nil, err
    }
    
    return &result, nil
}
{{end}}
`
```

### 9. **gen-validation**: Validation Code Generation
```go
// exp/gen/validation/generator.go
package validation

type ValidationGenerator struct {
    schema *jsonschema.Schema
}

// Generate validation code from schema
func (g *ValidationGenerator) Generate() (string, error) {
    var validations []string
    
    // Type validation
    validations = append(validations, g.generateTypeValidation())
    
    // Property validation
    if g.schema.Properties != nil {
        validations = append(validations, g.generatePropertyValidation())
    }
    
    // Pattern validation
    if g.schema.Pattern != "" {
        validations = append(validations, g.generatePatternValidation())
    }
    
    // Custom MCP validations
    if g.schema.Extensions["x-mcp-validations"] != nil {
        validations = append(validations, g.generateMCPValidations())
    }
    
    return fmt.Sprintf(`
func (v *Validator) Validate(value interface{}) error {
    %s
    return nil
}`, strings.Join(validations, "\n    ")), nil
}
```

### 10. **gen-notification**: Notification System Generation
```go
// exp/gen/notification/generator.go
package notification

type NotificationGenerator struct {
    notifications []mcp.NotificationType
}

// Generate notification handlers and dispatchers
func (g *NotificationGenerator) Generate() (string, error) {
    return g.executeTemplate(notificationTemplate, g.notifications)
}

const notificationTemplate = `
type NotificationDispatcher struct {
    handlers map[string][]NotificationHandler
}

{{range .}}
func (d *NotificationDispatcher) On{{.Name}}(handler func(*{{.Name}}Notification)) {
    d.handlers["{{.Name}}"] = append(d.handlers["{{.Name}}"], func(n Notification) {
        handler(n.(*{{.Name}}Notification))
    })
}

func (d *NotificationDispatcher) Emit{{.Name}}(notification *{{.Name}}Notification) {
    for _, handler := range d.handlers["{{.Name}}"] {
        go handler(notification)
    }
}
{{end}}
`
```

## 🎯 Complete MCP Coverage Examples

### Resource Support Generation
```go
// Generate complete resource support
func GenerateResourceSupport(pkg *types.Package) error {
    detector := capability.NewDetector()
    resources := detector.DetectResources(pkg)
    
    generator := gen.NewResourceGenerator()
    code := generator.Generate(resources)
    
    return writeFile("resources_gen.go", code)
}
```

### Sampling Implementation
```go
// Generate sampling support for LLM integration
func GenerateSamplingSupport(models []string) error {
    generator := gen.NewSamplingGenerator()
    code := generator.Generate(models)
    
    return writeFile("sampling_gen.go", code)
}
```

### Complete Server Generation
```go
// Generate server with all MCP capabilities
func GenerateCompleteServer(pkg *types.Package) error {
    reflector := reflect.NewMCPReflector()
    spec := reflector.ExtractSpec(pkg)
    
    generators := []Generator{
        gen.NewSchemaGenerator(),
        gen.NewHandlerGenerator(),
        gen.NewTransportGenerator(),
        gen.NewValidationGenerator(),
        gen.NewNotificationGenerator(),
    }
    
    for _, g := range generators {
        code, err := g.Generate(spec)
        if err != nil {
            return err
        }
        
        filename := fmt.Sprintf("%s_gen.go", g.Name())
        if err := writeFile(filename, code); err != nil {
            return err
        }
    }
    
    return nil
}
```

These enhanced reflection and generation tools ensure complete coverage of the MCP specification, including all content types, capabilities, error handling, and transport mechanisms. They enable automatic generation of fully-compliant MCP servers from existing Go code.