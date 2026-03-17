package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/jsonrpc2"
)

// TestClientAllMethods tests all client methods for coverage
func TestClientAllMethods(t *testing.T) {
	// Create a mock server that handles all MCP methods
	mockServer := &mockMCPServer{
		responses: make(map[string]interface{}),
	}

	// Set up default responses
	mockServer.responses["initialize"] = InitializeResult{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ServerInfo: Implementation{
			Name:    "mock-server",
			Version: "1.0.0",
		},
		Capabilities: ServerCapabilities{
			Tools: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{ListChanged: true},
			Resources: &struct {
				Subscribe   bool `json:"subscribe,omitempty"`
				ListChanged bool `json:"listChanged,omitempty"`
			}{ListChanged: true},
			Prompts: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{ListChanged: true},
		},
	}

	mockServer.responses["ping"] = struct{}{}

	mockServer.responses["tools/list"] = ListToolsResult{
		Tools: []Tool{
			{Name: "test-tool", Description: "A test tool"},
		},
	}

	mockServer.responses["tools/call"] = CallToolResult{
		Content: []any{
			TextContent{Type: "text", Text: "Tool result"},
		},
	}

	mockServer.responses["prompts/list"] = ListPromptsResult{
		Prompts: []Prompt{
			{Name: "test-prompt", Description: "A test prompt"},
		},
	}

	mockServer.responses["prompts/get"] = GetPromptResult{
		Messages: []PromptMessage{
			{Role: "user", Content: []any{TextContent{Type: "text", Text: "Test prompt"}}},
		},
	}

	mockServer.responses["resources/list"] = ListResourcesResult{
		Resources: []Resource{
			{URI: "test://resource", Description: "Test Resource"},
		},
	}

	mockServer.responses["resources/read"] = ReadResourceResult{
		Contents: []ResourceContents{
			TextResourceContents{URI: "test://resource", Text: "Resource content"},
		},
	}

	mockServer.responses["resources/templates/list"] = ListResourceTemplatesResult{
		Templates: []ResourceTemplate{
			{Template: "test://template/{id}", Description: "Test Template"},
		},
	}

	mockServer.responses["completion/complete"] = CompleteResult{}

	// Create client and server connection
	client, cleanup := createTestClientServer(t, mockServer)
	defer cleanup()

	// Test Initialize
	t.Run("Initialize", func(t *testing.T) {
		result, err := client.Initialize(context.Background(), InitializeRequest{
			ClientInfo: Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			},
		})
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		if result.ServerInfo.Name != "mock-server" {
			t.Errorf("Expected server name 'mock-server', got %s", result.ServerInfo.Name)
		}

		// Test double initialization
		_, err = client.Initialize(context.Background(), InitializeRequest{})
		if err == nil || err.Error() != "client already initialized" {
			t.Errorf("Expected 'client already initialized' error, got: %v", err)
		}
	})

	// Test checkInitialized
	t.Run("checkInitialized", func(t *testing.T) {
		// Create a new client that hasn't been initialized
		uninitClient := &Client{}
		err := uninitClient.checkInitialized()
		if err == nil || err.Error() != "client not initialized, call Initialize() first" {
			t.Errorf("Expected initialization error, got: %v", err)
		}
	})

	// Test Ping
	t.Run("Ping", func(t *testing.T) {
		err := client.Ping(context.Background())
		if err != nil {
			t.Errorf("Ping failed: %v", err)
		}
	})

	// Test ListTools
	t.Run("ListTools", func(t *testing.T) {
		result, err := client.ListTools(context.Background(), ListToolsRequest{})
		if err != nil {
			t.Fatalf("ListTools failed: %v", err)
		}

		if len(result.Tools) != 1 || result.Tools[0].Name != "test-tool" {
			t.Errorf("Unexpected tools result: %+v", result)
		}
	})

	// Test CallTool
	t.Run("CallTool", func(t *testing.T) {
		result, err := client.CallTool(context.Background(), CallToolRequest{
			Name:      "test-tool",
			Arguments: json.RawMessage(`{"arg": "value"}`),
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		if len(result.Content) != 1 {
			t.Errorf("Unexpected tool result length: %+v", result)
		}
		// Check content - it will be a map after JSON round-trip
		if content, ok := result.Content[0].(map[string]interface{}); ok {
			if content["type"] != "text" || content["text"] != "Tool result" {
				t.Errorf("Unexpected tool result content: %+v", result.Content[0])
			}
		} else {
			t.Errorf("Unexpected tool result type: %T", result.Content[0])
		}
	})

	// Test ListPrompts
	t.Run("ListPrompts", func(t *testing.T) {
		result, err := client.ListPrompts(context.Background(), ListPromptsRequest{})
		if err != nil {
			t.Fatalf("ListPrompts failed: %v", err)
		}

		if len(result.Prompts) != 1 || result.Prompts[0].Name != "test-prompt" {
			t.Errorf("Unexpected prompts result: %+v", result)
		}
	})

	// Test GetPrompt
	t.Run("GetPrompt", func(t *testing.T) {
		result, err := client.GetPrompt(context.Background(), GetPromptRequest{
			Name: "test-prompt",
		})
		if err != nil {
			t.Fatalf("GetPrompt failed: %v", err)
		}

		if len(result.Messages) != 1 || result.Messages[0].Role != "user" {
			t.Errorf("Unexpected prompt result: %+v", result)
		}
	})

	// Test ListResources
	t.Run("ListResources", func(t *testing.T) {
		result, err := client.ListResources(context.Background(), ListResourcesRequest{})
		if err != nil {
			t.Fatalf("ListResources failed: %v", err)
		}

		if len(result.Resources) != 1 || result.Resources[0].URI != "test://resource" {
			t.Errorf("Unexpected resources result: %+v", result)
		}
	})

	// Test ReadResource
	t.Run("ReadResource", func(t *testing.T) {
		result, err := client.ReadResource(context.Background(), ReadResourceRequest{
			URI: "test://resource",
		})
		if err != nil {
			t.Fatalf("ReadResource failed: %v", err)
		}

		if len(result.Contents) != 1 {
			t.Errorf("Unexpected resource result length: %+v", result)
		}
		// Check content - should be properly unmarshaled as TextResourceContents
		if tc, ok := result.Contents[0].(TextResourceContents); ok {
			if tc.Text != "Resource content" || tc.URI != "test://resource" {
				t.Errorf("Unexpected resource content: %+v", result.Contents[0])
			}
		} else {
			t.Errorf("Unexpected resource content type: %T, value: %+v", result.Contents[0], result.Contents[0])
		}
	})

	// Test ListResourceTemplates
	t.Run("ListResourceTemplates", func(t *testing.T) {
		result, err := client.ListResourceTemplates(context.Background(), ListResourceTemplatesRequest{})
		if err != nil {
			t.Fatalf("ListResourceTemplates failed: %v", err)
		}

		if len(result.Templates) != 1 || result.Templates[0].Template != "test://template/{id}" {
			t.Errorf("Unexpected templates result: %+v", result)
		}
	})

	t.Run("CallRaw", func(t *testing.T) {
		raw, err := client.CallRaw(context.Background(), string(MethodCompletionComplete), CompleteRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) == "" {
			t.Fatal("expected raw JSON")
		}
	})
}

// TestClientCancellation tests context cancellation handling
func TestClientCancellation(t *testing.T) {
	// Create a slow server that delays responses
	slowServer := &mockMCPServer{
		responses: make(map[string]interface{}),
		delay:     500 * time.Millisecond,
	}

	slowServer.responses["initialize"] = InitializeResult{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ServerInfo:      Implementation{Name: "slow-server", Version: "1.0.0"},
		Capabilities:    ServerCapabilities{},
	}

	slowServer.responses["tools/call"] = CallToolResult{
		Content: []any{TextContent{Type: "text", Text: "Slow result"}},
	}

	// Create client and server connection
	client, cleanup := createTestClientServer(t, slowServer)
	defer cleanup()

	// Initialize first
	ctx := context.Background()
	_, err := client.Initialize(ctx, InitializeRequest{})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test cancellation with cause
	t.Run("CancellationWithCause", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())

		// Start a call that will be cancelled
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel(errors.New("test cancellation"))
		}()

		_, err := client.CallTool(ctx, CallToolRequest{Name: "slow-tool"})
		if err == nil {
			t.Error("Expected error from cancelled context")
		}
	})

	// Test cancellation without cause
	t.Run("CancellationWithoutCause", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Start a call that will be cancelled
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		_, err := client.CallTool(ctx, CallToolRequest{Name: "slow-tool"})
		if err == nil {
			t.Error("Expected error from cancelled context")
		}
	})
}

// TestClientHandleMessage tests the handleMessage method
func TestClientHandleMessage(t *testing.T) {
	client := &Client{}

	// Test notification handling
	t.Run("Notification", func(t *testing.T) {
		notificationReceived := false
		client.OnNotification(func(n JSONRPCNotification) {
			notificationReceived = true
			if n.Method != "test/notification" {
				t.Errorf("Expected method 'test/notification', got %s", n.Method)
			}
		})

		// Create a request without ID (notification)
		req := &jsonrpc2.Request{
			Method: "test/notification",
			Params: json.RawMessage(`{"data": "test"}`),
		}

		result, err := client.handleMessage(context.Background(), req)
		if err != nil {
			t.Errorf("handleMessage returned unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result for notification, got %v", result)
		}
		if !notificationReceived {
			t.Error("Notification handler was not called")
		}
	})

	// Test request handling (should return error)
	t.Run("Request", func(t *testing.T) {
		req := &jsonrpc2.Request{
			ID:     jsonrpc2.Int64ID(1),
			Method: "some/method",
			Params: json.RawMessage(`{}`),
		}

		result, err := client.handleMessage(context.Background(), req)
		if err == nil || err.Error() != "method not implemented on client" {
			t.Errorf("Expected 'method not implemented' error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got %v", result)
		}
	})
}

// Helper functions

type mockMCPServer struct {
	responses map[string]interface{}
	mu        sync.RWMutex
	delay     time.Duration
}

func (m *mockMCPServer) handleRequest(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mu.RLock()
	response, ok := m.responses[req.Method]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("method not supported: %s", req.Method)
	}

	return response, nil
}

func createTestClientServer(t *testing.T, server *mockMCPServer) (*Client, func()) {
	// Create a pipe for bidirectional communication
	clientConn, serverConn := net.Pipe()

	// Start the mock server
	serverReady := make(chan struct{})
	serverDone := make(chan struct{})

	go func() {
		handler := jsonrpc2.HandlerFunc(server.handleRequest)
		serverReady <- struct{}{}

		conn, err := jsonrpc2.Dial(context.Background(), &ReadWriteCloserTransport{serverConn}, jsonrpc2.ConnectionOptions{
			Framer:  jsonrpc2.RawFramer(),
			Handler: handler,
		})
		if err != nil {
			t.Logf("Server dial error: %v", err)
			close(serverDone)
			return
		}
		conn.Wait()
		close(serverDone)
	}()

	// Wait for server to be ready
	<-serverReady
	time.Sleep(10 * time.Millisecond)

	// Create the client
	client, err := NewClient(&ReadWriteCloserTransport{clientConn})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		client.Close()
		clientConn.Close()
		serverConn.Close()
		<-serverDone
	}

	return client, cleanup
}
