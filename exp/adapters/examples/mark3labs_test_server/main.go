package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpSDK "github.com/tmc/mcp"
	"github.com/tmc/mcp/exp/adapters/mark3labs"
	"github.com/tmc/mcp/transport"
)

func main() {
	// Create the adapter to bridge mark3labs to SDK
	adapter := mark3labs.NewAdapter()

	// Create an SDK server that will use the adapter
	sdkServer := mcpSDK.NewServer(
		mcpSDK.WithName("mark3labs-test-server"),
		mcpSDK.WithVersion("1.0.0"),
		mcpSDK.WithAdapter(adapter),
	)

	// Initialize adapter with the SDK server
	ctx := context.Background()
	if err := adapter.Initialize(ctx, sdkServer); err != nil {
		log.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Register mark3labs components directly with the adapter
	registerMark3LabsComponents(adapter.(*mark3labs.Mark3LabsAdapter))

	// Create transport
	var t mcpSDK.Transport
	if len(os.Args) > 1 && os.Args[1] == "--stdio" {
		t = transport.NewStdIOTransport()
	} else {
		log.Fatal("Please specify --stdio flag")
	}

	// Serve
	log.Printf("Starting mark3labs test server via adapter...")
	if err := sdkServer.ServeTransport(ctx, t); err != nil {
		log.Fatal(err)
	}
}

// registerMark3LabsComponents registers mark3labs-style tools, resources, and prompts
func registerMark3LabsComponents(adapter *mark3labs.Mark3LabsAdapter) {
	// Register echo tool
	adapter.RegisterTool(
		mcp.Tool{
			Name:        "echo",
			Description: "Echoes back the input message",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message to echo",
					},
				},
				"required": []string{"message"},
			},
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			message, ok := req.Params.Arguments["message"].(string)
			if !ok {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: message parameter is required",
						},
					},
					IsError: true,
				}, nil
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Echo: %s", message),
					},
				},
			}, nil
		},
	)

	// Register calculate tool
	adapter.RegisterTool(
		mcp.Tool{
			Name:        "calculate",
			Description: "Performs basic arithmetic operations",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"add", "subtract", "multiply", "divide"},
						"description": "The operation to perform",
					},
					"a": map[string]interface{}{
						"type":        "number",
						"description": "First operand",
					},
					"b": map[string]interface{}{
						"type":        "number",
						"description": "Second operand",
					},
				},
				"required": []string{"operation", "a", "b"},
			},
		},
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			op, _ := req.Params.Arguments["operation"].(string)
			a, _ := req.Params.Arguments["a"].(float64)
			b, _ := req.Params.Arguments["b"].(float64)

			var result float64
			switch op {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{Type: "text", Text: "Error: division by zero"},
						},
						IsError: true,
					}, nil
				}
				result = a / b
			default:
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{Type: "text", Text: fmt.Sprintf("Unknown operation: %s", op)},
					},
					IsError: true,
				}, nil
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Result: %v", result),
					},
				},
			}, nil
		},
	)

	// Register test resource
	adapter.RegisterResource(
		mcp.Resource{
			URI:         "test://config",
			Name:        "Test Configuration",
			Description: "Returns test configuration data",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			config := map[string]interface{}{
				"version":   "1.0.0",
				"adapter":   "mark3labs",
				"timestamp": time.Now().Format(time.RFC3339),
			}

			jsonData, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return nil, err
			}

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "test://config",
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			}, nil
		},
	)

	// Register test prompt
	adapter.RegisterPrompt(
		mcp.Prompt{
			Name:        "test-prompt",
			Description: "A test prompt that generates a response based on topic",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "topic",
					Description: "The topic to generate content about",
					Required:    true,
				},
			},
		},
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			topic := req.Params.Arguments["topic"]

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Generated prompt about %s", topic),
				Messages: []mcp.PromptMessage{
					{
						Role: mcp.RoleUser,
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Tell me about %s", topic),
						},
					},
					{
						Role: mcp.RoleAssistant,
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Here's information about %s: This is a test implementation using the mark3labs adapter.", topic),
						},
					},
				},
			}, nil
		},
	)
}
