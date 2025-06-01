package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestServerBasicFunctionality(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// Check internal state is initialized
	if server.tools == nil {
		t.Error("tools map not initialized")
	}
	if server.resources == nil {
		t.Error("resources map not initialized")
	}
	if server.prompts == nil {
		t.Error("prompts map not initialized")
	}
}

func TestServerToolRegistration(t *testing.T) {
	server := NewServer("test", "1.0")

	tool := Tool{
		Name:        "echo",
		Description: "Echo tool",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}

	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []any{TextContent{Type: "text", Text: "echo response"}},
		}, nil
	}

	err := server.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	// Test duplicate registration
	err = server.RegisterTool(tool, handler)
	if err == nil {
		t.Error("Expected error for duplicate tool registration")
	}
}

func TestServerResourceRegistrationBasic(t *testing.T) {
	server := NewServer("test", "1.0")

	resource := Resource{
		URI:         "file://test.txt",
		Description: "Test file",
		MimeType:    "text/plain",
	}

	handler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{
			&TextResourceContents{Text: "test content"},
		}, nil
	}

	err := server.RegisterResource(resource, handler)
	if err != nil {
		t.Fatalf("RegisterResource failed: %v", err)
	}

	// Test duplicate registration
	err = server.RegisterResource(resource, handler)
	if err == nil {
		t.Error("Expected error for duplicate resource registration")
	}
}

func TestServerPromptRegistrationBasic(t *testing.T) {
	server := NewServer("test", "1.0")

	prompt := Prompt{
		Name:        "test-prompt",
		Description: "Test prompt",
		Arguments: []PromptArgument{
			{Name: "input", Description: "Input parameter", Required: true},
		},
	}

	handler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return &GetPromptResult{
			Messages: []PromptMessage{
				{Role: RoleUser, Content: []any{TextContent{Type: "text", Text: "test message"}}},
			},
		}, nil
	}

	err := server.RegisterPrompt(prompt, handler)
	if err != nil {
		t.Fatalf("RegisterPrompt failed: %v", err)
	}

	// Test duplicate registration
	err = server.RegisterPrompt(prompt, handler)
	if err == nil {
		t.Error("Expected error for duplicate prompt registration")
	}
}

func TestServerErrorHandlers(t *testing.T) {
	server := NewServer("test", "1.0")

	// Test tool that returns error
	tool := Tool{Name: "error-tool", Description: "Tool that errors"}
	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return nil, errors.New("tool error")
	}

	err := server.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	// Test resource that returns error
	resource := Resource{URI: "error://test", Description: "error"}
	resourceHandler := func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return nil, errors.New("resource error")
	}

	err = server.RegisterResource(resource, resourceHandler)
	if err != nil {
		t.Fatalf("RegisterResource failed: %v", err)
	}

	// Test prompt that returns error
	prompt := Prompt{Name: "error-prompt", Description: "Prompt that errors"}
	promptHandler := func(ctx context.Context, req GetPromptRequest) (*GetPromptResult, error) {
		return nil, errors.New("prompt error")
	}

	err = server.RegisterPrompt(prompt, promptHandler)
	if err != nil {
		t.Fatalf("RegisterPrompt failed: %v", err)
	}
}

func TestServerCapabilitiesUpdate(t *testing.T) {
	server := NewServer("test", "1.0")

	// Initially no capabilities
	if server.capabilities.Tools != nil {
		t.Error("Server should not have tools capability initially")
	}

	// Register a tool
	tool := Tool{Name: "test", Description: "test"}
	handler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: []any{}}, nil
	}

	err := server.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	// Should now have tools capability
	if server.capabilities.Tools == nil {
		t.Error("Server should have tools capability after registering tool")
	}
}

// TextResourceContents is defined in coverage_test.go
