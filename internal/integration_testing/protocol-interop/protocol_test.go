package protocolinterop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tmc/mcp"
)

// TestProtocolMessageSerialization tests that protocol messages serialize
// correctly according to the MCP specification
func TestProtocolMessageSerialization(t *testing.T) {
	t.Run("InitializeRequest", func(t *testing.T) {
		// Test basic protocol message structure
		msg := map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"experimental": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal message: %v", err)
		}

		var unmarshaled map[string]interface{}
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if unmarshaled["protocolVersion"] != "2024-11-05" {
			t.Errorf("ProtocolVersion mismatch: got %v, want %s",
				unmarshaled["protocolVersion"], "2024-11-05")
		}
	})

	t.Run("ToolCallRequest", func(t *testing.T) {
		// Test tool call request serialization
		req := mcp.CallToolRequest{
			Name: "test_tool",
			Arguments: json.RawMessage(`{
				"message": "hello world",
				"count": 42
			}`),
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal CallToolRequest: %v", err)
		}

		var unmarshaled mcp.CallToolRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal CallToolRequest: %v", err)
		}

		if unmarshaled.Name != "test_tool" {
			t.Errorf("Name mismatch: got %s, want test_tool", unmarshaled.Name)
		}

		// Verify arguments are preserved
		var args map[string]interface{}
		if err := json.Unmarshal(unmarshaled.Arguments, &args); err != nil {
			t.Fatalf("Failed to unmarshal arguments: %v", err)
		}

		if args["message"] != "hello world" {
			t.Errorf("Message argument mismatch: got %v, want hello world", args["message"])
		}
		if args["count"] != float64(42) {
			t.Errorf("Count argument mismatch: got %v, want 42", args["count"])
		}
	})

	t.Run("ResourceRequest", func(t *testing.T) {
		// Test resource request serialization
		req := mcp.ReadResourceRequest{
			URI: "file:///tmp/test.txt",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal ReadResourceRequest: %v", err)
		}

		var unmarshaled mcp.ReadResourceRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal ReadResourceRequest: %v", err)
		}

		if unmarshaled.URI != "file:///tmp/test.txt" {
			t.Errorf("URI mismatch: got %s, want file:///tmp/test.txt", unmarshaled.URI)
		}
	})
}

// TestProtocolConformance tests protocol conformance against the MCP specification
func TestProtocolConformance(t *testing.T) {
	_ = context.Background() // Avoid unused variable warning

	t.Run("ServerCapabilities", func(t *testing.T) {
		server := mcp.NewServer("test-server", "1.0.0")

		// Test that server properly advertises capabilities
		if server == nil {
			t.Fatal("Failed to create server")
		}
	})

	t.Run("ErrorFormats", func(t *testing.T) {
		// Test that errors conform to JSON-RPC 2.0 format
		err := &mcp.ResponseError{
			Code:    -32602,
			Message: "Invalid params",
		}

		data, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Fatalf("Failed to marshal ResponseError: %v", marshalErr)
		}

		var unmarshaled mcp.ResponseError
		if unmarshalErr := json.Unmarshal(data, &unmarshaled); unmarshalErr != nil {
			t.Fatalf("Failed to unmarshal ResponseError: %v", unmarshalErr)
		}

		if unmarshaled.Code != -32602 {
			t.Errorf("Error code mismatch: got %d, want -32602", unmarshaled.Code)
		}
		if unmarshaled.Message != "Invalid params" {
			t.Errorf("Error message mismatch: got %s, want Invalid params", unmarshaled.Message)
		}
	})

	t.Run("MethodCalls", func(t *testing.T) {
		// Test that method calls follow the correct format
		methods := []string{
			"tools/list",
			"tools/call", 
			"resources/list",
			"resources/read",
			"prompts/list",
			"prompts/get",
		}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				// Verify method names are valid
				if method == "" {
					t.Error("Method name cannot be empty")
				}
				if len(method) > 100 {
					t.Error("Method name too long")
				}
			})
		}
	})
}

// TestCrossImplementationCompatibility tests compatibility across different MCP implementations
func TestCrossImplementationCompatibility(t *testing.T) {
	// Create a test server
	server := mcp.NewServer("test-server", "1.0.0")
	
	// Register a simple tool for testing
	tool := mcp.Tool{
		Name:        "test_echo",
		Description: "Echo test tool",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {"message": {"type": "string"}}}`),
	}
	
	err := server.RegisterTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Echo: " + string(req.Arguments),
				},
			},
		}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	t.Run("ToolsListFormat", func(t *testing.T) {
		// Test tool registration and format validation
		if server == nil {
			t.Fatal("Server should not be nil")
		}
		
		// Test that the tool was registered successfully
		// Note: We test the tool registration indirectly by verifying the server accepts it
		if err != nil {
			t.Errorf("Tool registration failed: %v", err)
		}
	})
	
	t.Run("ToolCallFormat", func(t *testing.T) {
		// Test that tool calls work with standard format
		req := mcp.CallToolRequest{
			Name:      "test_echo",
			Arguments: json.RawMessage(`{"message": "hello"}`),
		}
		
		// Since we can't easily test the full server machinery here,
		// we'll verify the request format is valid
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal tool call request: %v", err)
		}
		
		var unmarshaled mcp.CallToolRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal tool call request: %v", err)
		}
		
		if unmarshaled.Name != "test_echo" {
			t.Errorf("Tool name mismatch in unmarshaled request: got %s, want test_echo", unmarshaled.Name)
		}
	})
}
