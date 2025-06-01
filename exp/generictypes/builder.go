package generictypes

import (
	"encoding/json"
	"github.com/tmc/mcp/modelcontextprotocol"
)

// Builder provides a fluent interface for constructing complex types.
type Builder[T any] struct {
	value T
	err   error
}

// NewBuilder creates a new Builder instance.
func NewBuilder[T any]() *Builder[T] {
	return &Builder[T]{}
}

// NewBuilderFrom creates a Builder with an initial value.
func NewBuilderFrom[T any](initial T) *Builder[T] {
	return &Builder[T]{value: initial}
}

// With applies a modification function to the value being built.
func (b *Builder[T]) With(fn func(*T) error) *Builder[T] {
	if b.err != nil {
		return b
	}
	b.err = fn(&b.value)
	return b
}

// WithValue applies a simple modification function.
func (b *Builder[T]) WithValue(fn func(*T)) *Builder[T] {
	if b.err != nil {
		return b
	}
	fn(&b.value)
	return b
}

// If conditionally applies a modification.
func (b *Builder[T]) If(condition bool, fn func(*T)) *Builder[T] {
	if b.err != nil || !condition {
		return b
	}
	fn(&b.value)
	return b
}

// IfError conditionally applies a modification if an error occurred.
func (b *Builder[T]) IfError(fn func(*T, error)) *Builder[T] {
	if b.err != nil {
		fn(&b.value, b.err)
	}
	return b
}

// Build returns the constructed value and any error.
func (b *Builder[T]) Build() (T, error) {
	return b.value, b.err
}

// MustBuild returns the value or panics if there's an error.
func (b *Builder[T]) MustBuild() T {
	if b.err != nil {
		panic(b.err)
	}
	return b.value
}

// --- Specific builders for common MCP types ---

// ResourceBuilder helps construct Resource types.
type ResourceBuilder struct {
	*Builder[modelcontextprotocol.Resource]
}

// NewResourceBuilder creates a new ResourceBuilder.
func NewResourceBuilder(uri, name string) *ResourceBuilder {
	return &ResourceBuilder{
		Builder: NewBuilderFrom(modelcontextprotocol.Resource{
			URI:  uri,
			Name: name,
		}),
	}
}

// WithDescription sets the description.
func (rb *ResourceBuilder) WithDescription(desc string) *ResourceBuilder {
	rb.WithValue(func(r *modelcontextprotocol.Resource) {
		r.Description = &desc
	})
	return rb
}

// WithMimeType sets the MIME type.
func (rb *ResourceBuilder) WithMimeType(mimeType string) *ResourceBuilder {
	rb.WithValue(func(r *modelcontextprotocol.Resource) {
		r.MimeType = &mimeType
	})
	return rb
}

// WithSize sets the size.
func (rb *ResourceBuilder) WithSize(size int64) *ResourceBuilder {
	rb.WithValue(func(r *modelcontextprotocol.Resource) {
		r.Size = &size
	})
	return rb
}

// WithAnnotations sets the annotations.
func (rb *ResourceBuilder) WithAnnotations(ann modelcontextprotocol.Annotations) *ResourceBuilder {
	rb.WithValue(func(r *modelcontextprotocol.Resource) {
		r.Annotations = &ann
	})
	return rb
}

// ToolBuilder helps construct Tool types.
type ToolBuilder struct {
	*Builder[modelcontextprotocol.Tool]
}

// NewToolBuilder creates a new ToolBuilder.
func NewToolBuilder(name string, inputSchema modelcontextprotocol.ToolSchema) *ToolBuilder {
	return &ToolBuilder{
		Builder: NewBuilderFrom(modelcontextprotocol.Tool{
			Name:        name,
			InputSchema: inputSchema,
		}),
	}
}

// WithDescription sets the description.
func (tb *ToolBuilder) WithDescription(desc string) *ToolBuilder {
	tb.WithValue(func(t *modelcontextprotocol.Tool) {
		t.Description = &desc
	})
	return tb
}

// WithAnnotations sets the annotations.
func (tb *ToolBuilder) WithAnnotations(ann modelcontextprotocol.ToolAnnotations) *ToolBuilder {
	tb.WithValue(func(t *modelcontextprotocol.Tool) {
		t.Annotations = &ann
	})
	return tb
}

// WithReadOnlyHint sets the read-only hint.
func (tb *ToolBuilder) WithReadOnlyHint(hint bool) *ToolBuilder {
	tb.WithValue(func(t *modelcontextprotocol.Tool) {
		if t.Annotations == nil {
			t.Annotations = &modelcontextprotocol.ToolAnnotations{}
		}
		t.Annotations.ReadOnlyHint = &hint
	})
	return tb
}

// --- Generic request builder ---

// RequestBuilder helps construct Request types.
type RequestBuilder[T any] struct {
	*Builder[Request[T]]
}

// NewRequestBuilder creates a new RequestBuilder.
func NewRequestBuilder[T any](params T) *RequestBuilder[T] {
	return &RequestBuilder[T]{
		Builder: NewBuilderFrom(Request[T]{Params: params}),
	}
}

// WithProgressToken sets the progress token.
func (rb *RequestBuilder[T]) WithProgressToken(token modelcontextprotocol.ProgressToken) *RequestBuilder[T] {
	rb.WithValue(func(r *Request[T]) {
		if r.Meta == nil {
			r.Meta = &modelcontextprotocol.RequestMeta{}
		}
		r.Meta.ProgressToken = &token
	})
	return rb
}

// --- Example usage ---

func ExampleBuilders() {
	// Building a Resource
	resource, _ := NewResourceBuilder("file:///example.txt", "Example File").
		WithDescription("An example text file").
		WithMimeType("text/plain").
		WithSize(1024).
		Build()
		
	// Building a Tool
	schema := modelcontextprotocol.ToolSchema{
		Type:       "object",
		Properties: make(map[string]json.RawMessage),
		Required:   []string{"input"},
	}
	
	tool, _ := NewToolBuilder("example_tool", schema).
		WithDescription("An example tool").
		WithReadOnlyHint(true).
		Build()
		
	// Building a Request
	params := struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{
		Name:  "test",
		Value: "value",
	}
	
	request, _ := NewRequestBuilder(params).
		WithProgressToken("token-123").
		Build()
		
	_ = resource
	_ = tool  
	_ = request
}

// ChainBuilder allows chaining multiple builders together.
type ChainBuilder[T any] struct {
	builders []*Builder[T]
}

// NewChainBuilder creates a new ChainBuilder.
func NewChainBuilder[T any]() *ChainBuilder[T] {
	return &ChainBuilder[T]{
		builders: make([]*Builder[T], 0),
	}
}

// Add adds a builder to the chain.
func (cb *ChainBuilder[T]) Add(builder *Builder[T]) *ChainBuilder[T] {
	cb.builders = append(cb.builders, builder)
	return cb
}

// BuildAll builds all items in the chain.
func (cb *ChainBuilder[T]) BuildAll() ([]T, error) {
	results := make([]T, len(cb.builders))
	for i, builder := range cb.builders {
		result, err := builder.Build()
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}