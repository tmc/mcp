package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// StreamableClientTransportOptions configures the streamable client transport
type StreamableClientTransportOptions struct {
	HTTPClient     *http.Client
	Logger         *slog.Logger
	SessionTimeout time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

// StreamableClientTransport implements streamable HTTP transport for MCP clients
type StreamableClientTransport struct {
	url  string
	opts StreamableClientTransportOptions
}

// NewStreamableClientTransport creates a new streamable client transport
func NewStreamableClientTransport(url string, opts *StreamableClientTransportOptions) *StreamableClientTransport {
	t := &StreamableClientTransport{url: url}
	
	if opts != nil {
		t.opts = *opts
	}
	
	// Set defaults
	if t.opts.HTTPClient == nil {
		t.opts.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	if t.opts.Logger == nil {
		t.opts.Logger = slog.Default()
	}
	if t.opts.SessionTimeout <= 0 {
		t.opts.SessionTimeout = 5 * time.Minute
	}
	if t.opts.RetryAttempts <= 0 {
		t.opts.RetryAttempts = 3
	}
	if t.opts.RetryDelay <= 0 {
		t.opts.RetryDelay = time.Second
	}
	
	return t
}

// Connect implements the StreamableTransport interface
func (t *StreamableClientTransport) Connect(ctx context.Context) (Connection, error) {
	return newStreamableClientConnection(ctx, t.url, t.opts)
}

// Dial implements the Transport interface for compatibility
func (t *StreamableClientTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	conn, err := t.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &streamableClientRWCAdapter{conn: conn.(*streamableClientConnection)}, nil
}

// streamableClientConnection implements the Connection interface for clients
type streamableClientConnection struct {
	url           string
	sessionID     string
	postURL       string
	opts          StreamableClientTransportOptions
	httpClient    *http.Client
	
	mu           sync.RWMutex
	closed       bool
	eventSource  *eventSource
	lastEventID  string
	
	// Message handling
	incomingCh   chan JSONRPCMessage
	errorCh      chan error
	closeCh      chan struct{}
}

// newStreamableClientConnection creates a new streamable client connection
func newStreamableClientConnection(ctx context.Context, baseURL string, opts StreamableClientTransportOptions) (*streamableClientConnection, error) {
	conn := &streamableClientConnection{
		url:        baseURL,
		opts:       opts,
		httpClient: opts.HTTPClient,
		incomingCh: make(chan JSONRPCMessage, 100),
		errorCh:    make(chan error, 10),
		closeCh:    make(chan struct{}),
	}
	
	// Start SSE connection
	if err := conn.startSSEConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to start SSE connection: %w", err)
	}
	
	return conn, nil
}

// startSSEConnection establishes the SSE connection and extracts the POST endpoint
func (c *streamableClientConnection) startSSEConnection(ctx context.Context) error {
	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	
	// Add session parameter if we have one
	query := u.Query()
	if c.sessionID != "" {
		query.Set("session", c.sessionID)
	}
	u.RawQuery = query.Encode()
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	
	// Set required headers
	req.Header.Set("Accept", "text/event-stream, application/json")
	req.Header.Set("Cache-Control", "no-cache")
	
	// Add Last-Event-ID for resumption
	if c.lastEventID != "" {
		req.Header.Set("Last-Event-ID", c.lastEventID)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE request failed: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("SSE request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Create event source
	c.eventSource = &eventSource{
		resp:   resp,
		logger: c.opts.Logger,
	}
	
	// Start processing events
	go c.processEvents(ctx)
	
	return nil
}

// processEvents processes incoming SSE events
func (c *streamableClientConnection) processEvents(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		if c.eventSource != nil {
			c.eventSource.close()
		}
		c.mu.Unlock()
	}()
	
	for evt, err := range scanEvents(c.eventSource.resp.Body) {
		if err != nil {
			c.opts.Logger.ErrorContext(ctx, "SSE event error", "error", err)
			select {
			case c.errorCh <- err:
			case <-ctx.Done():
				return
			case <-c.closeCh:
				return
			}
			continue
		}
		
		// Handle different event types
		switch evt.name {
		case "endpoint":
			if err := c.handleEndpointEvent(ctx, evt); err != nil {
				c.opts.Logger.ErrorContext(ctx, "Failed to handle endpoint event", "error", err)
			}
		case "":
			// Default event (JSON-RPC message)
			if err := c.handleMessageEvent(ctx, evt); err != nil {
				c.opts.Logger.ErrorContext(ctx, "Failed to handle message event", "error", err)
			}
		default:
			c.opts.Logger.DebugContext(ctx, "Ignoring unknown event type", "event", evt.name)
		}
		
		// Update last event ID
		if evt.id != "" {
			c.lastEventID = evt.id
		}
	}
}

// handleEndpointEvent handles the endpoint event to extract POST URL
func (c *streamableClientConnection) handleEndpointEvent(ctx context.Context, evt event) error {
	endpoint := strings.TrimSpace(string(evt.data))
	
	baseURL, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	
	postURL, err := baseURL.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	
	c.mu.Lock()
	c.postURL = postURL.String()
	c.mu.Unlock()
	
	c.opts.Logger.DebugContext(ctx, "Received POST endpoint", "url", postURL.String())
	return nil
}

// handleMessageEvent handles JSON-RPC message events
func (c *streamableClientConnection) handleMessageEvent(ctx context.Context, evt event) error {
	var msg JSONRPCMessage
	if err := json.Unmarshal(evt.data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	select {
	case c.incomingCh <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closeCh:
		return io.ErrClosedPipe
	}
}

// Read implements the Connection interface
func (c *streamableClientConnection) Read(ctx context.Context) (JSONRPCMessage, error) {
	select {
	case <-ctx.Done():
		return JSONRPCMessage{}, ctx.Err()
	case <-c.closeCh:
		return JSONRPCMessage{}, io.ErrClosedPipe
	case err := <-c.errorCh:
		return JSONRPCMessage{}, err
	case msg := <-c.incomingCh:
		return msg, nil
	}
}

// Write implements the Connection interface
func (c *streamableClientConnection) Write(ctx context.Context, msg JSONRPCMessage) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return io.ErrClosedPipe
	}
	postURL := c.postURL
	c.mu.RUnlock()
	
	if postURL == "" {
		return fmt.Errorf("POST endpoint not yet available")
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Send HTTP POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create POST request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// Close implements the Connection interface
func (c *streamableClientConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.closed = true
	close(c.closeCh)
	
	if c.eventSource != nil {
		c.eventSource.close()
	}
	
	return nil
}

// eventSource manages the SSE response stream
type eventSource struct {
	resp   *http.Response
	logger *slog.Logger
}

func (es *eventSource) close() {
	if es.resp != nil {
		es.resp.Body.Close()
	}
}

// streamableClientRWCAdapter adapts the streamable client connection to io.ReadWriteCloser
type streamableClientRWCAdapter struct {
	conn    *streamableClientConnection
	readBuf bytes.Buffer
	mu      sync.Mutex
}

func (a *streamableClientRWCAdapter) Read(p []byte) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.readBuf.Len() > 0 {
		return a.readBuf.Read(p)
	}
	
	msg, err := a.conn.Read(context.Background())
	if err != nil {
		return 0, err
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	
	data = append(data, '\n')
	a.readBuf.Write(data)
	
	return a.readBuf.Read(p)
}

func (a *streamableClientRWCAdapter) Write(p []byte) (n int, err error) {
	var msg JSONRPCMessage
	if err := json.Unmarshal(p, &msg); err != nil {
		return 0, err
	}
	
	if err := a.conn.Write(context.Background(), msg); err != nil {
		return 0, err
	}
	
	return len(p), nil
}

func (a *streamableClientRWCAdapter) Close() error {
	return a.conn.Close()
}