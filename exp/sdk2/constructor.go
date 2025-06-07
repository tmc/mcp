package sdk2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// NewClient creates a new Client with the given options.
// This follows the pattern of sql.Open, http.NewClient, etc.
func NewClient(opts ...ClientOption) Client {
	config := &ClientConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
		ClientInfo: ClientInfo{Name: "sdk2-client", Version: "0.1.0"},
	}

	for _, opt := range opts {
		opt(config)
	}

	return &client{
		config:  *config,
		pending: make(map[int64]chan *jsonrpcResponse),
		done:    make(chan struct{}),
	}
}

// NewServer creates a new Server with the given options.
// This follows the pattern of http.NewServer.
func NewServer(opts ...ServerOption) *Server {
	server := &Server{
		Handler: DefaultServeMux,
	}

	for _, opt := range opts {
		opt(server)
	}

	return server
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux {
	return &ServeMux{
		handlers: make(map[string]Handler),
	}
}

// NewRequest creates a new Request with the given method and parameters.
// This follows the pattern of http.NewRequest.
func NewRequest(method string, params interface{}) (*Request, error) {
	var paramBytes []byte
	var err error

	if params != nil {
		paramBytes, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
	}

	return &Request{
		Method:  method,
		Params:  json.RawMessage(paramBytes),
		Proto:   ProtocolVersion,
		Context: context.Background(),
	}, nil
}

// NewRequestWithContext creates a new Request with the given context.
func NewRequestWithContext(ctx context.Context, method string, params interface{}) (*Request, error) {
	req, err := NewRequest(method, params)
	if err != nil {
		return nil, err
	}
	req.Context = ctx
	return req, nil
}

// Content convenience constructors following the NewXxx pattern

// NewTextContent creates a new TextContent with validation.
func NewTextContent(text string) (TextContent, error) {
	content := TextContent{Text: text}
	if err := content.Valid(); err != nil {
		return TextContent{}, fmt.Errorf("invalid text content: %w", err)
	}
	return content, nil
}

// MustNewTextContent creates a new TextContent and panics on validation error.
// This follows the pattern of template.Must, regexp.MustCompile, etc.
func MustNewTextContent(text string) TextContent {
	content, err := NewTextContent(text)
	if err != nil {
		panic(err)
	}
	return content
}

// NewImageContent creates a new ImageContent with validation.
func NewImageContent(data, mimeType string) (ImageContent, error) {
	content := ImageContent{Data: data, MimeType: mimeType}
	if err := content.Valid(); err != nil {
		return ImageContent{}, fmt.Errorf("invalid image content: %w", err)
	}
	return content, nil
}

// MustNewImageContent creates a new ImageContent and panics on validation error.
func MustNewImageContent(data, mimeType string) ImageContent {
	content, err := NewImageContent(data, mimeType)
	if err != nil {
		panic(err)
	}
	return content
}

// NewResourceReferenceContent creates a new ResourceReferenceContent with validation.
func NewResourceReferenceContent(uri string) (ResourceReferenceContent, error) {
	content := ResourceReferenceContent{URI: uri}
	if err := content.Valid(); err != nil {
		return ResourceReferenceContent{}, fmt.Errorf("invalid resource content: %w", err)
	}
	return content, nil
}

// MustNewResourceReferenceContent creates a new ResourceReferenceContent and panics on validation error.
func MustNewResourceReferenceContent(uri string) ResourceReferenceContent {
	content, err := NewResourceReferenceContent(uri)
	if err != nil {
		panic(err)
	}
	return content
}

// Tool definition helpers

// NewTool creates a new Tool with the given schema.
func NewTool(name, description string, schema interface{}) (Tool, error) {
	var schemaBytes json.RawMessage
	var err error

	if schema != nil {
		schemaBytes, err = json.Marshal(schema)
		if err != nil {
			return Tool{}, fmt.Errorf("marshal schema: %w", err)
		}
	}

	return Tool{
		Name:        name,
		Description: description,
		InputSchema: schemaBytes,
	}, nil
}

// MustNewTool creates a new Tool and panics on error.
func MustNewTool(name, description string, schema interface{}) Tool {
	tool, err := NewTool(name, description, schema)
	if err != nil {
		panic(err)
	}
	return tool
}

// Resource definition helpers

// NewResource creates a new Resource.
func NewResource(uri, name, description, mimeType string) Resource {
	return Resource{
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    mimeType,
	}
}

// Prompt definition helpers

// NewPrompt creates a new Prompt.
func NewPrompt(name, description string, schema interface{}) (Prompt, error) {
	var schemaBytes json.RawMessage
	var err error

	if schema != nil {
		schemaBytes, err = json.Marshal(schema)
		if err != nil {
			return Prompt{}, fmt.Errorf("marshal schema: %w", err)
		}
	}

	return Prompt{
		Name:            name,
		Description:     description,
		ArgumentsSchema: schemaBytes,
	}, nil
}

// MustNewPrompt creates a new Prompt and panics on error.
func MustNewPrompt(name, description string, schema interface{}) Prompt {
	prompt, err := NewPrompt(name, description, schema)
	if err != nil {
		panic(err)
	}
	return prompt
}
