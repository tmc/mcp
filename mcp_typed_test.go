package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// Test input/output types for typed tool testing
type AddInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddOutput struct {
	Sum int `json:"sum"`
}

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Message string `json:"message"`
}

type ComplexInput struct {
	Text   string            `json:"text"`
	Number float64           `json:"number"`
	Flag   bool              `json:"flag"`
	Items  []string          `json:"items"`
	Meta   map[string]string `json:"meta"`
}

type ComplexOutput struct {
	Result string `json:"result"`
	Count  int    `json:"count"`
}

// Test RegisterTypedTool with simple types
func TestRegisterTypedTool_SimpleTypes(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Test successful registration
	err := RegisterTypedTool(server, "add", "Add two numbers", func(ctx context.Context, input AddInput) (AddOutput, error) {
		return AddOutput{Sum: input.A + input.B}, nil
	})

	if err != nil {
		t.Fatalf("Failed to register typed tool: %v", err)
	}

	// Verify the tool was registered
	if server.tools == nil {
		t.Fatal("Server tools map is nil")
	}

	toolDef, exists := server.tools["add"]
	if !exists {
		t.Fatal("Tool 'add' was not registered")
	}

	if toolDef.tool.Name != "add" {
		t.Errorf("Expected tool name 'add', got %s", toolDef.tool.Name)
	}

	if toolDef.tool.Description != "Add two numbers" {
		t.Errorf("Expected description 'Add two numbers', got %s", toolDef.tool.Description)
	}

	// Verify schema was generated
	if toolDef.tool.InputSchema == nil {
		t.Fatal("Input schema was not generated")
	}

	var schema map[string]any
	err = json.Unmarshal(toolDef.tool.InputSchema, &schema)
	if err != nil {
		t.Fatalf("Failed to unmarshal input schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("Expected schema type 'object', got %v", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties is not a map")
	}

	// Check that properties for a and b exist and are numbers
	aProperty, exists := properties["a"].(map[string]any)
	if !exists {
		t.Fatal("Property 'a' not found in schema")
	}
	if aProperty["type"] != "number" {
		t.Errorf("Expected property 'a' type 'number', got %v", aProperty["type"])
	}

	bProperty, exists := properties["b"].(map[string]any)
	if !exists {
		t.Fatal("Property 'b' not found in schema")
	}
	if bProperty["type"] != "number" {
		t.Errorf("Expected property 'b' type 'number', got %v", bProperty["type"])
	}
}

// Test RegisterTypedTool with string types
func TestRegisterTypedTool_StringTypes(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	err := RegisterTypedTool(server, "greet", "Greet someone", func(ctx context.Context, input GreetInput) (GreetOutput, error) {
		return GreetOutput{Message: "Hello, " + input.Name + "!"}, nil
	})

	if err != nil {
		t.Fatalf("Failed to register typed tool: %v", err)
	}

	toolDef, exists := server.tools["greet"]
	if !exists {
		t.Fatal("Tool 'greet' was not registered")
	}

	// Verify schema generation for string types
	var schema map[string]any
	err = json.Unmarshal(toolDef.tool.InputSchema, &schema)
	if err != nil {
		t.Fatalf("Failed to unmarshal input schema: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties is not a map")
	}

	nameProperty, exists := properties["name"].(map[string]any)
	if !exists {
		t.Fatal("Property 'name' not found in schema")
	}
	if nameProperty["type"] != "string" {
		t.Errorf("Expected property 'name' type 'string', got %v", nameProperty["type"])
	}
}

// Test RegisterTypedTool with complex types
func TestRegisterTypedTool_ComplexTypes(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	err := RegisterTypedTool(server, "complex", "Process complex input", func(ctx context.Context, input ComplexInput) (ComplexOutput, error) {
		return ComplexOutput{
			Result: input.Text + " processed",
			Count:  len(input.Items),
		}, nil
	})

	if err != nil {
		t.Fatalf("Failed to register typed tool: %v", err)
	}

	toolDef, exists := server.tools["complex"]
	if !exists {
		t.Fatal("Tool 'complex' was not registered")
	}

	// Verify schema generation for complex types
	var schema map[string]any
	err = json.Unmarshal(toolDef.tool.InputSchema, &schema)
	if err != nil {
		t.Fatalf("Failed to unmarshal input schema: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties is not a map")
	}

	// Check various property types
	textProp, exists := properties["text"].(map[string]any)
	if !exists {
		t.Fatal("Property 'text' not found")
	}
	if textProp["type"] != "string" {
		t.Errorf("Expected 'text' type 'string', got %v", textProp["type"])
	}

	numberProp, exists := properties["number"].(map[string]any)
	if !exists {
		t.Fatal("Property 'number' not found")
	}
	if numberProp["type"] != "number" {
		t.Errorf("Expected 'number' type 'number', got %v", numberProp["type"])
	}

	flagProp, exists := properties["flag"].(map[string]any)
	if !exists {
		t.Fatal("Property 'flag' not found")
	}
	if flagProp["type"] != "boolean" {
		t.Errorf("Expected 'flag' type 'boolean', got %v", flagProp["type"])
	}

	itemsProp, exists := properties["items"].(map[string]any)
	if !exists {
		t.Fatal("Property 'items' not found")
	}
	if itemsProp["type"] != "array" {
		t.Errorf("Expected 'items' type 'array', got %v", itemsProp["type"])
	}

	metaProp, exists := properties["meta"].(map[string]any)
	if !exists {
		t.Fatal("Property 'meta' not found")
	}
	if metaProp["type"] != "object" {
		t.Errorf("Expected 'meta' type 'object', got %v", metaProp["type"])
	}
}

// Test tool execution with valid input
func TestTypedTool_Execution_ValidInput(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	err := RegisterTypedTool(server, "add", "Add two numbers", func(ctx context.Context, input AddInput) (AddOutput, error) {
		return AddOutput{Sum: input.A + input.B}, nil
	})

	if err != nil {
		t.Fatalf("Failed to register typed tool: %v", err)
	}

	// Prepare a valid call request
	requestArgs, _ := json.Marshal(AddInput{A: 5, B: 3})
	request := CallToolRequest{
		Arguments: requestArgs,
	}

	// Execute the tool
	toolDef := server.tools["add"]
	result, err := toolDef.handler(context.Background(), request)

	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if result.IsError {
		t.Fatal("Tool returned error result")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	content, ok := result.Content[0].(map[string]any)
	if !ok {
		t.Fatal("Content is not a map")
	}

	if content["type"] != "text" {
		t.Errorf("Expected content type 'text', got %v", content["type"])
	}

	if content["format"] != "json" {
		t.Errorf("Expected content format 'json', got %v", content["format"])
	}

	// Parse the JSON result
	var output AddOutput
	err = json.Unmarshal([]byte(content["text"].(string)), &output)
	if err != nil {
		t.Fatalf("Failed to unmarshal output: %v", err)
	}

	if output.Sum != 8 {
		t.Errorf("Expected sum 8, got %d", output.Sum)
	}
}

// Test tool execution with invalid input
func TestTypedTool_Execution_InvalidInput(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	err := RegisterTypedTool(server, "add", "Add two numbers", func(ctx context.Context, input AddInput) (AddOutput, error) {
		return AddOutput{Sum: input.A + input.B}, nil
	})

	if err != nil {
		t.Fatalf("Failed to register typed tool: %v", err)
	}

	// Prepare an invalid call request (malformed JSON)
	request := CallToolRequest{
		Arguments: json.RawMessage(`{"a": "not-a-number", "b": true}`),
	}

	// Execute the tool
	toolDef := server.tools["add"]
	result, err := toolDef.handler(context.Background(), request)

	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if !result.IsError {
		t.Fatal("Expected error result for invalid input")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	content, ok := result.Content[0].(map[string]string)
	if !ok {
		t.Fatal("Content is not a string map")
	}

	if content["type"] != "text" {
		t.Errorf("Expected content type 'text', got %v", content["type"])
	}

	if !contains(content["text"], "Invalid input") {
		t.Errorf("Expected error message to contain 'Invalid input', got %s", content["text"])
	}
}

// Test tool execution with handler error
func TestTypedTool_Execution_HandlerError(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	err := RegisterTypedTool(server, "failing_add", "Add two numbers but fail", func(ctx context.Context, input AddInput) (AddOutput, error) {
		return AddOutput{}, errors.New("computation failed")
	})

	if err != nil {
		t.Fatalf("Failed to register typed tool: %v", err)
	}

	// Prepare a valid call request
	requestArgs, _ := json.Marshal(AddInput{A: 5, B: 3})
	request := CallToolRequest{
		Arguments: requestArgs,
	}

	// Execute the tool
	toolDef := server.tools["failing_add"]
	result, err := toolDef.handler(context.Background(), request)

	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if !result.IsError {
		t.Fatal("Expected error result for handler error")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	content, ok := result.Content[0].(map[string]string)
	if !ok {
		t.Fatal("Content is not a string map")
	}

	if !contains(content["text"], "computation failed") {
		t.Errorf("Expected error message to contain 'computation failed', got %s", content["text"])
	}
}

// Test createJSONSchema function with simple types
func TestCreateJSONSchema_SimpleTypes(t *testing.T) {
	// Test with struct type
	schema, err := createJSONSchema[AddInput]()
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	var schemaMap map[string]any
	err = json.Unmarshal(schema, &schemaMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	if schemaMap["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", schemaMap["type"])
	}

	// Test with string type
	schema, err = createJSONSchema[string]()
	if err != nil {
		t.Fatalf("Failed to create string schema: %v", err)
	}

	err = json.Unmarshal(schema, &schemaMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal string schema: %v", err)
	}

	if schemaMap["type"] != "string" {
		t.Errorf("Expected type 'string', got %v", schemaMap["type"])
	}
}

// Test createJSONSchema function with complex types
func TestCreateJSONSchema_ComplexTypes(t *testing.T) {
	schema, err := createJSONSchema[ComplexInput]()
	if err != nil {
		t.Fatalf("Failed to create complex schema: %v", err)
	}

	var schemaMap map[string]any
	err = json.Unmarshal(schema, &schemaMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal complex schema: %v", err)
	}

	if schemaMap["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", schemaMap["type"])
	}

	properties, ok := schemaMap["properties"].(map[string]any)
	if !ok {
		t.Fatal("Properties is not a map")
	}

	// Verify all expected properties are present with correct types
	expectedProps := map[string]string{
		"text":   "string",
		"number": "number",
		"flag":   "boolean",
		"items":  "array",
		"meta":   "object",
	}

	for propName, expectedType := range expectedProps {
		prop, exists := properties[propName].(map[string]any)
		if !exists {
			t.Errorf("Property '%s' not found in schema", propName)
			continue
		}

		if prop["type"] != expectedType {
			t.Errorf("Property '%s' expected type '%s', got %v", propName, expectedType, prop["type"])
		}
	}
}

// Test RegisterTypedTool with nil server
func TestRegisterTypedTool_NilServer(t *testing.T) {
	err := RegisterTypedTool[AddInput, AddOutput](nil, "add", "Add two numbers", func(ctx context.Context, input AddInput) (AddOutput, error) {
		return AddOutput{Sum: input.A + input.B}, nil
	})

	if err == nil {
		t.Fatal("Expected error when registering tool with nil server")
	}
}

// Helper function to check if a string contains a substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > 0 && len(substr) > 0 &&
			(str[:len(substr)] == substr ||
				str[len(str)-len(substr):] == substr ||
				containsSubstring(str, substr))))
}

func containsSubstring(str, substr string) bool {
	if len(substr) > len(str) {
		return false
	}
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
