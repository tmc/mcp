package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func main() {
	server := mcp.NewServer(mcp.ServerOptions{
		Name:        "json-server",
		Version:     "1.0.0",
		Description: "A server for JSON manipulation and validation operations",
	})

	// Add JSON validation tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "validate_json",
		Description: "Validates if a string is valid JSON",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json_string": map[string]interface{}{
					"type":        "string",
					"description": "The JSON string to validate",
				},
			},
			"required": []string{"json_string"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		jsonString := args["json_string"].(string)

		var result interface{}
		err := json.Unmarshal([]byte(jsonString), &result)

		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Invalid JSON: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Valid JSON",
				},
			},
		}, nil
	})

	// Add JSON formatting tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "format_json",
		Description: "Formats JSON with proper indentation",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json_string": map[string]interface{}{
					"type":        "string",
					"description": "The JSON string to format",
				},
				"indent": map[string]interface{}{
					"type":        "integer",
					"description": "Number of spaces for indentation (default: 2)",
					"default":     2,
				},
			},
			"required": []string{"json_string"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		jsonString := args["json_string"].(string)

		indent := 2
		if indentVal, ok := args["indent"]; ok {
			if indentFloat, ok := indentVal.(float64); ok {
				indent = int(indentFloat)
			}
		}

		var jsonObj interface{}
		err := json.Unmarshal([]byte(jsonString), &jsonObj)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Invalid JSON: %v", err),
					},
				},
			}, nil
		}

		indentStr := strings.Repeat(" ", indent)
		formatted, err := json.MarshalIndent(jsonObj, "", indentStr)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error formatting JSON: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(formatted),
				},
			},
		}, nil
	})

	// Add JSON minification tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "minify_json",
		Description: "Removes whitespace from JSON to minimize size",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json_string": map[string]interface{}{
					"type":        "string",
					"description": "The JSON string to minify",
				},
			},
			"required": []string{"json_string"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		jsonString := args["json_string"].(string)

		var jsonObj interface{}
		err := json.Unmarshal([]byte(jsonString), &jsonObj)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Invalid JSON: %v", err),
					},
				},
			}, nil
		}

		minified, err := json.Marshal(jsonObj)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error minifying JSON: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(minified),
				},
			},
		}, nil
	})

	// Add JSON path extraction tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "extract_json_path",
		Description: "Extracts value at specified JSON path",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json_string": map[string]interface{}{
					"type":        "string",
					"description": "The JSON string to extract from",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "JSON path (e.g., 'data.items[0].name')",
				},
			},
			"required": []string{"json_string", "path"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		jsonString := args["json_string"].(string)
		path := args["path"].(string)

		var jsonObj interface{}
		err := json.Unmarshal([]byte(jsonString), &jsonObj)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Invalid JSON: %v", err),
					},
				},
			}, nil
		}

		// Simple path extraction (supports dot notation and array indices)
		value, err := extractJSONPath(jsonObj, path)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Path extraction error: %v", err),
					},
				},
			}, nil
		}

		result, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error serializing result: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(result),
				},
			},
		}, nil
	})

	if err := server.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// extractJSONPath extracts a value from JSON object using dot notation
func extractJSONPath(obj interface{}, path string) (interface{}, error) {
	if path == "" {
		return obj, nil
	}

	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		// Handle array indices like "items[0]"
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			arrayPart := part[:strings.Index(part, "[")]
			indexPart := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]

			// Get the array field
			if arrayPart != "" {
				if m, ok := current.(map[string]interface{}); ok {
					current = m[arrayPart]
				} else {
					return nil, fmt.Errorf("expected object at path part: %s", arrayPart)
				}
			}

			// Get the array index
			if arr, ok := current.([]interface{}); ok {
				index, err := strconv.Atoi(indexPart)
				if err != nil {
					return nil, fmt.Errorf("invalid array index: %s", indexPart)
				}
				if index < 0 || index >= len(arr) {
					return nil, fmt.Errorf("array index out of bounds: %d", index)
				}
				current = arr[index]
			} else {
				return nil, fmt.Errorf("expected array at path part: %s", part)
			}
		} else {
			// Handle regular object field access
			if m, ok := current.(map[string]interface{}); ok {
				var exists bool
				current, exists = m[part]
				if !exists {
					return nil, fmt.Errorf("field not found: %s", part)
				}
			} else {
				return nil, fmt.Errorf("expected object at path part: %s", part)
			}
		}
	}

	return current, nil
}
