// Package sdk2 provides a stdlib-idiomatic, type-safe Go API for the Model Context Protocol (MCP).
//
// This package follows Go standard library patterns and idioms, mirroring the design of net/http,
// database/sql, and other successful stdlib packages. It provides a clean and ergonomic API for
// building MCP clients and servers with strong typing and familiar interfaces.
//
// Server Example (http.Server-inspired):
//
//	// Simple server with handler registration
//	mux := sdk2.NewServeMux()
//	mux.HandleFunc("tools/call", func(w sdk2.ResponseWriter, r *sdk2.Request) {
//		var call sdk2.ToolCall
//		json.Unmarshal(r.Params, &call)
//		result := processCall(call)
//		json.NewEncoder(w).Encode(result)
//	})
//	
//	server := &sdk2.Server{
//		Addr:    ":stdio",
//		Handler: mux,
//	}
//	log.Fatal(server.ListenAndServe())
//
// Client Example (net.Dial-inspired):
//
//	client, err := sdk2.Dial(ctx, "stdio", "")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//	
//	tools, err := client.ListTools(ctx)
//	result, err := client.CallTool(ctx, "echo", map[string]any{"message": "hello"})
//
package sdk2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"
)

// Core interfaces following stdlib patterns

// Client provides MCP client operations, mirroring http.Client.
// It's safe for concurrent use by multiple goroutines.
type Client interface {
	// Do sends an MCP request and returns an MCP response, following
	// the net/http.Client pattern. This is the primitive method that
	// other methods are built upon.
	Do(req *Request) (*Response, error)
	
	// ListTools lists available tools from the server
	ListTools(ctx context.Context) ([]Tool, error)
	
	// CallTool executes a tool with the given arguments
	CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
	
	// ListResources lists available resources from the server
	ListResources(ctx context.Context) ([]Resource, error)
	
	// ReadResource reads content from a specific resource
	ReadResource(ctx context.Context, uri string) (*ResourceContent, error)
	
	// ListPrompts lists available prompts from the server
	ListPrompts(ctx context.Context) ([]Prompt, error)
	
	// GetPrompt retrieves a specific prompt with optional arguments
	GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error)
	
	// Close closes the client connection and cleans up resources
	Close() error
	
	// Ping verifies connectivity to the server (like database/sql)
	Ping(ctx context.Context) error
}

// Server provides MCP server operations, similar to http.Server.
type Server struct {
	// Addr optionally specifies the TCP address for the server to listen on,
	// in the form "host:port". If empty, ":stdio" is used (stdio transport).
	Addr string
	
	// Handler to invoke for MCP requests. If nil, DefaultMux is used.
	Handler Handler
	
	// serverInfo holds server information for handshake
	serverInfo *ServerInfo
	
	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. Zero means no timeout.
	ReadTimeout time.Duration
	
	// WriteTimeout is the maximum duration before timing out
	// writes of the response. Zero means no timeout.
	WriteTimeout time.Duration
	
	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. Zero means no timeout.
	IdleTimeout time.Duration
	
	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and values.
	MaxHeaderBytes int
	
	// ConnState specifies an optional callback function that is
	// called when a client connection changes state.
	ConnState func(net.Conn, ConnState)
	
	// BaseContext optionally specifies a function that returns
	// the base context for incoming requests on this server.
	BaseContext func(net.Listener) context.Context
	
	// ConnContext optionally specifies a function that modifies
	// the context used for a new connection.
	ConnContext func(ctx context.Context, c net.Conn) context.Context
}

// ConnState represents the state of a client connection to a server.
// This follows the same pattern as http.ConnState.
type ConnState int

const (
	// StateNew represents a new connection that is expected to
	// send a request immediately.
	StateNew ConnState = iota
	
	// StateActive represents a connection that has read 1 or more
	// bytes of a request.
	StateActive
	
	// StateIdle represents a connection that has finished
	// handling a request and is in the keep-alive state.
	StateIdle
	
	// StateClosed represents a closed connection.
	StateClosed
)

// String returns a string representation of the connection state.
func (c ConnState) String() string {
	switch c {
	case StateNew:
		return "new"
	case StateActive:
		return "active"
	case StateIdle:
		return "idle"
	case StateClosed:
		return "closed"
	default:
		return fmt.Sprintf("ConnState(%d)", int(c))
	}
}

// Handler responds to MCP requests.
//
// ServeRequest should write reply to ResponseWriter and then return.
// Returning signals that the request is finished; it is not valid to use the
// ResponseWriter or read from the Request.Context after or concurrently with
// the completion of the ServeRequest call.
//
// This follows the exact same pattern as http.Handler.
type Handler interface {
	ServeRequest(ResponseWriter, *Request)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as MCP handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f. This follows the http.HandlerFunc pattern.
type HandlerFunc func(ResponseWriter, *Request)

// ServeRequest calls f(w, r).
func (f HandlerFunc) ServeRequest(w ResponseWriter, r *Request) {
	f(w, r)
}

// Request represents an MCP request received by a server or to be sent by a client.
// This follows the http.Request pattern closely.
type Request struct {
	// Method specifies the MCP method (e.g., "tools/list", "tools/call")
	Method string
	
	// Params holds the parameters for this request
	Params json.RawMessage
	
	// ID is the JSON-RPC request ID. Nil for notifications.
	ID *RequestID
	
	// Context is the request's context. To change the context, use WithContext.
	Context context.Context
	
	// Proto is the protocol version (e.g., "2025-03-26").
	Proto string
	
	// Header contains the request header fields
	Header Header
	
	// RemoteAddr allows HTTP servers and other software to record
	// the network address that sent the request
	RemoteAddr string
	
	// RequestURI is the unmodified request-target sent by the client
	RequestURI string
}

// WithContext returns a shallow copy of r with its context changed to ctx.
// This follows the http.Request.WithContext pattern.
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.Context = ctx
	return r2
}

// URL returns the parsed request URI as a URL. This is useful for
// resource URIs and other URL-based parameters.
func (r *Request) URL() (*url.URL, error) {
	if r.RequestURI == "" {
		return nil, fmt.Errorf("no request URI")
	}
	return url.Parse(r.RequestURI)
}

// RequestID represents a JSON-RPC request ID that can be a string, number, or null.
// This handles the polymorphic nature of JSON-RPC IDs properly.
type RequestID struct {
	Value any // string, int64, float64, or nil
}

// MarshalJSON implements json.Marshaler.
func (id RequestID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.Value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *RequestID) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &id.Value)
}

// String returns a string representation of the request ID.
func (id RequestID) String() string {
	if id.Value == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", id.Value)
}

// IsNil returns true if the request ID is nil (for notifications).
func (id RequestID) IsNil() bool {
	return id.Value == nil
}

// ResponseWriter interface is used by an MCP handler to construct an MCP response.
// This follows the http.ResponseWriter pattern exactly.
type ResponseWriter interface {
	// Header returns the header map that will be sent by WriteHeader.
	Header() Header
	
	// Write writes the data to the connection as part of an MCP reply.
	Write([]byte) (int, error)
	
	// WriteHeader sends an MCP response header with the provided status code.
	WriteHeader(statusCode int)
}

// Header represents the key-value pairs in an MCP header.
// This follows the http.Header pattern.
type Header map[string][]string

// Set sets the header entries associated with key to the single element value.
func (h Header) Set(key, value string) {
	h[key] = []string{value}
}

// Get gets the first value associated with the given key.
func (h Header) Get(key string) string {
	if h == nil {
		return ""
	}
	v := h[key]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// Add adds the key, value pair to the header.
func (h Header) Add(key, value string) {
	h[key] = append(h[key], value)
}

// Del deletes the values associated with key.
func (h Header) Del(key string) {
	delete(h, key)
}

// Values returns all values associated with the given key.
func (h Header) Values(key string) []string {
	if h == nil {
		return nil
	}
	return h[key]
}

// Has reports whether h has the given key defined.
func (h Header) Has(key string) bool {
	_, ok := h[key]
	return ok
}

// Clone returns a copy of h or nil if h is nil.
func (h Header) Clone() Header {
	if h == nil {
		return nil
	}
	h2 := make(Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}

// Response represents the response from an MCP request.
// This follows the http.Response pattern.
type Response struct {
	// Status is the response status text.
	Status string // e.g. "200 OK"
	
	// StatusCode is the response status code.
	StatusCode int // e.g. 200
	
	// Proto is the protocol version.
	Proto string // e.g. "MCP/2025-03-26"
	
	// Header maps header keys to values.
	Header Header
	
	// Body represents the response body.
	Body io.ReadCloser
	
	// ContentLength records the length of the associated content.
	ContentLength int64
	
	// Request is the request that was sent to obtain this Response.
	Request *Request
}

// Core protocol types

// Tool represents a tool definition with its schema and metadata.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Validate validates the tool definition.
func (t Tool) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if t.Description == "" {
		return fmt.Errorf("tool description cannot be empty")
	}
	// Could add schema validation here
	return nil
}

// ToolCall represents a tool execution request.
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// Validate validates the tool call.
func (tc ToolCall) Validate() error {
	if tc.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	return nil
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Validate validates the tool result.
func (tr ToolResult) Validate() error {
	if len(tr.Content) == 0 {
		return fmt.Errorf("tool result must have at least one content item")
	}
	for i, content := range tr.Content {
		if err := content.Valid(); err != nil {
			return fmt.Errorf("content item %d: %w", i, err)
		}
	}
	return nil
}

// Resource represents a resource definition.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Validate validates the resource definition.
func (r Resource) Validate() error {
	if r.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}
	if r.Name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}
	if _, err := url.Parse(r.URI); err != nil {
		return fmt.Errorf("invalid resource URI: %w", err)
	}
	return nil
}

// ResourceRequest represents a resource read request.
type ResourceRequest struct {
	URI string `json:"uri"`
}

// Validate validates the resource request.
func (rr ResourceRequest) Validate() error {
	if rr.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}
	if _, err := url.Parse(rr.URI); err != nil {
		return fmt.Errorf("invalid resource URI: %w", err)
	}
	return nil
}

// ResourceContent represents the content of a resource.
type ResourceContent struct {
	URI      string    `json:"uri"`
	MimeType string    `json:"mimeType,omitempty"`
	Content  []Content `json:"content"`
}

// Validate validates the resource content.
func (rc ResourceContent) Validate() error {
	if rc.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}
	if len(rc.Content) == 0 {
		return fmt.Errorf("resource content must have at least one content item")
	}
	for i, content := range rc.Content {
		if err := content.Valid(); err != nil {
			return fmt.Errorf("content item %d: %w", i, err)
		}
	}
	return nil
}

// Prompt represents a prompt definition.
type Prompt struct {
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	ArgumentsSchema json.RawMessage `json:"argumentsSchema,omitempty"`
}

// Validate validates the prompt definition.
func (p Prompt) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}
	return nil
}

// PromptRequest represents a prompt request.
type PromptRequest struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// Validate validates the prompt request.
func (pr PromptRequest) Validate() error {
	if pr.Name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}
	return nil
}

// PromptResult represents the result of a prompt request.
type PromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// Validate validates the prompt result.
func (pr PromptResult) Validate() error {
	if len(pr.Messages) == 0 {
		return fmt.Errorf("prompt result must have at least one message")
	}
	for i, msg := range pr.Messages {
		if err := msg.Validate(); err != nil {
			return fmt.Errorf("message %d: %w", i, err)
		}
	}
	return nil
}

// PromptMessage represents a message in a prompt result.
type PromptMessage struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// Validate validates the prompt message.
func (pm PromptMessage) Validate() error {
	if pm.Role == "" {
		return fmt.Errorf("message role cannot be empty")
	}
	if len(pm.Content) == 0 {
		return fmt.Errorf("message must have at least one content item")
	}
	for i, content := range pm.Content {
		if err := content.Valid(); err != nil {
			return fmt.Errorf("content item %d: %w", i, err)
		}
	}
	return nil
}

// Content represents various types of content (text, image, etc.).
// This is a sealed interface - only types in this package can implement it.
type Content interface {
	// ContentType returns the MIME type of the content
	ContentType() string
	// mcpContent is unexported to seal the interface
	mcpContent()
	// Valid validates the content structure
	Valid() error
}

// TextContent represents textual content.
type TextContent struct {
	Text string `json:"text"`
}

// ContentType implements Content.
func (TextContent) ContentType() string { return "text/plain" }

// mcpContent implements Content (sealed interface).
func (TextContent) mcpContent() {}

// Valid implements Content.
func (t TextContent) Valid() error {
	if t.Text == "" {
		return fmt.Errorf("text content cannot be empty")
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (t TextContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		Type: "text",
		Text: t.Text,
	})
}

// ImageContent represents image content.
type ImageContent struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

// ContentType implements Content.
func (i ImageContent) ContentType() string { return i.MimeType }

// mcpContent implements Content (sealed interface).
func (ImageContent) mcpContent() {}

// Valid implements Content.
func (i ImageContent) Valid() error {
	if i.Data == "" {
		return fmt.Errorf("image data cannot be empty")
	}
	if i.MimeType == "" {
		return fmt.Errorf("image mime type cannot be empty")
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (i ImageContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type     string `json:"type"`
		Data     string `json:"data"`
		MimeType string `json:"mimeType"`
	}{
		Type:     "image",
		Data:     i.Data,
		MimeType: i.MimeType,
	})
}

// ResourceReferenceContent represents content that references an external resource.
type ResourceReferenceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
}

// ContentType implements Content.
func (r ResourceReferenceContent) ContentType() string {
	if r.MimeType != "" {
		return r.MimeType
	}
	return "text/plain"
}

// mcpContent implements Content (sealed interface).
func (ResourceReferenceContent) mcpContent() {}

// Valid implements Content.
func (r ResourceReferenceContent) Valid() error {
	if r.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}
	if _, err := url.Parse(r.URI); err != nil {
		return fmt.Errorf("invalid resource URI: %w", err)
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (r ResourceReferenceContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type     string `json:"type"`
		URI      string `json:"uri"`
		MimeType string `json:"mimeType,omitempty"`
	}{
		Type:     "resource",
		URI:      r.URI,
		MimeType: r.MimeType,
	})
}

// UnmarshalContent unmarshals JSON data into the appropriate Content type.
func UnmarshalContent(data []byte) (Content, error) {
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("unmarshal content type: %w", err)
	}
	
	switch base.Type {
	case "text":
		var t struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("unmarshal text content: %w", err)
		}
		return TextContent{Text: t.Text}, nil
		
	case "image":
		var i struct {
			Data     string `json:"data"`
			MimeType string `json:"mimeType"`
		}
		if err := json.Unmarshal(data, &i); err != nil {
			return nil, fmt.Errorf("unmarshal image content: %w", err)
		}
		return ImageContent{Data: i.Data, MimeType: i.MimeType}, nil
		
	case "resource":
		var r struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType"`
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("unmarshal resource content: %w", err)
		}
		return ResourceReferenceContent{URI: r.URI, MimeType: r.MimeType}, nil
		
	default:
		return nil, fmt.Errorf("unknown content type: %s", base.Type)
	}
}

// Configuration types following stdlib patterns

// ClientConfig configures client behavior. This follows the pattern of
// tls.Config, http.Transport, etc. for configuration structs.
type ClientConfig struct {
	// Timeout for each request. Zero means no timeout.
	Timeout time.Duration
	
	// Maximum number of retry attempts.
	MaxRetries int
	
	// Delay between retry attempts.
	RetryDelay time.Duration
	
	// Handler for server notifications.
	NotificationHandler NotificationHandler
	
	// Client information sent during handshake.
	ClientInfo ClientInfo
	
	// Optional transport-specific configuration
	Transport RoundTripper
}

// RoundTripper is an interface representing the ability to execute a
// single MCP transaction, obtaining the Response for a given Request.
// This mirrors http.RoundTripper exactly.
type RoundTripper interface {
	// RoundTrip executes a single MCP transaction, returning
	// a Response for the provided Request.
	RoundTrip(*Request) (*Response, error)
}

// NotificationHandler handles notifications from the server.
type NotificationHandler interface {
	HandleNotification(ctx context.Context, method string, params json.RawMessage) error
}

// NotificationHandlerFunc is an adapter to allow the use of ordinary functions as NotificationHandlers.
type NotificationHandlerFunc func(context.Context, string, json.RawMessage) error

// HandleNotification calls f(ctx, method, params).
func (f NotificationHandlerFunc) HandleNotification(ctx context.Context, method string, params json.RawMessage) error {
	return f(ctx, method, params)
}

// ClientInfo contains information about the client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerConfig configures server behavior.
type ServerConfig struct {
	// Server information sent during handshake.
	ServerInfo ServerInfo
	
	// Capabilities supported by the server.
	Capabilities ServerCapabilities
}

// ServerInfo contains information about the server.
type ServerInfo struct {
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	Capabilities *ServerCapabilities `json:"capabilities,omitempty"`
}

// ServerCapabilities defines what capabilities the server supports.
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

// ToolsCapability defines tool-related capabilities.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability defines resource-related capabilities.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability defines prompt-related capabilities.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability defines logging-related capabilities.
type LoggingCapability struct {
	Level string `json:"level,omitempty"`
}

// Transport abstractions following net.Conn patterns

// Conn represents a generic stream-oriented network connection.
// This follows the net.Conn interface pattern exactly.
type Conn interface {
	io.Reader
	io.Writer
	io.Closer
	
	// LocalAddr returns the local network address.
	LocalAddr() net.Addr
	
	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr
	
	// SetDeadline sets the read and write deadlines associated
	// with the connection.
	SetDeadline(t time.Time) error
	
	// SetReadDeadline sets the deadline for future Read calls.
	SetReadDeadline(t time.Time) error
	
	// SetWriteDeadline sets the deadline for future Write calls.
	SetWriteDeadline(t time.Time) error
}

// Listener represents a generic network listener.
// This follows the net.Listener interface pattern exactly.
type Listener interface {
	// Accept waits for and returns the next connection to the listener.
	Accept() (Conn, error)
	
	// Close closes the listener.
	Close() error
	
	// Addr returns the listener's network address.
	Addr() net.Addr
}

// Dialer contains options for connecting to a server.
// This follows the net.Dialer pattern exactly.
type Dialer struct {
	// Timeout is the maximum amount of time a dial will wait for
	// a connect to complete.
	Timeout time.Duration
	
	// Deadline is the absolute point in time after which dials will fail.
	Deadline time.Time
	
	// LocalAddr is the local address to use when dialing an address.
	LocalAddr net.Addr
	
	// KeepAlive specifies the interval between keep-alive probes
	// for an active network connection.
	KeepAlive time.Duration
	
	// Control is called after creating the network connection
	// but before actually dialing.
	Control func(network, address string, c interface{}) error
}

// Constants for MCP protocol

const (
	// ProtocolVersion is the MCP protocol version this SDK supports.
	ProtocolVersion = "2025-03-26"
	
	// DefaultPort is the default port for MCP over TCP.
	DefaultPort = "3000"
)

// Well-known MCP methods
const (
	MethodInitialize     = "initialize"
	MethodInitialized    = "notifications/initialized"
	MethodToolsList      = "tools/list"
	MethodToolsCall      = "tools/call"
	MethodResourcesList  = "resources/list"
	MethodResourcesRead  = "resources/read"
	MethodPromptsList    = "prompts/list"
	MethodPromptsGet     = "prompts/get"
	MethodLoggingLog     = "logging/setLevel"
	MethodProgress       = "notifications/progress"
	MethodCancelled      = "notifications/cancelled"
)

// Option types following the functional options pattern

// Option configures Client and Server instances using the functional options pattern.
type Option func(interface{})

// ClientOption configures a Client.
type ClientOption func(*ClientConfig)

// WithTimeout sets the request timeout for the client.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *ClientConfig) {
		c.Timeout = timeout
	}
}

// WithRetries sets the retry policy for the client.
func WithRetries(maxRetries int, delay time.Duration) ClientOption {
	return func(c *ClientConfig) {
		c.MaxRetries = maxRetries
		c.RetryDelay = delay
	}
}

// WithClientInfo sets the client information sent during handshake.
func WithClientInfo(name, version string) ClientOption {
	return func(c *ClientConfig) {
		c.ClientInfo = ClientInfo{Name: name, Version: version}
	}
}

// WithNotificationHandler sets the handler for server notifications.
func WithNotificationHandler(handler NotificationHandler) ClientOption {
	return func(c *ClientConfig) {
		c.NotificationHandler = handler
	}
}

// WithTransport sets a custom transport for the client.
func WithTransport(transport RoundTripper) ClientOption {
	return func(c *ClientConfig) {
		c.Transport = transport
	}
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithServerInfo sets the server information sent during handshake.
func WithServerInfo(name, version string) ServerOption {
	return func(s *Server) {
		if s.serverInfo == nil {
			s.serverInfo = &ServerInfo{}
		}
		s.serverInfo.Name = name
		s.serverInfo.Version = version
	}
}

// WithCapabilities sets the server capabilities.
func WithCapabilities(caps *ServerCapabilities) ServerOption {
	return func(s *Server) {
		if s.serverInfo == nil {
			s.serverInfo = &ServerInfo{}
		}
		s.serverInfo.Capabilities = caps
	}
}

// WithHandler sets the request handler for the server.
func WithHandler(handler Handler) ServerOption {
	return func(s *Server) {
		s.Handler = handler
	}
}

// WithTimeouts sets read and write timeouts for the server.
func WithTimeouts(read, write time.Duration) ServerOption {
	return func(s *Server) {
		s.ReadTimeout = read
		s.WriteTimeout = write
	}
}