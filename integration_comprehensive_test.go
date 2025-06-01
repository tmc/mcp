package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"
)

// ptr returns a pointer to the given value
func ptr(s string) *string {
	return &s
}

// Comprehensive integration tests covering full MCP workflows

func TestFullClientServerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a simple server
	server := NewServer("integration-test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Register a test tool
	echoTool := Tool{
		Name:        "echo",
		Description: "Echo back the input",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {"message": {"type": "string"}}}`),
	}

	echoHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		var input map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &input); err != nil {
			return &CallToolResult{
				IsError: true,
				Content: []any{map[string]string{"type": "text", "text": "Invalid input"}},
			}, nil
		}

		message, ok := input["message"].(string)
		if !ok {
			message = "No message provided"
		}

		return &CallToolResult{
			Content: []any{map[string]string{
				"type": "text",
				"text": fmt.Sprintf("Echo: %s", message),
			}},
		}, nil
	}

	err := server.RegisterTool(echoTool, echoHandler)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Register a test prompt
	testPrompt := Prompt{
		Name:        "greeting",
		Description: "A greeting prompt",
		Arguments: []PromptArgument{
			{Name: "name", Description: "Name to greet", Required: true},
		},
	}

	promptHandler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		name, ok := req.Arguments["name"].(string)
		if !ok {
			name = "World"
		}

		return &GetPromptResult{
			Messages: []PromptMessage{
				{
					Role: RoleUser,
					Content: []any{map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Hello, %s!", name),
					}},
				},
			},
		}, nil
	}

	err = server.RegisterPrompt(testPrompt, promptHandler)
	if err != nil {
		t.Fatalf("Failed to register prompt: %v", err)
	}

	// Register a test resource
	testResource := Resource{
		URI:         "test://example",
		Description: "Test resource",
		MimeType:    "text/plain",
	}

	resourceHandler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{
			TextResourceContents{
				URI:      req.URI,
				MimeType: "text/plain",
				Text:     "This is test resource content",
			},
		}, nil
	}

	err = server.RegisterResource(testResource, resourceHandler)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}

	// Create transport using pipes
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	serverTransport := &ReadWriteCloserTransport{serverConn}

	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve(ctx, serverTransport)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create and initialize client
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize the connection
	initResult, err := client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo: Implementation{
			Name:    "integration-test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{},
	})
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	if initResult.ServerInfo.Name != "integration-test-server" {
		t.Errorf("Expected server name 'integration-test-server', got %s", initResult.ServerInfo.Name)
	}

	// Test tool listing
	toolsResult, err := client.ListTools(ctx, ListToolsRequest{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	if len(toolsResult.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolsResult.Tools))
	}

	if toolsResult.Tools[0].Name != "echo" {
		t.Errorf("Expected tool name 'echo', got %s", toolsResult.Tools[0].Name)
	}

	// Test tool calling
	callArgs, _ := json.Marshal(map[string]string{"message": "Hello, World!"})
	callResult, err := client.CallTool(ctx, CallToolRequest{
		Name:      "echo",
		Arguments: callArgs,
	})
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	if len(callResult.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(callResult.Content))
	}

	// Test prompt listing
	promptsResult, err := client.ListPrompts(ctx, ListPromptsRequest{})
	if err != nil {
		t.Fatalf("Failed to list prompts: %v", err)
	}

	if len(promptsResult.Prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(promptsResult.Prompts))
	}

	if promptsResult.Prompts[0].Name != "greeting" {
		t.Errorf("Expected prompt name 'greeting', got %s", promptsResult.Prompts[0].Name)
	}

	// Test prompt getting
	promptArgs := map[string]interface{}{"name": "Integration Test"}
	promptResult, err := client.GetPrompt(ctx, GetPromptRequest{
		Name:      "greeting",
		Arguments: promptArgs,
	})
	if err != nil {
		t.Fatalf("Failed to get prompt: %v", err)
	}

	if len(promptResult.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(promptResult.Messages))
	}

	// Test resource listing
	resourcesResult, err := client.ListResources(ctx, ListResourcesRequest{})
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	if len(resourcesResult.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resourcesResult.Resources))
	}

	if resourcesResult.Resources[0].URI != "test://example" {
		t.Errorf("Expected resource URI 'test://example', got %s", resourcesResult.Resources[0].URI)
	}

	// Test resource reading
	readResult, err := client.ReadResource(ctx, ReadResourceRequest{
		URI: "test://example",
	})
	if err != nil {
		t.Fatalf("Failed to read resource: %v", err)
	}

	if len(readResult.Contents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(readResult.Contents))
	}

	// Test ping
	err = client.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Cancel context to stop server
	cancel()

	// Wait for server to finish
	select {
	case serverErr := <-serverDone:
		if serverErr != nil && serverErr != context.Canceled {
			t.Errorf("Server error: %v", serverErr)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestNotificationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create server with notification support
	server := NewServer("notification-test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Create transport using pipes
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	serverTransport := &ReadWriteCloserTransport{serverConn}

	// Track received notifications
	var receivedNotifications []string
	var notificationMu sync.Mutex

	// Create client with notification handler
	client, err := NewClient(clientTransport, WithNotificationHandler(func(notif JSONRPCNotification) {
		notificationMu.Lock()
		defer notificationMu.Unlock()
		receivedNotifications = append(receivedNotifications, notif.Method)
	}))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Start server
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve(ctx, serverTransport)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Initialize client
	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo: Implementation{
			Name:    "notification-test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{},
	})
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Simulate server sending notifications (this would need more complex setup in real scenario)
	// For now, just verify the notification infrastructure is in place

	cancel()
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestErrorHandlingIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create server with error-prone handlers
	server := NewServer("error-test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Register tool that always fails
	errorTool := Tool{
		Name:        "error-tool",
		Description: "Tool that always returns error",
	}

	errorHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return nil, fmt.Errorf("tool execution failed")
	}

	err := server.RegisterTool(errorTool, errorHandler)
	if err != nil {
		t.Fatalf("Failed to register error tool: %v", err)
	}

	// Register prompt that returns error
	errorPrompt := Prompt{
		Name:        "error-prompt",
		Description: "Prompt that always returns error",
	}

	errorPromptHandler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return nil, fmt.Errorf("prompt execution failed")
	}

	err = server.RegisterPrompt(errorPrompt, errorPromptHandler)
	if err != nil {
		t.Fatalf("Failed to register error prompt: %v", err)
	}

	// Create transport
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	serverTransport := &ReadWriteCloserTransport{serverConn}

	// Start server
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve(ctx, serverTransport)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create and initialize client
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo: Implementation{
			Name:    "error-test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{},
	})
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Test tool error handling
	_, err = client.CallTool(ctx, CallToolRequest{
		Name:      "error-tool",
		Arguments: json.RawMessage(`{}`),
	})
	if err == nil {
		t.Error("Expected error when calling error tool")
	}

	// Test prompt error handling
	_, err = client.GetPrompt(ctx, GetPromptRequest{
		Name:      "error-prompt",
		Arguments: map[string]interface{}{},
	})
	if err == nil {
		t.Error("Expected error when getting error prompt")
	}

	// Test calling non-existent tool
	_, err = client.CallTool(ctx, CallToolRequest{
		Name:      "non-existent-tool",
		Arguments: json.RawMessage(`{}`),
	})
	if err == nil {
		t.Error("Expected error when calling non-existent tool")
	}

	cancel()
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestConcurrentClientServerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create server with counter tool
	server := NewServer("concurrent-test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	var counter int64
	var counterMu sync.Mutex

	counterTool := Tool{
		Name:        "increment",
		Description: "Increment a counter",
	}

	counterHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		counterMu.Lock()
		counter++
		currentValue := counter
		counterMu.Unlock()

		return &CallToolResult{
			Content: []any{map[string]interface{}{
				"type":  "text",
				"text":  fmt.Sprintf("Counter: %d", currentValue),
				"value": currentValue,
			}},
		}, nil
	}

	err := server.RegisterTool(counterTool, counterHandler)
	if err != nil {
		t.Fatalf("Failed to register counter tool: %v", err)
	}

	// Create multiple client-server pairs
	const numClients = 5
	const callsPerClient = 10

	var wg sync.WaitGroup
	wg.Add(numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			defer wg.Done()

			// Create dedicated connection for this client
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()

			clientTransport := &ReadWriteCloserTransport{clientConn}
			serverTransport := &ReadWriteCloserTransport{serverConn}

			// Start server for this client
			go func() {
				server.Serve(ctx, serverTransport)
			}()

			time.Sleep(50 * time.Millisecond)

			// Create and initialize client
			client, err := NewClient(clientTransport)
			if err != nil {
				t.Errorf("Client %d: Failed to create client: %v", clientID, err)
				return
			}
			defer client.Close()

			_, err = client.Initialize(ctx, InitializeRequest{
				ProtocolVersion: LATEST_PROTOCOL_VERSION,
				ClientInfo: Implementation{
					Name:    fmt.Sprintf("concurrent-test-client-%d", clientID),
					Version: "1.0.0",
				},
				Capabilities: ClientCapabilities{},
			})
			if err != nil {
				t.Errorf("Client %d: Failed to initialize: %v", clientID, err)
				return
			}

			// Make multiple calls
			for j := 0; j < callsPerClient; j++ {
				_, err := client.CallTool(ctx, CallToolRequest{
					Name:      "increment",
					Arguments: json.RawMessage(`{}`),
				})
				if err != nil {
					t.Errorf("Client %d, call %d: Failed to call tool: %v", clientID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final counter value
	counterMu.Lock()
	finalCounter := counter
	counterMu.Unlock()

	expectedCount := int64(numClients * callsPerClient)
	if finalCounter != expectedCount {
		t.Errorf("Expected counter value %d, got %d", expectedCount, finalCounter)
	}
}

func TestLongRunningIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create server with slow tool
	server := NewServer("long-running-test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	slowTool := Tool{
		Name:        "slow-operation",
		Description: "A tool that takes time to complete",
	}

	slowHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		// Simulate long-running operation
		select {
		case <-time.After(2 * time.Second):
			return &CallToolResult{
				Content: []any{map[string]string{
					"type": "text",
					"text": "Operation completed successfully",
				}},
			}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	err := server.RegisterTool(slowTool, slowHandler)
	if err != nil {
		t.Fatalf("Failed to register slow tool: %v", err)
	}

	// Create transport
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	serverTransport := &ReadWriteCloserTransport{serverConn}

	// Start server
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve(ctx, serverTransport)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create and initialize client
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo: Implementation{
			Name:    "long-running-test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{},
	})
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Test long-running operation
	start := time.Now()
	result, err := client.CallTool(ctx, CallToolRequest{
		Name:      "slow-operation",
		Arguments: json.RawMessage(`{}`),
	})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Long-running operation failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result from long-running operation")
	}

	if duration < 2*time.Second {
		t.Errorf("Operation completed too quickly: %v", duration)
	}

	cancel()
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestMemoryIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create server with memory-intensive tool
	server := NewServer("memory-test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	memoryTool := Tool{
		Name:        "memory-test",
		Description: "Tool that processes large data",
	}

	memoryHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		// Process large input data
		var input map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &input); err != nil {
			return nil, err
		}

		// Create large response
		largeData := make([]string, 1000)
		for i := range largeData {
			largeData[i] = fmt.Sprintf("Item %d with some data content", i)
		}

		return &CallToolResult{
			Content: []any{map[string]interface{}{
				"type": "text",
				"text": "Large data processed",
				"data": largeData,
			}},
		}, nil
	}

	err := server.RegisterTool(memoryTool, memoryHandler)
	if err != nil {
		t.Fatalf("Failed to register memory tool: %v", err)
	}

	// Create transport
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientTransport := &ReadWriteCloserTransport{clientConn}
	serverTransport := &ReadWriteCloserTransport{serverConn}

	// Start server
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve(ctx, serverTransport)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create and initialize client
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo: Implementation{
			Name:    "memory-test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{},
	})
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Test with large input data
	largeInput := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeInput[fmt.Sprintf("key%d", i)] = fmt.Sprintf("Large value %d with lots of content", i)
	}

	inputArgs, _ := json.Marshal(largeInput)
	result, err := client.CallTool(ctx, CallToolRequest{
		Name:      "memory-test",
		Arguments: inputArgs,
	})

	if err != nil {
		t.Errorf("Memory test failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result from memory test")
	}

	cancel()
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

// Test helper for creating paired connections
func createTestTransports() (client, server Transport) {
	clientConn, serverConn := net.Pipe()
	return &ReadWriteCloserTransport{clientConn}, &ReadWriteCloserTransport{serverConn}
}

// Test helper for basic server setup
func createTestServer(t interface{}) *Server {
	var server *Server
	switch v := t.(type) {
	case *testing.T:
		server = NewServer("test-server", "1.0.0", WithTestLogger(v, slog.LevelDebug))
	case *testing.B:
		server = NewServer("test-server", "1.0.0", WithTestLogger(v, slog.LevelDebug))
	default:
		server = NewServer("test-server", "1.0.0")
	}

	// Add basic echo tool
	tool := Tool{Name: "echo", Description: "Echo tool"}
	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []any{map[string]string{"type": "text", "text": "echo"}},
		}, nil
	}
	server.RegisterTool(tool, handler)

	return server
}

// Benchmark integration test
func BenchmarkClientServerRoundtrip(b *testing.B) {
	ctx := context.Background()
	server := createTestServer(b)

	clientTransport, serverTransport := createTestTransports()
	defer clientTransport.(*ReadWriteCloserTransport).Close()
	defer serverTransport.(*ReadWriteCloserTransport).Close()

	// Start server
	go server.Serve(ctx, serverTransport)
	time.Sleep(50 * time.Millisecond)

	// Create client
	client, err := NewClient(clientTransport)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	// Initialize
	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "bench-client", Version: "1.0.0"},
		Capabilities:    ClientCapabilities{},
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.CallTool(ctx, CallToolRequest{
			Name:      "echo",
			Arguments: json.RawMessage(`{}`),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
