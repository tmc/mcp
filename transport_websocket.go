package mcp

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketTransport implements Transport interface for WebSocket connections
type WebSocketTransport struct {
	url    string
	header http.Header
	dialer *websocket.Dialer
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(rawurl string) (*WebSocketTransport, error) {
	_, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	return &WebSocketTransport{
		url:    rawurl,
		header: make(http.Header),
		dialer: websocket.DefaultDialer,
	}, nil
}

// WithHeader sets a header for the WebSocket connection
func (t *WebSocketTransport) WithHeader(key, value string) *WebSocketTransport {
	t.header.Set(key, value)
	return t
}

// WithDialer sets a custom WebSocket dialer
func (t *WebSocketTransport) WithDialer(dialer *websocket.Dialer) *WebSocketTransport {
	t.dialer = dialer
	return t
}

// Dial implements Transport interface
func (t *WebSocketTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	conn, _, err := t.dialer.DialContext(ctx, t.url, t.header)
	if err != nil {
		return nil, err
	}

	return &WebSocketConn{conn: conn}, nil
}

// WebSocketConn wraps a websocket.Conn to implement io.ReadWriteCloser
type WebSocketConn struct {
	conn       *websocket.Conn
	readBuffer []byte
	readPos    int
	closed     bool
	mu         sync.Mutex
}

// Read implements io.Reader
func (c *WebSocketConn) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If we have buffered data, use it first
	if c.readPos < len(c.readBuffer) {
		n = copy(p, c.readBuffer[c.readPos:])
		c.readPos += n

		// If we've consumed all buffered data, reset
		if c.readPos >= len(c.readBuffer) {
			c.readBuffer = nil
			c.readPos = 0
		}

		return n, nil
	}

	// Read the next message
	messageType, message, err := c.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	// Only handle text messages for JSON-RPC
	if messageType != websocket.TextMessage {
		return 0, io.EOF
	}

	// Copy what we can to the output buffer
	n = copy(p, message)

	// If there's remaining data, buffer it
	if n < len(message) {
		c.readBuffer = make([]byte, len(message)-n)
		copy(c.readBuffer, message[n:])
		c.readPos = 0
	}

	return n, nil
}

// Write implements io.Writer
func (c *WebSocketConn) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.ErrClosedPipe
	}

	err = c.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close implements io.Closer
func (c *WebSocketConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	return c.conn.Close()
}

// String provides a string representation
func (t *WebSocketTransport) String() string {
	return "WebSocketTransport(" + t.url + ")"
}
