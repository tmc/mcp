package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"

	"golang.org/x/exp/jsonrpc2"
)

// Server implements a Model Context Protocol server that can handle various
// types of requests including resources, prompts, and tools.
type Server struct {
	name         string
	version      string
	capabilities ServerCapabilities
	instructions string

	tools         map[string]toolDefinition
	resources     map[string]resourceDefinition
	resourceTmpls map[string]resourceTemplateDefinition
	prompts       map[string]promptDefinition

	limiter  *RateLimiter
	dispatch *Dispatcher

	logLevel *slog.Level
	logger   *slog.Logger

	mu       sync.RWMutex
	handlers map[string]jsonrpc2.HandlerFunc
}

type toolDefinition struct {
	tool    Tool
	handler ToolHandlerFunc
}

type resourceDefinition struct {
	resource Resource
	handler  ReadResourceHandlerFunc
}

type resourceTemplateDefinition struct {
	template ResourceTemplate
	handler  ResourceTemplateHandlerFunc
}

type promptDefinition struct {
	prompt  Prompt
	handler GetPromptHandlerFunc
}

// Use the connectionBinder type defined in client.go

// WithServerName sets a custom server name.
func WithServerName(name string) ServerOption {
	return func(s *Server) {
		s.name = name
	}
}

// WithServerVersion sets a custom server version.
func WithServerVersion(version string) ServerOption {
	return func(s *Server) {
		s.version = version
	}
}

// WithServerInstructions sets custom server instructions.
func WithServerInstructions(instructions string) ServerOption {
	return func(s *Server) {
		s.instructions = instructions
	}
}

var defaultServerOptions = []ServerOption{
	withInferredServerName(),
	withInferredServerVersion(),
}

// NewServer creates a new MCP server.
func NewServer(name, version string, opts ...ServerOption) *Server {
	opts = append(defaultServerOptions, opts...)
	s := &Server{
		name:          name,
		version:       version,
		capabilities:  ServerCapabilities{},
		tools:         make(map[string]toolDefinition),
		resources:     make(map[string]resourceDefinition),
		resourceTmpls: make(map[string]resourceTemplateDefinition),
		prompts:       make(map[string]promptDefinition),
		handlers:      make(map[string]jsonrpc2.HandlerFunc),
		dispatch:      NewDispatcher(),
		logger:        slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Register standard handlers
	s.registerDefaultHandlers()

	return s
}

func (s *Server) Serve(ctx context.Context, transport Transport) error {
	s.logger.Debug("Starting MCP server", "name", s.name, "version", s.version)
	return fmt.Errorf("Serve method not implemented in this file, see simple_serve.go for implementation")
}

// flushingReadWriteCloser wraps an io.ReadWriteCloser to ensure flushing after writes
type flushingReadWriteCloser struct {
	io.ReadWriteCloser
	logger *slog.Logger
}

// Write writes data to the underlying writer and attempts to flush
func (f *flushingReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = f.ReadWriteCloser.Write(p)
	if err != nil {
		return n, err
	}

	// Try various flushing methods
	flushed := false

	// Try Flush() method
	if flusher, ok := f.ReadWriteCloser.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			f.logger.Warn("Failed to flush using Flush()", "error", err)
		} else {
			flushed = true
		}
	}

	// Try Sync() method
	if !flushed {
		if syncer, ok := f.ReadWriteCloser.(interface{ Sync() error }); ok {
			if err := syncer.Sync(); err != nil {
				f.logger.Warn("Failed to flush using Sync()", "error", err)
			} else {
				flushed = true
			}
		}
	}

	// If we couldn't flush, log it
	if !flushed {
		f.logger.Debug("Could not flush connection - no flush method available")
	}

	return n, nil
}

// singleConnListener implements jsonrpc2.Listener for a single client
type singleConnListener struct {
	conn     io.ReadWriteCloser
	done     chan struct{}
	returned bool
	logger   *slog.Logger
}

// Accept returns the single connection and allows processing multiple messages on the same connection
func (l *singleConnListener) Accept(ctx context.Context) (io.ReadWriteCloser, error) {
	// Check for connection closed or nil
	if l.conn == nil {
		l.logger.Debug("Accept called but connection is nil")
		return nil, io.EOF
	}

	// If we've already returned the connection, we should just return EOF
	// to signal that no more connections are available (jsonrpc2 library will properly handle this)
	if l.returned {
		l.logger.Debug("Connection already returned, signaling EOF")
		return nil, io.EOF
	}

	// Get the connection and mark as returned
	l.returned = true
	l.logger.Debug("Returning connection from listener")

	// Return the connection - we'll only accept once but keep it alive
	return l.conn, nil
}

// temporaryError is an error that signals a temporary condition
// that might resolve in the future
type temporaryError struct {
	msg string
}

func (e *temporaryError) Error() string {
	return e.msg
}

func (e *temporaryError) Temporary() bool {
	return true
}

// Close closes the listener
func (l *singleConnListener) Close() error {
	l.logger.Debug("Closing singleConnListener")
	// Just close the done channel - don't close the connection
	// as it's still being used by the server
	if l.done != nil {
		close(l.done)
		l.done = nil
	}
	return nil
}

// Dialer returns a dialer to connect to this listener
func (l *singleConnListener) Dialer() jsonrpc2.Dialer {
	return nil // We don't support creating a dialer for this listener
}

// The Serve function implementation is in simple_serve.go

// StdioTransport creates a transport that uses stdin/stdout
func StdioTransport() Transport {
	return &ReadWriteCloserTransport{
		ReadWriteCloser: struct {
			io.Reader
			io.Writer
			io.Closer
		}{
			Reader: os.Stdin,
			Writer: os.Stdout,
			Closer: io.NopCloser(nil), // We don't actually want to close stdin/stdout
		},
	}
}

// handleRequest implements the core JSON-RPC request handler
func (s *Server) handleRequest(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	// Debug log the entire request for troubleshooting
	reqJSON, _ := json.Marshal(req)
	fmt.Fprintf(os.Stderr, "DEBUG GOT REQUEST: %s\n", reqJSON)

	s.logger.Debug("Handling request",
		"method", req.Method,
		"id", req.ID)

	s.mu.RLock()
	handler, exists := s.handlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		s.logger.Warn("Method not supported", "method", req.Method)
		return nil, fmt.Errorf("method '%s' not supported", req.Method)
	}

	// Call the handler and get the result
	result, err := handler(ctx, req)
	if err != nil {
		s.logger.Warn("Method execution failed",
			"method", req.Method,
			"error", err)
	} else {
		s.logger.Debug("Method completed successfully",
			"method", req.Method)

		// Debug log the result
		resultJSON, _ := json.Marshal(result)
		fmt.Fprintf(os.Stderr, "DEBUG RESULT: %s\n", resultJSON)
	}

	return result, err
}

// registerDefaultHandlers sets up the standard MCP protocol handlers
func (s *Server) registerDefaultHandlers() {
	// Register initialize handler
	s.handlers[string(MethodInitialize)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params InitializeRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid initialize parameters: %w", err)
		}

		result := InitializeResult{
			ProtocolVersion: LATEST_PROTOCOL_VERSION,
			ServerInfo: Implementation{
				Name:    s.name,
				Version: s.version,
			},
			Capabilities: s.capabilities,
			Instructions: s.instructions,
		}

		return result, nil
	}

	// Register ping handler
	s.handlers[string(MethodPing)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		return struct{}{}, nil
	}

	// Register tools/list handler
	s.handlers[string(MethodToolsList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListToolsRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid tools/list parameters: %w", err)
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		tools := make([]Tool, 0, len(s.tools))
		for _, def := range s.tools {
			tools = append(tools, def.tool)
		}

		result := ListToolsResult{
			Tools: tools,
			// Implement cursor-based pagination in a future version
		}

		return result, nil
	}

	// Register tools/call handler
	s.handlers[string(MethodToolsCall)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params CallToolRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid tools/call parameters: %w", err)
		}

		s.mu.RLock()
		toolDef, exists := s.tools[params.Name]
		s.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("tool '%s' not found", params.Name)
		}

		result, err := toolDef.handler(ctx, params)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	// Register prompts/list handler
	s.handlers[string(MethodPromptsList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListPromptsRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid prompts/list parameters: %w", err)
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		prompts := make([]Prompt, 0, len(s.prompts))
		for _, def := range s.prompts {
			prompts = append(prompts, def.prompt)
		}

		result := ListPromptsResult{
			Prompts: prompts,
			// Implement cursor-based pagination in a future version
		}

		return result, nil
	}

	// Register prompts/get handler
	s.handlers[string(MethodPromptsGet)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params GetPromptRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid prompts/get parameters: %w", err)
		}

		s.mu.RLock()
		promptDef, exists := s.prompts[params.Name]
		s.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("prompt '%s' not found", params.Name)
		}

		result, err := promptDef.handler(ctx, params)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	// Register resources/list handler
	s.handlers[string(MethodResourcesList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListResourcesRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid resources/list parameters: %w", err)
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		resources := make([]Resource, 0, len(s.resources))
		for _, def := range s.resources {
			resources = append(resources, def.resource)
		}

		result := ListResourcesResult{
			Resources: resources,
			// Implement cursor-based pagination in a future version
		}

		return result, nil
	}

	// Register resources/read handler
	s.handlers[string(MethodResourcesRead)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ReadResourceRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid resources/read parameters: %w", err)
		}

		s.mu.RLock()
		resourceDef, exists := s.resources[params.URI]
		s.mu.RUnlock()

		if !exists {
			// Try to find a matching template
			s.mu.RLock()
			var matchedTemplate resourceTemplateDefinition
			var found bool

			// This is a simple exact match for now
			// In future, implement proper template matching with wildcards or regex
			for _, tmplDef := range s.resourceTmpls {
				if tmplDef.template.Template == params.URI {
					matchedTemplate = tmplDef
					found = true
					break
				}
			}
			s.mu.RUnlock()

			if !found {
				return nil, fmt.Errorf("resource '%s' not found", params.URI)
			}

			contents, err := matchedTemplate.handler(ctx, params)
			if err != nil {
				return nil, err
			}

			result := ReadResourceResult{
				Contents: contents,
			}

			return result, nil
		}

		contents, err := resourceDef.handler(ctx, params)
		if err != nil {
			return nil, err
		}

		result := ReadResourceResult{
			Contents: contents,
		}

		return result, nil
	}

	// Register resources/templates/list handler
	s.handlers[string(MethodResourcesTemplatesList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListResourceTemplatesRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid resources/templates/list parameters: %w", err)
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		templates := make([]ResourceTemplate, 0, len(s.resourceTmpls))
		for _, def := range s.resourceTmpls {
			templates = append(templates, def.template)
		}

		result := ListResourceTemplatesResult{
			Templates: templates,
			// Implement cursor-based pagination in a future version
		}

		return result, nil
	}
}

// RegisterTool adds a new tool to the server.
func (s *Server) RegisterTool(tool Tool, handler ToolHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering tool", "name", tool.Name)

	if _, exists := s.tools[tool.Name]; exists {
		s.logger.Warn("Tool already registered", "name", tool.Name)
		return fmt.Errorf("tool '%s' already registered", tool.Name)
	}

	s.tools[tool.Name] = toolDefinition{
		tool:    tool,
		handler: handler,
	}

	s.logger.Info("Tool registered successfully", "name", tool.Name)

	if s.capabilities.Tools != nil && s.capabilities.Tools.ListChanged {
		s.logger.Debug("Sending tool list changed notification")
		go s.dispatch.NotifyListChanged(MethodToolListChanged)
	}

	return nil
}

// RegisterPrompt adds a new prompt to the server.
func (s *Server) RegisterPrompt(prompt Prompt, handler GetPromptHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering prompt", "name", prompt.Name)

	if _, exists := s.prompts[prompt.Name]; exists {
		s.logger.Warn("Prompt already registered", "name", prompt.Name)
		return fmt.Errorf("prompt '%s' already registered", prompt.Name)
	}

	s.prompts[prompt.Name] = promptDefinition{
		prompt:  prompt,
		handler: handler,
	}

	s.logger.Info("Prompt registered successfully", "name", prompt.Name)
	s.logger.Debug("Sending prompt list changed notification")
	go s.dispatch.NotifyListChanged(MethodPromptListChanged)

	return nil
}

// RegisterResource adds a new resource to the server.
func (s *Server) RegisterResource(resource Resource, handler ReadResourceHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering resource", "uri", resource.URI)

	if _, exists := s.resources[resource.URI]; exists {
		s.logger.Warn("Resource already registered", "uri", resource.URI)
		return fmt.Errorf("resource '%s' already registered", resource.URI)
	}

	s.resources[resource.URI] = resourceDefinition{
		resource: resource,
		handler:  handler,
	}

	s.logger.Info("Resource registered successfully", "uri", resource.URI)
	s.logger.Debug("Sending resource list changed notification")
	go s.dispatch.NotifyListChanged(MethodResourceListChanged)

	return nil
}

// RegisterResourceTemplate adds a new resource template to the server.
func (s *Server) RegisterResourceTemplate(template ResourceTemplate, handler ResourceTemplateHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering resource template", "template", template.Template)

	if _, exists := s.resourceTmpls[template.Template]; exists {
		s.logger.Warn("Resource template already registered", "template", template.Template)
		return fmt.Errorf("resource template '%s' already registered", template.Template)
	}

	s.resourceTmpls[template.Template] = resourceTemplateDefinition{
		template: template,
		handler:  handler,
	}

	s.logger.Info("Resource template registered successfully", "template", template.Template)
	s.logger.Debug("Sending resource list changed notification")
	go s.dispatch.NotifyListChanged(MethodResourceListChanged)

	return nil
}

// withInferredServerName sets the server name to the default value using go build info.
func withInferredServerName() ServerOption {
	return func(s *Server) {
		if s.name == "" {
			s.name = inferServerName()
		}
	}
}

// withInferredServerVersion sets the server version to the default value using go build info.
func withInferredServerVersion() ServerOption {
	return func(s *Server) {
		s.version = inferServerVersion()
	}
}

// inferServerName infers the server name from the build info.
func inferServerName() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return os.Args[0]
	}
	if bi.Main.Path != "" {
		return bi.Main.Path
	}
	return os.Args[0]
}

// inferServerVersion infers the server version from the build info.
func inferServerVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	bij, _ := json.Marshal(bi)
	slog.Info("Build info", "info", bij)
	if bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "unknown"
}
