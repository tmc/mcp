# Type System Architecture

## Overview

The MCP Go implementation features a comprehensive type system that ensures protocol compliance, type safety, and extensibility. The type system is designed around JSON-RPC 2.0 and MCP protocol specifications while providing idiomatic Go interfaces.

## Type Hierarchy

```
MCP Types
├── Protocol Layer
│   ├── JSON-RPC Types
│   │   ├── JSONRPCRequest
│   │   ├── JSONRPCResponse  
│   │   ├── JSONRPCNotification
│   │   └── JSONRPCError
│   └── MCP Protocol Types
│       ├── InitializeRequest/Result
│       ├── Capabilities Types
│       └── Implementation
├── Content Layer
│   ├── Content (interface)
│   ├── TextContent
│   ├── ImageContent
│   └── ResourceContents (interface)
│       ├── TextResourceContents
│       └── BlobResourceContents
├── Resource Layer
│   ├── Tool & CallTool Types
│   ├── Prompt & GetPrompt Types
│   ├── Resource & ReadResource Types
│   └── ResourceTemplate Types
└── Handler Layer
    ├── ToolHandlerFunc
    ├── GetPromptHandlerFunc
    ├── ReadResourceHandlerFunc
    └── NotificationHandler
```

## Core Type Patterns

### 1. Interface-Based Polymorphism

The type system uses Go interfaces to enable polymorphic behavior while maintaining type safety:

```go
// Content interface allows multiple content types
type Content interface {
    content()
}

// ResourceContents interface supports text and binary resources
type ResourceContents interface {
    resourceContents()
}
```

**Benefits**:
- Type-safe polymorphism
- Extensible content types
- Clean JSON serialization
- Future protocol compatibility

### 2. Request/Response Pairs

Each MCP operation follows a consistent request/response pattern:

```go
// Tool operations
type CallToolRequest struct {
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments,omitempty"`
}

type CallToolResult struct {
    Content []any `json:"content"`
    IsError bool  `json:"isError,omitempty"`
    Meta    any   `json:"_meta,omitempty"`
}
```

**Design Principles**:
- Consistent naming convention (`*Request`/`*Result`)
- JSON-first serialization
- Optional fields with `omitempty`
- Extensible with metadata fields

### 3. Capability Declaration

Capabilities use embedded structs for clean organization:

```go
type ServerCapabilities struct {
    Experimental map[string]any `json:"experimental,omitempty"`
    Tools        *struct {
        ListChanged bool `json:"listChanged,omitempty"`
    } `json:"tools,omitempty"`
    Resources *struct {
        Subscribe   bool `json:"subscribe,omitempty"`
        ListChanged bool `json:"listChanged,omitempty"`
    } `json:"resources,omitempty"`
    Prompts *struct {
        ListChanged bool `json:"listChanged,omitempty"`
    } `json:"prompts,omitempty"`
}
```

**Advantages**:
- Null-safe capability checking
- Clear capability grouping
- Extensible experimental features
- Protocol-compliant JSON structure

## JSON Serialization Strategy

### Custom Marshaling

Several types implement custom JSON marshaling for protocol compliance:

```go
// ReadResourceResult needs custom unmarshaling for polymorphic contents
func (r *ReadResourceResult) UnmarshalJSON(data []byte) error {
    type Alias ReadResourceResult
    aux := &struct {
        Contents []json.RawMessage `json:"contents"`
        *Alias
    }{
        Alias: (*Alias)(r),
    }
    
    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }
    
    // Unmarshal each content item based on its type
    for _, rawContent := range aux.Contents {
        var content ResourceContents
        // Type detection and unmarshaling logic
        r.Contents = append(r.Contents, content)
    }
    
    return nil
}
```

### Type Detection Patterns

```go
func detectContentType(raw json.RawMessage) (Content, error) {
    var typeCheck struct {
        Type string `json:"type"`
    }
    
    if err := json.Unmarshal(raw, &typeCheck); err != nil {
        return nil, err
    }
    
    switch typeCheck.Type {
    case "text":
        var content TextContent
        err := json.Unmarshal(raw, &content)
        return content, err
    case "image":
        var content ImageContent
        err := json.Unmarshal(raw, &content)
        return content, err
    default:
        return nil, fmt.Errorf("unknown content type: %s", typeCheck.Type)
    }
}
```

## Handler Function Types

### Type-Safe Handler Signatures

```go
// Tool handler with clear input/output types
type ToolHandlerFunc func(ctx context.Context, request CallToolRequest) (*CallToolResult, error)

// Resource handler returns slice for multiple contents
type ReadResourceHandlerFunc func(ctx context.Context, request ReadResourceRequest) ([]ResourceContents, error)

// Prompt handler for template processing
type GetPromptHandlerFunc func(ctx context.Context, request GetPromptRequest) (*GetPromptResult, error)
```

**Design Benefits**:
- Clear function signatures
- Context support for cancellation
- Consistent error handling
- Type safety at compile time

### Generic Handler Pattern

For advanced use cases, the package provides generic handlers:

```go
func RegisterTypedTool[Input any, Output any](
    server *Server,
    name string,
    description string,
    handler func(context.Context, Input) (Output, error),
) error {
    // Automatic JSON schema generation
    // Type-safe argument parsing
    // Structured output formatting
}
```

## Error Type Hierarchy

### Standard Error Types

```go
var (
    ErrInvalidParams   = errors.New("mcp: invalid parameters")
    ErrNotFound        = errors.New("mcp: not found")
    ErrUnsupported     = errors.New("mcp: operation or capability not supported")
    ErrTransportClosed = errors.New("mcp: transport closed")
)
```

### JSON-RPC Error Structure

```go
type JSONRPCError struct {
    Code    int             `json:"code"`
    Message string          `json:"message"`
    Data    json.RawMessage `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
    if e.Data != nil {
        return e.Message + ": " + string(e.Data)
    }
    return e.Message
}
```

**Error Code Standards**:
- `-32700`: Parse error
- `-32600`: Invalid request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error

## Content Type System

### Base Content Interface

```go
type Content interface {
    content() // Marker method for type safety
}
```

### Text Content

```go
type TextContent struct {
    Type string `json:"type"`           // Always "text"
    Text string `json:"text"`           // The text content
}

func (TextContent) content() {}
```

### Image Content

```go
type ImageContent struct {
    Type     string `json:"type"`       // Always "image"
    Data     []byte `json:"data,omitempty"`
    MimeType string `json:"mimeType,omitempty"`
}

func (ImageContent) content() {}
```

### Resource Contents

```go
type ResourceContents interface {
    resourceContents() // Marker method
}

type TextResourceContents struct {
    URI      string `json:"uri"`
    MimeType string `json:"mimeType,omitempty"`
    Text     string `json:"text"`
}

type BlobResourceContents struct {
    URI      string `json:"uri"`
    MimeType string `json:"mimeType,omitempty"`
    Blob     string `json:"blob"` // base64 encoded
}
```

## Protocol Version Management

### Version Constants

```go
const (
    LATEST_PROTOCOL_VERSION = "2025-03-26"
    JSONRPC_VERSION         = "2.0"
)
```

### Version Compatibility

```go
type InitializeRequest struct {
    ProtocolVersion string             `json:"protocolVersion"`
    ClientInfo      Implementation     `json:"clientInfo"`
    Capabilities    ClientCapabilities `json:"capabilities"`
}

func (r *InitializeRequest) IsVersionSupported() bool {
    return r.ProtocolVersion == LATEST_PROTOCOL_VERSION
}
```

## Schema Generation

### Automatic Schema Creation

```go
func createJSONSchema[T any]() (json.RawMessage, error) {
    var example T
    typeName := fmt.Sprintf("%T", example)
    
    // Handle primitive types
    switch typeName {
    case "string":
        return json.Marshal(map[string]any{"type": "string"})
    case "int", "int32", "int64", "float32", "float64":
        return json.Marshal(map[string]any{"type": "number"})
    case "bool":
        return json.Marshal(map[string]any{"type": "boolean"})
    }
    
    // Generate object schema from struct
    return generateObjectSchema(example)
}
```

## Type Safety Features

### Compile-Time Guarantees

1. **Interface Compliance**: Types must implement required interfaces
2. **Method Signatures**: Handler functions have enforced signatures
3. **Field Types**: JSON tags ensure correct serialization
4. **Protocol Compliance**: Types match the MCP specification

### Runtime Validation

```go
func validateToolRequest(req CallToolRequest) error {
    if req.Name == "" {
        return ErrInvalidParams
    }
    // Additional validation logic
    return nil
}
```

## Extensibility Patterns

### Custom Content Types

```go
type CustomContent struct {
    Type string `json:"type"`
    Data any    `json:"data"`
}

func (CustomContent) content() {}

// Register with content system
func init() {
    registerContentType("custom", func() Content {
        return &CustomContent{}
    })
}
```

### Experimental Features

```go
type ServerCapabilities struct {
    Experimental map[string]any `json:"experimental,omitempty"`
    // ... standard fields
}

// Add experimental capability
capabilities.Experimental["customFeature"] = map[string]any{
    "version": "1.0",
    "enabled": true,
}
```

## Memory Management

### Efficient Patterns

1. **Slice Reuse**: Pre-allocate slices for known sizes
2. **String Interning**: Reuse common strings (method names, types)
3. **Pool Usage**: Object pools for frequently created types
4. **Lazy Loading**: Initialize expensive fields only when needed

### Resource Cleanup

```go
type ResourceHandle struct {
    resource Resource
    cleanup  func()
}

func (h *ResourceHandle) Close() error {
    if h.cleanup != nil {
        h.cleanup()
        h.cleanup = nil
    }
    return nil
}
```

## Testing Support

### Mock Types

```go
type MockContent struct {
    TypeValue string
    Data      any
}

func (m MockContent) content() {}
func (m MockContent) MarshalJSON() ([]byte, error) {
    return json.Marshal(map[string]any{
        "type": m.TypeValue,
        "data": m.Data,
    })
}
```

### Test Builders

```go
func NewTestTool(name string) Tool {
    return Tool{
        Name:        name,
        Description: "Test tool: " + name,
        InputSchema: json.RawMessage(`{"type": "object"}`),
    }
}

func NewTestCallRequest(toolName string, args any) CallToolRequest {
    argsJSON, _ := json.Marshal(args)
    return CallToolRequest{
        Name:      toolName,
        Arguments: argsJSON,
    }
}
```

This type system provides a robust foundation for MCP implementations with strong typing, protocol compliance, and extensibility while maintaining performance and ease of use.