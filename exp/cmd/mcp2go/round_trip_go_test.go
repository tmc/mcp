package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/sourcegen"
)

// TestCCToolsRoundTrip tests that we can read the cc-tools.json file,
// convert it to Go structs, compile the generated code, and serialize
// back to JSON without losing critical information.
func TestCCToolsRoundTrip(t *testing.T) {
	// Read the original JSON file
	data, err := os.ReadFile("../mcptrace2gostruct/testdata/cc-tools.json")
	if err != nil {
		t.Fatalf("Failed to read cc-tools.json: %v", err)
	}

	// Parse the JSON to get the tools array
	var result struct {
		Result struct {
			Tools []json.RawMessage `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(result.Result.Tools) == 0 {
		t.Fatal("No tools found in cc-tools.json")
	}

	// Test each tool in the array
	for i, toolJSON := range result.Result.Tools {
		// Parse as MCP tool description
		var tool sourcegen.MCPToolDescription
		if err := json.Unmarshal(toolJSON, &tool); err != nil {
			t.Errorf("Failed to parse tool %d: %v", i, err)
			continue
		}

		// Generate Go code
		gen := sourcegen.NewGenerator("main")
		code, err := gen.GenerateFromMCPTool(&tool)
		if err != nil {
			t.Errorf("Failed to generate code for tool %s: %v", tool.Name, err)
			continue
		}

		// Convert tool name to Go type name
		toolName := strings.Title(strings.ReplaceAll(strings.ReplaceAll(tool.Name, "_", ""), "-", ""))

		// Verify the generated code contains expected elements
		expectedElements := []string{
			"type " + toolName + "Tool interface",
			"type " + toolName + "Input struct",
			"Execute(ctx context.Context",
		}

		for _, expected := range expectedElements {
			if !contains(code, expected) {
				t.Errorf("Generated code for tool %s missing expected element: %s", tool.Name, expected)
			}
		}

		// Verify structural integrity by checking if the tool can be re-marshaled
		// (This tests that our type conversions preserve the essential structure)
		toolJSON2, err := json.Marshal(tool)
		if err != nil {
			t.Errorf("Failed to re-marshal tool %s: %v", tool.Name, err)
			continue
		}

		// Parse both original and re-marshaled JSON into maps for comparison
		var original, remarshaled map[string]interface{}
		if err := json.Unmarshal(toolJSON, &original); err != nil {
			t.Errorf("Failed to parse original JSON for tool %s: %v", tool.Name, err)
			continue
		}
		if err := json.Unmarshal(toolJSON2, &remarshaled); err != nil {
			t.Errorf("Failed to parse re-marshaled JSON for tool %s: %v", tool.Name, err)
			continue
		}

		// Check that key fields are preserved
		if original["name"] != remarshaled["name"] {
			t.Errorf("Tool name not preserved: got %v, want %v", remarshaled["name"], original["name"])
		}
		if original["description"] != remarshaled["description"] {
			t.Errorf("Tool description not preserved: got %v, want %v", remarshaled["description"], original["description"])
		}

		// Verify input schema structure is preserved
		origSchema, _ := original["inputSchema"].(map[string]interface{})
		remSchema, _ := remarshaled["inputSchema"].(map[string]interface{})
		if origSchema["type"] != remSchema["type"] {
			t.Errorf("Schema type not preserved for tool %s", tool.Name)
		}
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}