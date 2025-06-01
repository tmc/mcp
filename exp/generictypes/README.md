# Generic Types for Model Context Protocol

This experimental package explores how Go generics could improve the Model Context Protocol API design. It demonstrates several patterns that could reduce code duplication and improve type safety.

## Key Improvements

### 1. Generic List Results

Instead of having multiple list result types:
```go
// Current approach - repetitive
type ListResourcesResult struct { ... }
type ListPromptsResult struct { ... }
type ListToolsResult struct { ... }
```

We can use a single generic type:
```go
// Generic approach
type ListResult[T any] struct {
    Meta       map[string]any `json:"_meta,omitempty"`
    Items      []T            `json:"items"`
    NextCursor *Cursor        `json:"nextCursor,omitempty"`
}
```

### 2. Generic Request/Response Wrappers

Simplify request and response types with consistent metadata handling:
```go
type Request[T any] struct {
    Meta   *RequestMeta `json:"_meta,omitempty"`
    Params T            `json:",inline"`
}

type Result[T any] struct {
    Meta map[string]any `json:"_meta,omitempty"`
    Data T              `json:",inline"`
}
```

### 3. Type-Safe Union Unmarshaling

Replace repetitive switch statements with a generic union unmarshaler:
```go
contentUnion := NewTypedUnion[Content]("type").
    Register("text", unmarshalAs[TextContent]).
    Register("image", unmarshalAs[ImageContent]).
    Register("audio", unmarshalAs[AudioContent])
```

### 4. Optional Type for Nullable Fields

Replace pointer fields with an explicit Optional type:
```go
// Before
type Resource struct {
    Description *string `json:"description,omitempty"`
    MimeType    *string `json:"mimeType,omitempty"`
}

// After
type Resource struct {
    Description Optional[string] `json:"description,omitempty"`
    MimeType    Optional[string] `json:"mimeType,omitempty"`
}
```

### 5. Generic Builder Pattern

Provide a fluent interface for constructing complex types:
```go
resource, _ := NewResourceBuilder("file:///example.txt", "Example").
    WithDescription("An example file").
    WithMimeType("text/plain").
    WithSize(1024).
    Build()
```

## Benefits

1. **Reduced Code Duplication**: Eliminate repetitive type definitions and unmarshaling logic
2. **Improved Type Safety**: Stronger compile-time guarantees
3. **Better Composability**: Generic functions can work with multiple types
4. **Cleaner API**: More consistent patterns across the codebase
5. **Easier Maintenance**: Changes to common patterns only need to be made once

## Migration Path

The generic types can coexist with existing types, allowing for gradual migration:

```go
// Convert existing types to generic equivalents
func ConvertResourceList(old ListResourcesResult) ListResult[Resource] {
    return ListResult[Resource]{
        Meta:       old.Meta,
        Items:      old.Resources,
        NextCursor: old.NextCursor,
    }
}
```

## Performance Considerations

Generic implementations should have minimal runtime overhead compared to the current approach. The main benefits are:
- Reduced binary size due to less code duplication
- Better inlining opportunities
- No runtime type assertions needed in many cases

## Future Enhancements

1. **Generic Validators**: Type-safe validation rules
2. **Generic Serializers**: Custom marshaling/unmarshaling logic
3. **Generic Middleware**: Request/response interceptors
4. **Generic Caching**: Type-safe caching layers

This experimental package demonstrates the potential for using Go generics to create a more maintainable and type-safe MCP implementation.