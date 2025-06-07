// Package testutil provides testing helpers for MCP server and client setup.
package testutil

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"

	"github.com/tmc/mcp"
)

// testLogHandler implements slog.Handler that redirects to t.Log
type testLogHandler struct {
	t       *testing.T
	level   slog.Level
	verbose bool
}

func (h *testLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	// Without -v: Show INFO and above
	// With -v: Show DEBUG and above
	if !h.verbose {
		return level >= slog.LevelInfo
	}
	return level >= h.level
}

func (h *testLogHandler) Handle(_ context.Context, record slog.Record) error {
	h.t.Logf("[%s] %s", record.Level, record.Message)
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

// WithTestLogger creates a server option that redirects logs to t.Log
// Default behavior:
// - Without -v: Shows INFO and above
// - With -v: Shows DEBUG and above
// - With MCP_TEST_DEBUG=1: Always shows DEBUG and above
func WithTestLogger(t *testing.T, level slog.Level) mcp.ServerOption {
	// Check for debug environment variable
	verbose := testing.Verbose() || os.Getenv("MCP_TEST_DEBUG") == "1"

	return mcp.WithLogger(slog.New(&testLogHandler{
		t:       t,
		level:   level,
		verbose: verbose,
	}))
}

// TestLogger creates a logger that redirects to t.Log with appropriate filtering
// This can be used in test handlers to ensure logs go through the test output
func TestLogger(t *testing.T) *slog.Logger {
	verbose := testing.Verbose() || os.Getenv("MCP_TEST_DEBUG") == "1"

	return slog.New(&testLogHandler{
		t:       t,
		level:   slog.LevelDebug,
		verbose: verbose,
	})
}

// ServerClientPair represents a connected server and client pair for testing.
type ServerClientPair struct {
	Server *mcp.Server
	Client *mcp.Client

	// Cleanup should be called to clean up resources
	Cleanup func()
}

// NewServerClientPair creates a connected server and client pair using bidirectional pipes.
// This is useful for testing server implementations with real client interactions.
// The server's logger is automatically configured to use t.Log for clean test output.
//
// Example usage:
//
//	pair, err := mcp.NewServerClientPair(t, context.Background(), myServer)
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer pair.Cleanup()
//
//	// Use pair.Client to interact with the server
//	result, err := pair.Client.CallTool(ctx, request)
//	if err != nil {
//	    t.Fatal(err)
//	}
func NewServerClientPair(t *testing.T, ctx context.Context, server *mcp.Server) (*ServerClientPair, error) {
	// Configure server to use test logger if t is provided
	if t != nil {
		// Use the WithLogger option to set the test logger
		verbose := testing.Verbose() || os.Getenv("MCP_TEST_DEBUG") == "1"
		opt := mcp.WithLogger(slog.New(&testLogHandler{t: t, level: slog.LevelDebug, verbose: verbose}))
		opt(server)
	}
	// Create bidirectional pipes
	serverConn, clientConn := net.Pipe()

	// Create server transport from the server side of the pipe
	serverTransport := &mcp.ReadWriteCloserTransport{
		ReadWriteCloser: serverConn,
	}

	// Create client transport from the client side of the pipe
	clientTransport := &mcp.ReadWriteCloserTransport{
		ReadWriteCloser: clientConn,
	}

	// Start server in a goroutine
	serverCtx, serverCancel := context.WithCancel(ctx)
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve(serverCtx, serverTransport)
	}()

	// Create and initialize client
	client, err := mcp.NewClient(clientTransport)
	if err != nil {
		serverCancel()
		_ = serverConn.Close()
		_ = clientConn.Close()
		return nil, err
	}

	// Initialize the client
	initReq := mcp.InitializeRequest{
		ClientInfo: mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
	}

	if _, err := client.Initialize(ctx, initReq); err != nil {
		serverCancel()
		_ = client.Close()
		_ = serverConn.Close()
		_ = clientConn.Close()
		return nil, err
	}

	return &ServerClientPair{
		Server: server,
		Client: client,
		Cleanup: func() {
			// Cleanup in reverse order
			_ = client.Close()
			serverCancel()
			<-serverDone // Wait for server to finish
			_ = serverConn.Close()
			_ = clientConn.Close()
		},
	}, nil
}

// NewServerClientPairWithOptions creates a connected server and client pair with custom options.
func NewServerClientPairWithOptions(ctx context.Context, server *mcp.Server, serverOpts []mcp.ServerOption, clientOpts []mcp.ClientOption) (*ServerClientPair, error) {
	// Apply server options
	for _, opt := range serverOpts {
		opt(server)
	}

	// Create the basic pair
	pair, err := NewServerClientPair(nil, ctx, server)
	if err != nil {
		return nil, err
	}

	// Apply client options (if we need to recreate the client)
	// For now, options are typically applied during client creation
	// This is here for future extensibility

	return pair, nil
}

// TestServerConfig provides common server configuration for tests.
type TestServerConfig struct {
	Name    string
	Version string
	Tools   []mcp.Tool
	Prompts []mcp.Prompt
	Options []mcp.ServerOption
}

// NewTestServer creates a new server configured for testing.
// The server's logger is automatically configured to use t.Log for clean test output.
func NewTestServer(t *testing.T, config *TestServerConfig) *mcp.Server {
	// Set defaults
	if config.Name == "" {
		config.Name = "test-server"
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}

	// Create server with base configuration
	options := []mcp.ServerOption{
		mcp.WithServerName(config.Name),
		mcp.WithServerVersion(config.Version),
		WithTestLogger(t, slog.LevelInfo),
	}

	// Add any custom options
	options = append(options, config.Options...)

	server := mcp.NewServer(config.Name, config.Version, options...)

	// Register tools - we'll need to register handlers separately
	// since Server likely has a RegisterToolHandler method

	// Register prompts - we'll need to register handlers separately
	// since Server likely has a RegisterPromptHandler method

	return server
}
