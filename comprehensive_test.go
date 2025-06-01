package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// TestComprehensiveMCPFlow tests a complete MCP workflow
func TestComprehensiveMCPFlow(t *testing.T) {
	server := NewServer("comprehensive-test-server", "1.0.0")

	// Register multiple tools
	calculator := Tool{
		Name:        "calculator",
		Description: "Basic arithmetic calculator",
		InputSchema: mustMarshalJSON(map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type": "string",
					"enum": []string{"add", "subtract", "multiply", "divide"},
				},
				"a": map[string]interface{}{
					"type": "number",
				},
				"b": map[string]interface{}{
					"type": "number",
				},
			},
			"required": []string{"operation", "a", "b"},
		}),
	}

	calculatorHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		var args struct {
			Operation string  `json:"operation"`
			A         float64 `json:"a"`
			B         float64 `json:"b"`
		}

		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return &CallToolResult{IsError: true}, fmt.Errorf("invalid arguments: %v", err)
		}

		var result float64
		switch args.Operation {
		case "add":
			result = args.A + args.B
		case "subtract":
			result = args.A - args.B
		case "multiply":
			result = args.A * args.B
		case "divide":
			if args.B == 0 {
				return &CallToolResult{IsError: true}, fmt.Errorf("division by zero")
			}
			result = args.A / args.B
		default:
			return &CallToolResult{IsError: true}, fmt.Errorf("unknown operation: %s", args.Operation)
		}

		return &CallToolResult{
			Content: []any{
				map[string]interface{}{
					"type":   "text",
					"text":   fmt.Sprintf("%.2f %s %.2f = %.2f", args.A, args.Operation, args.B, result),
					"result": result,
				},
			},
		}, nil
	}

	err := server.RegisterTool(calculator, calculatorHandler)
	if err != nil {
		t.Fatalf("Failed to register calculator tool: %v", err)
	}

	// Register a counter tool for concurrency testing
	var counter int
	var mu sync.Mutex

	counterTool := Tool{
		Name:        "counter",
		Description: "Thread-safe counter",
		InputSchema: mustMarshalJSON(map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"increment": map[string]interface{}{
					"type":    "number",
					"default": 1,
				},
			},
		}),
	}

	counterHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		var args struct {
			Increment int `json:"increment"`
		}
		args.Increment = 1 // default

		if len(req.Arguments) > 0 {
			if err := json.Unmarshal(req.Arguments, &args); err != nil {
				return &CallToolResult{IsError: true}, fmt.Errorf("invalid arguments: %v", err)
			}
		}

		mu.Lock()
		counter += args.Increment
		currentCount := counter
		mu.Unlock()

		return &CallToolResult{
			Content: []any{
				map[string]interface{}{
					"type":  "text",
					"text":  fmt.Sprintf("Counter: %d", currentCount),
					"count": currentCount,
				},
			},
		}, nil
	}

	err = server.RegisterTool(counterTool, counterHandler)
	if err != nil {
		t.Fatalf("Failed to register counter tool: %v", err)
	}

	// Register resources
	configResource := Resource{
		URI:         "config://settings.json",
		Description: "Application configuration",
		MimeType:    "application/json",
	}

	configHandler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		config := map[string]interface{}{
			"app_name":    "test-app",
			"version":     "1.0.0",
			"debug_mode":  true,
			"max_retries": 3,
		}

		configJSON, _ := json.Marshal(config)

		return []ResourceContents{
			TextResourceContents{
				URI:      req.URI,
				MimeType: "application/json",
				Text:     string(configJSON),
			},
		}, nil
	}

	err = server.RegisterResource(configResource, configHandler)
	if err != nil {
		t.Fatalf("Failed to register config resource: %v", err)
	}

	logResource := Resource{
		URI:         "logs://app.log",
		Description: "Application logs",
		MimeType:    "text/plain",
	}

	logHandler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		logs := []string{
			"2024-01-01 10:00:00 INFO  Application started",
			"2024-01-01 10:00:01 DEBUG Connecting to database",
			"2024-01-01 10:00:02 INFO  Database connection established",
			"2024-01-01 10:00:03 WARN  High memory usage detected",
		}

		var logText string
		for _, log := range logs {
			logText += log + "\n"
		}

		return []ResourceContents{
			TextResourceContents{
				URI:      req.URI,
				MimeType: "text/plain",
				Text:     logText,
			},
		}, nil
	}

	err = server.RegisterResource(logResource, logHandler)
	if err != nil {
		t.Fatalf("Failed to register log resource: %v", err)
	}

	// Test the complete flow
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

	// Create and initialize client
	clientTransport := &ReadWriteCloserTransport{clientConn}
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	initReq := InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "comprehensive-test-client", Version: "1.0.0"},
		Capabilities:    ClientCapabilities{},
	}

	initResult, err := client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Verify initialization
	if initResult.ServerInfo.Name != "comprehensive-test-server" {
		t.Errorf("Expected server name 'comprehensive-test-server', got '%s'", initResult.ServerInfo.Name)
	}

	// Test tool operations
	t.Run("Calculator Operations", func(t *testing.T) {
		tests := []struct {
			name      string
			operation string
			a, b      float64
			expected  float64
			wantError bool
		}{
			{"addition", "add", 10, 5, 15, false},
			{"subtraction", "subtract", 10, 3, 7, false},
			{"multiplication", "multiply", 4, 7, 28, false},
			{"division", "divide", 15, 3, 5, false},
			{"division by zero", "divide", 10, 0, 0, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				args, _ := json.Marshal(map[string]interface{}{
					"operation": tt.operation,
					"a":         tt.a,
					"b":         tt.b,
				})

				result, err := client.CallTool(context.Background(), CallToolRequest{
					Name:      "calculator",
					Arguments: args,
				})

				if tt.wantError {
					if err == nil {
						t.Error("Expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				if len(result.Content) == 0 {
					t.Fatal("Expected content in result")
				}

				content := result.Content[0].(map[string]interface{})
				resultValue := content["result"].(float64)

				if resultValue != tt.expected {
					t.Errorf("Expected result %.2f, got %.2f", tt.expected, resultValue)
				}
			})
		}
	})

	// Test counter concurrency
	t.Run("Concurrent Counter Operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		resultsChannel := make(chan int, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				args, _ := json.Marshal(map[string]interface{}{
					"increment": 1,
				})

				result, err := client.CallTool(context.Background(), CallToolRequest{
					Name:      "counter",
					Arguments: args,
				})

				if err != nil {
					t.Errorf("Unexpected error in goroutine: %v", err)
					return
				}

				if len(result.Content) > 0 {
					content := result.Content[0].(map[string]interface{})
					if count, ok := content["count"].(float64); ok {
						resultsChannel <- int(count)
					}
				}
			}()
		}

		wg.Wait()
		close(resultsChannel)

		// Collect all results
		var counts []int
		for count := range resultsChannel {
			counts = append(counts, count)
		}

		if len(counts) != numGoroutines {
			t.Errorf("Expected %d results, got %d", numGoroutines, len(counts))
		}

		// All counts should be unique (demonstrating thread safety)
		countMap := make(map[int]bool)
		for _, count := range counts {
			if countMap[count] {
				t.Errorf("Duplicate count found: %d (indicates race condition)", count)
			}
			countMap[count] = true
		}
	})

	// Test resource operations
	t.Run("Resource Operations", func(t *testing.T) {
		// List resources
		resourcesResult, err := client.ListResources(context.Background(), ListResourcesRequest{})
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(resourcesResult.Resources) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(resourcesResult.Resources))
		}

		// Test config resource
		configResult, err := client.ReadResource(context.Background(), ReadResourceRequest{
			URI: "config://settings.json",
		})
		if err != nil {
			t.Fatalf("Failed to read config resource: %v", err)
		}

		if len(configResult.Contents) == 0 {
			t.Fatal("Expected content in config resource")
		}

		configContent := configResult.Contents[0].(TextResourceContents)
		var config map[string]interface{}
		err = json.Unmarshal([]byte(configContent.Text), &config)
		if err != nil {
			t.Fatalf("Failed to parse config JSON: %v", err)
		}

		if config["app_name"] != "test-app" {
			t.Errorf("Expected app_name 'test-app', got '%v'", config["app_name"])
		}

		// Test log resource
		logResult, err := client.ReadResource(context.Background(), ReadResourceRequest{
			URI: "logs://app.log",
		})
		if err != nil {
			t.Fatalf("Failed to read log resource: %v", err)
		}

		if len(logResult.Contents) == 0 {
			t.Fatal("Expected content in log resource")
		}

		logContent := logResult.Contents[0].(TextResourceContents)
		if len(logContent.Text) == 0 {
			t.Error("Expected log content")
		}

		// Check that log contains expected entries
		if !containsString(logContent.Text, "Application started") {
			t.Error("Expected 'Application started' in logs")
		}

		if !containsString(logContent.Text, "High memory usage detected") {
			t.Error("Expected 'High memory usage detected' in logs")
		}
	})

	// Test error scenarios
	t.Run("Error Scenarios", func(t *testing.T) {
		// Test calling non-existent tool
		_, err := client.CallTool(context.Background(), CallToolRequest{
			Name:      "non-existent-tool",
			Arguments: json.RawMessage(`{}`),
		})
		if err == nil {
			t.Error("Expected error when calling non-existent tool")
		}

		// Test invalid arguments
		_, err = client.CallTool(context.Background(), CallToolRequest{
			Name:      "calculator",
			Arguments: json.RawMessage(`{"invalid": "args"}`),
		})
		if err == nil {
			t.Error("Expected error with invalid calculator arguments")
		}

		// Test reading non-existent resource
		_, err = client.ReadResource(context.Background(), ReadResourceRequest{
			URI: "non://existent",
		})
		if err == nil {
			t.Error("Expected error when reading non-existent resource")
		}
	})

	// Test context cancellation
	t.Run("Context Cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := client.ListTools(ctx, ListToolsRequest{})
		if err == nil {
			t.Error("Expected error due to cancelled context")
		}
	})
}

// Helper function to marshal JSON and panic on error (for tests)
func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findString(s, substr) >= 0
}

func findString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestMCPProtocolCompliance tests protocol compliance
func TestMCPProtocolCompliance(t *testing.T) {
	server := NewServer("protocol-test-server", "1.0.0")

	// Test that server properly supports all required protocol methods
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	go func() {
		transport := &ReadWriteCloserTransport{serverConn}
		_ = server.Serve(context.Background(), transport)
	}()

	client, err := NewClient(&ReadWriteCloserTransport{clientConn})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test initialization
	initReq := InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "protocol-test-client", Version: "1.0.0"},
		Capabilities:    ClientCapabilities{},
	}

	initResult, err := client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if initResult.ProtocolVersion != LATEST_PROTOCOL_VERSION {
		t.Errorf("Expected protocol version %s, got %s", LATEST_PROTOCOL_VERSION, initResult.ProtocolVersion)
	}

	// Test that all required methods are available
	t.Run("Required Methods", func(t *testing.T) {
		// ping method should work
		err := client.Ping(context.Background())
		if err != nil {
			t.Errorf("Ping failed: %v", err)
		}

		// tools/list should work (even if empty)
		_, err = client.ListTools(context.Background(), ListToolsRequest{})
		if err != nil {
			t.Errorf("ListTools failed: %v", err)
		}

		// resources/list should work (even if empty)
		_, err = client.ListResources(context.Background(), ListResourcesRequest{})
		if err != nil {
			t.Errorf("ListResources failed: %v", err)
		}

		// prompts/list should work (even if empty)
		_, err = client.ListPrompts(context.Background(), ListPromptsRequest{})
		if err != nil {
			t.Errorf("ListPrompts failed: %v", err)
		}
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	server := NewServer("edge-case-server", "1.0.0")

	// Register a tool that tests various edge cases
	edgeCaseTool := Tool{
		Name:        "edge-case-tool",
		Description: "Tool for testing edge cases",
		InputSchema: mustMarshalJSON(map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}),
	}

	edgeCaseHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		// Test large response
		largeText := ""
		for i := 0; i < 1000; i++ {
			largeText += fmt.Sprintf("Line %d with some content to make it longer\n", i)
		}

		return &CallToolResult{
			Content: []any{
				map[string]interface{}{
					"type": "text",
					"text": largeText,
					"size": len(largeText),
				},
			},
		}, nil
	}

	err := server.RegisterTool(edgeCaseTool, edgeCaseHandler)
	if err != nil {
		t.Fatalf("Failed to register edge case tool: %v", err)
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	go func() {
		transport := &ReadWriteCloserTransport{serverConn}
		_ = server.Serve(context.Background(), transport)
	}()

	client, err := NewClient(&ReadWriteCloserTransport{clientConn})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	initReq := InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "edge-case-client", Version: "1.0.0"},
		Capabilities:    ClientCapabilities{},
	}

	_, err = client.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Run("Large Response", func(t *testing.T) {
		result, err := client.CallTool(context.Background(), CallToolRequest{
			Name:      "edge-case-tool",
			Arguments: json.RawMessage(`{}`),
		})

		if err != nil {
			t.Fatalf("Failed to call edge case tool: %v", err)
		}

		if len(result.Content) == 0 {
			t.Fatal("Expected content in result")
		}

		content := result.Content[0].(map[string]interface{})
		size := int(content["size"].(float64))

		if size <= 0 {
			t.Error("Expected large response size")
		}

		text := content["text"].(string)
		if len(text) != size {
			t.Errorf("Expected text length %d, got %d", size, len(text))
		}
	})

	t.Run("Empty Arguments", func(t *testing.T) {
		// Test with empty arguments
		result, err := client.CallTool(context.Background(), CallToolRequest{
			Name:      "edge-case-tool",
			Arguments: json.RawMessage(`{}`),
		})

		if err != nil {
			t.Fatalf("Failed to call tool with empty arguments: %v", err)
		}

		if len(result.Content) == 0 {
			t.Error("Expected content even with empty arguments")
		}
	})

	t.Run("Timeout Handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// This should timeout quickly
		_, err := client.CallTool(ctx, CallToolRequest{
			Name:      "edge-case-tool",
			Arguments: json.RawMessage(`{}`),
		})

		// We expect this to either timeout or complete very quickly
		// Don't fail the test if it completes quickly, as that's also valid
		if err != nil && err != context.DeadlineExceeded {
			t.Logf("Got error (expected timeout or success): %v", err)
		}
	})
}
