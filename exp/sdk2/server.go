package sdk2

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
)

// Note: NewServer is defined in constructor.go to avoid duplication

// ListenAndServe listens on the TCP network address srv.Addr and then
// calls Serve to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
//
// If srv.Addr is blank, ":stdio" is used.
//
// ListenAndServe always returns a non-nil error. After Shutdown or Close,
// the returned error is ErrServerClosed.
func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":stdio"
	}
	
	if addr == ":stdio" {
		// Use stdio listener
		ln := newStdioListener()
		return srv.Serve(ln)
	}
	
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(&netListener{Listener: ln})
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each. The service goroutines read requests and
// then call srv.Handler to reply to them.
//
// Serve always returns a non-nil error and closes l.
func (srv *Server) Serve(l Listener) error {
	defer l.Close()
	
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		
		go srv.handleConnection(conn)
	}
}

// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = NewServeMux()

// Package-level convenience functions following http package patterns

// Handle registers the handler for the given pattern in the DefaultServeMux.
// The documentation for ServeMux explains how patterns are matched.
func Handle(pattern string, handler Handler) {
	DefaultServeMux.Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern in DefaultServeMux.
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}

// ListenAndServe listens on the address and serves using DefaultServeMux.
// If addr is blank, ":stdio" is used.
//
// Example:
//   sdk2.HandleFunc("tools/call", myHandler)
//   sdk2.ListenAndServe(":stdio")
func ListenAndServe(addr string) error {
	server := &Server{
		Addr:    addr,
		Handler: DefaultServeMux,
	}
	return server.ListenAndServe()
}

// Serve accepts incoming connections on the listener l and serves using DefaultServeMux.
func Serve(l Listener) error {
	server := &Server{Handler: DefaultServeMux}
	return server.Serve(l)
}

// Error replies to the request with the specified error message and status code.
// It does not otherwise end the request; the caller should ensure no further
// writes are done to w. Like http.Error.
func Error(w ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	errResp := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": error,
		},
	}
	json.NewEncoder(w).Encode(errResp)
}

// NotFound replies to the request with an MCP 404 not found error.
// Like http.NotFound.
func NotFound(w ResponseWriter, r *Request) {
	Error(w, "not found", StatusNotFound)
}

// MethodNotAllowed replies to the request with an MCP 405 method not allowed error.
// Like http.MethodNotAllowed.
func MethodNotAllowed(w ResponseWriter, r *Request) {
	Error(w, "method not allowed", StatusMethodNotAllowed)
}

// ServeMux is an MCP request multiplexer.
type ServeMux struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// Note: NewServeMux is defined in constructor.go to avoid duplication

// Handle registers the handler for the given pattern.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	
	if pattern == "" {
		panic("mcp: invalid pattern")
	}
	if handler == nil {
		panic("mcp: nil handler")
	}
	
	mux.handlers[pattern] = handler
}

// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("mcp: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler))
}

// ServeRequest dispatches the request to the handler whose
// pattern most closely matches the request method.
func (mux *ServeMux) ServeRequest(w ResponseWriter, r *Request) {
	mux.mu.RLock()
	handler, exists := mux.handlers[r.Method]
	mux.mu.RUnlock()
	
	if !exists {
		// Try DefaultMux as fallback
		NotFound(w, r)
		return
	}
	
	handler.ServeRequest(w, r)
}

// Handle registers the handler for the given pattern in the DefaultServeMux.
func Handle(pattern string, handler Handler) {
	DefaultServeMux.Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern in the DefaultServeMux.
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}

// A HandlerFunc is an adapter to allow the use of ordinary functions as MCP handlers.
type HandlerFunc func(ResponseWriter, *Request)

// ServeRequest calls f(w, r).
func (f HandlerFunc) ServeRequest(w ResponseWriter, r *Request) {
	f(w, r)
}

// Error replies to the request with the specified error message and MCP code.
func Error(w ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintln(w, error)
}

// NotFound replies to the request with an MCP 404 not found error.
func NotFound(w ResponseWriter, r *Request) {
	Error(w, "404 method not found", StatusNotFound)
}

// NotFoundHandler returns a simple request handler that replies to each request with a "404 method not found" reply.
func NotFoundHandler() Handler {
	return HandlerFunc(NotFound)
}

// handleConnection handles a single connection
func (srv *Server) handleConnection(conn Conn) {
	defer conn.Close()
	
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	
	// Simple JSON-RPC message handling
	for {
		// Read a line (JSON-RPC message)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			slog.Error("Failed to read message", "error", err)
			return
		}
		
		// Parse JSON-RPC request
		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			slog.Error("Failed to parse JSON-RPC request", "error", err)
			continue
		}
		
		// Handle the request
		srv.handleRequest(writer, &req)
	}
}

// handleRequest processes a single JSON-RPC request
func (srv *Server) handleRequest(writer *bufio.Writer, req *jsonrpcRequest) {
	// Create MCP request
	mcpReq := &Request{
		Method: req.Method,
		Params: req.Params,
		Proto:  ProtocolVersion,
	}
	if req.ID != nil {
		mcpReq.ID = &RequestID{Value: req.ID}
	}
	mcpReq.Context = context.Background()
	
	// Create response writer
	rw := &responseWriter{
		writer: writer,
		header: make(Header),
		id:     req.ID,
	}
	
	// Handle special methods
	switch req.Method {
	case MethodInitialize:
		srv.handleInitialize(rw, mcpReq)
	case MethodInitialized:
		srv.handleInitialized(rw, mcpReq)
	default:
		// Use the handler
		handler := srv.Handler
		if handler == nil {
			handler = DefaultServeMux
		}
		handler.ServeRequest(rw, mcpReq)
	}
	
	writer.Flush()
}

// handleInitialize handles the initialize request
func (srv *Server) handleInitialize(w ResponseWriter, r *Request) {
	// Parse initialize parameters
	var params struct {
		ProtocolVersion string     `json:"protocolVersion"`
		ClientInfo      ClientInfo `json:"clientInfo"`
	}
	
	if r.Params != nil {
		if err := json.Unmarshal(r.Params, &params); err != nil {
			Error(w, "Invalid initialize parameters", StatusBadRequest)
			return
		}
	}
	
	// Create response using server info or defaults
	result := ServerInfo{
		Name:    "sdk2-server",
		Version: "0.1.0",
		Capabilities: &ServerCapabilities{
			Tools:     &ToolsCapability{},
			Resources: &ResourcesCapability{},
			Prompts:   &PromptsCapability{},
			Logging:   &LoggingCapability{},
		},
	}
	
	// Override with configured server info if available
	if srv.serverInfo != nil {
		if srv.serverInfo.Name != "" {
			result.Name = srv.serverInfo.Name
		}
		if srv.serverInfo.Version != "" {
			result.Version = srv.serverInfo.Version
		}
		if srv.serverInfo.Capabilities != nil {
			result.Capabilities = srv.serverInfo.Capabilities
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleInitialized handles the initialized notification
func (srv *Server) handleInitialized(w ResponseWriter, r *Request) {
	// This is a notification, no response needed
	slog.Info("Client initialized")
}

// responseWriter implements ResponseWriter
type responseWriter struct {
	writer *bufio.Writer
	header Header
	id     any
	
	wroteHeader bool
	statusCode  int
}

func (w *responseWriter) Header() Header {
	return w.header
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(StatusOK)
	}
	
	// For JSON-RPC, we need to wrap the response
	if w.id != nil {
		resp := jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      w.id,
			Result:  json.RawMessage(data),
		}
		
		respData, err := json.Marshal(resp)
		if err != nil {
			return 0, err
		}
		
		// Add newline for line-delimited JSON
		respData = append(respData, '\n')
		return w.writer.Write(respData)
	}
	
	return w.writer.Write(data)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = statusCode
	
	// For error status codes, send JSON-RPC error
	if statusCode >= 400 && w.id != nil {
		resp := jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      w.id,
			Error: &jsonrpcError{
				Code:    statusCode,
				Message: StatusText(statusCode),
			},
		}
		
		respData, _ := json.Marshal(resp)
		respData = append(respData, '\n')
		w.writer.Write(respData)
	}
}

// netListener wraps net.Listener to implement our Listener interface
type netListener struct {
	net.Listener
}

func (l *netListener) Accept() (Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &netConn{Conn: conn}, nil
}

func (l *netListener) Addr() net.Addr {
	return l.Listener.Addr()
}

// stdioListener implements Listener for stdio transport
type stdioListener struct {
	once   sync.Once
	closed bool
	mu     sync.Mutex
}

func newStdioListener() *stdioListener {
	return &stdioListener{}
}

func (l *stdioListener) Accept() (Conn, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, fmt.Errorf("listener closed")
	}
	l.mu.Unlock()
	
	// For stdio, we only accept one connection
	var conn Conn
	l.once.Do(func() {
		conn = &stdioConn{
			reader: os.Stdin,
			writer: os.Stdout,
		}
	})
	
	if conn == nil {
		// Already served one connection
		return nil, fmt.Errorf("stdio listener only accepts one connection")
	}
	
	return conn, nil
}

func (l *stdioListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}

func (l *stdioListener) Addr() net.Addr {
	return &stdioAddr{}
}

// Status codes for MCP responses
const (
	StatusOK                  = 200
	StatusBadRequest          = 400
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusInternalServerError = 500
	StatusServiceUnavailable  = 503
)

var statusText = map[int]string{
	StatusOK:                  "OK",
	StatusBadRequest:          "Bad Request",
	StatusNotFound:            "Not Found",
	StatusMethodNotAllowed:    "Method Not Allowed",
	StatusInternalServerError: "Internal Server Error",
	StatusServiceUnavailable:  "Service Unavailable",
}

// StatusText returns a text for the MCP status code.
func StatusText(code int) string {
	if text, ok := statusText[code]; ok {
		return text
	}
	return fmt.Sprintf("Unknown Status Code %d", code)
}

// WithContext returns a shallow copy of r with its context changed to ctx.
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.Context = ctx
	return r2
}