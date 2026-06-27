package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/exp/adapters/golang_tools"
	protocol "github.com/tmc/mcp/modelcontextprotocol"
	"github.com/tmc/mcp/server"
	"github.com/tmc/mcp/transport"
)

func main() {
	// Create a standard SDK server with tools and prompts
	sdkServer := server.NewServer(
		server.WithName("golang-tools-test-server"),
		server.WithVersion("1.0.0"),
		server.WithInstructions("Test server demonstrating golang-tools adapter"),
	)

	// Add test tools to the SDK server
	sdkServer.AddTool(
		"echo",
		"Echoes back the input message",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string",
					"description": "Message to echo"
				}
			},
			"required": ["message"]
		}`),
		echoToolHandler,
	)

	sdkServer.AddTool(
		"calculate",
		"Performs basic arithmetic operations",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["add", "subtract", "multiply", "divide"],
					"description": "The operation to perform"
				},
				"a": {
					"type": "number",
					"description": "First operand"
				},
				"b": {
					"type": "number",
					"description": "Second operand"
				}
			},
			"required": ["operation", "a", "b"]
		}`),
		calculateToolHandler,
	)

	// Add test prompt
	sdkServer.AddPrompt(
		"test-prompt",
		"A test prompt that generates a response based on topic",
		[]server.PromptArgument{
			{
				Name:        "topic",
				Description: "The topic to generate content about",
				Required:    true,
			},
		},
		testPromptHandler,
	)

	// Create golang-tools adapter
	adapter := golang_tools.NewAdapter()

	// Initialize adapter with the SDK server
	ctx := context.Background()
	if err := adapter.Initialize(ctx, sdkServer); err != nil {
		log.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Create server with adapter
	adapterServer := server.NewServer(
		server.WithName("golang-tools-adapted"),
		server.WithVersion("1.0.0"),
		server.WithAdapter(adapter),
	)

	// Create transport
	var t mcp.Transport
	if len(os.Args) > 1 && os.Args[1] == "--stdio" {
		t = transport.NewStdIOTransport()
	} else {
		log.Fatal("Please specify --stdio flag")
	}

	// Serve using the adapter
	log.Printf("Starting golang-tools test server via adapter...")
	if err := adapterServer.ServeTransport(ctx, t); err != nil {
		log.Fatal(err)
	}
}

// Tool handlers
func echoToolHandler(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, err
	}

	message, ok := args["message"].(string)
	if !ok {
		return protocol.CallToolResult{
			Content: []protocol.Content{
				protocol.TextContent{
					Type: "text",
					Text: "Error: message parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	return protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Echo: %s", message),
			},
		},
	}, nil
}

func calculateToolHandler(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, err
	}

	op, _ := args["operation"].(string)
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)

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
			return protocol.CallToolResult{
				Content: []protocol.Content{
					protocol.TextContent{Type: "text", Text: "Error: division by zero"},
				},
				IsError: true,
			}, nil
		}
		result = a / b
	default:
		return protocol.CallToolResult{
			Content: []protocol.Content{
				protocol.TextContent{Type: "text", Text: fmt.Sprintf("Unknown operation: %s", op)},
			},
			IsError: true,
		}, nil
	}

	return protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Result: %v", result),
			},
		},
	}, nil
}

// Prompt handler
func testPromptHandler(ctx context.Context, name string, arguments map[string]string) (interface{}, error) {
	topic := arguments["topic"]

	return protocol.GetPromptResult{
		Description: fmt.Sprintf("Generated prompt about %s", topic),
		Messages: []protocol.PromptMessage{
			{
				Role: protocol.RoleUser,
				Content: protocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Tell me about %s", topic),
				},
			},
			{
				Role: protocol.RoleAssistant,
				Content: protocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Here's information about %s: This is a test implementation using the golang-tools adapter.", topic),
				},
			},
		},
	}, nil
}
