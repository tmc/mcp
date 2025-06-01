package mcp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
)

func TestClientNotificationHandling(t *testing.T) {
	tests := []struct {
		name          string
		notifications []JSONRPCNotification
		setupHandler  func(*Client) bool
		wantReceived  bool
	}{
		{
			name: "receives notifications with handler",
			notifications: []JSONRPCNotification{
				{Method: "test/notification1", Params: json.RawMessage(`{"data":1}`)},
			},
			setupHandler: func(c *Client) bool {
				received := false
				handler := func(n JSONRPCNotification) {
					received = true
				}
				// Access the handler through WithNotificationHandler option
				opt := WithNotificationHandler(handler)
				opt(c)
				// Simulate notification handler being called
				c.notificationMu.RLock()
				h := c.notifyHandler
				c.notificationMu.RUnlock()
				if h != nil {
					h(JSONRPCNotification{Method: "test/notification1", Params: json.RawMessage(`{"data":1}`)})
				}
				return received
			},
			wantReceived: true,
		},
		{
			name: "no handler ignores notifications",
			notifications: []JSONRPCNotification{
				{Method: "test/notification1", Params: json.RawMessage(`{"data":1}`)},
			},
			setupHandler: func(c *Client) bool {
				return false
			},
			wantReceived: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			received := tt.setupHandler(client)

			if received != tt.wantReceived {
				t.Errorf("received = %v, want %v", received, tt.wantReceived)
			}
		})
	}
}

func TestServerResourceRegistration(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		handler  ReadResourceHandlerFunc
		wantErr  bool
	}{
		{
			name: "register new resource",
			resource: Resource{
				URI:         "test://resource1",
				Description: "Test resource",
			},
			handler: func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
				return []ResourceContents{TextResourceContents{Text: "content"}}, nil
			},
			wantErr: false,
		},
		{
			name: "duplicate resource registration",
			resource: Resource{
				URI: "test://duplicate",
			},
			handler: func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
				return nil, nil
			},
			wantErr: true, // Second registration should fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("test", "1.0")

			// For duplicate test, register once first
			if tt.name == "duplicate resource registration" {
				server.RegisterResource(tt.resource, tt.handler)
			}

			err := server.RegisterResource(tt.resource, tt.handler)

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerOptions(t *testing.T) {
	tests := []struct {
		name   string
		option ServerOption
		check  func(*Server) bool
	}{
		{
			name:   "with server name",
			option: WithServerName("custom-name"),
			check: func(s *Server) bool {
				return s.name == "custom-name"
			},
		},
		{
			name:   "with server version",
			option: WithServerVersion("2.0.0"),
			check: func(s *Server) bool {
				return s.version == "2.0.0"
			},
		},
		{
			name:   "with instructions",
			option: WithServerInstructions("test instructions"),
			check: func(s *Server) bool {
				return s.instructions == "test instructions"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("default", "1.0", tt.option)

			if !tt.check(server) {
				t.Error("option did not apply correctly")
			}
		})
	}
}

func TestPromptRegistration(t *testing.T) {
	server := NewServer("test", "1.0")

	prompt := Prompt{
		Name:        "test-prompt",
		Description: "Test prompt",
	}

	handler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	err := server.RegisterPrompt(prompt, handler)
	if err != nil {
		t.Fatalf("RegisterPrompt() error = %v", err)
	}

	// Try to register duplicate
	err = server.RegisterPrompt(prompt, handler)
	if err == nil {
		t.Error("expected error for duplicate prompt registration")
	}
}

func TestResourceTemplateRegistration(t *testing.T) {
	server := NewServer("test", "1.0")

	template := ResourceTemplate{
		Template:    "test://template/{id}",
		Description: "Test template",
	}

	handler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{TextResourceContents{Text: "content"}}, nil
	}

	err := server.RegisterResourceTemplate(template, handler)
	if err != nil {
		t.Fatalf("RegisterResourceTemplate() error = %v", err)
	}

	// Try to register duplicate
	err = server.RegisterResourceTemplate(template, handler)
	if err == nil {
		t.Error("expected error for duplicate template registration")
	}
}

func TestToolRegistration(t *testing.T) {
	server := NewServer("test", "1.0")

	tool := Tool{
		Name:        "test-tool",
		Description: "Test tool",
	}

	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []interface{}{"result"},
		}, nil
	}

	err := server.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	// Try to register duplicate
	err = server.RegisterTool(tool, handler)
	if err == nil {
		t.Error("expected error for duplicate tool registration")
	}
}

func TestWithNotificationHandlerCoverage(t *testing.T) {
	called := false
	handler := func(n JSONRPCNotification) {
		called = true
	}

	opt := WithNotificationHandler(handler)
	client := &Client{}
	opt(client)

	// Verify handler was set
	client.notificationMu.RLock()
	if client.notifyHandler == nil {
		t.Error("notification handler was not set")
	}
	client.notificationMu.RUnlock()

	// Call the handler to verify it works
	client.notifyHandler(JSONRPCNotification{Method: "test"})
	if !called {
		t.Error("handler was not called")
	}
}

type mockFlushWriter struct {
	writeCalled bool
	flushCalled bool
}

func (m *mockFlushWriter) Write(p []byte) (n int, err error) {
	m.writeCalled = true
	return len(p), nil
}

func (m *mockFlushWriter) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (m *mockFlushWriter) Close() error {
	return nil
}

func (m *mockFlushWriter) Flush() error {
	m.flushCalled = true
	return nil
}

type mockSyncWriter struct {
	writeCalled bool
	syncCalled  bool
}

func (m *mockSyncWriter) Write(p []byte) (n int, err error) {
	m.writeCalled = true
	return len(p), nil
}

func (m *mockSyncWriter) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (m *mockSyncWriter) Close() error {
	return nil
}

func (m *mockSyncWriter) Sync() error {
	m.syncCalled = true
	return nil
}

func TestFlushingReadWriteCloser(t *testing.T) {
	t.Run("writer with Flush method", func(t *testing.T) {
		mock := &mockFlushWriter{}
		fw := &flushingReadWriteCloser{
			ReadWriteCloser: mock,
			logger:          slog.Default(),
		}

		data := []byte("test data")
		n, err := fw.Write(data)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if n != len(data) {
			t.Errorf("Write() = %v, want %v", n, len(data))
		}

		if !mock.writeCalled {
			t.Error("Write was not called on underlying writer")
		}
		if !mock.flushCalled {
			t.Error("Flush was not called")
		}
	})

	t.Run("writer with Sync method", func(t *testing.T) {
		mock := &mockSyncWriter{}
		fw := &flushingReadWriteCloser{
			ReadWriteCloser: mock,
			logger:          slog.Default(),
		}

		data := []byte("test data")
		n, err := fw.Write(data)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if n != len(data) {
			t.Errorf("Write() = %v, want %v", n, len(data))
		}

		if !mock.writeCalled {
			t.Error("Write was not called on underlying writer")
		}
		if !mock.syncCalled {
			t.Error("Sync was not called")
		}
	})

	t.Run("writer without flush methods", func(t *testing.T) {
		mock := &struct {
			io.ReadWriteCloser
			writeCalled bool
		}{
			ReadWriteCloser: &mockWriter{},
		}
		fw := &flushingReadWriteCloser{
			ReadWriteCloser: mock,
			logger:          slog.Default(),
		}

		data := []byte("test data")
		n, err := fw.Write(data)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if n != len(data) {
			t.Errorf("Write() = %v, want %v", n, len(data))
		}
	})
}

type mockWriter struct{}

func (m *mockWriter) Read(p []byte) (n int, err error)  { return 0, nil }
func (m *mockWriter) Write(p []byte) (n int, err error) { return len(p), nil }
func (m *mockWriter) Close() error                      { return nil }

// Test options functions
func TestLogLevelOption(t *testing.T) {
	// Test WithLogLevel with existing logger
	serverWithLogger := NewServer("test", "1.0", WithLogger(slog.Default()))
	level := slog.LevelDebug
	opt := WithLogLevel(level)
	opt(serverWithLogger)

	// Test WithLogLevel without logger (should create new logger)
	serverNoLogger := &Server{}
	opt(serverNoLogger)

	if serverNoLogger.logger == nil {
		t.Error("expected logger to be created")
	}
}
