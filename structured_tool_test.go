package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

type structuredAddArgs struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type structuredAddResult struct {
	Sum float64 `json:"sum"`
}

func TestRegisterTypedToolStructuredOutput(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	err := RegisterTypedToolWithServer(server, "structured_add", "Add numbers",
		func(ctx context.Context, args structuredAddArgs) (structuredAddResult, error) {
			return structuredAddResult{Sum: args.A + args.B}, nil
		})
	if err != nil {
		t.Fatalf("RegisterTypedToolWithServer() error = %v", err)
	}

	toolDef, ok := server.tools["structured_add"]
	if !ok {
		t.Fatal("tool was not registered")
	}
	if len(toolDef.tool.OutputSchema) == 0 {
		t.Fatal("OutputSchema is empty")
	}

	var outputSchema map[string]any
	if err := json.Unmarshal(toolDef.tool.OutputSchema, &outputSchema); err != nil {
		t.Fatalf("unmarshal output schema: %v", err)
	}
	if outputSchema["type"] != "object" {
		t.Fatalf("output schema type = %v, want object", outputSchema["type"])
	}
	properties, ok := outputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("output schema properties is not an object")
	}
	sum, ok := properties["sum"].(map[string]any)
	if !ok {
		t.Fatal("output schema is missing sum property")
	}
	if sum["type"] != "number" {
		t.Fatalf("sum schema type = %v, want number", sum["type"])
	}

	toolJSON, err := json.Marshal(toolDef.tool)
	if err != nil {
		t.Fatalf("marshal tool: %v", err)
	}
	var toolWire struct {
		OutputSchema map[string]any `json:"outputSchema"`
	}
	if err := json.Unmarshal(toolJSON, &toolWire); err != nil {
		t.Fatalf("unmarshal tool wire JSON: %v", err)
	}
	if toolWire.OutputSchema == nil {
		t.Fatal("tool wire JSON is missing outputSchema")
	}

	result, err := toolDef.handler(context.Background(), CallToolRequest{
		Name:      "structured_add",
		Arguments: json.RawMessage(`{"a":2,"b":3}`),
	})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result.StructuredContent == nil {
		t.Fatal("StructuredContent is nil")
	}

	var structured structuredAddResult
	structuredJSON, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if err := json.Unmarshal(structuredJSON, &structured); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	if structured.Sum != 5 {
		t.Fatalf("structured sum = %v, want 5", structured.Sum)
	}

	if len(result.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(result.Content))
	}
	content, ok := result.Content[0].(map[string]any)
	if !ok {
		t.Fatalf("Content[0] has type %T, want map[string]any", result.Content[0])
	}
	text, ok := content["text"].(string)
	if !ok {
		t.Fatalf("Content[0].text has type %T, want string", content["text"])
	}
	var compat structuredAddResult
	if err := json.Unmarshal([]byte(text), &compat); err != nil {
		t.Fatalf("unmarshal compatibility content: %v", err)
	}
	if compat != structured {
		t.Fatalf("compat content = %+v, want %+v", compat, structured)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var resultWire struct {
		StructuredContent map[string]float64 `json:"structuredContent"`
	}
	if err := json.Unmarshal(resultJSON, &resultWire); err != nil {
		t.Fatalf("unmarshal result wire JSON: %v", err)
	}
	if resultWire.StructuredContent["sum"] != 5 {
		t.Fatalf("wire structured sum = %v, want 5", resultWire.StructuredContent["sum"])
	}
}
