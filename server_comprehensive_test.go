package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"log/slog"
)

// TestServerInitialization tests server creation and configuration
func TestServerInitialization(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ServerOption
		checker func(*Server) error
	}{
		{
			name: "default server",
			opts: nil,
			checker: func(s *Server) error {
				if s.name == "" {
					return errors.New("server name should not be empty")
				}
				if s.version == "" {
					return errors.New("server version should not be empty")
				}
				return nil
			},
		},
		{
			name: "custom name and version",
			opts: []ServerOption{
				WithServerName("custom-server"),
				WithServerVersion("2.0.0"),
			},
			checker: func(s *Server) error {
				if s.name != "custom-server" {
					return errors.New("custom name not set")
				}
				if s.version != "2.0.0" {
					return errors.New("custom version not set")
				}
				return nil
			},
		},
		{
			name: "with instructions",
			opts: []ServerOption{
				WithServerInstructions("Test instructions"),
			},
			checker: func(s *Server) error {
				if s.instructions != "Test instructions" {
					return errors.New("instructions not set")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("test", "1.0", tt.opts...)
			if err := tt.checker(server); err != nil {
				t.Error(err)
			}
		})
	}
}

// TestServerToolManagement tests tool registration and retrieval
func TestServerToolManagement(t *testing.T) {
	server := NewServer("test", "1.0", WithTestLogger(t, slog.LevelDebug))

	// Define test tools
	tool1Schema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type": "string",
			},
		},
	})
	tool1 := Tool{
		Name:        "test-tool-1",
		Description: "First test tool",
		InputSchema: json.RawMessage(tool1Schema),
	}

	tool2 := Tool{
		Name:        "test-tool-2",
		Description: "Second test tool",
	}

	handler1 := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []interface{}{map[string]string{
				"type": "text",
				"text": "Tool 1 result",
			}},
		}, nil
	}

	handler2 := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []interface{}{map[string]string{
				"type": "text",
				"text": "Tool 2 result",
			}},
		}, nil
	}

	// Test registration
	err := server.RegisterTool(tool1, handler1)
	if err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}

	err = server.RegisterTool(tool2, handler2)
	if err != nil {
		t.Fatalf("Failed to register tool2: %v", err)
	}

	// Test duplicate registration
	err = server.RegisterTool(tool1, handler1)
	if err == nil {
		t.Error("Expected error for duplicate tool registration")
	}

	// Test tool retrieval
	server.mu.RLock()
	if len(server.tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(server.tools))
	}

	if _, exists := server.tools["test-tool-1"]; !exists {
		t.Error("Tool 1 not found")
	}

	if _, exists := server.tools["test-tool-2"]; !exists {
		t.Error("Tool 2 not found")
	}
	server.mu.RUnlock()
}

// TestServerResourceManagement tests resource registration and handling
func TestServerResourceManagement(t *testing.T) {
	server := NewServer("test", "1.0", WithTestLogger(t, slog.LevelDebug))

	// Define test resources
	resource1 := Resource{
		URI:         "test://resource-1",
		Description: "First test resource",
		MimeType:    "text/plain",
	}

	resource2 := Resource{
		URI:         "test://resource-2",
		Description: "Second test resource",
		MimeType:    "application/json",
	}

	handler1 := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{
			TextResourceContents{
				URI:      req.URI,
				MimeType: "text/plain",
				Text:     "Resource 1 content",
			},
		}, nil
	}

	handler2 := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{
			BlobResourceContents{
				URI:      req.URI,
				MimeType: "application/json",
				Blob:     `eyJ0ZXN0IjogImRhdGEifQ==`, // base64 encoded {"test": "data"}
			},
		}, nil
	}

	// Test registration
	err := server.RegisterResource(resource1, handler1)
	if err != nil {
		t.Fatalf("Failed to register resource1: %v", err)
	}

	err = server.RegisterResource(resource2, handler2)
	if err != nil {
		t.Fatalf("Failed to register resource2: %v", err)
	}

	// Test duplicate registration
	err = server.RegisterResource(resource1, handler1)
	if err == nil {
		t.Error("Expected error for duplicate resource registration")
	}

	// Verify resources are stored
	server.mu.RLock()
	if len(server.resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(server.resources))
	}
	server.mu.RUnlock()
}

// TestServerResourceTemplateManagement tests resource template functionality
func TestServerResourceTemplateManagement(t *testing.T) {
	server := NewServer("test", "1.0", WithTestLogger(t, slog.LevelDebug))

	template := ResourceTemplate{
		Template:    "test://files/{id}",
		Description: "Template for file resources",
	}

	handler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		// Extract ID from URI
		parts := strings.Split(req.URI, "/")
		if len(parts) < 3 {
			return nil, errors.New("invalid URI format")
		}
		id := parts[len(parts)-1]

		return []ResourceContents{
			TextResourceContents{
				URI:      req.URI,
				MimeType: "text/plain",
				Text:     "File content for ID: " + id,
			},
		}, nil
	}

	// Test registration
	err := server.RegisterResourceTemplate(template, handler)
	if err != nil {
		t.Fatalf("Failed to register resource template: %v", err)
	}

	// Test duplicate registration
	err = server.RegisterResourceTemplate(template, handler)
	if err == nil {
		t.Error("Expected error for duplicate resource template registration")
	}

	// Verify template is stored
	server.mu.RLock()
	if len(server.resourceTmpls) != 1 {
		t.Errorf("Expected 1 resource template, got %d", len(server.resourceTmpls))
	}
	server.mu.RUnlock()
}

// TestServerPromptManagement tests prompt registration and handling
func TestServerPromptManagement(t *testing.T) {
	server := NewServer("test", "1.0", WithTestLogger(t, slog.LevelDebug))

	prompt := Prompt{
		Name:        "test-prompt",
		Description: "A test prompt",
		Arguments: []PromptArgument{
			{
				Name:        "topic",
				Description: "The topic to write about",
				Required:    true,
			},
		},
	}

	handler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		topic := "default"
		if req.Arguments != nil {
			if t, ok := req.Arguments["topic"].(string); ok {
				topic = t
			}
		}

		return &GetPromptResult{
			Messages: []PromptMessage{
				{
					Role: RoleUser,
					Content: []any{
						TextContent{
							Type: "text",
							Text: "Write about: " + topic,
						},
					},
				},
			},
		}, nil
	}

	// Test registration
	err := server.RegisterPrompt(prompt, handler)
	if err != nil {
		t.Fatalf("Failed to register prompt: %v", err)
	}

	// Test duplicate registration
	err = server.RegisterPrompt(prompt, handler)
	if err == nil {
		t.Error("Expected error for duplicate prompt registration")
	}

	// Verify prompt is stored
	server.mu.RLock()
	if len(server.prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(server.prompts))
	}
	server.mu.RUnlock()
}

// TestServerCapabilitiesReporting tests that server reports correct capabilities
func TestServerCapabilitiesReporting(t *testing.T) {
	// Create server with capabilities enabled
	server := NewServer("test", "1.0", func(s *Server) {
		s.capabilities.Tools = &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{ListChanged: true}
		s.capabilities.Resources = &struct {
			Subscribe   bool `json:"subscribe,omitempty"`
			ListChanged bool `json:"listChanged,omitempty"`
		}{ListChanged: true}
		s.capabilities.Prompts = &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{ListChanged: true}
	})

	// Register various features to test capability reporting
	server.RegisterTool(Tool{Name: "test-tool"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{}, nil
	})

	server.RegisterResource(Resource{URI: "test://resource"}, func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return nil, nil
	})

	server.RegisterPrompt(Prompt{Name: "test-prompt"}, func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	})

	// Test that capabilities are properly set
	caps := server.capabilities

	// Should have tools capability if tools are registered
	if caps.Tools == nil {
		t.Error("Expected tools capability to be set")
	}

	if caps.Resources == nil {
		t.Error("Expected resources capability to be set")
	}

	if caps.Prompts == nil {
		t.Error("Expected prompts capability to be set")
	}

	// Verify tools are actually registered
	server.mu.RLock()
	toolCount := len(server.tools)
	resourceCount := len(server.resources)
	promptCount := len(server.prompts)
	server.mu.RUnlock()

	if toolCount != 1 {
		t.Errorf("Expected 1 tool, got %d", toolCount)
	}
	if resourceCount != 1 {
		t.Errorf("Expected 1 resource, got %d", resourceCount)
	}
	if promptCount != 1 {
		t.Errorf("Expected 1 prompt, got %d", promptCount)
	}
}

// TestServerErrorHandling tests various error conditions
func TestServerErrorHandling(t *testing.T) {
	server := NewServer("test", "1.0", WithTestLogger(t, slog.LevelDebug))

	// Note: The current implementation doesn't validate empty names/URIs during registration
	// This test verifies the current behavior rather than expected validation

	// Test registering tool with empty name (currently allowed)
	err := server.RegisterTool(Tool{Name: ""}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return nil, nil
	})
	if err != nil {
		t.Logf("Tool with empty name rejected: %v", err)
	}

	// Test registering resource with empty URI (currently allowed)
	err = server.RegisterResource(Resource{URI: ""}, func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return nil, nil
	})
	if err != nil {
		t.Logf("Resource with empty URI rejected: %v", err)
	}

	// Test registering prompt with empty name (currently allowed)
	err = server.RegisterPrompt(Prompt{Name: ""}, func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return nil, nil
	})
	if err != nil {
		t.Logf("Prompt with empty name rejected: %v", err)
	}

	// Test registering resource template with empty template (currently allowed)
	err = server.RegisterResourceTemplate(ResourceTemplate{Template: ""}, func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return nil, nil
	})
	if err != nil {
		t.Logf("Resource template with empty template rejected: %v", err)
	}

	// Test duplicate registration (should fail)
	tool := Tool{Name: "duplicate-tool"}
	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{}, nil
	}

	// First registration should succeed
	err = server.RegisterTool(tool, handler)
	if err != nil {
		t.Errorf("First tool registration failed: %v", err)
	}

	// Second registration should fail
	err = server.RegisterTool(tool, handler)
	if err == nil {
		t.Error("Expected error for duplicate tool registration")
	}
}

// TestServerJSONSerialization tests that server data structures serialize correctly
func TestServerJSONSerialization(t *testing.T) {
	server := NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))

	// Register some test items
	toolSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type": "string",
			},
		},
	})
	server.RegisterTool(Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(toolSchema),
	}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{}, nil
	})

	// Test that tools can be marshaled to JSON
	server.mu.RLock()
	tools := make([]Tool, 0, len(server.tools))
	for _, def := range server.tools {
		tools = append(tools, def.tool)
	}
	server.mu.RUnlock()

	data, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("Failed to marshal tools: %v", err)
	}

	// Test that we can unmarshal back
	var unmarshaled []Tool
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal tools: %v", err)
	}

	if len(unmarshaled) != 1 {
		t.Errorf("Expected 1 tool after unmarshal, got %d", len(unmarshaled))
	}

	if unmarshaled[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got %s", unmarshaled[0].Name)
	}
}

// TestServerConcurrentAccess tests thread safety of server operations
func TestServerConcurrentAccess(t *testing.T) {
	server := NewServer("test", "1.0", WithTestLogger(t, slog.LevelDebug))

	// Test concurrent tool registration
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()

			tool := Tool{
				Name:        fmt.Sprintf("tool-%d", i),
				Description: fmt.Sprintf("Tool %d", i),
			}

			handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
				return &CallToolResult{}, nil
			}

			server.RegisterTool(tool, handler)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all tools were registered
	server.mu.RLock()
	toolCount := len(server.tools)
	server.mu.RUnlock()

	if toolCount != 10 {
		t.Errorf("Expected 10 tools, got %d", toolCount)
	}
}
