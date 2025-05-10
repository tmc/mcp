package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Support for creating typed tool handlers that automatically handle JSON serialization/deserialization.
// This allows for a more idiomatic Go API when registering tools.

// RegisterTypedTool registers a type-safe tool handler with automatic JSON marshaling/unmarshaling.
// Input is the Go type for the tool's input, and Output is the Go type for the tool's output.
func RegisterTypedTool[Input any, Output any](
	server *Server,
	name string,
	description string,
	handler func(context.Context, Input) (Output, error),
) error {
	// Create a JSON schema from the Input type if possible
	inputSchema, err := createJSONSchema[Input]()
	if err != nil {
		return fmt.Errorf("failed to create input schema: %w", err)
	}

	// Register the tool with the server
	toolHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		// Parse the input
		var input Input
		if err := json.Unmarshal(req.Arguments, &input); err != nil {
			return &CallToolResult{
				IsError: true,
				Content: []any{
					map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Invalid input: %v", err),
					},
				},
			}, nil
		}

		// Call the handler
		output, err := handler(ctx, input)
		if err != nil {
			return &CallToolResult{
				IsError: true,
				Content: []any{
					map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Error: %v", err),
					},
				},
			}, nil
		}

		// Convert the output to a generic map for the content
		outputJSON, err := json.Marshal(output)
		if err != nil {
			return &CallToolResult{
				IsError: true,
				Content: []any{
					map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Failed to marshal output: %v", err),
					},
				},
			}, nil
		}

		var outputMap map[string]any
		if err := json.Unmarshal(outputJSON, &outputMap); err != nil {
			// If it can't be unmarshaled as a map, use it as a text result
			return &CallToolResult{
				Content: []any{
					map[string]string{
						"type": "text",
						"text": string(outputJSON),
					},
				},
			}, nil
		}

		// Return the result
		return &CallToolResult{
			Content: []any{
				map[string]any{
					"type":   "text",
					"format": "json",
					"text":   string(outputJSON),
				},
			},
		}, nil
	}

	// Add the tool with its handler
	tool := Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
	return server.RegisterTool(tool, toolHandler)
}

// createJSONSchema generates a simple JSON schema representation for the given type.
// This is a basic implementation that could be enhanced in the future.
func createJSONSchema[T any]() (json.RawMessage, error) {
	// For now, just create a simple schema with example values
	var example T
	exampleJSON, err := json.Marshal(example)
	if err != nil {
		return nil, err
	}

	var exampleMap map[string]any
	if err := json.Unmarshal(exampleJSON, &exampleMap); err != nil {
		// If it's not a struct that can be represented as a map, return a simpler schema
		return json.Marshal(map[string]any{
			"type": "string", // Default to string, could be improved with reflection
		})
	}

	// Create a schema based on the example
	schema := map[string]any{
		"type": "object",
		"properties": func() map[string]any {
			props := make(map[string]any)
			for k, v := range exampleMap {
				propType := "string" // Default
				switch v.(type) {
				case float64:
					propType = "number"
				case bool:
					propType = "boolean"
				case map[string]any:
					propType = "object"
				case []any:
					propType = "array"
				}
				props[k] = map[string]any{"type": propType}
			}
			return props
		}(),
	}

	return json.Marshal(schema)
}
