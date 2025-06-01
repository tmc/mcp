package golang_tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tmc/mcp/protocol"
	"github.com/tmc/mcp/server"
)

// mockServer is a mock implementation of server.Server for testing
type mockServer struct {
	info    server.ServerInfo
	tools   []server.Tool
	prompts []server.Prompt
}

func (m *mockServer) GetServerInfo() server.ServerInfo {
	return m.info
}

func (m *mockServer) GetTools() []server.Tool {
	return m.tools
}

func (m *mockServer) GetPrompts() []server.Prompt {
	return m.prompts
}

func (m *mockServer) CallTool(ctx context.Context, name string, args map[string]json.RawMessage) (any, error) {
	// Mock tool call result
	return protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: "Tool result",
			},
		},
		IsError: false,
	}, nil
}

func (m *mockServer) GetPrompt(ctx context.Context, name string, args map[string]string) (any, error) {
	// Mock prompt result
	return protocol.GetPromptResult{
		Description: "Test prompt",
		Messages: []protocol.PromptMessage{
			{
				Role: protocol.RoleUser,
				Content: protocol.TextContent{
					Type: "text",
					Text: "Test message",
				},
			},
		},
	}, nil
}

func TestGolangToolsAdapter_Initialize(t *testing.T) {
	adapter := NewAdapter()
	
	mockSrv := &mockServer{
		info: server.ServerInfo{
			Name:            "test-server",
			Version:         "1.0.0",
			Instructions:    "Test instructions",
			ProtocolVersion: "2024-11-05",
		},
		tools: []server.Tool{
			{
				Name:        "test-tool",
				Description: "A test tool",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		},
		prompts: []server.Prompt{
			{
				Name:        "test-prompt",
				Description: "A test prompt",
				Arguments: []protocol.PromptArgument{
					{
						Name:        "arg1",
						Description: "First argument",
						Required:    true,
					},
				},
			},
		},
	}
	
	ctx := context.Background()
	if err := adapter.Initialize(ctx, mockSrv); err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}
	
	// Test GetCapabilities
	caps := adapter.GetCapabilities()
	if caps.Tools == nil {
		t.Error("Expected tools capability to be set")
	}
	if caps.Prompts == nil {
		t.Error("Expected prompts capability to be set")
	}
}

func TestGolangToolsAdapter_HandleRequest(t *testing.T) {
	adapter := NewAdapter()
	
	mockSrv := &mockServer{
		info: server.ServerInfo{
			Name:            "test-server",
			Version:         "1.0.0",
			Instructions:    "Test instructions",
			ProtocolVersion: "2024-11-05",
		},
		tools: []server.Tool{
			{
				Name:        "test-tool",
				Description: "A test tool",
			},
		},
	}
	
	ctx := context.Background()
	if err := adapter.Initialize(ctx, mockSrv); err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}
	
	// Test initialize
	result, err := adapter.HandleRequest(ctx, "initialize", nil)
	if err != nil {
		t.Fatalf("Failed to handle initialize: %v", err)
	}
	
	initResult, ok := result.(protocol.InitializeResult)
	if !ok {
		t.Fatalf("Expected InitializeResult, got %T", result)
	}
	
	if initResult.ServerInfo.Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got %s", initResult.ServerInfo.Name)
	}
	
	// Test list tools
	result, err = adapter.HandleRequest(ctx, "tools/list", nil)
	if err != nil {
		t.Fatalf("Failed to handle tools/list: %v", err)
	}
	
	listResult, ok := result.(protocol.ListToolsResult)
	if !ok {
		t.Fatalf("Expected ListToolsResult, got %T", result)
	}
	
	if len(listResult.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(listResult.Tools))
	}
	
	// Test call tool
	params := map[string]interface{}{
		"name":      "test-tool",
		"arguments": map[string]interface{}{"key": "value"},
	}
	
	result, err = adapter.HandleRequest(ctx, "tools/call", params)
	if err != nil {
		t.Fatalf("Failed to handle tools/call: %v", err)
	}
	
	callResult, ok := result.(protocol.CallToolResult)
	if !ok {
		t.Fatalf("Expected CallToolResult, got %T", result)
	}
	
	if len(callResult.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(callResult.Content))
	}
}

func TestGolangToolsAdapter_ContentConversion(t *testing.T) {
	// Test text content conversion
	textContent := protocol.TextContent{
		Type: "text",
		Text: "Hello",
	}
	
	golangContent := convertSDKContentToGolang(textContent)
	if golangContent.Type != "text" {
		t.Errorf("Expected type 'text', got %s", golangContent.Type)
	}
	if golangContent.Text != "Hello" {
		t.Errorf("Expected text 'Hello', got %s", golangContent.Text)
	}
	
	// Test image content conversion
	imageContent := protocol.ImageContent{
		Type:     "image",
		Data:     "base64data",
		MimeType: "image/png",
	}
	
	golangContent = convertSDKContentToGolang(imageContent)
	if golangContent.Type != "image" {
		t.Errorf("Expected type 'image', got %s", golangContent.Type)
	}
	if golangContent.Data != "base64data" {
		t.Errorf("Expected data 'base64data', got %s", golangContent.Data)
	}
	if golangContent.MIMEType != "image/png" {
		t.Errorf("Expected mime type 'image/png', got %s", golangContent.MIMEType)
	}
}