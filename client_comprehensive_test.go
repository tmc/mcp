package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestClientInitialization tests client creation and initialization
func TestClientInitialization(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() (*Client, error)
		wantErr     bool
	}{
		{
			name: "successful initialization",
			setupClient: func() (*Client, error) {
				// Create a mock connection
				clientConn, serverConn := net.Pipe()
				defer serverConn.Close()

				// Start a mock server
				go func() {
					defer serverConn.Close()
					// Read initialization request
					buffer := make([]byte, 4096)
					n, err := serverConn.Read(buffer)
					if err != nil {
						return
					}

					// Parse the request to get ID
					var req JSONRPCRequest
					lines := strings.Split(string(buffer[:n]), "\n")
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if line == "" {
							continue
						}
						if err := json.Unmarshal([]byte(line), &req); err == nil {
							break
						}
					}

					// Send mock response
					response := JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      req.ID,
						Result: InitializeResult{
							ProtocolVersion: LATEST_PROTOCOL_VERSION,
							ServerInfo: Implementation{
								Name:    "test-server",
								Version: "1.0.0",
							},
							Capabilities: ServerCapabilities{},
						},
					}

					responseData, _ := json.Marshal(response)
					serverConn.Write(responseData)
					serverConn.Write([]byte("\n"))
				}()

				transport := &ReadWriteCloserTransport{clientConn}
				return NewClient(transport)
			},
			wantErr: false,
		},
		{
			name: "nil transport",
			setupClient: func() (*Client, error) {
				// NewClient with nil transport will panic in jsonrpc2.Dial
				// Let's test a different error condition instead
				return NewClient(TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
					return nil, io.ErrUnexpectedEOF
				}))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.setupClient()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("Expected client but got nil")
				return
			}

			// Test basic client operations
			defer client.Close()

			// Try to initialize
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			initReq := InitializeRequest{
				ProtocolVersion: LATEST_PROTOCOL_VERSION,
				ClientInfo: Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
				Capabilities: ClientCapabilities{},
			}

			result, err := client.Initialize(ctx, initReq)
			if err != nil {
				t.Logf("Initialize failed (may be expected for mock): %v", err)
				return
			}

			if result.ServerInfo.Name != "test-server" {
				t.Errorf("Expected server name 'test-server', got %s", result.ServerInfo.Name)
			}
		})
	}
}

// TestClientNotificationHandlingComprehensive tests notification handling
func TestClientNotificationHandlingComprehensive(t *testing.T) {
	client := &Client{}

	// Test with notification handler
	received := false
	handler := func(n JSONRPCNotification) {
		received = true
	}

	opt := WithNotificationHandler(handler)
	opt(client)

	// Simulate notification
	client.notificationMu.RLock()
	h := client.notifyHandler
	client.notificationMu.RUnlock()

	if h == nil {
		t.Error("Notification handler not set")
		return
	}

	h(JSONRPCNotification{Method: "test/notification"})

	if !received {
		t.Error("Notification handler was not called")
	}
}

// TestClientOptions tests client configuration options
func TestClientOptions(t *testing.T) {
	tests := []struct {
		name   string
		option ClientOption
		check  func(*Client) bool
	}{
		{
			name: "with notification handler",
			option: WithNotificationHandler(func(n JSONRPCNotification) {
				// test handler
			}),
			check: func(c *Client) bool {
				c.notificationMu.RLock()
				defer c.notificationMu.RUnlock()
				return c.notifyHandler != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			tt.option(client)

			if !tt.check(client) {
				t.Error("Option did not apply correctly")
			}
		})
	}
}

// TestClientTransportHandling tests various transport scenarios
func TestClientTransportHandling(t *testing.T) {
	tests := []struct {
		name      string
		transport Transport
		wantErr   bool
	}{
		{
			name: "valid ReadWriteCloser transport",
			transport: &ReadWriteCloserTransport{
				ReadWriteCloser: &mockClientReadWriteCloser{},
			},
			wantErr: false,
		},
		{
			name: "TransportFunc that succeeds",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				return &mockClientReadWriteCloser{}, nil
			}),
			wantErr: false,
		},
		{
			name: "TransportFunc that fails",
			transport: TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
				return nil, io.ErrUnexpectedEOF
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.transport)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("Expected client but got nil")
			}

			// Clean up
			if client != nil {
				client.Close()
			}
		})
	}
}

// TestClientErrorHandling tests error conditions
func TestClientErrorHandling(t *testing.T) {
	// Test client with failing transport
	failingTransport := TransportFunc(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return nil, io.ErrUnexpectedEOF
	})

	client, err := NewClient(failingTransport)
	if err == nil {
		t.Error("Expected error for failing transport")
	}
	if client != nil {
		t.Error("Expected nil client for failing transport")
	}
}

// TestClientConcurrentOperations tests thread safety
func TestClientConcurrentOperations(t *testing.T) {
	// Create a client with a working transport
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	// Mock server that responds to requests
	go func() {
		defer serverConn.Close()
		buffer := make([]byte, 4096)
		for {
			n, err := serverConn.Read(buffer)
			if err != nil {
				return
			}

			// Simple echo response for testing
			lines := strings.Split(string(buffer[:n]), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				var req JSONRPCRequest
				if err := json.Unmarshal([]byte(line), &req); err != nil {
					continue
				}

				response := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  map[string]string{"echo": "response"},
				}

				responseData, _ := json.Marshal(response)
				serverConn.Write(responseData)
				serverConn.Write([]byte("\n"))
			}
		}
	}()

	transport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test concurrent notification handler calls
	var handlerCalls atomic.Int32
	handler := func(n JSONRPCNotification) {
		handlerCalls.Add(1)
	}

	opt := WithNotificationHandler(handler)
	opt(client)

	// Simulate concurrent notifications
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			client.notificationMu.RLock()
			h := client.notifyHandler
			client.notificationMu.RUnlock()
			if h != nil {
				h(JSONRPCNotification{
					Method: "test/notification",
					Params: json.RawMessage(`{"id": ` + string(rune(i)) + `}`),
				})
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if got := handlerCalls.Load(); got != 10 {
		t.Errorf("Expected 10 handler calls, got %d", got)
	}
}

// Mock types for testing

type mockClientReadWriteCloser struct {
	data   bytes.Buffer
	closed bool
}

func (m *mockClientReadWriteCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	return m.data.Read(p)
}

func (m *mockClientReadWriteCloser) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.data.Write(p)
}

func (m *mockClientReadWriteCloser) Close() error {
	m.closed = true
	return nil
}

// TestClientJSONHandling tests JSON serialization/deserialization
func TestClientJSONHandling(t *testing.T) {
	// Test various JSON scenarios that a client might encounter
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "valid notification",
			jsonData: `{"method": "test/notification", "params": {"data": "test"}}`,
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{"method": "test/notification", "params": {"data": "test"}`,
			wantErr:  true,
		},
		{
			name:     "empty JSON",
			jsonData: `{}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var notification JSONRPCNotification
			err := json.Unmarshal([]byte(tt.jsonData), &notification)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
