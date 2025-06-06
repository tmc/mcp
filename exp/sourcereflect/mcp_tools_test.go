package sourcereflect_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/sourcereflect"
)

func TestToMCPTool(t *testing.T) {
	// Test function
	testFunc := func(name string, age int, tags []string) error {
		return nil
	}

	funcType := reflect.TypeOf(testFunc)
	tool, err := sourcereflect.ToMCPTool("testFunc", funcType)
	if err != nil {
		t.Fatalf("Failed to create MCP tool: %v", err)
	}

	// Validate tool structure
	if tool.Name != "testFunc" {
		t.Errorf("Expected name 'testFunc', got '%s'", tool.Name)
	}

	if tool.InputSchema == nil {
		t.Fatal("InputSchema should not be nil")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("Expected input schema type 'object', got '%s'", tool.InputSchema.Type)
	}

	// Check parameters
	if len(tool.InputSchema.Properties) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(tool.InputSchema.Properties))
	}

	if len(tool.InputSchema.Required) != 3 {
		t.Errorf("Expected 3 required fields, got %d", len(tool.InputSchema.Required))
	}
}

func TestMCPToolJSON(t *testing.T) {
	testFunc := func(name string, active bool) {}

	funcType := reflect.TypeOf(testFunc)
	tool, err := sourcereflect.ToMCPTool("exampleFunc", funcType)
	if err != nil {
		t.Fatalf("Failed to create MCP tool: %v", err)
	}

	// Set some hints
	readOnly := true
	openWorld := false
	tool.Hints = &sourcereflect.MCPToolHints{
		ReadOnlyHint:  &readOnly,
		OpenWorldHint: &openWorld,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(tool, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	jsonStr := string(jsonData)

	// Check JSON structure
	if !contains(jsonStr, `"name": "exampleFunc"`) {
		t.Error("JSON should contain the function name")
	}

	if !contains(jsonStr, `"readOnlyHint": true`) {
		t.Error("JSON should contain readOnlyHint")
	}

	if !contains(jsonStr, `"openWorldHint": false`) {
		t.Error("JSON should contain openWorldHint")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}
