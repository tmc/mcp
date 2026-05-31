package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	// Create a mock transport
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}

	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.conn == nil {
		t.Error("Client connection is nil")
	}

	if client.initialized {
		t.Error("Client should not be initialized")
	}

	client.Close()
}

func TestClientDefaultFramerWritesLine(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client, err := NewClient(&ReadWriteCloserTransport{clientConn})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	gotReq := make(chan string, 1)
	serverErr := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(serverConn).ReadString('\n')
		if err != nil {
			serverErr <- err
			return
		}
		gotReq <- line

		var req map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			serverErr <- err
			return
		}
		id, ok := req["id"]
		if !ok {
			serverErr <- errors.New("missing request id")
			return
		}
		_, err = fmt.Fprintf(serverConn, `{"jsonrpc":"2.0","id":%s,"result":{}}`+"\n", id)
		serverErr <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	select {
	case line := <-gotReq:
		if !strings.HasSuffix(line, "\n") {
			t.Fatalf("request = %q, want newline suffix", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive request")
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server failed: %v", err)
	}
}

func TestClientWithOptions(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}

	handler := func(notification JSONRPCNotification) {
		// Notification received
	}

	client, err := NewClient(transport, WithNotificationHandler(handler))
	if err != nil {
		t.Fatalf("NewClient with options failed: %v", err)
	}

	// Verify handler was set
	client.notificationMu.RLock()
	hasHandler := client.notifyHandler != nil
	client.notificationMu.RUnlock()

	if !hasHandler {
		t.Error("Notification handler not set")
	}

	client.Close()
}

func TestClientWithRawFramer(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client, err := NewClient(&ReadWriteCloserTransport{clientConn}, WithFramer(RawFramer()))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if _, ok := client.framer.(lineFramer); ok {
		t.Fatal("client framer is LineFramer, want raw framer")
	}
}

func TestClientOnNotification(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}

	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	client.OnNotification(func(notification JSONRPCNotification) {
		// Notification received
	})

	// Verify handler was set
	client.notificationMu.RLock()
	hasHandler := client.notifyHandler != nil
	client.notificationMu.RUnlock()

	if !hasHandler {
		t.Error("OnNotification did not set handler")
	}
}

func TestClientInitialize(t *testing.T) {
	// Create a basic server to respond to initialization
	server := NewServer("test-server", "1.0.0")

	clientConn, serverConn := net.Pipe()

	// Start server
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		serverTransport := &ReadWriteCloserTransport{serverConn}
		server.Serve(serverCtx, serverTransport)
	}()

	// Create client
	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Test initialization
	initReq := InitializeRequest{
		ClientInfo: Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	}

	result, err := client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if result == nil {
		t.Fatal("Initialize result is nil")
	}

	if result.ServerInfo.Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got %s", result.ServerInfo.Name)
	}

	if !client.initialized {
		t.Error("Client should be marked as initialized")
	}
}

func TestClientInitializeError(t *testing.T) {
	// Create a transport that will fail
	transport := &failingTransport{}

	_, err := NewClient(transport)
	if err == nil {
		t.Error("Expected error from failing transport")
	}
}

func TestClientDoubleInitialize(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	clientConn, serverConn := net.Pipe()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		serverTransport := &ReadWriteCloserTransport{serverConn}
		server.Serve(serverCtx, serverTransport)
	}()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	initReq := InitializeRequest{
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	}

	// First initialization should succeed
	_, err = client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("First initialize failed: %v", err)
	}

	// Second initialization should fail
	_, err = client.Initialize(context.Background(), initReq)
	if err == nil {
		t.Error("Expected error from double initialization")
	}
}

func TestClientContextCancellation(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	clientConn, serverConn := net.Pipe()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		serverTransport := &ReadWriteCloserTransport{serverConn}
		server.Serve(serverCtx, serverTransport)
	}()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Initialize first
	initReq := InitializeRequest{
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	}
	_, err = client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.ListTools(ctx, ListToolsRequest{})
	if err == nil {
		t.Error("Expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestClientListTools(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a test tool
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}
	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: []any{}}, nil
	}
	server.RegisterTool(tool, handler)

	clientConn, serverConn := net.Pipe()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		serverTransport := &ReadWriteCloserTransport{serverConn}
		server.Serve(serverCtx, serverTransport)
	}()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Initialize
	initReq := InitializeRequest{
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	}
	_, err = client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// List tools
	toolsResult, err := client.ListTools(context.Background(), ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(toolsResult.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolsResult.Tools))
	}

	if toolsResult.Tools[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got %s", toolsResult.Tools[0].Name)
	}
}

func TestClientCallTool(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a test tool
	tool := Tool{
		Name:        "echo",
		Description: "Echo tool",
	}
	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []any{TextContent{Type: "text", Text: "echo response"}},
		}, nil
	}
	server.RegisterTool(tool, handler)

	clientConn, serverConn := net.Pipe()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		serverTransport := &ReadWriteCloserTransport{serverConn}
		server.Serve(serverCtx, serverTransport)
	}()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Initialize
	initReq := InitializeRequest{
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	}
	_, err = client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Call tool
	result, err := client.CallTool(context.Background(), CallToolRequest{
		Name: "echo",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result == nil {
		t.Fatal("CallTool result is nil")
	}

	if len(result.Content) == 0 {
		t.Error("CallTool result has no content")
	}
}

func TestClientClose(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Close should not error
	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Second close should not error
	err = client.Close()
	if err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

// Test helper types

type failingTransport struct{}

func (f *failingTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return nil, errors.New("transport failure")
}

func TestClientBeforeInitialize(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Try to call methods before initialize
	_, err = client.ListTools(context.Background(), ListToolsRequest{})
	if err == nil {
		t.Error("Expected error when calling ListTools before initialize")
	}

	_, err = client.CallTool(context.Background(), CallToolRequest{Name: "test"})
	if err == nil {
		t.Error("Expected error when calling CallTool before initialize")
	}
}

func TestClientTimeout(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	transport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create a short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should timeout since no server is responding
	go func() {
		// Close the connection to force an error quickly
		time.Sleep(2 * time.Millisecond)
		client.Close()
	}()

	_, err = client.Initialize(ctx, InitializeRequest{
		ClientInfo:      Implementation{Name: "test", Version: "1.0"},
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
	})

	if err == nil {
		t.Error("Expected timeout or connection error")
	}

	// Accept various error types since this could be timeout or connection error
	t.Logf("Got expected error: %v", err)
}
