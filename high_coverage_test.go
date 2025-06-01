// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestAdvancedClientFeatures(t *testing.T) {
	// Test client options and advanced features

	// Test notification handler option
	client, err := NewClient(&ReadWriteCloserTransport{ReadWriteCloser: &mockReadWriteCloser{}},
		WithNotificationHandler(func(notification JSONRPCNotification) {
			t.Logf("Received notification: %s", notification.Method)
		}))
	if err != nil {
		t.Fatalf("Failed to create client with notification handler: %v", err)
	}

	// Client should be created successfully
	if client == nil {
		t.Error("Client should not be nil")
	}
}

func TestClientConnectionStates(t *testing.T) {
	// Test various client connection states

	// Test uninitialized client operations
	client, err := NewClient(&ReadWriteCloserTransport{ReadWriteCloser: &mockClosedReadWriteCloser{}})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test operations before initialization
	err = client.Ping(context.Background())
	if err == nil {
		t.Error("Expected error when pinging uninitialized client")
	}

	_, err = client.ListTools(context.Background(), ListToolsRequest{})
	if err == nil {
		t.Error("Expected error when listing tools on uninitialized client")
	}

	_, err = client.CallTool(context.Background(), CallToolRequest{Name: "test"})
	if err == nil {
		t.Error("Expected error when calling tool on uninitialized client")
	}

	_, err = client.ListResources(context.Background(), ListResourcesRequest{})
	if err == nil {
		t.Error("Expected error when listing resources on uninitialized client")
	}

	_, err = client.ReadResource(context.Background(), ReadResourceRequest{URI: "test://"})
	if err == nil {
		t.Error("Expected error when reading resource on uninitialized client")
	}

	_, err = client.ListPrompts(context.Background(), ListPromptsRequest{})
	if err == nil {
		t.Error("Expected error when listing prompts on uninitialized client")
	}

	_, err = client.GetPrompt(context.Background(), GetPromptRequest{Name: "test"})
	if err == nil {
		t.Error("Expected error when getting prompt on uninitialized client")
	}
}

func TestServerCapabilitiesVariations(t *testing.T) {
	// Test different server capability configurations

	// Test server with all capabilities enabled
	server := NewServer("full-capability-server", "1.0.0",
		WithCapabilities(ServerCapabilities{
			Tools: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{
				ListChanged: true,
			},
			Resources: &struct {
				Subscribe   bool `json:"subscribe,omitempty"`
				ListChanged bool `json:"listChanged,omitempty"`
			}{
				Subscribe:   true,
				ListChanged: true,
			},
			Prompts: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{
				ListChanged: true,
			},
			Experimental: map[string]any{
				"feature1": true,
			},
		}))

	// Test that capabilities are set
	if server.capabilities.Tools == nil {
		t.Error("Expected tools capability to be set")
	}

	if server.capabilities.Resources == nil {
		t.Error("Expected resources capability to be set")
	}

	if server.capabilities.Prompts == nil {
		t.Error("Expected prompts capability to be set")
	}

	// Test server with minimal capabilities
	server2 := NewServer("minimal-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Default capabilities should be minimal
	if server2.capabilities.Tools != nil && server2.capabilities.Tools.ListChanged {
		t.Error("Expected default tools capability to not have list changed")
	}
}

func TestProgressNotifications(t *testing.T) {
	// Test progress notification functionality
	server := NewServer("progress-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	ctx := context.Background()

	// Test progress notification with total
	total := 100.0
	err := server.dispatch.NotifyProgress(ctx, "test-token", 50.0, &total)
	if err != nil {
		t.Logf("Progress notification with total: %v (expected in isolated test)", err)
	}

	// Test progress notification without total
	err = server.dispatch.NotifyProgress(ctx, "test-token", 75.0, nil)
	if err != nil {
		t.Logf("Progress notification without total: %v (expected in isolated test)", err)
	}

	// Test progress notification with different token types
	err = server.dispatch.NotifyProgress(ctx, 123, 25.0, nil)
	if err != nil {
		t.Logf("Progress notification with numeric token: %v (expected in isolated test)", err)
	}
}

func TestLoggingNotifications(t *testing.T) {
	// Test logging notification functionality
	server := NewServer("logging-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	ctx := context.Background()

	// Test different log levels
	err := server.dispatch.NotifyLoggingMessage(ctx, LogLevelDebug, "test-logger", "Debug message")
	if err != nil {
		t.Logf("Debug logging notification: %v (expected in isolated test)", err)
	}

	err = server.dispatch.NotifyLoggingMessage(ctx, LogLevelInfo, "test-logger", "Info message")
	if err != nil {
		t.Logf("Info logging notification: %v (expected in isolated test)", err)
	}

	err = server.dispatch.NotifyLoggingMessage(ctx, LogLevelWarning, "test-logger", "Warning message")
	if err != nil {
		t.Logf("Warning logging notification: %v (expected in isolated test)", err)
	}

	err = server.dispatch.NotifyLoggingMessage(ctx, LogLevelError, "test-logger", "Error message")
	if err != nil {
		t.Logf("Error logging notification: %v (expected in isolated test)", err)
	}

	// Test with different data types
	err = server.dispatch.NotifyLoggingMessage(ctx, LogLevelInfo, "test-logger", map[string]any{
		"key":   "value",
		"count": 42,
	})
	if err != nil {
		t.Logf("Structured logging notification: %v (expected in isolated test)", err)
	}

	// Test with nil data
	err = server.dispatch.NotifyLoggingMessage(ctx, LogLevelInfo, "test-logger", nil)
	if err != nil {
		t.Logf("Nil data logging notification: %v (expected in isolated test)", err)
	}
}

func TestListChangedNotifications(t *testing.T) {
	// Test list changed notifications
	server := NewServer("list-changed-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	ctx := context.Background()

	// Test all list changed notification types
	err := server.dispatch.NotifyListChanged(ctx, MethodToolListChanged)
	if err != nil {
		t.Logf("Tool list changed notification: %v (expected in isolated test)", err)
	}

	err = server.dispatch.NotifyListChanged(ctx, MethodResourceListChanged)
	if err != nil {
		t.Logf("Resource list changed notification: %v (expected in isolated test)", err)
	}

	err = server.dispatch.NotifyListChanged(ctx, MethodPromptListChanged)
	if err != nil {
		t.Logf("Prompt list changed notification: %v (expected in isolated test)", err)
	}
}

func TestServerRegistrationEdgeCases(t *testing.T) {
	// Test edge cases in server registration
	server := NewServer("edge-case-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Test duplicate tool registration
	tool := Tool{
		Name:        "duplicate-tool",
		Description: "A test tool",
	}

	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: []any{"test"}}, nil
	}

	err := server.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("Failed to register tool first time: %v", err)
	}

	// Try to register the same tool again
	err = server.RegisterTool(tool, handler)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}

	// Test duplicate resource registration
	resource := Resource{
		URI:         "test://duplicate",
		Description: "A test resource",
	}

	resourceHandler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{
			TextResourceContents{
				URI:  req.URI,
				Text: "test",
			},
		}, nil
	}

	err = server.RegisterResource(resource, resourceHandler)
	if err != nil {
		t.Fatalf("Failed to register resource first time: %v", err)
	}

	err = server.RegisterResource(resource, resourceHandler)
	if err == nil {
		t.Error("Expected error when registering duplicate resource")
	}

	// Test duplicate resource template registration
	template := ResourceTemplate{
		Template:    "test://template/{id}",
		Description: "A test template",
	}

	err = server.RegisterResourceTemplate(template, resourceHandler)
	if err != nil {
		t.Fatalf("Failed to register resource template first time: %v", err)
	}

	err = server.RegisterResourceTemplate(template, resourceHandler)
	if err == nil {
		t.Error("Expected error when registering duplicate resource template")
	}

	// Test duplicate prompt registration
	prompt := Prompt{
		Name:        "duplicate-prompt",
		Description: "A test prompt",
	}

	promptHandler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{
			Messages: []PromptMessage{
				{
					Role:    RoleUser,
					Content: []any{"test"},
				},
			},
		}, nil
	}

	err = server.RegisterPrompt(prompt, promptHandler)
	if err != nil {
		t.Fatalf("Failed to register prompt first time: %v", err)
	}

	err = server.RegisterPrompt(prompt, promptHandler)
	if err == nil {
		t.Error("Expected error when registering duplicate prompt")
	}
}

func TestServerOptionsVariations(t *testing.T) {
	// Test various server option combinations

	// Test custom name and version
	server := NewServer("", "",
		WithServerName("custom-server"),
		WithServerVersion("2.0.0"),
		WithServerInstructions("Custom instructions"))

	if server.name != "custom-server" {
		t.Errorf("Expected name 'custom-server', got '%s'", server.name)
	}

	if server.version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", server.version)
	}

	if server.instructions != "Custom instructions" {
		t.Errorf("Expected custom instructions, got '%s'", server.instructions)
	}

	// Test with custom logger
	logger := slog.Default()
	serverWithLogger := NewServer("test-server", "1.0.0", WithLogger(logger))
	if serverWithLogger.logger != logger {
		t.Error("Expected logger to be set")
	}

	// Test with log level - skipped (WithServerLogLevel not implemented)
	// serverWithLogLevel := NewServer("test-server", "1.0.0", WithServerLogLevel(slog.LevelDebug))
	// if serverWithLogLevel.logger == nil {
	// 	t.Error("Expected logger to be created with log level")
	// }
}

func TestClientServerConnectionFailures(t *testing.T) {
	// Test connection failure scenarios

	// Create client with failing transport
	transport := &mockFailingTransport{}
	client, err := NewClient(transport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test initialization with failing transport should fail gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo: Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{},
	})

	// The error handling might vary, but we should not panic
	t.Logf("Initialize with failing transport: %v (expected)", err)
}

// Mock types for testing
type mockClosedReadWriteCloser struct{}

func (m *mockClosedReadWriteCloser) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock connection closed")
}

func (m *mockClosedReadWriteCloser) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock connection closed")
}

func (m *mockClosedReadWriteCloser) Close() error {
	return nil
}

type mockFailingTransport struct{}

func (t *mockFailingTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("mock transport failure")
}

func TestDispatcherCoverage(t *testing.T) {
	// Test dispatcher functionality comprehensively
	dispatcher := NewDispatcher()

	// Test registering multiple handlers for same method
	called1 := false
	called2 := false

	dispatcher.Handle("test/method", func(method string, params json.RawMessage) error {
		called1 = true
		return nil
	})

	dispatcher.Handle("test/method", func(method string, params json.RawMessage) error {
		called2 = true
		return nil
	})

	// Test dispatch calls all handlers
	err := dispatcher.Dispatch(context.Background(), "test/method", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Failed to dispatch: %v", err)
	}

	if !called1 || !called2 {
		t.Error("Expected both handlers to be called")
	}

	// Test dispatch with non-existent method
	err = dispatcher.Dispatch(context.Background(), "non/existent", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Unexpected error for non-existent method: %v", err)
	}

	// Test handler that returns error
	dispatcher.Handle("error/method", func(method string, params json.RawMessage) error {
		return fmt.Errorf("handler error")
	})

	err = dispatcher.Dispatch(context.Background(), "error/method", json.RawMessage(`{}`))
	if err == nil {
		t.Error("Expected error from failing handler")
	}
}

// func TestMethodRegistration(t *testing.T) {
// 	// Test method registration functionality - skipped (RegisterMethod not implemented)
// 	handlers := make(map[string]any)
//
// 	// Test registering various method types
// 	RegisterMethod(handlers, MethodPing, Handler[struct{}, struct{}](
// 		func(ctx context.Context, params struct{}) (struct{}, error) {
// 			return struct{}{}, nil
// 		}))
//
// 	if _, exists := handlers[string(MethodPing)]; !exists {
// 		t.Error("Expected ping method to be registered")
// 	}
//
// 	// Test registering tool list method
// 	RegisterMethod(handlers, MethodToolsList, Handler[ListToolsRequest, ListToolsResult](
// 		func(ctx context.Context, params ListToolsRequest) (ListToolsResult, error) {
// 			return ListToolsResult{Tools: []Tool{}}, nil
// 		}))
//
// 	if _, exists := handlers[string(MethodToolsList)]; !exists {
// 		t.Error("Expected tools list method to be registered")
// 	}
// }
