package sdk2

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultClient is the default Client used by helper functions.
// It's similar to http.DefaultClient.
var DefaultClient = &client{
	config: ClientConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
		ClientInfo: ClientInfo{Name: "sdk2-client", Version: "0.1.0"},
	},
}

// Package-level convenience functions using DefaultClient

// ListTools lists tools using the default client (like http.Get)
func ListTools(ctx context.Context) ([]Tool, error) {
	if DefaultClient == nil {
		return nil, ErrClientClosed
	}
	return DefaultClient.ListTools(ctx)
}

// CallTool calls a tool using the default client
func CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error) {
	if DefaultClient == nil {
		return nil, ErrClientClosed
	}
	return DefaultClient.CallTool(ctx, name, args)
}

// ListResources lists resources using the default client
func ListResources(ctx context.Context) ([]Resource, error) {
	if DefaultClient == nil {
		return nil, ErrClientClosed
	}
	return DefaultClient.ListResources(ctx)
}

// ReadResource reads a resource using the default client
func ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	if DefaultClient == nil {
		return nil, ErrClientClosed
	}
	return DefaultClient.ReadResource(ctx, uri)
}

// Ping checks connectivity using the default client
func Ping(ctx context.Context) error {
	if DefaultClient == nil {
		return ErrClientClosed
	}
	return DefaultClient.Ping(ctx)
}

// Dial connects to the MCP server at the given address using stdlib dial patterns.
// The network and address parameters follow net.Dial conventions.
//
// Example:
//
//	client, err := sdk2.Dial(ctx, "stdio", "")
//	client, err := sdk2.Dial(ctx, "tcp", "localhost:3000")
func Dial(ctx context.Context, network, address string) (Client, error) {
	d := &Dialer{}
	return d.DialContext(ctx, network, address)
}

// MustDial is like Dial but panics on error (like template.Must)
func MustDial(ctx context.Context, network, address string) Client {
	client, err := Dial(ctx, network, address)
	if err != nil {
		panic(err)
	}
	return client
}

// DialStdio is a convenience function for dialing stdio (like net.DialUnix, etc.)
func DialStdio(ctx context.Context) (Client, error) {
	return Dial(ctx, "stdio", "")
}

// MustDialStdio is like DialStdio but panics on error
func MustDialStdio(ctx context.Context) Client {
	client, err := DialStdio(ctx)
	if err != nil {
		panic(err)
	}
	return client
}

// DialConfig connects to the MCP server using the provided configuration.
// This follows the functional options pattern common in Go.
func DialConfig(ctx context.Context, network, address string, opts ...ClientOption) (Client, error) {
	config := &ClientConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
		ClientInfo: ClientInfo{Name: "sdk2-client", Version: "0.1.0"},
	}

	for _, opt := range opts {
		opt(config)
	}

	d := &Dialer{}
	return d.dialWithConfig(ctx, network, address, config)
}

// DialContext connects to the MCP server using the dialer.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (Client, error) {
	config := &ClientConfig{
		Timeout:    d.Timeout,
		MaxRetries: 3,
		RetryDelay: time.Second,
		ClientInfo: ClientInfo{Name: "sdk2-client", Version: "0.1.0"},
	}
	return d.dialWithConfig(ctx, network, address, config)
}

// dialWithConfig connects using a specific config.
func (d *Dialer) dialWithConfig(ctx context.Context, network, address string, config *ClientConfig) (Client, error) {
	// Handle special case for stdio
	if network == "stdio" {
		return dialStdio(ctx, config)
	}

	// Use context deadline if set
	deadline := d.Deadline
	if d.Timeout > 0 {
		timeoutDeadline := time.Now().Add(d.Timeout)
		if deadline.IsZero() || timeoutDeadline.Before(deadline) {
			deadline = timeoutDeadline
		}
	}

	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}

	// Dial the connection
	conn, err := (&net.Dialer{
		Timeout:   d.Timeout,
		KeepAlive: d.KeepAlive,
		Control:   d.Control,
	}).DialContext(ctx, network, address)
	if err != nil {
		return nil, NewConnError("dial", network, address, err)
	}

	// Wrap in our Conn interface
	mcpConn := &netConn{Conn: conn}

	return newClient(ctx, mcpConn, config)
}

// netConn wraps net.Conn to implement our Conn interface
type netConn struct {
	net.Conn
}

func (c *netConn) LocalAddr() net.Addr  { return c.Conn.LocalAddr() }
func (c *netConn) RemoteAddr() net.Addr { return c.Conn.RemoteAddr() }

// dialStdio creates a client using stdio transport
func dialStdio(ctx context.Context, config *ClientConfig) (Client, error) {
	conn := &stdioConn{
		reader: os.Stdin,
		writer: os.Stdout,
	}
	return newClient(ctx, conn, config)
}

// stdioConn implements Conn for stdio transport
type stdioConn struct {
	reader io.Reader
	writer io.Writer
}

func (c *stdioConn) Read(p []byte) (int, error) {
	if c.reader == nil {
		return 0, fmt.Errorf("stdio reader not available")
	}
	return c.reader.Read(p)
}

func (c *stdioConn) Write(p []byte) (int, error) {
	if c.writer == nil {
		return 0, fmt.Errorf("stdio writer not available")
	}
	return c.writer.Write(p)
}

func (c *stdioConn) Close() error { return nil }

func (c *stdioConn) LocalAddr() net.Addr  { return &stdioAddr{} }
func (c *stdioConn) RemoteAddr() net.Addr { return &stdioAddr{} }

func (c *stdioConn) SetDeadline(t time.Time) error      { return nil }
func (c *stdioConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *stdioConn) SetWriteDeadline(t time.Time) error { return nil }

// stdioAddr implements net.Addr for stdio
type stdioAddr struct{}

func (a *stdioAddr) Network() string { return "stdio" }
func (a *stdioAddr) String() string  { return "stdio" }

// client implements the Client interface using stdlib patterns
type client struct {
	conn   Conn
	config ClientConfig

	// JSON-RPC state
	nextID    int64
	pending   map[int64]chan *jsonrpcResponse
	pendingMu sync.RWMutex

	// Connection state
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex // protects writing

	// Lifecycle
	once sync.Once
	done chan struct{}

	// Protocol state
	initialized  int32 // atomic
	serverInfo   *ServerInfo
	capabilities *ServerCapabilities
	handshakeMu  sync.Mutex
}

// Do implements Client.Do - the low-level request method following http.Client pattern
func (c *client) Do(req *Request) (*Response, error) {
	if req.Context == nil {
		req.Context = context.Background()
	}

	// Convert to JSON-RPC request
	var id any
	if req.ID != nil {
		id = req.ID.Value
	}

	jsonReq := &jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  req.Method,
		Params:  req.Params,
	}

	// Send request and get response
	jsonResp, err := c.sendRequest(req.Context, jsonReq)
	if err != nil {
		return nil, err
	}

	// Convert to MCP response
	resp := &Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      ProtocolVersion,
		Header:     make(Header),
		Request:    req,
	}

	if jsonResp.Error != nil {
		resp.Status = fmt.Sprintf("%d %s", jsonResp.Error.Code, StatusText(jsonResp.Error.Code))
		resp.StatusCode = jsonResp.Error.Code
	}

	if jsonResp.Result != nil {
		resp.Body = io.NopCloser(jsonReaderFromBytes(jsonResp.Result))
		resp.ContentLength = int64(len(jsonResp.Result))
	}

	return resp, nil
}

// jsonReaderFromBytes creates an io.Reader from JSON bytes
func jsonReaderFromBytes(data json.RawMessage) io.Reader {
	return &jsonReader{data: data}
}

type jsonReader struct {
	data json.RawMessage
	pos  int
}

func (r *jsonReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// newClient creates a new client with the given connection and config
func newClient(ctx context.Context, conn Conn, config *ClientConfig) (Client, error) {
	if config == nil {
		config = &ClientConfig{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RetryDelay: time.Second,
			ClientInfo: ClientInfo{Name: "sdk2-client", Version: "0.1.0"},
		}
	}

	c := &client{
		conn:    conn,
		config:  *config,
		pending: make(map[int64]chan *jsonrpcResponse),
		reader:  bufio.NewReader(conn),
		writer:  bufio.NewWriter(conn),
		done:    make(chan struct{}),
	}

	// Start the read loop
	go c.readLoop(ctx)

	return c, nil
}

// Ping implements Client.Ping - verifies connectivity to the server
func (c *client) Ping(ctx context.Context) error {
	// For MCP, we can use a simple tools/list request as a ping
	_, err := c.ListTools(ctx)
	return err
}

// ListTools implements Client.ListTools
func (c *client) ListTools(ctx context.Context) ([]Tool, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	req := &Request{
		Method:  MethodToolsList,
		Context: ctx,
		Proto:   ProtocolVersion,
		Header:  make(Header),
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal tools response: %w", err)
	}

	return result.Tools, nil
}

// CallTool implements Client.CallTool
func (c *client) CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	params := ToolCall{
		Name:      name,
		Arguments: args,
	}

	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal parameters: %w", err)
	}

	req := &Request{
		Method:  MethodToolsCall,
		Params:  json.RawMessage(paramBytes),
		Context: ctx,
		Proto:   ProtocolVersion,
		Header:  make(Header),
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tools/call failed: %w", err)
	}
	defer resp.Body.Close()

	var result ToolResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal tool result: %w", err)
	}

	return &result, nil
}

// ListResources implements Client.ListResources
func (c *client) ListResources(ctx context.Context) ([]Resource, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	req := &Request{
		Method:  MethodResourcesList,
		Context: ctx,
		Proto:   ProtocolVersion,
		Header:  make(Header),
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resources/list failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Resources []Resource `json:"resources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal resources response: %w", err)
	}

	return result.Resources, nil
}

// ReadResource implements Client.ReadResource
func (c *client) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	params := ResourceRequest{URI: uri}

	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal parameters: %w", err)
	}

	req := &Request{
		Method:  MethodResourcesRead,
		Params:  json.RawMessage(paramBytes),
		Context: ctx,
		Proto:   ProtocolVersion,
		Header:  make(Header),
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resources/read failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Contents []ResourceContent `json:"contents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal resource content: %w", err)
	}

	if len(result.Contents) == 0 {
		return nil, Errorf("resource not found: %s", uri)
	}

	return &result.Contents[0], nil
}

// ListPrompts implements Client.ListPrompts
func (c *client) ListPrompts(ctx context.Context) ([]Prompt, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	req := &Request{
		Method:  MethodPromptsList,
		Context: ctx,
		Proto:   ProtocolVersion,
		Header:  make(Header),
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prompts/list failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Prompts []Prompt `json:"prompts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal prompts response: %w", err)
	}

	return result.Prompts, nil
}

// GetPrompt implements Client.GetPrompt
func (c *client) GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	params := PromptRequest{
		Name:      name,
		Arguments: args,
	}

	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal parameters: %w", err)
	}

	req := &Request{
		Method:  MethodPromptsGet,
		Params:  json.RawMessage(paramBytes),
		Context: ctx,
		Proto:   ProtocolVersion,
		Header:  make(Header),
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prompts/get failed: %w", err)
	}
	defer resp.Body.Close()

	var result PromptResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal prompt result: %w", err)
	}

	return &result, nil
}

// Close implements Client.Close
func (c *client) Close() error {
	var err error
	c.once.Do(func() {
		close(c.done)
		err = c.conn.Close()
	})
	return err
}

// ensureInitialized performs the MCP handshake if not already done
func (c *client) ensureInitialized(ctx context.Context) error {
	if atomic.LoadInt32(&c.initialized) == 1 {
		return nil
	}

	c.handshakeMu.Lock()
	defer c.handshakeMu.Unlock()

	// Double-check after acquiring lock
	if atomic.LoadInt32(&c.initialized) == 1 {
		return nil
	}

	// Send initialize request
	params := struct {
		ProtocolVersion string     `json:"protocolVersion"`
		Capabilities    struct{}   `json:"capabilities"`
		ClientInfo      ClientInfo `json:"clientInfo"`
	}{
		ProtocolVersion: ProtocolVersion,
		ClientInfo:      c.config.ClientInfo,
	}

	paramBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal initialize params: %w", err)
	}

	req := &jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      atomic.AddInt64(&c.nextID, 1),
		Method:  MethodInitialize,
		Params:  json.RawMessage(paramBytes),
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return NewProtocolError("initialize", "initialize request failed", err)
	}

	if resp.Error != nil {
		return NewMCPError("initialize", MethodInitialize, resp.Error.Code, resp.Error.Message, nil)
	}

	var result ServerInfo
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return NewProtocolError("initialize", "unmarshal initialize response", err)
	}

	c.serverInfo = &result
	c.capabilities = result.Capabilities

	// Send initialized notification
	notif := &jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  MethodInitialized,
		// No ID for notifications
	}

	if err := c.writeMessage(notif); err != nil {
		return NewProtocolError("initialize", "send initialized notification", err)
	}

	atomic.StoreInt32(&c.initialized, 1)
	return nil
}

// sendRequest sends a JSON-RPC request and waits for the response
func (c *client) sendRequest(ctx context.Context, req *jsonrpcRequest) (*jsonrpcResponse, error) {
	if req.ID == nil {
		// This is a notification, no response expected
		return nil, c.writeMessage(req)
	}

	// Extract ID for response tracking
	var id int64
	switch v := req.ID.(type) {
	case int64:
		id = v
	case float64:
		id = int64(v)
	default:
		return nil, fmt.Errorf("unsupported request ID type: %T", req.ID)
	}

	// Create response channel
	respChan := make(chan *jsonrpcResponse, 1)

	c.pendingMu.Lock()
	c.pending[id] = respChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	// Send request
	if err := c.writeMessage(req); err != nil {
		return nil, NewConnError("write", "mcp", "", err)
	}

	// Wait for response with timeout
	timeout := c.config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClientClosed
	}
}

// writeMessage writes a JSON-RPC message to the connection
func (c *client) writeMessage(msg any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Add newline for line-delimited JSON
	data = append(data, '\n')

	if _, err := c.writer.Write(data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return c.writer.Flush()
}

// readLoop reads and processes incoming messages
func (c *client) readLoop(ctx context.Context) {
	defer func() {
		// Clean up pending requests on exit
		c.pendingMu.Lock()
		for _, ch := range c.pending {
			close(ch)
		}
		c.pendingMu.Unlock()
	}()

	for {
		select {
		case <-c.done:
			return
		case <-ctx.Done():
			return
		default:
			// Read a line
			line, err := c.reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				// Log error and continue
				continue
			}

			// Parse JSON-RPC message
			var msg jsonrpcResponse
			if err := json.Unmarshal(line, &msg); err != nil {
				// Skip invalid messages
				continue
			}

			// Handle response or notification
			if msg.ID != nil {
				c.handleResponse(&msg)
			} else {
				c.handleNotification(ctx, &msg)
			}
		}
	}
}

// handleResponse routes a response to the appropriate pending request
func (c *client) handleResponse(resp *jsonrpcResponse) {
	// Extract ID - it could be string, number, or null
	var id int64
	switch v := resp.ID.(type) {
	case float64:
		id = int64(v)
	case int64:
		id = v
	default:
		return // Skip unknown ID types
	}

	c.pendingMu.RLock()
	ch, exists := c.pending[id]
	c.pendingMu.RUnlock()

	if !exists {
		return // No pending request for this ID
	}

	select {
	case ch <- resp:
	default:
		// Channel full, skip
	}
}

// handleNotification processes server notifications
func (c *client) handleNotification(ctx context.Context, resp *jsonrpcResponse) {
	if c.config.NotificationHandler == nil {
		return
	}

	// Extract method from notification (not in standard response format)
	// In a real implementation, notifications would have method field
	method := "notification"

	// Handle the notification
	_ = c.config.NotificationHandler.HandleNotification(ctx, method, resp.Result)
}

// JSON-RPC message types
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}
