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
	"time"

	"golang.org/x/exp/jsonrpc2"
)

// serverBinder implements jsonrpc2.Binder with cancellation support
type serverBinder struct {
	handler jsonrpc2.Handler
	logger  *slog.Logger
	framer  jsonrpc2.Framer
}

func (b serverBinder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	framer := b.framer
	if framer == nil {
		framer = defaultFramer()
	}
	return jsonrpc2.ConnectionOptions{
		Handler: b.handler,
		Framer:  framer,
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

	// serverRequestTimeout bounds how long a server-initiated request
	// (sampling, elicitation, roots/list) waits for a client response when the
	// caller's context has no earlier deadline. Zero means no added deadline.
	serverRequestTimeout time.Duration

	mu            sync.RWMutex // Protects the following fields:
	tools         map[string]toolDefinition
	resources     map[string]resourceDefinition
	resourceTmpls map[string]resourceTemplateDefinition
	prompts       map[string]promptDefinition
	subscriptions map[string]bool
	conn          *jsonrpc2.Connection
	clientCaps    ClientCapabilities
	completion    CompletionHandlerFunc
	handlers      map[string]jsonrpc2.HandlerFunc
	activeTools   map[string]context.CancelFunc
	framer        jsonrpc2.Framer
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

// WithServerRequestTimeout sets how long a server-initiated request (sampling,
// elicitation, roots/list) waits for a client response when the caller's context
// has no earlier deadline. A zero or negative duration disables the added
// deadline. The default is 30 seconds.
func WithServerRequestTimeout(d time.Duration) ServerOption {
	return func(s *Server) {
		s.serverRequestTimeout = d
	}
}

// WithServerRawFraming uses the undelimited JSON-RPC framing used by older
// versions of this package.
func WithServerRawFraming() ServerOption {
	return withServerFramer(jsonrpc2.RawFramer())
}

func withServerFramer(framer jsonrpc2.Framer) ServerOption {
	return func(s *Server) {
		s.framer = framer
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
		name:                 name,
		version:              version,
		capabilities:         ServerCapabilities{},
		tools:                make(map[string]toolDefinition),
		resources:            make(map[string]resourceDefinition),
		resourceTmpls:        make(map[string]resourceTemplateDefinition),
		prompts:              make(map[string]promptDefinition),
		subscriptions:        make(map[string]bool),
		handlers:             make(map[string]jsonrpc2.HandlerFunc),
		dispatch:             NewDispatcher(),
		validator:            NewParameterValidator(DefaultValidationConfig()),
		logger:               defaultLogger,
		activeTools:          make(map[string]context.CancelFunc),
		framer:               defaultFramer(),
		serverRequestTimeout: 30 * time.Second,
		mu:                   sync.RWMutex{},
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

// flushingDialer wraps a Dialer to return a flushingReadWriteCloser
type flushingDialer struct {
	dialer interface {
		Dial(context.Context) (io.ReadWriteCloser, error)
	}
	logger *slog.Logger
}

func (d *flushingDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	rwc, err := d.dialer.Dial(ctx)
	if err != nil {
		return nil, err
	}
	return &flushingReadWriteCloser{
		ReadWriteCloser: rwc,
		logger:          d.logger,
	}, nil
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
		// Recover panics from the user handler so a single bad request
		// degrades to an error instead of crashing the whole server. The
		// recover must live in this goroutine; RecoveryMiddleware runs in the
		// caller's goroutine and cannot catch a panic here. Reporting the
		// panic as an error also unblocks the select below, which would
		// otherwise wait on the result channel until ctx cancellation.
		defer func() {
			if r := recover(); r != nil {
				err := recoverPanic(r)
				s.logger.Error("recovered panic in handler",
					"method", req.Method, "error", err)
				errChan <- fmt.Errorf("handler %s: %w", req.Method, err)
			}
		}()
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
	s.registerLoggingHandler()
	s.registerCompletionHandler()
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

		s.mu.Lock()
		s.clientCaps = params.Capabilities
		s.mu.Unlock()

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

const (
	slogLevelNotice    = (slog.LevelInfo + slog.LevelWarn) / 2
	slogLevelCritical  = slog.LevelError + 4
	slogLevelAlert     = slog.LevelError + 8
	slogLevelEmergency = slog.LevelError + 12
)

func slogLevelForLoggingLevel(level LoggingLevel) (slog.Level, bool) {
	switch level {
	case LogLevelDebug:
		return slog.LevelDebug, true
	case LogLevelInfo:
		return slog.LevelInfo, true
	case LogLevelNotice:
		return slogLevelNotice, true
	case LogLevelWarning:
		return slog.LevelWarn, true
	case LogLevelError:
		return slog.LevelError, true
	case LogLevelCritical:
		return slogLevelCritical, true
	case LogLevelAlert:
		return slogLevelAlert, true
	case LogLevelEmergency:
		return slogLevelEmergency, true
	default:
		return 0, false
	}
}

func (s *Server) registerLoggingHandler() {
	s.capabilities.Logging = &struct{}{}
	// Default to Info so protocol logging messages flow before a client sends
	// logging/setLevel. Without a default, NotifyLoggingMessage would silently
	// drop everything until the first setLevel request.
	if s.logLevel == nil {
		defaultLevel := slog.LevelInfo
		s.logLevel = &defaultLevel
	}
	s.handlers[string(MethodLoggingSetLevel)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		if len(req.Params) == 0 || strings.TrimSpace(string(req.Params)) == "null" {
			return nil, NewParameterError(string(MethodLoggingSetLevel), "params", "missing required params", nil)
		}
		if err := s.validator.ValidateRequest(string(MethodLoggingSetLevel), req.Params); err != nil {
			return nil, err
		}

		var params SetLevelRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodLoggingSetLevel), err)
		}

		level, ok := slogLevelForLoggingLevel(params.Level)
		if !ok {
			return nil, NewParameterError(string(MethodLoggingSetLevel), "level", "unsupported logging level", nil)
		}

		s.mu.Lock()
		s.logLevel = &level
		s.mu.Unlock()

		return struct{}{}, nil
	}
}

func (s *Server) registerCompletionHandler() {
	if s.completion != nil {
		s.capabilities.Completions = &struct{}{}
	}
	s.handlers[string(MethodCompletionComplete)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		if s.completion == nil {
			return nil, jsonrpc2.ErrMethodNotFound
		}
		if len(req.Params) == 0 || strings.TrimSpace(string(req.Params)) == "null" {
			return nil, NewParameterError(string(MethodCompletionComplete), "params", "missing required params", nil)
		}
		if err := s.validator.ValidateRequest(string(MethodCompletionComplete), req.Params); err != nil {
			return nil, err
		}

		var params CompleteRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, NewParameterErrorFromJSON(string(MethodCompletionComplete), err)
		}
		return s.completion(ctx, params)
	}
}

// registerToolHandlers registers the tool management handlers (list and call)
func (s *Server) registerToolHandlers() {
	// Register tools/list handler
	s.handlers[string(MethodToolsList)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params ListToolsRequest
		if err := unmarshalOptionalParams(string(MethodToolsList), req.Params, &params); err != nil {
			return nil, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		tools := make([]Tool, 0, len(s.tools))
		for _, def := range s.tools {
			tools = append(tools, def.tool)
		}

		page, next := paginate(tools, params.Cursor, func(t Tool) string { return t.Name })
		return ListToolsResult{Tools: page, NextCursor: next}, nil
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
		if err := unmarshalOptionalParams(string(MethodPromptsList), req.Params, &params); err != nil {
			return nil, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		prompts := make([]Prompt, 0, len(s.prompts))
		for _, def := range s.prompts {
			prompts = append(prompts, def.prompt)
		}

		page, next := paginate(prompts, params.Cursor, func(p Prompt) string { return p.Name })
		return ListPromptsResult{Prompts: page, NextCursor: next}, nil
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
		if err := unmarshalOptionalParams(string(MethodResourcesList), req.Params, &params); err != nil {
			return nil, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		resources := make([]Resource, 0, len(s.resources))
		for _, def := range s.resources {
			resources = append(resources, def.resource)
		}

		page, next := paginate(resources, params.Cursor, func(r Resource) string { return r.URI })
		return ListResourcesResult{Resources: page, NextCursor: next}, nil
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
		if err := unmarshalOptionalParams(string(MethodResourcesTemplatesList), req.Params, &params); err != nil {
			return nil, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		templates := make([]ResourceTemplate, 0, len(s.resourceTmpls))
		for _, def := range s.resourceTmpls {
			templates = append(templates, def.template)
		}

		page, next := paginate(templates, params.Cursor, func(t ResourceTemplate) string { return t.Template })
		return ListResourceTemplatesResult{Templates: page, NextCursor: next}, nil
	}

	s.handlers[string(MethodResourcesSubscribe)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params SubscribeResourceRequest
		if err := unmarshalRequiredParams(string(MethodResourcesSubscribe), req.Params, &params); err != nil {
			return nil, err
		}
		if err := s.validator.ValidateResourceSubscription(MethodResourcesSubscribe, params.URI); err != nil {
			return nil, err
		}

		s.mu.Lock()
		s.subscriptions[params.URI] = true
		s.mu.Unlock()
		return struct{}{}, nil
	}

	s.handlers[string(MethodResourcesUnsubscribe)] = func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		var params UnsubscribeResourceRequest
		if err := unmarshalRequiredParams(string(MethodResourcesUnsubscribe), req.Params, &params); err != nil {
			return nil, err
		}
		if err := s.validator.ValidateResourceSubscription(MethodResourcesUnsubscribe, params.URI); err != nil {
			return nil, err
		}

		s.mu.Lock()
		delete(s.subscriptions, params.URI)
		s.mu.Unlock()
		return struct{}{}, nil
	}
}

func unmarshalOptionalParams(method string, params json.RawMessage, dst any) error {
	if len(params) == 0 || strings.TrimSpace(string(params)) == "null" {
		return nil
	}
	if err := json.Unmarshal(params, dst); err != nil {
		return NewParameterErrorFromJSON(method, err)
	}
	return nil
}

func unmarshalRequiredParams(method string, params json.RawMessage, dst any) error {
	if len(params) == 0 || strings.TrimSpace(string(params)) == "null" {
		return NewParameterError(method, "params", "missing required params", nil)
	}
	if err := json.Unmarshal(params, dst); err != nil {
		return NewParameterErrorFromJSON(method, err)
	}
	return nil
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
	go s.notifyListChanged(MethodToolListChanged)

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
	go s.notifyListChanged(MethodPromptListChanged)

	return nil
}

// RegisterResource adds a new resource to the server.
func (s *Server) RegisterResource(resource Resource, handler ReadResourceHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug("Registering resource", "uri", resource.URI)
	if resource.Name == "" {
		resource.Name = resource.URI
	}

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
	s.capabilities.Resources.Subscribe = true

	s.logger.Info("Resource registered successfully", "uri", resource.URI)
	s.logger.Debug("Sending resource list changed notification")
	go s.notifyListChanged(MethodResourceListChanged)

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
	if s.capabilities.Resources == nil {
		s.capabilities.Resources = &struct {
			Subscribe   bool `json:"subscribe,omitempty"`
			ListChanged bool `json:"listChanged,omitempty"`
		}{}
	}
	s.capabilities.Resources.ListChanged = true
	s.capabilities.Resources.Subscribe = true

	s.logger.Info("Resource template registered successfully", "template", template.Template)
	s.logger.Debug("Sending resource list changed notification")
	go s.notifyListChanged(MethodResourceListChanged)

	return nil
}

// ResourceUpdated notifies subscribed clients that a resource changed.
func (s *Server) ResourceUpdated(ctx context.Context, params ResourceUpdatedNotificationParams) error {
	if err := s.validator.ValidateResourceSubscription(MethodResourceUpdated, params.URI); err != nil {
		return err
	}

	s.mu.RLock()
	subscribed := s.subscriptions[params.URI]
	s.mu.RUnlock()
	if !subscribed {
		return nil
	}

	return s.notify(ctx, MethodResourceUpdated, params)
}

// NotifyProgress sends a progress notification to the connected client.
func (s *Server) NotifyProgress(ctx context.Context, token any, progress float64, total *float64) error {
	return s.notify(ctx, MethodProgress, ProgressNotification{
		ProgressToken: token,
		Progress:      progress,
		Total:         total,
	})
}

// NotifyLoggingMessage sends a protocol logging notification to the connected client.
func (s *Server) NotifyLoggingMessage(ctx context.Context, level LoggingLevel, logger string, data any) error {
	levelValue, ok := slogLevelForLoggingLevel(level)
	if !ok {
		return NewParameterError(string(MethodLogging), "level", "unsupported logging level", nil)
	}

	s.mu.RLock()
	minLevel := s.logLevel
	s.mu.RUnlock()
	if minLevel == nil || levelValue < *minLevel {
		return nil
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal logging data: %w", err)
	}
	return s.notify(ctx, MethodLogging, LoggingMessageNotification{
		Level:  level,
		Logger: logger,
		Data:   dataJSON,
	})
}

// NotifyElicitationComplete tells the client an out-of-band elicitation completed.
func (s *Server) NotifyElicitationComplete(ctx context.Context, elicitationID string) error {
	if elicitationID == "" {
		return NewParameterError(string(MethodElicitationComplete), "elicitationId", "missing required elicitation id", nil)
	}
	return s.notify(ctx, MethodElicitationComplete, struct {
		ElicitationID string `json:"elicitationId"`
	}{
		ElicitationID: elicitationID,
	})
}

// CreateMessage sends a sampling request to the connected client.
func (s *Server) CreateMessage(ctx context.Context, request CreateMessageRequest) (*CreateMessageResult, error) {
	if s == nil {
		return nil, fmt.Errorf("server is nil")
	}

	s.mu.RLock()
	conn := s.conn
	supported := s.clientCaps.Sampling != nil
	s.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("mcp: client connection is not established")
	}
	if !supported {
		return nil, fmt.Errorf("%w: client does not support sampling", ErrUnsupported)
	}
	if request.Messages == nil {
		request.Messages = []SamplingMessage{}
	}

	ctx, cancel := s.requestContext(ctx)
	defer cancel()

	var result CreateMessageResult
	if err := conn.Call(ctx, string(MethodSamplingCreateMessage), request).Await(ctx, &result); err != nil {
		return nil, fmt.Errorf("sampling/createMessage: %w", err)
	}
	return &result, nil
}

// Elicit asks the connected client to collect non-sensitive information from the user.
func (s *Server) Elicit(ctx context.Context, request ElicitRequest) (*ElicitResult, error) {
	if s == nil {
		return nil, fmt.Errorf("server is nil")
	}

	s.mu.RLock()
	conn := s.conn
	caps := s.clientCaps.Elicitation
	s.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("mcp: client connection is not established")
	}
	if caps == nil {
		return nil, fmt.Errorf("%w: client does not support elicitation", ErrUnsupported)
	}
	if request.Mode == "" {
		if request.URL != "" || request.ElicitationID != "" {
			request.Mode = "url"
		} else {
			request.Mode = "form"
		}
	}
	switch request.Mode {
	case "form":
		if caps.Form == nil && caps.URL != nil {
			return nil, fmt.Errorf("%w: client does not support form elicitation", ErrUnsupported)
		}
	case "url":
		if caps.URL == nil {
			return nil, fmt.Errorf("%w: client does not support url elicitation", ErrUnsupported)
		}
	default:
		return nil, NewParameterError(string(MethodElicitationCreate), "mode", "unsupported elicitation mode", nil)
	}

	ctx, cancel := s.requestContext(ctx)
	defer cancel()

	var result ElicitResult
	if err := conn.Call(ctx, string(MethodElicitationCreate), request).Await(ctx, &result); err != nil {
		return nil, fmt.Errorf("elicitation/create: %w", err)
	}
	return &result, nil
}

// ListRoots asks the connected client for its current set of filesystem roots.
// The client must have advertised the roots capability during initialization.
func (s *Server) ListRoots(ctx context.Context) (*ListRootsResult, error) {
	if s == nil {
		return nil, fmt.Errorf("server is nil")
	}

	s.mu.RLock()
	conn := s.conn
	supported := s.clientCaps.Roots != nil
	s.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("mcp: client connection is not established")
	}
	if !supported {
		return nil, fmt.Errorf("%w: client does not support roots", ErrUnsupported)
	}

	ctx, cancel := s.requestContext(ctx)
	defer cancel()

	var result ListRootsResult
	if err := conn.Call(ctx, string(MethodRootsList), ListRootsRequest{}).Await(ctx, &result); err != nil {
		return nil, fmt.Errorf("roots/list: %w", err)
	}
	return &result, nil
}

// requestContext derives a context for a server-initiated request, applying the
// configured serverRequestTimeout unless the caller's context already carries an
// earlier deadline. The returned cancel func must always be called.
func (s *Server) requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	s.mu.RLock()
	timeout := s.serverRequestTimeout
	s.mu.RUnlock()
	if timeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= timeout {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *Server) notify(ctx context.Context, method MCPMethod, params any) error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.Notify(ctx, string(method), params)
}

// notifyListChanged sends a list_changed notification to the connected client.
// It no-ops when no client is connected (for example, when a tool, prompt, or
// resource is registered before Serve), so startup-time registration is silent.
func (s *Server) notifyListChanged(method MCPMethod) {
	if err := s.notify(context.Background(), method, struct{}{}); err != nil {
		s.logger.Debug("failed to send list changed notification", "method", string(method), "error", err)
	}
}

// Serve starts serving MCP requests using the provided transport.
// It establishes a JSON-RPC connection and handles incoming requests.
func (s *Server) Serve(ctx context.Context, transport Transport) error {
	// Default to stdio transport if none provided
	if transport == nil {
		transport = StdioTransport()
	}

	// Wrap transport to ensure flushing
	// We need to implement the Dialer interface which matches Transport's Dial method signature
	// but jsonrpc2.Dial expects a specific interface.
	// Since Transport matches Dialer interface, we can just wrap it.
	flushingd := &flushingDialer{
		dialer: transport,
		logger: s.logger,
	}

	// Create the connection with cancellation support
	handler := jsonrpc2.HandlerFunc(s.handleRequest)
	binder := serverBinder{handler: handler, logger: s.logger, framer: s.framer}
	conn, err := jsonrpc2.Dial(ctx, flushingd, binder)
	if err != nil {
		return fmt.Errorf("failed to establish connection: %w", err)
	}
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
		}
		s.mu.Unlock()
		conn.Close()
	}()

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
