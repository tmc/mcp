package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// StdioFactory creates stdio transports.
type StdioFactory struct{}

// Create creates a new stdio transport.
func (f *StdioFactory) Create(config Config) (Transport, error) {
	return &StdioTransport{config: config}, nil
}

// Type returns the transport type.
func (f *StdioFactory) Type() string {
	return "stdio"
}

// ValidateConfig validates stdio transport configuration.
func (f *StdioFactory) ValidateConfig(config Config) error {
	// Stdio transport requires minimal configuration
	return nil
}

// StdioTransport implements stdio transport.
type StdioTransport struct {
	config Config
	cmd    *exec.Cmd
}

// Dial establishes a stdio connection.
func (t *StdioTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	// Check if command is specified
	if command, exists := t.config.Parameters["command"]; exists {
		cmdStr := command.(string)
		var args []string
		
		if argsParam, exists := t.config.Parameters["args"]; exists {
			if argsStr, ok := argsParam.(string); ok {
				args = strings.Fields(argsStr)
			}
		}
		
		// Create command
		t.cmd = exec.CommandContext(ctx, cmdStr, args...)
		
		// Get pipes
		stdin, err := t.cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
		}
		
		stdout, err := t.cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		
		// Start command
		if err := t.cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start command: %w", err)
		}
		
		return &stdioConn{
			stdin:  stdin,
			stdout: stdout,
			cmd:    t.cmd,
		}, nil
	}
	
	// Use current process stdin/stdout
	return &stdioConn{
		stdin:  os.Stdin,
		stdout: os.Stdout,
	}, nil
}

// Name returns the transport name.
func (t *StdioTransport) Name() string {
	return "stdio"
}

// Type returns the transport type.
func (t *StdioTransport) Type() string {
	return "stdio"
}

// Config returns the transport configuration.
func (t *StdioTransport) Config() Config {
	return t.config
}

// Health checks the transport health.
func (t *StdioTransport) Health(ctx context.Context) error {
	if t.cmd != nil {
		// Check if command is still running
		if t.cmd.Process == nil {
			return fmt.Errorf("command not started")
		}
		
		// Check process state
		if t.cmd.ProcessState != nil && t.cmd.ProcessState.Exited() {
			return fmt.Errorf("command exited")
		}
	}
	
	return nil
}

// Close closes the transport.
func (t *StdioTransport) Close() error {
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}

// stdioConn represents a stdio connection.
type stdioConn struct {
	stdin  io.Reader
	stdout io.Writer
	cmd    *exec.Cmd
}

// Read reads from stdin.
func (c *stdioConn) Read(p []byte) (int, error) {
	return c.stdin.Read(p)
}

// Write writes to stdout.
func (c *stdioConn) Write(p []byte) (int, error) {
	return c.stdout.Write(p)
}

// Close closes the connection.
func (c *stdioConn) Close() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// HTTPFactory creates HTTP transports.
type HTTPFactory struct{}

// Create creates a new HTTP transport.
func (f *HTTPFactory) Create(config Config) (Transport, error) {
	return &HTTPTransport{config: config}, nil
}

// Type returns the transport type.
func (f *HTTPFactory) Type() string {
	return "http"
}

// ValidateConfig validates HTTP transport configuration.
func (f *HTTPFactory) ValidateConfig(config Config) error {
	if _, exists := config.Parameters["url"]; !exists {
		return fmt.Errorf("url parameter is required for HTTP transport")
	}
	return nil
}

// HTTPTransport implements HTTP transport.
type HTTPTransport struct {
	config Config
	client *http.Client
}

// Dial establishes an HTTP connection.
func (t *HTTPTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	urlStr := t.config.Parameters["url"].(string)
	
	// Create HTTP client if not exists
	if t.client == nil {
		t.client = &http.Client{
			Timeout: t.config.Timeout,
		}
	}
	
	return &httpConn{
		url:    urlStr,
		client: t.client,
		ctx:    ctx,
	}, nil
}

// Name returns the transport name.
func (t *HTTPTransport) Name() string {
	return "http"
}

// Type returns the transport type.
func (t *HTTPTransport) Type() string {
	return "http"
}

// Config returns the transport configuration.
func (t *HTTPTransport) Config() Config {
	return t.config
}

// Health checks the transport health.
func (t *HTTPTransport) Health(ctx context.Context) error {
	urlStr := t.config.Parameters["url"].(string)
	
	req, err := http.NewRequestWithContext(ctx, "HEAD", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// Close closes the transport.
func (t *HTTPTransport) Close() error {
	if t.client != nil {
		t.client.CloseIdleConnections()
	}
	return nil
}

// httpConn represents an HTTP connection.
type httpConn struct {
	url    string
	client *http.Client
	ctx    context.Context
}

// Read reads from HTTP response.
func (c *httpConn) Read(p []byte) (int, error) {
	// HTTP is request-response, so this is not directly applicable
	return 0, fmt.Errorf("HTTP transport does not support streaming reads")
}

// Write writes HTTP request.
func (c *httpConn) Write(p []byte) (int, error) {
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.url, strings.NewReader(string(p)))
	if err != nil {
		return 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}
	
	return len(p), nil
}

// Close closes the connection.
func (c *httpConn) Close() error {
	return nil
}

// WebSocketFactory creates WebSocket transports.
type WebSocketFactory struct{}

// Create creates a new WebSocket transport.
func (f *WebSocketFactory) Create(config Config) (Transport, error) {
	return &WebSocketTransport{config: config}, nil
}

// Type returns the transport type.
func (f *WebSocketFactory) Type() string {
	return "websocket"
}

// ValidateConfig validates WebSocket transport configuration.
func (f *WebSocketFactory) ValidateConfig(config Config) error {
	if _, exists := config.Parameters["url"]; !exists {
		return fmt.Errorf("url parameter is required for WebSocket transport")
	}
	return nil
}

// WebSocketTransport implements WebSocket transport.
type WebSocketTransport struct {
	config Config
	dialer *websocket.Dialer
}

// Dial establishes a WebSocket connection.
func (t *WebSocketTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	urlStr := t.config.Parameters["url"].(string)
	
	// Create dialer if not exists
	if t.dialer == nil {
		t.dialer = &websocket.Dialer{
			HandshakeTimeout: t.config.Timeout,
		}
	}
	
	conn, _, err := t.dialer.DialContext(ctx, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial WebSocket: %w", err)
	}
	
	return &websocketConn{conn: conn}, nil
}

// Name returns the transport name.
func (t *WebSocketTransport) Name() string {
	return "websocket"
}

// Type returns the transport type.
func (t *WebSocketTransport) Type() string {
	return "websocket"
}

// Config returns the transport configuration.
func (t *WebSocketTransport) Config() Config {
	return t.config
}

// Health checks the transport health.
func (t *WebSocketTransport) Health(ctx context.Context) error {
	// Try to establish a connection
	conn, err := t.Dial(ctx)
	if err != nil {
		return fmt.Errorf("WebSocket health check failed: %w", err)
	}
	defer conn.Close()
	
	return nil
}

// Close closes the transport.
func (t *WebSocketTransport) Close() error {
	return nil
}

// websocketConn represents a WebSocket connection.
type websocketConn struct {
	conn *websocket.Conn
}

// Read reads from WebSocket.
func (c *websocketConn) Read(p []byte) (int, error) {
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return 0, fmt.Errorf("failed to read WebSocket message: %w", err)
	}
	
	n := copy(p, message)
	return n, nil
}

// Write writes to WebSocket.
func (c *websocketConn) Write(p []byte) (int, error) {
	if err := c.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, fmt.Errorf("failed to write WebSocket message: %w", err)
	}
	return len(p), nil
}

// Close closes the connection.
func (c *websocketConn) Close() error {
	return c.conn.Close()
}

// TCPFactory creates TCP transports.
type TCPFactory struct{}

// Create creates a new TCP transport.
func (f *TCPFactory) Create(config Config) (Transport, error) {
	return &TCPTransport{config: config}, nil
}

// Type returns the transport type.
func (f *TCPFactory) Type() string {
	return "tcp"
}

// ValidateConfig validates TCP transport configuration.
func (f *TCPFactory) ValidateConfig(config Config) error {
	if _, exists := config.Parameters["address"]; !exists {
		return fmt.Errorf("address parameter is required for TCP transport")
	}
	return nil
}

// TCPTransport implements TCP transport.
type TCPTransport struct {
	config Config
}

// Dial establishes a TCP connection.
func (t *TCPTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	address := t.config.Parameters["address"].(string)
	network := "tcp"
	
	if networkParam, exists := t.config.Parameters["network"]; exists {
		network = networkParam.(string)
	}
	
	var dialer net.Dialer
	if t.config.Timeout > 0 {
		dialer.Timeout = t.config.Timeout
	}
	
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial TCP: %w", err)
	}
	
	return conn, nil
}

// Name returns the transport name.
func (t *TCPTransport) Name() string {
	return "tcp"
}

// Type returns the transport type.
func (t *TCPTransport) Type() string {
	return "tcp"
}

// Config returns the transport configuration.
func (t *TCPTransport) Config() Config {
	return t.config
}

// Health checks the transport health.
func (t *TCPTransport) Health(ctx context.Context) error {
	// Try to establish a connection
	conn, err := t.Dial(ctx)
	if err != nil {
		return fmt.Errorf("TCP health check failed: %w", err)
	}
	defer conn.Close()
	
	return nil
}

// Close closes the transport.
func (t *TCPTransport) Close() error {
	return nil
}

// UnixFactory creates Unix domain socket transports.
type UnixFactory struct{}

// Create creates a new Unix transport.
func (f *UnixFactory) Create(config Config) (Transport, error) {
	return &UnixTransport{config: config}, nil
}

// Type returns the transport type.
func (f *UnixFactory) Type() string {
	return "unix"
}

// ValidateConfig validates Unix transport configuration.
func (f *UnixFactory) ValidateConfig(config Config) error {
	if _, exists := config.Parameters["path"]; !exists {
		return fmt.Errorf("path parameter is required for Unix transport")
	}
	return nil
}

// UnixTransport implements Unix domain socket transport.
type UnixTransport struct {
	config Config
}

// Dial establishes a Unix domain socket connection.
func (t *UnixTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	path := t.config.Parameters["path"].(string)
	
	var dialer net.Dialer
	if t.config.Timeout > 0 {
		dialer.Timeout = t.config.Timeout
	}
	
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to dial Unix socket: %w", err)
	}
	
	return conn, nil
}

// Name returns the transport name.
func (t *UnixTransport) Name() string {
	return "unix"
}

// Type returns the transport type.
func (t *UnixTransport) Type() string {
	return "unix"
}

// Config returns the transport configuration.
func (t *UnixTransport) Config() Config {
	return t.config
}

// Health checks the transport health.
func (t *UnixTransport) Health(ctx context.Context) error {
	// Try to establish a connection
	conn, err := t.Dial(ctx)
	if err != nil {
		return fmt.Errorf("Unix socket health check failed: %w", err)
	}
	defer conn.Close()
	
	return nil
}

// Close closes the transport.
func (t *UnixTransport) Close() error {
	return nil
}

// Helper functions

// parseAddress parses a network address.
func parseAddress(address string) (host string, port int, err error) {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format: %s", address)
	}
	
	host = parts[0]
	port, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %w", err)
	}
	
	return host, port, nil
}

// validateURL validates a URL.
func validateURL(rawURL string) error {
	_, err := url.Parse(rawURL)
	return err
}