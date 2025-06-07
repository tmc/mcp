package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/jsonrpc2"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		sName   string
		version string
		opts    []ServerOption
		want    func(*Server) bool
	}{
		{
			name:    "basic server creation",
			sName:   "test-server",
			version: "1.0.0",
			want: func(s *Server) bool {
				return s.name == "test-server" && s.version == "1.0.0"
			},
		},
		{
			name:    "server with custom name option",
			sName:   "original",
			version: "1.0.0",
			opts:    []ServerOption{WithServerName("custom-name")},
			want: func(s *Server) bool {
				return s.name == "custom-name" && s.version == "1.0.0"
			},
		},
		{
			name:    "server with custom version option",
			sName:   "test-server",
			version: "1.0.0",
			opts:    []ServerOption{WithServerVersion("2.0.0")},
			want: func(s *Server) bool {
				return s.name == "test-server" && s.version == "2.0.0"
			},
		},
		{
			name:    "server with instructions",
			sName:   "test-server",
			version: "1.0.0",
			opts:    []ServerOption{WithServerInstructions("Test instructions")},
			want: func(s *Server) bool {
				return s.instructions == "Test instructions"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.sName, tt.version, tt.opts...)

			if server == nil {
				t.Fatal("NewServer returned nil")
			}

			if !tt.want(server) {
				t.Errorf("Server doesn't match expectations")
			}

			// Verify basic initialization
			if server.tools == nil {
				t.Error("Server tools map is nil")
			}
			if server.resources == nil {
				t.Error("Server resources map is nil")
			}
			if server.resourceTmpls == nil {
				t.Error("Server resource templates map is nil")
			}
			if server.prompts == nil {
				t.Error("Server prompts map is nil")
			}
			if server.handlers == nil {
				t.Error("Server handlers map is nil")
			}
			if server.dispatch == nil {
				t.Error("Server dispatcher is nil")
			}
		})
	}
}

func TestServerRegisterTool(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	inputSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "Test input",
			},
		},
	})
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: inputSchema,
	}

	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []any{
				map[string]interface{}{
					"type": "text",
					"text": fmt.Sprintf("Tool called with: %v", req.Arguments),
				},
			},
		}, nil
	}

	// Test successful registration
	err := server.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	// Verify tool is registered
	server.mu.RLock()
	toolDef, exists := server.tools[tool.Name]
	server.mu.RUnlock()

	if !exists {
		t.Error("Tool was not registered")
	}

	if toolDef.tool.Name != tool.Name {
		t.Errorf("Expected tool name %s, got %s", tool.Name, toolDef.tool.Name)
	}

	// Test duplicate registration error
	err = server.RegisterTool(tool, handler)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}

	// Test capabilities update
	if server.capabilities.Tools == nil || !server.capabilities.Tools.ListChanged {
		t.Error("Expected tools capability to be set")
	}
}

func TestServerRegisterResource(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	resource := Resource{
		URI:         "test://resource",
		Description: "A test resource",
		MimeType:    "text/plain",
	}

	handler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{
			TextResourceContents{
				URI:      req.URI,
				MimeType: "text/plain",
				Text:     "Resource content",
			},
		}, nil
	}

	// Test successful registration
	err := server.RegisterResource(resource, handler)
	if err != nil {
		t.Fatalf("RegisterResource failed: %v", err)
	}

	// Verify resource is registered
	server.mu.RLock()
	resourceDef, exists := server.resources[resource.URI]
	server.mu.RUnlock()

	if !exists {
		t.Error("Resource was not registered")
	}

	if resourceDef.resource.URI != resource.URI {
		t.Errorf("Expected resource URI %s, got %s", resource.URI, resourceDef.resource.URI)
	}

	// Test duplicate registration error
	err = server.RegisterResource(resource, handler)
	if err == nil {
		t.Error("Expected error when registering duplicate resource")
	}

	// Test capabilities update
	if server.capabilities.Resources == nil || !server.capabilities.Resources.ListChanged {
		t.Error("Expected resources capability to be set")
	}
}

func TestServerRegisterPrompt(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	prompt := Prompt{
		Name:        "test-prompt",
		Description: "A test prompt",
		Arguments: []PromptArgument{
			{
				Name:        "input",
				Description: "Test input",
				Required:    true,
			},
		},
	}

	handler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{
			Messages: []PromptMessage{
				{
					Role: RoleUser,
					Content: []any{
						map[string]interface{}{
							"type": "text",
							"text": fmt.Sprintf("Prompt with args: %v", req.Arguments),
						},
					},
				},
			},
		}, nil
	}

	// Test successful registration
	err := server.RegisterPrompt(prompt, handler)
	if err != nil {
		t.Fatalf("RegisterPrompt failed: %v", err)
	}

	// Verify prompt is registered
	server.mu.RLock()
	promptDef, exists := server.prompts[prompt.Name]
	server.mu.RUnlock()

	if !exists {
		t.Error("Prompt was not registered")
	}

	if promptDef.prompt.Name != prompt.Name {
		t.Errorf("Expected prompt name %s, got %s", prompt.Name, promptDef.prompt.Name)
	}

	// Test duplicate registration error
	err = server.RegisterPrompt(prompt, handler)
	if err == nil {
		t.Error("Expected error when registering duplicate prompt")
	}

	// Test capabilities update
	if !server.capabilities.Prompts.ListChanged {
		t.Error("Expected prompts capability to be set")
	}
}

func TestServerHandleInitialize(t *testing.T) {
	tests := []struct {
		name    string
		request JSONRPCRequest
		wantErr bool
	}{
		{
			name: "valid initialize request",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Method:  "initialize",
				Params: json.RawMessage(`{
					"protocolVersion": "2024-11-05",
					"capabilities": {
						"roots": {"listChanged": true},
						"sampling": {}
					},
					"clientInfo": {
						"name": "test-client",
						"version": "1.0.0"
					}
				}`),
			},
			wantErr: false,
		},
		{
			name: "invalid initialize request",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Method:  "initialize",
				Params:  json.RawMessage(`{invalid json}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("test-server", "1.0.0")

			// Create mock connection
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()

			ctx := context.Background()

			// Convert to jsonrpc2.Request
			req := &jsonrpc2.Request{
				Method: tt.request.Method,
				Params: tt.request.Params,
				ID:     tt.request.ID,
			}

			handler, exists := server.handlers[string(MethodInitialize)]
			if !exists {
				t.Fatal("Initialize handler not registered")
			}

			result, err := handler(ctx, req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify result structure
			initResult, ok := result.(InitializeResult)
			if !ok {
				t.Errorf("Expected InitializeResult, got %T", result)
				return
			}

			if initResult.ProtocolVersion != PROTOCOL_VERSION {
				t.Errorf("Expected protocol version %s, got %s", PROTOCOL_VERSION, initResult.ProtocolVersion)
			}

			if initResult.ServerInfo.Name != server.name {
				t.Errorf("Expected server name %s, got %s", server.name, initResult.ServerInfo.Name)
			}
		})
	}
}

func TestServerToolsListHandler(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register some test tools
	tool1 := Tool{Name: "tool1", Description: "First tool"}
	tool2 := Tool{Name: "tool2", Description: "Second tool"}

	handler := func(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
		return CallToolResult{}, nil
	}

	_ = server.RegisterTool(tool1, handler)
	_ = server.RegisterTool(tool2, handler)

	ctx := context.Background()
	req := &jsonrpc2.Request{
		Method: string(MethodToolsList),
		Params: json.RawMessage(`{}`),
	}

	listHandler, exists := server.handlers[string(MethodToolsList)]
	if !exists {
		t.Fatal("Tools list handler not registered")
	}

	result, err := listHandler(ctx, req)
	if err != nil {
		t.Fatalf("Tools list handler failed: %v", err)
	}

	listResult, ok := result.(ListToolsResult)
	if !ok {
		t.Fatalf("Expected ListToolsResult, got %T", result)
	}

	if len(listResult.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(listResult.Tools))
	}

	// Verify tools are included
	toolNames := make(map[string]bool)
	for _, tool := range listResult.Tools {
		toolNames[tool.Name] = true
	}

	if !toolNames["tool1"] || !toolNames["tool2"] {
		t.Error("Expected tools not found in result")
	}
}

func TestServerToolsCallHandler(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a test tool
	tool := Tool{
		Name:        "echo-tool",
		Description: "Echoes input",
	}

	handler := func(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
		if req.Name != "echo-tool" {
			return CallToolResult{}, fmt.Errorf("unexpected tool name: %s", req.Name)
		}

		input, ok := req.Arguments["input"].(string)
		if !ok {
			return CallToolResult{}, fmt.Errorf("missing or invalid input")
		}

		return CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": fmt.Sprintf("Echo: %s", input),
				},
			},
		}, nil
	}

	_ = server.RegisterTool(tool, handler)

	tests := []struct {
		name    string
		request string
		wantErr bool
		want    string
	}{
		{
			name: "successful tool call",
			request: `{
				"name": "echo-tool",
				"arguments": {"input": "hello world"}
			}`,
			wantErr: false,
			want:    "Echo: hello world",
		},
		{
			name: "tool not found",
			request: `{
				"name": "nonexistent-tool",
				"arguments": {}
			}`,
			wantErr: true,
		},
		{
			name:    "invalid request format",
			request: `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &jsonrpc2.Request{
				Method: string(MethodToolsCall),
				Params: json.RawMessage(tt.request),
			}

			callHandler, exists := server.handlers[string(MethodToolsCall)]
			if !exists {
				t.Fatal("Tools call handler not registered")
			}

			result, err := callHandler(ctx, req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			callResult, ok := result.(CallToolResult)
			if !ok {
				t.Fatalf("Expected CallToolResult, got %T", result)
			}

			if len(callResult.Content) == 0 {
				t.Error("Expected content in result")
				return
			}

			// Check the echoed content
			content := callResult.Content[0].(map[string]interface{})
			text := content["text"].(string)
			if !strings.Contains(text, tt.want) {
				t.Errorf("Expected result to contain %q, got %q", tt.want, text)
			}
		})
	}
}

func TestServerConcurrentOperations(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register multiple tools concurrently
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			tool := Tool{
				Name:        fmt.Sprintf("tool-%d", id),
				Description: fmt.Sprintf("Tool %d", id),
			}

			handler := func(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
				return CallToolResult{}, nil
			}

			if err := server.RegisterTool(tool, handler); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case err := <-errors:
			t.Fatalf("Concurrent registration failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent registrations")
		}
	}

	// Verify all tools were registered
	server.mu.RLock()
	toolCount := len(server.tools)
	server.mu.RUnlock()

	if toolCount != 10 {
		t.Errorf("Expected 10 tools, got %d", toolCount)
	}
}

func TestServerErrorHandling(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a tool that returns an error
	tool := Tool{Name: "error-tool", Description: "Always errors"}
	handler := func(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
		return CallToolResult{}, fmt.Errorf("tool error: %s", req.Name)
	}

	_ = server.RegisterTool(tool, handler)

	ctx := context.Background()
	req := &jsonrpc2.Request{
		Method: string(MethodToolsCall),
		Params: json.RawMessage(`{"name": "error-tool", "arguments": {}}`),
	}

	callHandler := server.handlers[string(MethodToolsCall)]
	result, err := callHandler(ctx, req)

	if err == nil {
		t.Error("Expected error from tool handler")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	if !strings.Contains(err.Error(), "tool error") {
		t.Errorf("Expected tool error message, got: %v", err)
	}
}

func TestServerContextCancellation(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a tool that checks context cancellation
	tool := Tool{Name: "long-running-tool", Description: "Long running operation"}
	handler := func(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
		select {
		case <-ctx.Done():
			return CallToolResult{}, ctx.Err()
		case <-time.After(1 * time.Second):
			return CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "completed",
					},
				},
			}, nil
		}
	}

	_ = server.RegisterTool(tool, handler)

	ctx, cancel := context.WithCancel(context.Background())
	req := &jsonrpc2.Request{
		Method: string(MethodToolsCall),
		Params: json.RawMessage(`{"name": "long-running-tool", "arguments": {}}`),
	}

	// Cancel context immediately
	cancel()

	callHandler := server.handlers[string(MethodToolsCall)]
	result, err := callHandler(ctx, req)

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if result != nil {
		t.Error("Expected nil result on cancellation")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}
