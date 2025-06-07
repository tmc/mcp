package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tmc/mcp/exp/sdk2"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "server" {
		runServer()
	} else {
		runClient()
	}
}

// runServer demonstrates the ultra-simple server using stdlib patterns
func runServer() {
	fmt.Println("🚀 Ultra-Simple MCP Server (stdlib patterns)")
	fmt.Println("===========================================")

	// Register handlers exactly like http.HandleFunc
	sdk2.HandleFunc("tools/list", listTools)
	sdk2.HandleFunc("tools/call", callTool)

	fmt.Println("✅ Handlers registered")
	fmt.Println("📡 Starting server on stdio...")

	// Start server exactly like http.ListenAndServe
	log.Fatal(sdk2.ListenAndServe(":stdio"))
}

func listTools(w sdk2.ResponseWriter, r *sdk2.Request) {
	tools := []sdk2.Tool{
		sdk2.MustNewTool("echo", "Echoes back your message", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
			},
			"required": []string{"message"},
		}),
		sdk2.MustNewTool("greet", "Greets a person", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
			"required": []string{"name"},
		}),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sdk2.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"tools": tools})
}

func callTool(w sdk2.ResponseWriter, r *sdk2.Request) {
	var call sdk2.ToolCall
	if err := json.Unmarshal(r.Params, &call); err != nil {
		sdk2.Error(w, "Invalid parameters", sdk2.StatusBadRequest)
		return
	}

	switch call.Name {
	case "echo":
		message, ok := call.Arguments["message"].(string)
		if !ok {
			sdk2.Error(w, "Missing message parameter", sdk2.StatusBadRequest)
			return
		}

		result := &sdk2.ToolResult{
			Content: []sdk2.Content{
				sdk2.MustNewTextContent("Echo: " + message),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(result)

	case "greet":
		name, ok := call.Arguments["name"].(string)
		if !ok {
			sdk2.Error(w, "Missing name parameter", sdk2.StatusBadRequest)
			return
		}

		result := &sdk2.ToolResult{
			Content: []sdk2.Content{
				sdk2.MustNewTextContent(fmt.Sprintf("Hello, %s! Nice to meet you.", name)),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(result)

	default:
		sdk2.NotFound(w, r)
	}
}

// runClient demonstrates the ultra-simple client using stdlib patterns
func runClient() {
	fmt.Println("🖥️  Ultra-Simple MCP Client (stdlib patterns)")
	fmt.Println("=============================================")

	ctx := context.Background()

	// Method 1: Simple dial (like net.Dial)
	fmt.Println("📞 Method 1: Simple dial (like net.Dial)")
	client, err := sdk2.Dial(ctx, "stdio", "")
	if err != nil {
		fmt.Printf("❌ Dial failed: %v\n", err)
		return
	}
	defer client.Close()
	fmt.Println("✅ Connected via simple dial")

	// Method 2: Must dial (like template.Must)
	fmt.Println("\n💥 Method 2: Must dial (like template.Must)")
	// client2 := sdk2.MustDial(ctx, "stdio", "")
	fmt.Println("✅ Would connect with MustDial (panics on error)")

	// Method 3: Dial with config (like grpc.Dial)
	fmt.Println("\n🔧 Method 3: Dial with config (like grpc.Dial)")
	client3, err := sdk2.DialConfig(ctx, "stdio", "",
		sdk2.WithTimeout(30*time.Second),
		sdk2.WithClientInfo("demo-client", "1.0.0"),
	)
	if err != nil {
		fmt.Printf("❌ DialConfig failed: %v\n", err)
	} else {
		defer client3.Close()
		fmt.Println("✅ Connected with advanced config")
	}

	// Method 4: Client operations (database/sql style)
	fmt.Println("\n🌐 Method 4: Client operations (database/sql style)")

	// List tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		fmt.Printf("❌ ListTools failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Found %d tools\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("   - %s: %s\n", tool.Name, tool.Description)
	}

	// Call tools
	result, err := client.CallTool(ctx, "echo", map[string]any{
		"message": "Hello from stdlib-style client!",
	})
	if err != nil {
		fmt.Printf("❌ CallTool failed: %v\n", err)
		return
	}
	if len(result.Content) > 0 {
		fmt.Printf("✅ Tool result: %+v\n", result.Content[0])
	}

	// Test connectivity (like database/sql Ping)
	if err := client.Ping(ctx); err != nil {
		fmt.Printf("❌ Ping failed: %v\n", err)
	} else {
		fmt.Println("✅ Ping successful")
	}

	// Demonstrate error handling
	fmt.Println("\n❗ Error handling demo:")
	_, err = client.CallTool(ctx, "nonexistent", map[string]any{})
	if err != nil {
		fmt.Printf("✅ Expected error: %v\n", err)

		// Check error types (stdlib patterns)
		if sdk2.IsRetryable(err) {
			fmt.Println("   This error is retryable")
		}
		if sdk2.IsTimeout(err) {
			fmt.Println("   This error is a timeout")
		}
	}

	fmt.Println("\n🎉 Demo completed successfully!")
}
