package main

import (
	"encoding/json"
	"testing"

	"github.com/tmc/mcp/exp/sourcegen"
)

func TestQuickCCTools(t *testing.T) {
	// Read just a sample tool
	data := []byte(`{
		"name": "Task",
		"description": "Launch a new task",
		"inputSchema": {
			"type": "object",
			"properties": {
				"description": {
					"type": "string",
					"description": "A short (3-5 word) description of the task"
				},
				"prompt": {
					"type": "string",
					"description": "The task for the agent to perform"
				}
			},
			"required": ["description", "prompt"],
			"additionalProperties": false,
			"$schema": "http://json-schema.org/draft-07/schema#"
		}
	}`)

	var tool sourcegen.MCPToolDescription
	if err := json.Unmarshal(data, &tool); err != nil {
		t.Fatalf("Failed to parse tool: %v", err)
	}

	gen := sourcegen.NewGenerator("main")
	code, err := gen.GenerateFromMCPTool(&tool)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	t.Logf("Generated code:\n%s", code)
}
