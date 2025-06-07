// Package main demonstrates a simple calculator server using SDK2.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/tmc/mcp/exp/sdk2"
	"github.com/tmc/mcp/exp/sdk2/transport"
)

// CalculatorTool implements a basic calculator.
type CalculatorTool struct{}

// Handle executes calculator operations.
func (c *CalculatorTool) Handle(ctx context.Context, arguments map[string]any) (*sdk2.ToolResult, error) {
	operation, ok := arguments["operation"].(string)
	if !ok {
		return &sdk2.ToolResult{
			IsError: true,
			Content: []sdk2.Content{
				sdk2.TextContent{Text: "operation parameter is required and must be a string"},
			},
		}, nil
	}

	a, ok := arguments["a"].(float64)
	if !ok {
		return &sdk2.ToolResult{
			IsError: true,
			Content: []sdk2.Content{
				sdk2.TextContent{Text: "parameter 'a' is required and must be a number"},
			},
		}, nil
	}

	b, ok := arguments["b"].(float64)
	if !ok {
		return &sdk2.ToolResult{
			IsError: true,
			Content: []sdk2.Content{
				sdk2.TextContent{Text: "parameter 'b' is required and must be a number"},
			},
		}, nil
	}

	var result float64
	var err error

	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return &sdk2.ToolResult{
				IsError: true,
				Content: []sdk2.Content{
					sdk2.TextContent{Text: "division by zero is not allowed"},
				},
			}, nil
		}
		result = a / b
	default:
		return &sdk2.ToolResult{
			IsError: true,
			Content: []sdk2.Content{
				sdk2.TextContent{Text: fmt.Sprintf("unknown operation: %s", operation)},
			},
		}, nil
	}

	return &sdk2.ToolResult{
		Content: []sdk2.Content{
			sdk2.TextContent{Text: fmt.Sprintf("%.2f", result)},
		},
	}, err
}

// Description returns the tool description.
func (c *CalculatorTool) Description() string {
	return "A simple calculator that can perform basic arithmetic operations"
}

// Schema returns the JSON schema for the tool parameters.
func (c *CalculatorTool) Schema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
				"description": "The arithmetic operation to perform",
			},
			"a": map[string]any{
				"type":        "number",
				"description": "The first number",
			},
			"b": map[string]any{
				"type":        "number",
				"description": "The second number",
			},
		},
		"required": []string{"operation", "a", "b"},
	}

	bytes, _ := json.Marshal(schema)
	return json.RawMessage(bytes)
}

func main() {
	// Create a new server
	server := sdk2.NewServer("calculator-server", "1.0.0")

	// Add the calculator tool
	server.AddTool("calculator", &CalculatorTool{})

	// Create stdio transport
	transport := transport.NewStdio()

	// Serve using the transport
	ctx := context.Background()
	if err := server.Serve(ctx, transport); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
