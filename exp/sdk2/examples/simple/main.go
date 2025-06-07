// Package main demonstrates a simple usage example of SDK2.
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tmc/mcp/exp/sdk2"
)

// This example shows the stdlib-idiomatic patterns in SDK2,
// demonstrating how it feels like a natural extension of Go's standard library.

func main() {
	fmt.Println("SDK2 Simple Example")
	fmt.Println("===================")

	// Create a server using stdlib patterns
	server := sdk2.NewServer()

	// Register handlers using http.HandleFunc-like patterns
	sdk2.HandleFunc("tools/list", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		tools := []sdk2.Tool{{
			Name:        "greeting",
			Description: "Generates a personalized greeting message",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "The name of the person to greet"}
				},
				"required": ["name"]
			}`),
		}}
		json.NewEncoder(w).Encode(map[string]any{"tools": tools})
	})

	sdk2.HandleFunc("tools/call", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		var call sdk2.ToolCall
		json.Unmarshal(r.Params, &call)

		if call.Name == "greeting" {
			name, ok := call.Arguments["name"].(string)
			if !ok {
				result := &sdk2.ToolResult{
					IsError: true,
					Content: []sdk2.Content{
						sdk2.TextContent{Text: "name parameter is required and must be a string"},
					},
				}
				json.NewEncoder(w).Encode(result)
				return
			}

			greeting := fmt.Sprintf("Hello, %s! Welcome to MCP SDK2.", name)
			result := &sdk2.ToolResult{
				Content: []sdk2.Content{
					sdk2.TextContent{Text: greeting},
				},
			}
			json.NewEncoder(w).Encode(result)
		}
	})

	fmt.Println("\n1. Server Configuration:")
	fmt.Printf("   Type: %T\n", server)
	fmt.Printf("   Handler: %T\n", server.Handler)

	fmt.Println("\n2. Content Types:")

	// Demonstrate different content types
	textContent := sdk2.TextContent{Text: "This is text content"}
	imageContent := sdk2.ImageContent{
		Data:     "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
		MimeType: "image/png",
	}

	textJSON, _ := json.Marshal(textContent)
	imageJSON, _ := json.Marshal(imageContent)

	fmt.Printf("   Text Content: %s\n", string(textJSON))
	fmt.Printf("   Image Content: %s\n", string(imageJSON))

	fmt.Println("\n3. Client Configuration:")

	// Demonstrate client options using functional pattern
	config := &sdk2.ClientConfig{}
	sdk2.WithTimeout(30 * time.Second)(config)
	sdk2.WithRetries(3, time.Second)(config)
	sdk2.WithClientInfo("demo-client", "1.0.0")(config)

	fmt.Printf("   Timeout: %v\n", config.Timeout)
	fmt.Printf("   Max Retries: %d\n", config.MaxRetries)
	fmt.Printf("   Retry Delay: %v\n", config.RetryDelay)
	fmt.Printf("   Client Name: %s\n", config.ClientInfo.Name)

	fmt.Println("\n4. Request ID Examples:")

	// Demonstrate flexible RequestID
	ids := []sdk2.RequestID{
		{Value: "string-id"},
		{Value: int64(123)},
		{Value: nil},
	}

	for _, id := range ids {
		data, _ := json.Marshal(id)
		fmt.Printf("   ID %s -> JSON: %s\n", id.String(), string(data))
	}

	fmt.Println("\nSDK2 demonstrates:")
	fmt.Println("- http.Server-like server patterns")
	fmt.Println("- net.Dial-like client patterns")
	fmt.Println("- http.Handler-like request handling")
	fmt.Println("- Strong typing with sealed interfaces")
	fmt.Println("- Functional options configuration")
	fmt.Println("- Type-safe content handling")
	fmt.Println("- Clean error handling")
	fmt.Println("- Extensible architecture")
}
