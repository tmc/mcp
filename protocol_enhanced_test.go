package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestProtocolTypes tests the core protocol type definitions
func TestProtocolTypes(t *testing.T) {
	t.Run("JSONRPCRequest", func(t *testing.T) {
		// Test valid JSONRPC request
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`"test-123"`),
			Method:  "test/method",
			Params:  json.RawMessage(`{"param": "value"}`),
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal JSONRPCRequest: %v", err)
		}

		var decoded JSONRPCRequest
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal JSONRPCRequest: %v", err)
		}

		if decoded.JSONRPC != req.JSONRPC {
			t.Errorf("Expected JSONRPC %s, got %s", req.JSONRPC, decoded.JSONRPC)
		}
		if decoded.Method != req.Method {
			t.Errorf("Expected Method %s, got %s", req.Method, decoded.Method)
		}
	})

	t.Run("JSONRPCResponse", func(t *testing.T) {
		// Test successful response
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`"test-123"`),
			Result:  json.RawMessage(`{"success": true}`),
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal JSONRPCResponse: %v", err)
		}

		var decoded JSONRPCResponse
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal JSONRPCResponse: %v", err)
		}

		if decoded.JSONRPC != resp.JSONRPC {
			t.Errorf("Expected JSONRPC %s, got %s", resp.JSONRPC, decoded.JSONRPC)
		}

		// Test error response
		errorResp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`"test-123"`),
			Error: &JSONRPCError{
				Code:    -32600,
				Message: "Invalid Request",
				Data:    json.RawMessage(`{"details": "malformed request"}`),
			},
		}

		data, err = json.Marshal(errorResp)
		if err != nil {
			t.Fatalf("Failed to marshal error JSONRPCResponse: %v", err)
		}

		var decodedError JSONRPCResponse
		err = json.Unmarshal(data, &decodedError)
		if err != nil {
			t.Fatalf("Failed to unmarshal error JSONRPCResponse: %v", err)
		}

		if decodedError.Error == nil {
			t.Error("Expected error in response")
		} else {
			if decodedError.Error.Code != errorResp.Error.Code {
				t.Errorf("Expected error code %d, got %d", errorResp.Error.Code, decodedError.Error.Code)
			}
			if decodedError.Error.Message != errorResp.Error.Message {
				t.Errorf("Expected error message %s, got %s", errorResp.Error.Message, decodedError.Error.Message)
			}
		}
	})
}

func TestInitializeRequest(t *testing.T) {
	tests := []struct {
		name    string
		request InitializeRequest
		wantErr bool
	}{
		{
			name: "complete initialize request",
			request: InitializeRequest{
				ProtocolVersion: "2024-11-05",
				Capabilities: ClientCapabilities{
					Sampling: &struct{}{},
				},
				ClientInfo: Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
			},
			wantErr: false,
		},
		{
			name: "minimal initialize request",
			request: InitializeRequest{
				ProtocolVersion: "2024-11-05",
				Capabilities:    ClientCapabilities{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal InitializeRequest: %v", err)
			}

			var decoded InitializeRequest
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Failed to unmarshal InitializeRequest: %v", err)
				}
				return
			}

			if tt.wantErr {
				t.Error("Expected error but got none")
				return
			}

			if decoded.ProtocolVersion != tt.request.ProtocolVersion {
				t.Errorf("Expected protocol version %s, got %s", tt.request.ProtocolVersion, decoded.ProtocolVersion)
			}

			// Test capabilities marshaling
			if tt.request.Capabilities.Roots != nil {
				if decoded.Capabilities.Roots == nil {
					t.Error("Expected roots capability in decoded request")
				} else if *decoded.Capabilities.Roots.ListChanged != *tt.request.Capabilities.Roots.ListChanged {
					t.Error("Roots capability not preserved in marshaling")
				}
			}

			// Test client info marshaling
			if tt.request.ClientInfo != nil {
				if decoded.ClientInfo == nil {
					t.Error("Expected client info in decoded request")
				} else {
					if decoded.ClientInfo.Name != tt.request.ClientInfo.Name {
						t.Errorf("Expected client name %s, got %s", tt.request.ClientInfo.Name, decoded.ClientInfo.Name)
					}
					if decoded.ClientInfo.Version != tt.request.ClientInfo.Version {
						t.Errorf("Expected client version %s, got %s", tt.request.ClientInfo.Version, decoded.ClientInfo.Version)
					}
				}
			}
		})
	}
}

func TestToolDefinitions(t *testing.T) {
	t.Run("Tool marshaling", func(t *testing.T) {
		tool := Tool{
			Name:        "test-tool",
			Description: "A test tool for validation",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "string",
						"description": "Test input parameter",
					},
					"count": map[string]interface{}{
						"type":        "integer",
						"minimum":     1,
						"description": "Count parameter",
					},
				},
				"required": []string{"input"},
			},
		}

		data, err := json.Marshal(tool)
		if err != nil {
			t.Fatalf("Failed to marshal Tool: %v", err)
		}

		var decoded Tool
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal Tool: %v", err)
		}

		if decoded.Name != tool.Name {
			t.Errorf("Expected tool name %s, got %s", tool.Name, decoded.Name)
		}
		if decoded.Description != tool.Description {
			t.Errorf("Expected tool description %s, got %s", tool.Description, decoded.Description)
		}

		// Verify schema structure
		schema := decoded.InputSchema.(map[string]interface{})
		if schema["type"] != "object" {
			t.Error("Expected schema type to be 'object'")
		}

		properties := schema["properties"].(map[string]interface{})
		inputProp := properties["input"].(map[string]interface{})
		if inputProp["type"] != "string" {
			t.Error("Expected input property type to be 'string'")
		}
	})

	t.Run("CallToolRequest", func(t *testing.T) {
		req := CallToolRequest{
			Name: "test-tool",
			Arguments: map[string]interface{}{
				"input": "test value",
				"count": 42,
				"flag":  true,
			},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal CallToolRequest: %v", err)
		}

		var decoded CallToolRequest
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal CallToolRequest: %v", err)
		}

		if decoded.Name != req.Name {
			t.Errorf("Expected tool name %s, got %s", req.Name, decoded.Name)
		}

		// Verify arguments
		if decoded.Arguments["input"] != req.Arguments["input"] {
			t.Error("Input argument not preserved")
		}
		if decoded.Arguments["count"] != req.Arguments["count"] {
			t.Error("Count argument not preserved")
		}
		if decoded.Arguments["flag"] != req.Arguments["flag"] {
			t.Error("Flag argument not preserved")
		}
	})

	t.Run("CallToolResult", func(t *testing.T) {
		result := CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Tool execution successful",
				},
				map[string]interface{}{
					"type":     "image",
					"data":     "base64data",
					"mimeType": "image/png",
				},
			},
			IsError: BoolPtr(false),
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal CallToolResult: %v", err)
		}

		var decoded CallToolResult
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal CallToolResult: %v", err)
		}

		if len(decoded.Content) != len(result.Content) {
			t.Errorf("Expected %d content items, got %d", len(result.Content), len(decoded.Content))
		}

		if decoded.IsError == nil || *decoded.IsError != false {
			t.Error("IsError flag not preserved correctly")
		}

		// Verify content structure
		textContent := decoded.Content[0].(map[string]interface{})
		if textContent["type"] != "text" {
			t.Error("Expected first content type to be 'text'")
		}
		if textContent["text"] != "Tool execution successful" {
			t.Error("Text content not preserved")
		}

		imageContent := decoded.Content[1].(map[string]interface{})
		if imageContent["type"] != "image" {
			t.Error("Expected second content type to be 'image'")
		}
	})
}

func TestResourceDefinitions(t *testing.T) {
	t.Run("Resource marshaling", func(t *testing.T) {
		resource := Resource{
			URI:         "file:///test/path/data.txt",
			Name:        "Test Data File",
			Description: StringPtr("A test data file for validation"),
			MimeType:    StringPtr("text/plain"),
		}

		data, err := json.Marshal(resource)
		if err != nil {
			t.Fatalf("Failed to marshal Resource: %v", err)
		}

		var decoded Resource
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal Resource: %v", err)
		}

		if decoded.URI != resource.URI {
			t.Errorf("Expected resource URI %s, got %s", resource.URI, decoded.URI)
		}
		if decoded.Name != resource.Name {
			t.Errorf("Expected resource name %s, got %s", resource.Name, decoded.Name)
		}
		if decoded.Description == nil || *decoded.Description != *resource.Description {
			t.Error("Resource description not preserved")
		}
		if decoded.MimeType == nil || *decoded.MimeType != *resource.MimeType {
			t.Error("Resource mime type not preserved")
		}
	})

	t.Run("ResourceTemplate", func(t *testing.T) {
		template := ResourceTemplate{
			URITemplate: "file:///{path}",
			Name:        "Dynamic File Resource",
			Description: StringPtr("A templated file resource"),
			MimeType:    StringPtr("application/octet-stream"),
		}

		data, err := json.Marshal(template)
		if err != nil {
			t.Fatalf("Failed to marshal ResourceTemplate: %v", err)
		}

		var decoded ResourceTemplate
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ResourceTemplate: %v", err)
		}

		if decoded.URITemplate != template.URITemplate {
			t.Errorf("Expected URI template %s, got %s", template.URITemplate, decoded.URITemplate)
		}
		if decoded.Name != template.Name {
			t.Errorf("Expected template name %s, got %s", template.Name, decoded.Name)
		}
	})

	t.Run("ReadResourceResult", func(t *testing.T) {
		result := ReadResourceResult{
			Contents: []interface{}{
				map[string]interface{}{
					"uri":      "file:///test/data.txt",
					"mimeType": "text/plain",
					"text":     "This is test file content",
				},
				map[string]interface{}{
					"uri":      "file:///test/binary.dat",
					"mimeType": "application/octet-stream",
					"blob":     "YmluYXJ5IGRhdGE=", // base64: "binary data"
				},
			},
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal ReadResourceResult: %v", err)
		}

		var decoded ReadResourceResult
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ReadResourceResult: %v", err)
		}

		if len(decoded.Contents) != len(result.Contents) {
			t.Errorf("Expected %d content items, got %d", len(result.Contents), len(decoded.Contents))
		}

		// Verify text content
		textContent := decoded.Contents[0].(map[string]interface{})
		if textContent["mimeType"] != "text/plain" {
			t.Error("Text content mime type not preserved")
		}
		if textContent["text"] != "This is test file content" {
			t.Error("Text content not preserved")
		}

		// Verify binary content
		binaryContent := decoded.Contents[1].(map[string]interface{})
		if binaryContent["mimeType"] != "application/octet-stream" {
			t.Error("Binary content mime type not preserved")
		}
		if binaryContent["blob"] != "YmluYXJ5IGRhdGE=" {
			t.Error("Binary content not preserved")
		}
	})
}

func TestPromptDefinitions(t *testing.T) {
	t.Run("Prompt marshaling", func(t *testing.T) {
		prompt := Prompt{
			Name:        "test-prompt",
			Description: "A test prompt for validation",
			Arguments: []PromptArgument{
				{
					Name:        "topic",
					Description: "The topic to discuss",
					Required:    BoolPtr(true),
				},
				{
					Name:        "style",
					Description: "Writing style preference",
					Required:    BoolPtr(false),
				},
			},
		}

		data, err := json.Marshal(prompt)
		if err != nil {
			t.Fatalf("Failed to marshal Prompt: %v", err)
		}

		var decoded Prompt
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal Prompt: %v", err)
		}

		if decoded.Name != prompt.Name {
			t.Errorf("Expected prompt name %s, got %s", prompt.Name, decoded.Name)
		}
		if decoded.Description != prompt.Description {
			t.Errorf("Expected prompt description %s, got %s", prompt.Description, decoded.Description)
		}

		if len(decoded.Arguments) != len(prompt.Arguments) {
			t.Errorf("Expected %d arguments, got %d", len(prompt.Arguments), len(decoded.Arguments))
		}

		// Verify required argument
		requiredArg := decoded.Arguments[0]
		if requiredArg.Name != "topic" {
			t.Error("Required argument name not preserved")
		}
		if requiredArg.Required == nil || *requiredArg.Required != true {
			t.Error("Required flag not preserved")
		}

		// Verify optional argument
		optionalArg := decoded.Arguments[1]
		if optionalArg.Name != "style" {
			t.Error("Optional argument name not preserved")
		}
		if optionalArg.Required == nil || *optionalArg.Required != false {
			t.Error("Optional flag not preserved")
		}
	})

	t.Run("GetPromptResult", func(t *testing.T) {
		result := GetPromptResult{
			Description: StringPtr("Generated prompt for the topic"),
			Messages: []PromptMessage{
				{
					Role: "system",
					Content: map[string]interface{}{
						"type": "text",
						"text": "You are a helpful assistant.",
					},
				},
				{
					Role: "user",
					Content: map[string]interface{}{
						"type": "text",
						"text": "Please help me with the topic.",
					},
				},
			},
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal GetPromptResult: %v", err)
		}

		var decoded GetPromptResult
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal GetPromptResult: %v", err)
		}

		if decoded.Description == nil || *decoded.Description != *result.Description {
			t.Error("Prompt description not preserved")
		}

		if len(decoded.Messages) != len(result.Messages) {
			t.Errorf("Expected %d messages, got %d", len(result.Messages), len(decoded.Messages))
		}

		// Verify system message
		systemMsg := decoded.Messages[0]
		if systemMsg.Role != "system" {
			t.Error("System message role not preserved")
		}
		systemContent := systemMsg.Content.(map[string]interface{})
		if systemContent["type"] != "text" {
			t.Error("System message content type not preserved")
		}

		// Verify user message
		userMsg := decoded.Messages[1]
		if userMsg.Role != "user" {
			t.Error("User message role not preserved")
		}
	})
}

func TestCapabilityNegotiation(t *testing.T) {
	t.Run("ClientCapabilities", func(t *testing.T) {
		caps := ClientCapabilities{
			Experimental: map[string]interface{}{
				"customFeature": true,
				"betaVersion":   "2.0",
			},
			Roots: &RootsCapability{
				ListChanged: BoolPtr(true),
			},
			Sampling: &SamplingCapability{},
		}

		data, err := json.Marshal(caps)
		if err != nil {
			t.Fatalf("Failed to marshal ClientCapabilities: %v", err)
		}

		var decoded ClientCapabilities
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ClientCapabilities: %v", err)
		}

		// Verify experimental capabilities
		if decoded.Experimental["customFeature"] != true {
			t.Error("Custom experimental feature not preserved")
		}
		if decoded.Experimental["betaVersion"] != "2.0" {
			t.Error("Beta version experimental feature not preserved")
		}

		// Verify standard capabilities
		if decoded.Roots == nil || decoded.Roots.ListChanged == nil || *decoded.Roots.ListChanged != true {
			t.Error("Roots capability not preserved")
		}

		if decoded.Sampling == nil {
			t.Error("Sampling capability not preserved")
		}
	})

	t.Run("ServerCapabilities", func(t *testing.T) {
		caps := ServerCapabilities{
			Experimental: map[string]interface{}{
				"serverFeature": "enabled",
			},
			Logging: &LoggingCapability{},
			Prompts: &PromptsCapability{
				ListChanged: true,
			},
			Resources: &ResourcesCapability{
				Subscribe:   BoolPtr(true),
				ListChanged: true,
			},
			Tools: &ToolsCapability{
				ListChanged: true,
			},
		}

		data, err := json.Marshal(caps)
		if err != nil {
			t.Fatalf("Failed to marshal ServerCapabilities: %v", err)
		}

		var decoded ServerCapabilities
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ServerCapabilities: %v", err)
		}

		// Verify experimental capabilities
		if decoded.Experimental["serverFeature"] != "enabled" {
			t.Error("Server experimental feature not preserved")
		}

		// Verify standard capabilities
		if decoded.Logging == nil {
			t.Error("Logging capability not preserved")
		}

		if decoded.Prompts == nil || !decoded.Prompts.ListChanged {
			t.Error("Prompts capability not preserved")
		}

		if decoded.Resources == nil || !decoded.Resources.ListChanged {
			t.Error("Resources capability not preserved")
		}
		if decoded.Resources.Subscribe == nil || *decoded.Resources.Subscribe != true {
			t.Error("Resources subscribe capability not preserved")
		}

		if decoded.Tools == nil || !decoded.Tools.ListChanged {
			t.Error("Tools capability not preserved")
		}
	})
}

func TestProtocolConstants(t *testing.T) {
	// Test that protocol constants are properly defined
	expectedMethods := map[string]Method{
		"initialize":               MethodInitialize,
		"initialized":              MethodInitialized,
		"ping":                     MethodPing,
		"tools/list":               MethodToolsList,
		"tools/call":               MethodToolsCall,
		"prompts/list":             MethodPromptsList,
		"prompts/get":              MethodPromptsGet,
		"resources/list":           MethodResourcesList,
		"resources/read":           MethodResourcesRead,
		"resources/templates/list": MethodResourceTemplatesList,
		"notifications/cancelled":  MethodNotificationCancelled,
	}

	for expected, actual := range expectedMethods {
		if string(actual) != expected {
			t.Errorf("Expected method constant %s to equal %q, got %q", reflect.TypeOf(actual).Name(), expected, string(actual))
		}
	}

	// Test protocol version constant
	if PROTOCOL_VERSION == "" {
		t.Error("PROTOCOL_VERSION constant should not be empty")
	}
}

// Helper functions for pointer types
func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}
