package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/mcpspec"
	"github.com/tmc/mcp/exp/sourcegen"
)

func TestQuick(t *testing.T) {
	// Read just a sample tool
	toolJSON := `{
		"name": "Task",
		"description": "Launch a new task",
		"inputSchema": {
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "Command to run"
				},
				"args": {
					"type": "array",
					"items": { "type": "string" },
					"description": "Command arguments"
				}
			},
			"required": ["command"],
			"additionalProperties": false,
			"$schema": "http://json-schema.org/draft-07/schema#"
		}
	}`

	// Parse tool
	var tool mcpspec.ToolDefinition
	if err := json.Unmarshal([]byte(toolJSON), &tool); err != nil {
		t.Fatalf("Failed to parse tool JSON: %v", err)
	}

	// Generate code
	gen := sourcegen.NewGenerator("generated")
	output, err := gen.GenerateFromTool(&tool)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	if !strings.Contains(output, "type TaskTool interface") {
		t.Error("Generated code missing interface definition")
	}
	t.Logf("Generated code:\n%s", output)
}
