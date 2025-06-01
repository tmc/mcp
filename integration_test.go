package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"
)

// TestBasicClientServerIntegration tests basic client-server communication
func TestBasicClientServerIntegration(t *testing.T) {
	server := NewServer("integration-test-server", "1.0.0")

	// Register a test tool
	inputSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Message to echo",
			},
		},
		"required": []string{"message"},
	})
	tool := Tool{
		Name:        "echo",
		Description: "Echoes the input",
		InputSchema: inputSchema,
	}

	toolHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		var args map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return &CallToolResult{}, fmt.Errorf("invalid arguments: %v", err)
		}

		message, ok := args["message"].(string)
		if !ok {
			return &CallToolResult{}, fmt.Errorf("missing or invalid message")
		}

		return &CallToolResult{
			Content: []any{
				map[string]interface{}{
					"type": "text",
					"text": fmt.Sprintf("Echo: %s", message),
				},
			},
		}, nil
	}

	err := server.RegisterTool(tool, toolHandler)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Register a test resource
	resource := Resource{
		URI:         "test://data.txt",
		Description: "Test data resource",
		MimeType:    "text/plain",
	}

	resourceHandler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{
			TextResourceContents{
				URI:      req.URI,
				MimeType: "text/plain",
				Text:     "This is test data content",
			},
		}, nil
	}

	err = server.RegisterResource(resource, resourceHandler)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}

	// Create client-server connection
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Start server
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		transport := &ReadWriteCloserTransport{serverConn}
		serverDone <- server.Serve(context.Background(), transport)
	}()

	// Create client
	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize client
	initReq := InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		Capabilities:    ClientCapabilities{},
	}

	_, err = client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Test tool listing
	toolsResult, err := client.ListTools(context.Background(), ListToolsRequest{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	if len(toolsResult.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolsResult.Tools))
	}

	if toolsResult.Tools[0].Name != "echo" {
		t.Errorf("Expected tool name 'echo', got '%s'", toolsResult.Tools[0].Name)
	}

	// Test tool call with proper JSON marshaling
	args, _ := json.Marshal(map[string]interface{}{
		"message": "Hello, World!",
	})

	toolResult, err := client.CallTool(context.Background(), CallToolRequest{
		Name:      "echo",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	if len(toolResult.Content) == 0 {
		t.Error("Expected content in tool result")
	} else {
		content := toolResult.Content[0].(map[string]interface{})
		text := content["text"].(string)
		if text != "Echo: Hello, World!" {
			t.Errorf("Expected 'Echo: Hello, World!', got '%s'", text)
		}
	}

	// Test resource listing
	resourcesResult, err := client.ListResources(context.Background(), ListResourcesRequest{})
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	if len(resourcesResult.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resourcesResult.Resources))
	}

	if resourcesResult.Resources[0].URI != "test://data.txt" {
		t.Errorf("Expected resource URI 'test://data.txt', got '%s'", resourcesResult.Resources[0].URI)
	}

	// Test resource reading
	readResult, err := client.ReadResource(context.Background(), ReadResourceRequest{
		URI: "test://data.txt",
	})
	if err != nil {
		t.Fatalf("Failed to read resource: %v", err)
	}

	if len(readResult.Contents) == 0 {
		t.Error("Expected content in resource read result")
	} else {
		textContent := readResult.Contents[0].(TextResourceContents)
		if textContent.Text != "This is test data content" {
			t.Errorf("Expected 'This is test data content', got '%s'", textContent.Text)
		}
	}
}

// TestClientDoubleInitializationError tests that double initialization returns error
func TestClientDoubleInitializationError(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Start server
	go func() {
		transport := &ReadWriteCloserTransport{serverConn}
		_ = server.Serve(context.Background(), transport)
	}()

	// Create client
	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	initReq := InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		Capabilities:    ClientCapabilities{},
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

// TestContextTimeout tests context cancellation and timeouts
func TestContextTimeout(t *testing.T) {
	server := NewServer("timeout-test-server", "1.0.0")

	// Register a slow tool
	tool := Tool{
		Name:        "slow-tool",
		Description: "A tool that takes time to complete",
	}

	toolHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		// Simulate slow operation
		select {
		case <-time.After(2 * time.Second):
			return &CallToolResult{
				Content: []any{
					map[string]interface{}{
						"type": "text",
						"text": "completed",
					},
				},
			}, nil
		case <-ctx.Done():
			return &CallToolResult{}, ctx.Err()
		}
	}

	err := server.RegisterTool(tool, toolHandler)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Start server
	go func() {
		transport := &ReadWriteCloserTransport{serverConn}
		_ = server.Serve(context.Background(), transport)
	}()

	// Create and initialize client
	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	initReq := InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
		Capabilities:    ClientCapabilities{},
	}

	_, err = client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err = client.CallTool(ctx, CallToolRequest{
		Name:      "slow-tool",
		Arguments: json.RawMessage("{}"),
	})

	if err == nil {
		t.Error("Expected timeout error")
	}

	if err != context.DeadlineExceeded {
		t.Logf("Got error: %v (type: %T)", err, err)
		// Context cancellation might be wrapped
	}
}
