// Package transport provides transport implementations for SDK2.
// This follows Go stdlib patterns like net/http/httptrace.
package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// Transport defines the interface for MCP communication transports.
// This follows the io interfaces pattern in Go stdlib.
type Transport interface {
	io.ReadWriteCloser
	
	// Dial establishes a connection for client use (for network transports)
	Dial(ctx context.Context) (Conn, error)
	
	// Listen starts listening for server connections (for network transports)
	Listen(ctx context.Context) (Listener, error)
}

// Conn represents a bidirectional connection for MCP communication.
// This follows the net.Conn pattern but is specialized for MCP.
type Conn interface {
	io.ReadWriteCloser
	
	// LocalAddr returns the local address
	LocalAddr() net.Addr
	
	// RemoteAddr returns the remote address  
	RemoteAddr() net.Addr
	
	// SetDeadline sets read and write deadlines
	SetDeadline(t time.Time) error
	
	// SetReadDeadline sets the read deadline
	SetReadDeadline(t time.Time) error
	
	// SetWriteDeadline sets the write deadline
	SetWriteDeadline(t time.Time) error
}

// Listener accepts incoming connections.
// This follows the net.Listener pattern.
type Listener interface {
	// Accept waits for and returns the next connection
	Accept() (Conn, error)
	
	// Close closes the listener
	Close() error
	
	// Addr returns the listener's address
	Addr() net.Addr
}

// StdioTransport implements Transport using stdin/stdout.
type StdioTransport struct {
	reader *bufio.Reader
	writer *bufio.Writer
	once   sync.Once
	conn   *stdioConn
}

// NewStdio creates a new stdio transport.
func NewStdio() *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
}

// NewStdioWithReadWriter creates a stdio transport with custom reader/writer.
// This is useful for testing or custom stdio handling.
func NewStdioWithReadWriter(r io.Reader, w io.Writer) *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(r),
		writer: bufio.NewWriter(w),
	}
}

// Read reads data from stdin.
func (t *StdioTransport) Read(p []byte) (int, error) {
	return t.reader.Read(p)
}

// Write writes data to stdout.
func (t *StdioTransport) Write(p []byte) (int, error) {
	n, err := t.writer.Write(p)
	if err != nil {
		return n, err
	}
	
	// Auto-flush for line-based protocols like JSON-RPC
	if len(p) > 0 && p[len(p)-1] == '\n' {
		if flushErr := t.writer.Flush(); flushErr != nil {
			return n, flushErr
		}
	}
	
	return n, nil
}

// Close closes the transport (flushes stdout for stdio).
func (t *StdioTransport) Close() error {
	if t.writer != nil {
		return t.writer.Flush()
	}
	return nil
}

// Dial establishes a connection (for stdio, returns the transport as a connection)
func (t *StdioTransport) Dial(ctx context.Context) (Conn, error) {
	var conn *stdioConn
	t.once.Do(func() {
		conn = &stdioConn{
			transport: t,
		}
		t.conn = conn
	})
	
	if t.conn == nil {
		return nil, fmt.Errorf("stdio transport already used")
	}
	
	return t.conn, nil
}

// Listen starts listening (for stdio, creates a listener that accepts one connection)
func (t *StdioTransport) Listen(ctx context.Context) (Listener, error) {
	return &stdioListener{
		transport: t,
		accepted:  make(chan struct{}),
	}, nil
}

// stdioConn implements Conn for stdio
type stdioConn struct {
	transport *StdioTransport
	closed    bool
	mu        sync.Mutex
}

func (c *stdioConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, io.EOF
	}
	c.mu.Unlock()
	
	return c.transport.Read(p)
}

func (c *stdioConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, fmt.Errorf("connection closed")
	}
	c.mu.Unlock()
	
	return c.transport.Write(p)
}

func (c *stdioConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	c.closed = true
	
	return c.transport.Close()
}

func (c *stdioConn) LocalAddr() net.Addr {
	return &stdioAddr{}
}

func (c *stdioConn) RemoteAddr() net.Addr {
	return &stdioAddr{}
}

func (c *stdioConn) SetDeadline(t time.Time) error {
	// Stdio doesn't support deadlines
	return nil
}

func (c *stdioConn) SetReadDeadline(t time.Time) error {
	// Stdio doesn't support deadlines
	return nil
}

func (c *stdioConn) SetWriteDeadline(t time.Time) error {
	// Stdio doesn't support deadlines
	return nil
}

// stdioListener implements Listener for stdio
type stdioListener struct {
	transport *StdioTransport
	accepted  chan struct{}
	closed    bool
	mu        sync.Mutex
}

func (l *stdioListener) Accept() (Conn, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, fmt.Errorf("listener closed")
	}
	l.mu.Unlock()
	
	// For stdio, we only accept one connection
	select {
	case <-l.accepted:
		return nil, fmt.Errorf("stdio listener only accepts one connection")
	default:
		close(l.accepted)
	}
	
	return l.transport.Dial(context.Background())
}

func (l *stdioListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.closed {
		return nil
	}
	l.closed = true
	
	return l.transport.Close()
}

func (l *stdioListener) Addr() net.Addr {
	return &stdioAddr{}
}

// stdioAddr implements net.Addr for stdio
type stdioAddr struct{}

func (a *stdioAddr) Network() string { return "stdio" }
func (a *stdioAddr) String() string  { return "stdio" }

// NewReadWriteCloser creates a transport from any ReadWriteCloser.
func NewReadWriteCloser(rwc io.ReadWriteCloser) *ReadWriteCloserTransport {
	return &ReadWriteCloserTransport{rwc: rwc}
}

// ReadWriteCloserTransport wraps any ReadWriteCloser as a Transport.
type ReadWriteCloserTransport struct {
	rwc io.ReadWriteCloser
}

// Read implements Transport.
func (t *ReadWriteCloserTransport) Read(p []byte) (int, error) {
	return t.rwc.Read(p)
}

// Write implements Transport.
func (t *ReadWriteCloserTransport) Write(p []byte) (int, error) {
	return t.rwc.Write(p)
}

// Close implements Transport.
func (t *ReadWriteCloserTransport) Close() error {
	return t.rwc.Close()
}

// Dial for ReadWriteCloser returns the wrapped connection
func (t *ReadWriteCloserTransport) Dial(ctx context.Context) (Conn, error) {
	return &rwcConn{rwc: t.rwc}, nil
}

// Listen for ReadWriteCloser creates a listener (mainly for testing)
func (t *ReadWriteCloserTransport) Listen(ctx context.Context) (Listener, error) {
	return &rwcListener{transport: t}, nil
}

// rwcConn wraps ReadWriteCloser as Conn
type rwcConn struct {
	rwc io.ReadWriteCloser
}

func (c *rwcConn) Read(p []byte) (int, error)  { return c.rwc.Read(p) }
func (c *rwcConn) Write(p []byte) (int, error) { return c.rwc.Write(p) }
func (c *rwcConn) Close() error                { return c.rwc.Close() }
func (c *rwcConn) LocalAddr() net.Addr         { return &rwcAddr{} }
func (c *rwcConn) RemoteAddr() net.Addr        { return &rwcAddr{} }
func (c *rwcConn) SetDeadline(t time.Time) error     { return nil }
func (c *rwcConn) SetReadDeadline(t time.Time) error { return nil }
func (c *rwcConn) SetWriteDeadline(t time.Time) error { return nil }

// rwcListener implements Listener for ReadWriteCloser
type rwcListener struct {
	transport *ReadWriteCloserTransport
	accepted  bool
	mu        sync.Mutex
}

func (l *rwcListener) Accept() (Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.accepted {
		return nil, fmt.Errorf("ReadWriteCloser listener only accepts one connection")
	}
	l.accepted = true
	
	return &rwcConn{rwc: l.transport.rwc}, nil
}

func (l *rwcListener) Close() error {
	return l.transport.Close()
}

func (l *rwcListener) Addr() net.Addr {
	return &rwcAddr{}
}

// rwcAddr implements net.Addr for ReadWriteCloser
type rwcAddr struct{}

func (a *rwcAddr) Network() string { return "rwc" }
func (a *rwcAddr) String() string  { return "readwritecloser" }

// Message represents a JSON-RPC message for transport
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// WriteMessage writes a JSON-RPC message to a writer with line-delimited JSON format
func WriteMessage(w io.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	
	// Add newline for line-delimited JSON
	data = append(data, '\n')
	
	_, err = w.Write(data)
	return err
}

// ReadMessage reads a JSON-RPC message from a reader
func ReadMessage(r *bufio.Reader) (*Message, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read line: %w", err)
	}
	
	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}
	
	return &msg, nil
}