package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tmc/mcp/exp/sdk2"
)

// This example demonstrates the stdlib-idiomatic design of sdk2
func main() {
	fmt.Println("🚀 SDK2 Idiomatic MCP API Demo")
	fmt.Println("=====================================")

	// Choose demo mode
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "server":
			runServerDemo()
		case "client":
			runClientDemo()
		case "helpers":
			runHelpersDemo()
		case "errors":
			runErrorsDemo()
		default:
			fmt.Println("Usage: go run main.go [server|client|helpers|errors]")
		}
		return
	}

	// Run all demos
	fmt.Println("\n📡 1. Ultra-Simple Server (http.HandleFunc style)")
	runServerDemo()

	fmt.Println("\n🖥️  2. Ultra-Simple Client (net.Dial style)")
	runClientDemo()

	fmt.Println("\n🔧 3. Helper Functions (stdlib convenience)")
	runHelpersDemo()

	fmt.Println("\n❗ 4. Error Handling (stdlib patterns)")
	runErrorsDemo()
}

func runServerDemo() {
	fmt.Println("   Setting up handlers with familiar patterns...")

	// Register handlers exactly like http.HandleFunc
	sdk2.HandleFunc("tools/list", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		tools := []sdk2.Tool{
			{
				Name:        "echo",
				Description: "Echoes back the input",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}`),
			},
			{
				Name:        "math",
				Description: "Performs basic math operations",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"operation":{"type":"string"},"a":{"type":"number"},"b":{"type":"number"}},"required":["operation","a","b"]}`),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"tools": tools})
	})

	sdk2.HandleFunc("tools/call", func(w sdk2.ResponseWriter, r *sdk2.Request) {
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

		case "math":
			op, _ := call.Arguments["operation"].(string)
			a, _ := call.Arguments["a"].(float64)
			b, _ := call.Arguments["b"].(float64)

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
					sdk2.Error(w, "Division by zero", sdk2.StatusBadRequest)
					return
				}
				result = a / b
			default:
				sdk2.Error(w, "Unknown operation", sdk2.StatusBadRequest)
				return
			}

			toolResult := &sdk2.ToolResult{
				Content: []sdk2.Content{
					sdk2.MustNewTextContent(fmt.Sprintf("%.2f %s %.2f = %.2f", a, op, b, result)),
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(sdk2.StatusOK)
			json.NewEncoder(w).Encode(toolResult)

		default:
			sdk2.NotFound(w, r)
		}
	})

	fmt.Println("   ✅ Handlers registered with http.HandleFunc patterns")
	fmt.Println("   📝 Ready to serve with: sdk2.ListenAndServe(\":stdio\")")

	// In a real server, you'd call:
	// log.Fatal(sdk2.ListenAndServe(":stdio"))
}

func runClientDemo() {
	fmt.Println("   Creating client with stdlib patterns...")

	ctx := context.Background()

	// Simple dial (like net.Dial)
	fmt.Println("   📞 Simple dial: sdk2.Dial(ctx, \"stdio\", \"\")")

	// Advanced dial with options (like grpc.Dial)
	fmt.Println("   🔧 Advanced dial with options:")
	fmt.Println("      client, err := sdk2.DialConfig(ctx, \"stdio\", \"\",")
	fmt.Println("          sdk2.WithTimeout(30*time.Second),")
	fmt.Println("          sdk2.WithRetries(3, time.Second),")
	fmt.Println("          sdk2.WithClientInfo(\"demo-client\", \"1.0.0\"),")
	fmt.Println("      )")

	// Must functions (like template.Must)
	fmt.Println("   💥 Must functions: sdk2.MustDial(ctx, \"stdio\", \"\")")

	// Convenience functions
	fmt.Println("   🎯 Convenience: sdk2.DialStdio(ctx)")

	// High-level operations (like sql.DB)
	fmt.Println("   📋 High-level operations:")
	fmt.Println("      tools, err := client.ListTools(ctx)")
	fmt.Println("      result, err := client.CallTool(ctx, \"echo\", args)")
	fmt.Println("      err := client.Ping(ctx)  // like database/sql")

	fmt.Println("   ✅ Client patterns mirror net, http, and database/sql")
}

func runHelpersDemo() {
	fmt.Println("   Package-level helpers using DefaultClient...")

	ctx := context.Background()

	// Package-level functions (like http.Get)
	fmt.Println("   🌐 Package-level functions (like http.Get):")
	fmt.Println("      tools, err := sdk2.ListTools(ctx)")
	fmt.Println("      result, err := sdk2.CallTool(ctx, \"echo\", args)")
	fmt.Println("      err := sdk2.Ping(ctx)")

	// Error helpers (like fmt.Errorf)
	fmt.Println("   ❗ Error helpers (like fmt.Errorf):")
	fmt.Println("      err := sdk2.Errorf(\"something failed: %s\", reason)")
	fmt.Println("      err := sdk2.TimeoutErrorf(\"dial\", \"30s\", \"connection failed\")")

	// Parser helpers (like url.Parse)
	fmt.Println("   🔍 Parser helpers (like url.Parse):")
	fmt.Println("      tool, err := sdk2.ParseTool(jsonData)")
	fmt.Println("      tool := sdk2.MustParseTool(jsonData)  // panics on error")

	// Content helpers (sealed interface)
	fmt.Println("   📝 Content helpers (type-safe, validated):")
	fmt.Println("      text := sdk2.MustNewTextContent(\"Hello, world!\")")
	fmt.Println("      image := sdk2.MustNewImageContent(data, \"image/png\")")

	fmt.Println("   ✅ All helpers follow stdlib naming and behavior patterns")
}

func runErrorsDemo() {
	fmt.Println("   Error handling with stdlib patterns...")

	// Common error variables (like os.ErrNotExist, io.EOF)
	fmt.Println("   📋 Common error variables (like os.ErrNotExist):")
	fmt.Println("      sdk2.ErrTimeout, sdk2.ErrClosed, sdk2.ErrHandshake")
	fmt.Println("      sdk2.ErrToolNotFound, sdk2.ErrBadRequest, sdk2.ErrInvalid")

	// Error wrapping (Go 1.13+ patterns)
	fmt.Println("   🔗 Error wrapping (Go 1.13+ patterns):")
	fmt.Println("      if errors.Is(err, sdk2.ErrTimeout) { /* handle timeout */ }")
	fmt.Println("      var mcpErr *sdk2.MCPError")
	fmt.Println("      if errors.As(err, &mcpErr) { /* handle MCP error */ }")

	// Error checking functions
	fmt.Println("   ✅ Error checking functions:")
	fmt.Println("      if sdk2.IsTimeout(err) { /* retry */ }")
	fmt.Println("      if sdk2.IsRetryable(err) { /* retry */ }")

	// Error construction
	fmt.Println("   🏗️  Error construction:")
	fmt.Println("      err := sdk2.Errorf(\"operation failed: %s\", reason)")
	fmt.Println("      err := sdk2.NewConnError(\"dial\", \"tcp\", \"localhost:3000\", cause)")

	fmt.Println("   ✅ All error patterns mirror stdlib (errors, fmt, net)")
}
