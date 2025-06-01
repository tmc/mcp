// Package main demonstrates using the stdlib-idiomatic SDK2 client API.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tmc/mcp/exp/sdk2"
)

func main() {
	ctx := context.Background()

	// Connect to an MCP server using stdio transport
	client, err := sdk2.Dial(ctx, "stdio", "")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Alternative: Connect via TCP
	// client, err := sdk2.Dial(ctx, "tcp", "localhost:3000")

	log.Println("Connected to MCP server")

	// List available tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("Available tools (%d):\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("  %d. %s - %s\n", i+1, tool.Name, tool.Description)
	}

	// Call the echo tool if available
	for _, tool := range tools {
		if tool.Name == "echo" {
			fmt.Println("\nCalling echo tool...")
			result, err := client.CallTool(ctx, "echo", map[string]any{
				"message": "Hello from SDK2 client!",
			})
			if err != nil {
				log.Fatalf("Failed to call echo tool: %v", err)
			}

			fmt.Printf("Tool result (error: %v):\n", result.IsError)
			for i, content := range result.Content {
				if textContent, ok := content.(sdk2.TextContent); ok {
					fmt.Printf("  Content %d: %s\n", i+1, textContent.Text)
				} else {
					fmt.Printf("  Content %d: %s (%T)\n", i+1, content.ContentType(), content)
				}
			}
			break
		}
	}

	// List available resources
	resources, err := client.ListResources(ctx)
	if err != nil {
		log.Printf("Failed to list resources: %v", err)
	} else {
		fmt.Printf("\nAvailable resources (%d):\n", len(resources))
		for i, resource := range resources {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, resource.Name, resource.URI, resource.Description)
		}

		// Read the first resource if available
		if len(resources) > 0 {
			fmt.Printf("\nReading resource: %s\n", resources[0].URI)
			content, err := client.ReadResource(ctx, resources[0].URI)
			if err != nil {
				log.Printf("Failed to read resource: %v", err)
			} else {
				fmt.Printf("Resource content (MIME: %s):\n", content.MimeType)
				for i, c := range content.Content {
					if textContent, ok := c.(sdk2.TextContent); ok {
						fmt.Printf("  Content %d: %s\n", i+1, textContent.Text)
					} else {
						fmt.Printf("  Content %d: %s (%T)\n", i+1, c.ContentType(), c)
					}
				}
			}
		}
	}

	// List available prompts
	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		log.Printf("Failed to list prompts: %v", err)
	} else {
		fmt.Printf("\nAvailable prompts (%d):\n", len(prompts))
		for i, prompt := range prompts {
			fmt.Printf("  %d. %s - %s\n", i+1, prompt.Name, prompt.Description)
		}
	}

	fmt.Println("\nDemo completed successfully!")
}

// Example of using the client with custom configuration
func advancedClientExample() {
	ctx := context.Background()

	// Create a client with custom configuration
	config := &sdk2.ClientConfig{
		Timeout:    10 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
		ClientInfo: sdk2.ClientInfo{
			Name:    "my-mcp-client",
			Version: "1.0.0",
		},
		NotificationHandler: sdk2.NotificationHandlerFunc(func(ctx context.Context, method string, params []byte) error {
			fmt.Printf("Notification: %s -> %s\n", method, string(params))
			return nil
		}),
	}

	client, err := sdk2.DialContext(ctx, "tcp", "localhost:3000", config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Use the client...
	tools, err := client.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("Found %d tools\n", len(tools))
}