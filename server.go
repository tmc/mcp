package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"testing"

	"golang.org/x/exp/jsonrpc2"
)

// serverBinder is a custom JSON-RPC binder that provides enhanced server functionality.
// It wraps the standard JSON-RPC handler with additional capabilities including
// cancellation support through the CancellablePreempter, which enables proper
// handling of client-initiated request cancellations.
type serverBinder struct {
	handler jsonrpc2.Handler
	logger  *slog.Logger
}

// Bind implements the jsonrpc2.Binder interface
func (b *serverBinder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	return jsonrpc2.ConnectionOptions{
		Handler: b.handler,
		Preempter: &CancellablePreempter{
			Conn:   conn,
			Logger: b.logger,
		},
	}, nil
}

// Server implements a Model Context Protocol server that can handle various
// types of requests including resources, prompts, and tools.
type Server struct {
	name         string
	version      string
	capabilities ServerCapabilities
	instructions string

	dispatch  *Dispatcher
	validator *ParameterValidator

	logLevel *slog.Level
	logger   *slog.Logger

	mu            sync.RWMutex // Protects the following fields:
	tools         map[string]toolDefinition
	resources     map[string]resourceDefinition
	resourceTmpls map[string]resourceTemplateDefinition
	prompts       map[string]promptDefinition
	handlers      map[string]jsonrpc2.HandlerFunc
	activeTools   map[string]context.CancelFunc
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

// WithValidationConfig sets a custom validation configuration.
func WithValidationConfig(config *ValidationConfig) ServerOption {
	return func(s *Server) {
		if config != nil {
			s.validator = NewParameterValidator(config)
		}
	}
}

var defaultServerOptions = []ServerOption{
	withInferredServerName(),
	withInferredServerVersion(),
}

// NewServer creates a new MCP server.
func NewServer(name, version string, opts ...ServerOption) *Server {
	opts = append(defaultServerOptions, opts...)

	// Create a test-aware default logger
	var defaultLogger *slog.Logger
	if isInTest() {
		// In test mode, use a quiet logger to avoid cluttering test output
		defaultLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
	} else {
		defaultLogger = slog.Default()
	}

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
		validator:     NewParameterValidator(DefaultValidationConfig()),
		logger:        defaultLogger,
		activeTools:   make(map[string]context.CancelFunc),
		mu:            sync.RWMutex{},
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Register standard handlers
	s.registerDefaultHandlers()

	return s
}

// flushingReadWriteCloser wraps an io.ReadWriteCloser to ensure immediate flushing after writes.
// This is essential for MCP communication to ensure messages are delivered promptly rather than
// being buffered. It attempts multiple flushing strategies (Flush(), Sync()) to accommodate
// different underlying transport implementations.
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

// singleConnListener implements jsonrpc2.Listener for single-connection MCP servers.
// Unlike traditional servers that accept multiple connections, MCP servers typically
// handle a single long-lived connection (e.g., stdin/stdout). This listener manages
// the lifecycle of that single connection and properly signals EOF when no more
// connections are available.
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

// handleRequest implements the core JSON-RPC request handler with enhanced validation and monitoring
func (s *Server) handleRequest(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	// Apply request size limits before processing
	if req.Params != nil {
		if err := s.validator.ValidateRequest(req.Method, req.Params); err != nil {
			s.logger.Warn("Request validation failed",
				"method", req.Method,
				"error", err)
			return nil, err
		}
	}

	// Debug log the entire request for troubleshooting (but only if debug enabled)
	if s.logger.Enabled(ctx, slog.LevelDebug) {
		// Limit debug logging for large requests
		if len(req.Params) > 10*1024 { // 10KB limit for debug logging
			s.logger.Debug("Got large request",
				"method", req.Method,
				"id", req.ID,
				"params_size", len(req.Params))
		} else {
			reqJSON, _ := json.Marshal(req)
			s.logger.Debug("Got request",
				"request", string(reqJSON),
				"method", req.Method,
				"id", req.ID)
		}
	}

	s.mu.RLock()
	handler, exists := s.handlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		s.logger.Warn("Method not supported", "method", req.Method)
		return nil, NewNotFoundError("method", req.Method)
	}

	// Call the handler and get the result with timeout protection
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := handler(ctx, req)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for result or context cancellation
	select {
	case <-ctx.Done():
		s.logger.Warn("Request cancelled", "method", req.Method)
		return nil, ctx.Err()
	case err := <-errChan:
		s.logger.Warn("Method execution failed",
			"method", req.Method,
			"error", err)
		return nil, err
	case result := <-resultChan:
		s.logger.Debug("Method completed successfully",
			"method", req.Method)

		// Debug log the result (with size limits)
		if s.logger.Enabled(ctx, slog.LevelDebug) {
			resultJSON, _ := json.Marshal(result)
			if len(resultJSON) > 10*1024 { // 10KB limit for debug logging
				s.logger.Debug("Got large result",
					"method", req.Method,
					"result_size", len(resultJSON))
			} else {
				s.logger.Debug("Got result", "result", string(resultJSON))
			}
		}

		return result, nil
	}
}

// registerDefaultHandlers sets up the standard MCP protocol handlers required by the specification.
// This function orchestrates the registration of all standard protocol handlers by calling
// individual registration functions for each handler category.
func (s *Server) registerDefaultHandlers() {
	s.registerInitializeHandler()
	s.registerPingHandler()
	s.registerToolHandlers()
	s.registerPromptHandlers()
	s.registerResourceHandlers()
}

// registerInitializeHandler registers the initialize protocol handler for handshake and capability negotiation
func (s *Server) registerInitializeHandler() {
	s.handlers[string(MethodInitialize)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		// Validate request format first
		if err := s.validator.ValidateRequest(string(MethodInitialize), req.Params); err != nil {
			return nil, err
		}

		var params InitializeRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodInitialize), err)
		}

		// Validate request parameters
		if err := s.validator.ValidateInitializeRequest(params); err != nil {
			return nil, err
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
}

// registerPingHandler registers the ping handler for server liveness checks
func (s *Server) registerPingHandler() {
	s.handlers[string(MethodPing)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		return struct{}{}, nil
	}
}

// registerToolHandlers registers the tool management handlers (list and call)
func (s *Server) registerToolHandlers() {
	// Register tools/list handler
	s.handlers[string(MethodToolsList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListToolsRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodToolsList), err)
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
		// Validate request format first
		if err := s.validator.ValidateRequest(string(MethodToolsCall), req.Params); err != nil {
			return nil, err
		}

		var params CallToolRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodToolsCall), err)
		}

		// Validate request parameters
		if err := s.validator.ValidateCallToolRequest(params); err != nil {
			return nil, err
		}

		s.mu.RLock()
		toolDef, exists := s.tools[params.Name]
		s.mu.RUnlock()

		if !exists {
			return nil, NewNotFoundError("tool", params.Name)
		}

		result, err := toolDef.handler(ctx, params)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// registerPromptHandlers registers the prompt management handlers (list and get)
func (s *Server) registerPromptHandlers() {
	// Register prompts/list handler
	s.handlers[string(MethodPromptsList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListPromptsRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodPromptsList), err)
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
		// Validate request format first
		if err := s.validator.ValidateRequest(string(MethodPromptsGet), req.Params); err != nil {
			return nil, err
		}

		var params GetPromptRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodPromptsGet), err)
		}

		// Validate request parameters
		if err := s.validator.ValidateGetPromptRequest(params); err != nil {
			return nil, err
		}

		s.mu.RLock()
		promptDef, exists := s.prompts[params.Name]
		s.mu.RUnlock()

		if !exists {
			return nil, NewNotFoundError("prompt", params.Name)
		}

		result, err := promptDef.handler(ctx, params)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// registerResourceHandlers registers the resource management handlers (list, read, templates/list)
func (s *Server) registerResourceHandlers() {
	// Register resources/list handler
	s.handlers[string(MethodResourcesList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListResourcesRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodResourcesList), err)
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
		// Validate request format first
		if err := s.validator.ValidateRequest(string(MethodResourcesRead), req.Params); err != nil {
			return nil, err
		}

		var params ReadResourceRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodResourcesRead), err)
		}

		// Validate request parameters
		if err := s.validator.ValidateReadResourceRequest(params); err != nil {
			return nil, err
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
				return nil, NewNotFoundError("resource", params.URI)
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
			return nil, NewParameterErrorFromJSON(string(MethodResourcesTemplatesList), err)
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
	if s == nil {
		return fmt.Errorf("server is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering tool", "name", tool.Name)

	if _, exists := s.tools[tool.Name]; exists {
		s.logger.Warn("Tool already registered", "name", tool.Name)
		return NewAlreadyExistsError("tool", tool.Name)
	}

	s.tools[tool.Name] = toolDefinition{
		tool:    tool,
		handler: handler,
	}

	// Initialize and set tools capability
	if s.capabilities.Tools == nil {
		s.capabilities.Tools = &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{}
	}
	s.capabilities.Tools.ListChanged = true

	s.logger.Info("Tool registered successfully", "name", tool.Name)
	s.logger.Debug("Sending tool list changed notification")
	go s.dispatch.NotifyListChanged(context.Background(), MethodToolListChanged)

	return nil
}

// RegisterPrompt adds a new prompt to the server.
func (s *Server) RegisterPrompt(prompt Prompt, handler GetPromptHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering prompt", "name", prompt.Name)

	if _, exists := s.prompts[prompt.Name]; exists {
		s.logger.Warn("Prompt already registered", "name", prompt.Name)
		return NewAlreadyExistsError("prompt", prompt.Name)
	}

	s.prompts[prompt.Name] = promptDefinition{
		prompt:  prompt,
		handler: handler,
	}

	// Initialize and set prompts capability
	if s.capabilities.Prompts == nil {
		s.capabilities.Prompts = &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{}
	}
	s.capabilities.Prompts.ListChanged = true

	s.logger.Info("Prompt registered successfully", "name", prompt.Name)
	s.logger.Debug("Sending prompt list changed notification")
	go s.dispatch.NotifyListChanged(context.Background(), MethodPromptListChanged)

	return nil
}

// RegisterResource adds a new resource to the server.
func (s *Server) RegisterResource(resource Resource, handler ReadResourceHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering resource", "uri", resource.URI)

	if _, exists := s.resources[resource.URI]; exists {
		s.logger.Warn("Resource already registered", "uri", resource.URI)
		return NewAlreadyExistsError("resource", resource.URI)
	}

	s.resources[resource.URI] = resourceDefinition{
		resource: resource,
		handler:  handler,
	}

	// Initialize and set resources capability
	if s.capabilities.Resources == nil {
		s.capabilities.Resources = &struct {
			Subscribe   bool `json:"subscribe,omitempty"`
			ListChanged bool `json:"listChanged,omitempty"`
		}{}
	}
	s.capabilities.Resources.ListChanged = true

	s.logger.Info("Resource registered successfully", "uri", resource.URI)
	s.logger.Debug("Sending resource list changed notification")
	go s.dispatch.NotifyListChanged(context.Background(), MethodResourceListChanged)

	return nil
}

// RegisterResourceTemplate adds a new resource template to the server.
func (s *Server) RegisterResourceTemplate(template ResourceTemplate, handler ResourceTemplateHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering resource template", "template", template.Template)

	if _, exists := s.resourceTmpls[template.Template]; exists {
		s.logger.Warn("Resource template already registered", "template", template.Template)
		return NewAlreadyExistsError("resource template", template.Template)
	}

	s.resourceTmpls[template.Template] = resourceTemplateDefinition{
		template: template,
		handler:  handler,
	}

	s.logger.Info("Resource template registered successfully", "template", template.Template)
	s.logger.Debug("Sending resource list changed notification")
	go s.dispatch.NotifyListChanged(context.Background(), MethodResourceListChanged)

	return nil
}

// Serve starts serving MCP requests using the provided transport.
// It establishes a JSON-RPC connection and handles incoming requests.
func (s *Server) Serve(ctx context.Context, transport Transport) error {
	// Default to stdio transport if none provided
	if transport == nil {
		transport = StdioTransport()
	}

	// Create a handler for the connection
	handler := jsonrpc2.HandlerFunc(s.handleRequest)

	// Create a custom binder that includes cancellation support
	binder := &serverBinder{
		handler: handler,
		logger:  s.logger,
	}

	// Create the connection
	conn, err := jsonrpc2.Dial(ctx, transport, binder)
	if err != nil {
		return fmt.Errorf("failed to establish connection: %w", err)
	}
	defer conn.Close()

	// Wait for either context cancellation or connection to finish
	// The connection will automatically handle incoming requests via the handler
	done := make(chan error, 1)
	go func() {
		done <- conn.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		// Connection finished, return any error
		if err != nil {
			return fmt.Errorf("connection error: %w", err)
		}
		return nil
	}
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
		if s.version == "" {
			s.version = inferServerVersion()
		}
	}
}

// inferServerName infers the server name from the build info.
func inferServerName() string {
	if len(os.Args) == 0 {
		return ""
	}

	// Extract the base name from the first argument
	return filepath.Base(os.Args[0])
}

// inferServerVersion infers the server version from the build info.
func inferServerVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	if bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "unknown"
}

// isInTest returns true if we're running in a test environment
func isInTest() bool {
	return strings.HasSuffix(os.Args[0], ".test") ||
		strings.Contains(os.Args[0], "/_test/") ||
		os.Getenv("GOTEST") == "1"
}

// isShortTest returns true if we're running tests in short mode
func isShortTest() bool {
	return testing.Short()
}
