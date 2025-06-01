package protocol

import (
	"encoding/json"
	"testing"
)

func TestServerCapabilities(t *testing.T) {
	capabilities := ServerCapabilities{
		Tools: &ToolsCapability{
			ListChanged: true,
		},
		Resources: &ResourcesCapability{
			Subscribe:   true,
			ListChanged: true,
		},
		Prompts: &PromptsCapability{
			ListChanged: false,
		},
		Logging: &LoggingCapability{},
	}

	// Test JSON marshaling
	data, err := json.Marshal(capabilities)
	if err != nil {
		t.Fatalf("Failed to marshal capabilities: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled ServerCapabilities
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal capabilities: %v", err)
	}

	// Verify fields
	if unmarshaled.Tools == nil {
		t.Error("Tools capability should not be nil")
	} else if !unmarshaled.Tools.ListChanged {
		t.Error("Tools.ListChanged should be true")
	}

	if unmarshaled.Resources == nil {
		t.Error("Resources capability should not be nil")
	} else {
		if !unmarshaled.Resources.Subscribe {
			t.Error("Resources.Subscribe should be true")
		}
		if !unmarshaled.Resources.ListChanged {
			t.Error("Resources.ListChanged should be true")
		}
	}

	if unmarshaled.Prompts == nil {
		t.Error("Prompts capability should not be nil")
	} else if unmarshaled.Prompts.ListChanged {
		t.Error("Prompts.ListChanged should be false")
	}

	if unmarshaled.Logging == nil {
		t.Error("Logging capability should not be nil")
	}
}

func TestImplementation(t *testing.T) {
	impl := Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	data, err := json.Marshal(impl)
	if err != nil {
		t.Fatalf("Failed to marshal implementation: %v", err)
	}

	var unmarshaled Implementation
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal implementation: %v", err)
	}

	if unmarshaled.Name != "test-server" {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, "test-server")
	}
	if unmarshaled.Version != "1.0.0" {
		t.Errorf("Version = %v, want %v", unmarshaled.Version, "1.0.0")
	}
}

func TestInitializeResult(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		},
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: true,
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal initialize result: %v", err)
	}

	var unmarshaled InitializeResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal initialize result: %v", err)
	}

	if unmarshaled.ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %v, want %v", unmarshaled.ProtocolVersion, "2024-11-05")
	}
	if unmarshaled.ServerInfo.Name != "test-server" {
		t.Errorf("ServerInfo.Name = %v, want %v", unmarshaled.ServerInfo.Name, "test-server")
	}
}

func TestTool(t *testing.T) {
	inputSchema := json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`)
	tool := Tool{
		Name:        "echo",
		Description: "Echo tool",
		InputSchema: inputSchema,
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Failed to marshal tool: %v", err)
	}

	var unmarshaled Tool
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal tool: %v", err)
	}

	if unmarshaled.Name != "echo" {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, "echo")
	}
	if unmarshaled.Description != "Echo tool" {
		t.Errorf("Description = %v, want %v", unmarshaled.Description, "Echo tool")
	}
	if string(unmarshaled.InputSchema) != string(inputSchema) {
		t.Errorf("InputSchema mismatch")
	}
}

func TestListToolsResult(t *testing.T) {
	result := ListToolsResult{
		Tools: []Tool{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal list tools result: %v", err)
	}

	var unmarshaled ListToolsResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal list tools result: %v", err)
	}

	if len(unmarshaled.Tools) != 2 {
		t.Errorf("Tools length = %v, want %v", len(unmarshaled.Tools), 2)
	}
	if unmarshaled.Tools[0].Name != "tool1" {
		t.Errorf("First tool name = %v, want %v", unmarshaled.Tools[0].Name, "tool1")
	}
}

func TestContentInterface(t *testing.T) {
	// Test that content types implement the Content interface
	var content Content

	textContent := TextContent{
		Type: "text",
		Text: "Hello, world!",
	}
	content = textContent
	content.isContent() // Should not panic

	imageContent := ImageContent{
		Type:     "image",
		Data:     "base64data",
		MimeType: "image/png",
	}
	content = imageContent
	content.isContent() // Should not panic

	resourceContent := ResourceContent{
		Type: "resource",
		Resource: TextResourceContents{
			URI:  "file://test.txt",
			Text: "resource content",
		},
	}
	content = resourceContent
	content.isContent() // Should not panic
}

func TestCallToolResult(t *testing.T) {
	result := CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: "Result text",
			},
		},
		IsError: false,
	}

	// Test JSON marshaling with interface types
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal call tool result: %v", err)
	}

	t.Logf("CallToolResult JSON: %s", string(data))
}

func TestResource(t *testing.T) {
	resource := Resource{
		URI:         "file://test.txt",
		Name:        "test.txt",
		Description: "Test file",
		MimeType:    "text/plain",
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}

	var unmarshaled Resource
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal resource: %v", err)
	}

	if unmarshaled.URI != "file://test.txt" {
		t.Errorf("URI = %v, want %v", unmarshaled.URI, "file://test.txt")
	}
	if unmarshaled.Name != "test.txt" {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, "test.txt")
	}
}

func TestResourceContentsInterface(t *testing.T) {
	// Test that resource content types implement the ResourceContents interface
	var contents ResourceContents

	textContents := TextResourceContents{
		URI:      "file://test.txt",
		MimeType: "text/plain",
		Text:     "file content",
	}
	contents = textContents
	contents.isResourceContents() // Should not panic

	blobContents := BlobResourceContents{
		URI:      "file://test.bin",
		MimeType: "application/octet-stream",
		Blob:     "YmluYXJ5IGRhdGE=", // base64
	}
	contents = blobContents
	contents.isResourceContents() // Should not panic
}

func TestPrompt(t *testing.T) {
	prompt := Prompt{
		Name:        "test-prompt",
		Description: "A test prompt",
		Arguments: []PromptArgument{
			{
				Name:        "input",
				Description: "Input parameter",
				Required:    true,
			},
			{
				Name:        "optional",
				Description: "Optional parameter",
				Required:    false,
			},
		},
	}

	data, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("Failed to marshal prompt: %v", err)
	}

	var unmarshaled Prompt
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal prompt: %v", err)
	}

	if unmarshaled.Name != "test-prompt" {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, "test-prompt")
	}
	if len(unmarshaled.Arguments) != 2 {
		t.Errorf("Arguments length = %v, want %v", len(unmarshaled.Arguments), 2)
	}
	if !unmarshaled.Arguments[0].Required {
		t.Error("First argument should be required")
	}
	if unmarshaled.Arguments[1].Required {
		t.Error("Second argument should not be required")
	}
}

func TestRole(t *testing.T) {
	// Test role constants
	if RoleUser != "user" {
		t.Errorf("RoleUser = %v, want %v", RoleUser, "user")
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %v, want %v", RoleAssistant, "assistant")
	}
	if RoleSystem != "system" {
		t.Errorf("RoleSystem = %v, want %v", RoleSystem, "system")
	}
}

func TestPromptMessage(t *testing.T) {
	message := PromptMessage{
		Role: RoleUser,
		Content: TextContent{
			Type: "text",
			Text: "Hello",
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Failed to marshal prompt message: %v", err)
	}

	t.Logf("PromptMessage JSON: %s", string(data))
}

func TestGetPromptResult(t *testing.T) {
	result := GetPromptResult{
		Description: "Test prompt result",
		Messages: []PromptMessage{
			{
				Role: RoleSystem,
				Content: TextContent{
					Type: "text",
					Text: "System message",
				},
			},
			{
				Role: RoleUser,
				Content: TextContent{
					Type: "text",
					Text: "User message",
				},
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal get prompt result: %v", err)
	}

	t.Logf("GetPromptResult JSON: %s", string(data))
}
