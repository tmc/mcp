// Package transport handles network transport for mcpd.
package transport

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// TraceLogger logs interactions to an MCP trace file
type TraceLogger struct {
	File       *os.File
	TimeFormat string
}

// NewTraceLogger creates a new trace logger
func NewTraceLogger(file *os.File) *TraceLogger {
	return &TraceLogger{
		File:       file,
		TimeFormat: "milli", // Default to milliseconds
	}
}

// LogClientToServer logs a message from client to server
func (t *TraceLogger) LogClientToServer(data []byte) error {
	if t.File == nil {
		return nil
	}
	
	timestamp := formatTimestamp(time.Now(), t.TimeFormat)
	_, err := fmt.Fprintf(t.File, "mcp-recv %s # %s\n", data, timestamp)
	return err
}

// LogServerToClient logs a message from server to client
func (t *TraceLogger) LogServerToClient(data []byte) error {
	if t.File == nil {
		return nil
	}
	
	timestamp := formatTimestamp(time.Now(), t.TimeFormat)
	_, err := fmt.Fprintf(t.File, "mcp-send %s # %s\n", data, timestamp)
	return err
}

// formatTimestamp formats a timestamp according to the specified format
func formatTimestamp(t time.Time, format string) string {
	switch format {
	case "milli":
		return fmt.Sprintf("%d.%03d", t.Unix(), t.Nanosecond()/1000000)
	case "micro":
		return fmt.Sprintf("%d.%06d", t.Unix(), t.Nanosecond()/1000)
	case "nano":
		return fmt.Sprintf("%d.%09d", t.Unix(), t.Nanosecond())
	default:
		return fmt.Sprintf("%d.%03d", t.Unix(), t.Nanosecond()/1000000)
	}
}

// Listener manages incoming connections
type Listener struct {
	Addr           string
	Network        string
	SocketPath     string
	TraceLogger    *TraceLogger
	PromptHandler  *InteractivePromptHandler

	listener       net.Listener
	shutdownCh     chan struct{}
	connections    map[net.Conn]struct{}
	mu             sync.Mutex
}

// NewListener creates a new transport listener
func NewListener(addr string) (*Listener, error) {
	// Parse address
	network, address, err := parseAddr(addr)
	if err != nil {
		return nil, err
	}
	
	t := &Listener{
		Addr:        addr,
		Network:     network,
		connections: make(map[net.Conn]struct{}),
		shutdownCh:  make(chan struct{}),
	}
	
	// Store socket path for cleanup if using Unix sockets
	if network == "unix" {
		t.SocketPath = address
	}
	
	return t, nil
}

// WithTraceLogger sets a trace logger
func (t *Listener) WithTraceLogger(logger *TraceLogger) *Listener {
	t.TraceLogger = logger
	return t
}

// WithPromptHandler sets an interactive prompt handler
func (t *Listener) WithPromptHandler(handler *InteractivePromptHandler) *Listener {
	t.PromptHandler = handler

	// Link the trace logger to the prompt handler if available
	if t.TraceLogger != nil && handler != nil {
		handler.WithTraceLogger(t.TraceLogger)
	}

	return t
}

// Listen starts listening for connections
func (t *Listener) Listen() error {
	// Create the listener
	var err error
	t.listener, err = net.Listen(t.Network, t.getAddress())
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", t.Addr, err)
	}
	
	slog.Info("Listening for connections", "network", t.Network, "address", t.getAddress())
	return nil
}

// getAddress returns the address portion without the network prefix
func (t *Listener) getAddress() string {
	if t.Network == "unix" {
		return t.SocketPath
	}
	
	// For TCP, extract address from "tcp://address"
	parts := strings.SplitN(t.Addr, "://", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	
	return t.Addr
}

// Accept accepts a new connection
func (t *Listener) Accept() (net.Conn, error) {
	if t.listener == nil {
		return nil, fmt.Errorf("listener not started")
	}
	
	// Accept a connection
	conn, err := t.listener.Accept()
	if err != nil {
		return nil, err
	}
	
	// Track the connection
	t.mu.Lock()
	t.connections[conn] = struct{}{}
	t.mu.Unlock()
	
	slog.Info("Accepted connection", "remote", conn.RemoteAddr().String())
	return conn, nil
}

// Close closes the listener and all active connections
func (t *Listener) Close() error {
	if t.shutdownCh == nil {
		return nil
	}

	// Use mutex to protect shutdownCh
	t.mu.Lock()
	if t.shutdownCh != nil {
		// Signal shutdown
		close(t.shutdownCh)
		t.shutdownCh = nil
	}

	// Close all active connections
	for conn := range t.connections {
		conn.Close()
		delete(t.connections, conn)
	}
	t.mu.Unlock()

	// Close the listener
	if t.listener != nil {
		err := t.listener.Close()

		// Clean up Unix socket file if needed
		if t.Network == "unix" && t.SocketPath != "" {
			if err := os.Remove(t.SocketPath); err != nil && !os.IsNotExist(err) {
				slog.Error("Failed to remove socket file", "path", t.SocketPath, "error", err)
			}
		}

		return err
	}

	return nil
}

// RemoveConnection removes a connection from tracking
func (t *Listener) RemoveConnection(conn net.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	delete(t.connections, conn)
}

// The HandleConnection function implementation has been moved to handler.go for better organization
// and to support interactive features.

// parseAddr parses a network address string
func parseAddr(addr string) (network, address string, err error) {
	parts := strings.SplitN(addr, "://", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid address format: %s, expected network://address", addr)
	}
	
	network, address = parts[0], parts[1]
	
	switch network {
	case "tcp", "tcp4", "tcp6", "unix":
		// These are supported
	default:
		return "", "", fmt.Errorf("unsupported network: %s", network)
	}
	
	return network, address, nil
}