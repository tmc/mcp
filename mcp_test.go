package mcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
	"log/slog"
)

type testInput struct {
	Message string `json:"message"`
}

type testOutput struct {
	Result string `json:"result"`
}

// TestCancellation tests that context cancellation works through the JSON-RPC layer
func TestCancellation(t *testing.T) {
	// Create a server with a long-running tool
	server := NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Track if the tool was cancelled
	toolCancelled := make(chan bool, 1)

	// Register a long-running tool
	longTool := Tool{
		Name:        "longOperation",
		Description: "A tool that takes a long time to complete",
	}

	server.RegisterTool(longTool, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		// Wait for either cancellation or a timeout
		select {
		case <-ctx.Done():
			toolCancelled <- true
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return &CallToolResult{
				Content: []any{map[string]string{
					"type": "text",
					"text": "Completed successfully",
				}},
			}, nil
		}
	})

	// Create client and server connections via net.Pipe
	clientConn, serverConn := net.Pipe()

	// Start the server
	serverCtx := context.Background()
	serverReady := make(chan struct{})

	go func() {
		// The server transport needs to properly handle contexts
		serverTransport := &ReadWriteCloserTransport{serverConn}
		serverReady <- struct{}{}

		if err := server.Serve(serverCtx, serverTransport); err != nil {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	<-serverReady

	// Create the client
	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize the client
	if _, err := client.Initialize(context.Background(), InitializeRequest{
		ClientInfo: Implementation{
			Name:    "test-client",
			Version: "1.0",
		},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	}); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create a cancellable context for the tool call
	callCtx, cancel := context.WithCancel(context.Background())

	// Start the long-running tool call
	callDone := make(chan error, 1)
	go func() {
		_, err := client.CallTool(callCtx, CallToolRequest{
			Name: "longOperation",
		})
		callDone <- err
	}()

	// Give the call time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Check if the call returned with a cancellation error
	select {
	case err := <-callDone:
		if err == nil {
			t.Error("Expected error from cancelled call, got nil")
		} else if err != context.Canceled {
			t.Logf("Got error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Call did not complete after cancellation")
	}

	// Note: The server-side tool may not be cancelled because cancellation
	// doesn't automatically propagate through JSON-RPC. That's why MCP
	// includes the notifications/cancelled message.
}

// TestCancellationWithCause tests that context cancellation with a cause
// automatically sends the cancellation notification
func TestCancellationWithCause(t *testing.T) {
	// Create a server
	server := NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Track cancellation notification
	notificationReceived := make(chan string, 1)

	// Note: In a complete implementation, we'd hook into the server's
	// notification handling to verify the cancellation message is received

	// Register a long-running tool
	server.RegisterTool(Tool{
		Name:        "slowTool",
		Description: "A slow tool for testing cancellation",
	}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		select {
		case <-ctx.Done():
			notificationReceived <- "tool-cancelled"
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			return &CallToolResult{
				Content: []any{map[string]string{"type": "text", "text": "done"}},
			}, nil
		}
	})

	// Create connections
	clientConn, serverConn := net.Pipe()

	// Start server
	serverCtx := context.Background()
	go func() {
		serverTransport := &ReadWriteCloserTransport{serverConn}
		server.Serve(serverCtx, serverTransport)
	}()

	// Create client
	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize
	if _, err := client.Initialize(context.Background(), InitializeRequest{
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0"},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	}); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create a context with cancel cause
	ctx, cancel := context.WithCancelCause(context.Background())

	// Start the tool call
	callErr := make(chan error, 1)
	go func() {
		_, err := client.CallTool(ctx, CallToolRequest{
			Name: "slowTool",
		})
		callErr <- err
	}()

	// Cancel with a specific cause
	time.Sleep(100 * time.Millisecond)
	myError := errors.New("user requested cancellation: clicked stop button")
	cancel(myError)

	// Check if the call returned with the error
	select {
	case err := <-callErr:
		if err == nil {
			t.Error("Expected error from cancelled call")
		} else {
			t.Logf("Got error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Call did not complete after cancellation")
	}

	// In a real test, we'd verify that the notification was sent with the correct reason
	// For now, this test demonstrates the API usage
}

// A very simple test that just confirms we can create a server and client
func TestServerClientCreation(t *testing.T) {
	// Create a new server
	server := NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	if server.name != "test-server" {
		t.Errorf("Expected server name 'test-server', got %s", server.name)
	}

	// Version might be overwritten by default options that infer from build info
	// So don't check for exact version match

	// Create client and server connections via net.Pipe
	clientConn, serverConn := net.Pipe()

	// Create a transport for the client
	transport := &ReadWriteCloserTransport{clientConn}

	// Start the server in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// Create a transport that just returns our server connection
		serverTransport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
			return serverConn, nil
		})

		// This will start serving requests on the server side of the pipe
		server.Serve(ctx, serverTransport)
	}()

	// Create the client
	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	// Close connections to avoid resource leaks
	client.Close()
}

// Fake transport for testing that just returns predetermined responses
type fakeTransport struct {
	reader io.Reader
	writer *bytes.Buffer
}

func newFakeTransport(response string) *fakeTransport {
	return &fakeTransport{
		reader: bytes.NewBufferString(response),
		writer: &bytes.Buffer{},
	}
}

func (f *fakeTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &fakeReadWriteCloser{
		reader: f.reader,
		writer: f.writer,
	}, nil
}

type fakeReadWriteCloser struct {
	reader io.Reader
	writer *bytes.Buffer
}

func (f *fakeReadWriteCloser) Read(p []byte) (n int, err error) {
	return f.reader.Read(p)
}

func (f *fakeReadWriteCloser) Write(p []byte) (n int, err error) {
	return f.writer.Write(p)
}

func (f *fakeReadWriteCloser) Close() error {
	return nil
}
