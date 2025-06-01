// Package main demonstrates the stdlib-idiomatic patterns in SDK2.
// This showcases how familiar the API feels to any Go developer.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/tmc/mcp/exp/sdk2"
	"github.com/tmc/mcp/exp/sdk2/transport"
)

func main() {
	// Show different patterns based on args
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "server":
			runServer()
		case "client":
			runClient()
		case "middleware":
			runMiddleware()
		case "typed":
			runTypedHandlers()
		case "demo":
			runDemo()
		default:
			fmt.Printf("Usage: %s [server|client|middleware|typed|demo]\n", os.Args[0])
		}
		return
	}
	
	fmt.Println("SDK2 Stdlib Showcase - Choose a demo:")
	fmt.Println("  go run main.go demo       - Interactive demonstration")
	fmt.Println("  go run main.go server     - http.Server-like patterns")
	fmt.Println("  go run main.go client     - net.Dial-like patterns")
	fmt.Println("  go run main.go middleware - http.Handler middleware")
	fmt.Println("  go run main.go typed      - Type-safe tool handlers")
}

// runDemo shows all the patterns without running actual servers
func runDemo() {
	fmt.Println("=== SDK2 STDLIB-IDIOMATIC API SHOWCASE ===")
	fmt.Println()

	demonstrateServerPatterns()
	demonstrateClientPatterns() 
	demonstrateHandlerPatterns()
	demonstrateFunctionalOptions()
	demonstrateTypeSafety()
	demonstrateTransportAbstraction()
}

// runServer demonstrates http.Server-like patterns
func runServer() {
	fmt.Println("=== HTTP.SERVER-LIKE PATTERNS ===")
	
	// Pattern 1: Simple function handlers (like http.HandleFunc)
	sdk2.HandleFunc("tools/list", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		tools := []sdk2.Tool{
			{
				Name:        "echo",
				Description: "Echoes back the input",
				InputSchema: echoSchema(),
			},
			{
				Name:        "math",
				Description: "Performs mathematical calculations",
				InputSchema: mathSchema(),
			},
		}
		
		result := map[string]any{"tools": tools}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(result)
	})
	
	// Pattern 2: Tool call handler
	sdk2.HandleFunc("tools/call", handleToolCall)
	
	// Pattern 3: Custom server configuration (like http.Server)
	server := &sdk2.Server{
		Addr:         ":stdio",             // Like http.Server.Addr
		Handler:      nil,                  // Uses DefaultServeMux like http
		ReadTimeout:  10 * time.Second,     // Like http.Server timeouts
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  2 * time.Minute,
	}
	
	slog.Info("Starting server", "addr", server.Addr)
	
	// Pattern 4: ListenAndServe (exactly like http.Server)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

// runClient demonstrates net.Dial-like patterns
func runClient() {
	fmt.Println("=== NET.DIAL-LIKE PATTERNS ===")
	
	ctx := context.Background()
	
	// Pattern 1: Simple dial (like net.Dial)
	fmt.Println("Simple dial pattern:")
	client, err := sdk2.Dial(ctx, "stdio", "")
	if err != nil {
		log.Fatal("Dial failed:", err)
	}
	defer client.Close()
	
	// Pattern 2: Dial with configuration (like tls.Dial, grpc.Dial)
	fmt.Println("Configured dial pattern:")
	client2, err := sdk2.DialConfig(ctx, "stdio", "",
		sdk2.WithTimeout(30*time.Second),
		sdk2.WithRetries(5, 2*time.Second),
		sdk2.WithClientInfo("showcase-client", "1.0.0"),
		sdk2.WithNotificationHandler(sdk2.NotificationHandlerFunc(
			func(ctx context.Context, method string, params json.RawMessage) error {
				slog.Info("Notification received", "method", method)
				return nil
			},
		)),
	)
	if err != nil {
		log.Fatal("DialConfig failed:", err)
	}
	defer client2.Close()
	
	// Pattern 3: High-level operations (like database/sql)
	fmt.Println("High-level operations:")
	tools, err := client.ListTools(ctx)
	if err != nil {
		log.Printf("ListTools failed: %v", err)
		return
	}
	
	fmt.Printf("Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	
	// Pattern 4: Tool execution
	result, err := client.CallTool(ctx, "echo", map[string]any{
		"message": "Hello from stdlib-idiomatic client!",
	})
	if err != nil {
		log.Printf("CallTool failed: %v", err)
		return
	}
	
	fmt.Printf("Tool result: %+v\n", result)
}

// runMiddleware demonstrates http.Handler middleware patterns
func runMiddleware() {
	fmt.Println("=== HTTP.HANDLER MIDDLEWARE PATTERNS ===")
	
	// Pattern 1: Logging middleware (like http middleware)
	loggingMiddleware := func(next sdk2.Handler) sdk2.Handler {
		return sdk2.HandlerFunc(func(w sdk2.ResponseWriter, r *sdk2.Request) {
			start := time.Now()
			slog.Info("Request started", "method", r.Method, "id", r.ID)
			
			next.ServeRequest(w, r)
			
			duration := time.Since(start)
			slog.Info("Request completed", "method", r.Method, "duration", duration)
		})
	}
	
	// Pattern 2: Authentication middleware
	authMiddleware := func(next sdk2.Handler) sdk2.Handler {
		return sdk2.HandlerFunc(func(w sdk2.ResponseWriter, r *sdk2.Request) {
			// In a real app, you'd check authentication here
			if r.Method != "initialize" && r.Method != "initialized" {
				// Add user info to context (like http.Request.Context)
				ctx := context.WithValue(r.Context, "user", "authenticated-user")
				r = r.WithContext(ctx)
			}
			
			next.ServeRequest(w, r)
		})
	}
	
	// Pattern 3: Recovery middleware
	recoveryMiddleware := func(next sdk2.Handler) sdk2.Handler {
		return sdk2.HandlerFunc(func(w sdk2.ResponseWriter, r *sdk2.Request) {
			defer func() {
				if err := recover(); err != nil {
					slog.Error("Handler panic", "error", err, "method", r.Method)
					sdk2.Error(w, "Internal server error", sdk2.StatusInternalServerError)
				}
			}()
			
			next.ServeRequest(w, r)
		})
	}
	
	// Pattern 4: Chain middleware (like http middleware)
	mux := sdk2.NewServeMux()
	mux.HandleFunc("tools/call", handleToolCall)
	
	// Apply middleware in order
	handler := recoveryMiddleware(authMiddleware(loggingMiddleware(mux)))
	
	server := sdk2.NewServer(
		sdk2.WithHandler(handler),
		sdk2.WithServerInfo("middleware-demo", "1.0.0"),
		sdk2.WithTimeouts(10*time.Second, 10*time.Second),
	)
	
	server.Addr = ":stdio"
	
	slog.Info("Starting server with middleware")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

// Typed handlers for demonstration (defined at package level)

// Calculator demonstrates typed tool handlers
type Calculator struct{}

func (c *Calculator) HandleAdd(ctx context.Context, a, b float64) (float64, error) {
	return a + b, nil
}

func (c *Calculator) HandleMultiply(ctx context.Context, a, b float64) (float64, error) {
	return a * b, nil
}

// FileSystem demonstrates resource handlers
type FileSystem struct {
	Root string
}

func (fs *FileSystem) HandleListResources(ctx context.Context) ([]sdk2.Resource, error) {
	return []sdk2.Resource{
		{
			URI:         "file:///example.txt",
			Name:        "example.txt",
			Description: "Example text file",
			MimeType:    "text/plain",
		},
	}, nil
}

func (fs *FileSystem) HandleReadResource(ctx context.Context, uri string) (*sdk2.ResourceContent, error) {
	if uri == "file:///example.txt" {
		return &sdk2.ResourceContent{
			URI:      uri,
			MimeType: "text/plain",
			Content: []sdk2.Content{
				sdk2.TextContent{Text: "Hello from file system!"},
			},
		}, nil
	}
	return nil, fmt.Errorf("resource not found: %s", uri)
}

// runTypedHandlers demonstrates type-safe handler patterns
func runTypedHandlers() {
	fmt.Println("=== TYPE-SAFE HANDLER PATTERNS ===")
	
	// Pattern 1: Typed tool handler
	// (types are defined at package level above)
	
	// Use typed handlers
	calc := &Calculator{}
	fs := &FileSystem{Root: "/tmp"}
	_ = fs // Used in demonstration
	
	// Register handlers with automatic parameter binding
	mux := sdk2.NewServeMux()
	
	// This would be enhanced in a full implementation
	mux.HandleFunc("tools/call", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		var call sdk2.ToolCall
		if err := json.Unmarshal(r.Params, &call); err != nil {
			sdk2.Error(w, "Invalid parameters", sdk2.StatusBadRequest)
			return
		}
		
		var result *sdk2.ToolResult
		var err error
		
		switch call.Name {
		case "add":
			a := call.Arguments["a"].(float64)
			b := call.Arguments["b"].(float64)
			sum, calcErr := calc.HandleAdd(r.Context, a, b)
			if calcErr != nil {
				err = calcErr
			} else {
				result = &sdk2.ToolResult{
					Content: []sdk2.Content{
						sdk2.TextContent{Text: fmt.Sprintf("%.2f", sum)},
					},
				}
			}
		case "multiply":
			a := call.Arguments["a"].(float64)
			b := call.Arguments["b"].(float64)
			product, calcErr := calc.HandleMultiply(r.Context, a, b)
			if calcErr != nil {
				err = calcErr
			} else {
				result = &sdk2.ToolResult{
					Content: []sdk2.Content{
						sdk2.TextContent{Text: fmt.Sprintf("%.2f", product)},
					},
				}
			}
		default:
			err = fmt.Errorf("unknown tool: %s", call.Name)
		}
		
		if err != nil {
			result = &sdk2.ToolResult{
				Content: []sdk2.Content{
					sdk2.TextContent{Text: err.Error()},
				},
				IsError: true,
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(result)
	})
	
	server := sdk2.NewServer(
		sdk2.WithHandler(mux),
		sdk2.WithServerInfo("typed-demo", "1.0.0"),
	)
	server.Addr = ":stdio"
	
	slog.Info("Starting typed handlers server")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

// demonstrateServerPatterns shows http.Server-like patterns
func demonstrateServerPatterns() {
	fmt.Println("1. SERVER PATTERNS (like net/http)")
	fmt.Println("==================================")
	
	// Simple server - just like http.Server
	server1 := &sdk2.Server{
		Addr:         ":stdio",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  2 * time.Minute,
	}
	fmt.Printf("✓ Simple server struct (like http.Server): %T\n", server1)
	
	// Server with functional options - like many stdlib packages
	server2 := sdk2.NewServer(
		sdk2.WithTimeouts(5*time.Second, 5*time.Second),
		sdk2.WithServerInfo("demo-server", "2.0.0"),
	)
	fmt.Printf("✓ Configured server with options: %T\n", server2)
	
	// Register handlers just like http.HandleFunc
	sdk2.HandleFunc("demo/method", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello!"})
	})
	fmt.Println("✓ Handler registration (like http.HandleFunc)")
	
	// ListenAndServe pattern
	fmt.Println("✓ server.ListenAndServe() - exactly like http.Server")
	fmt.Println()
}

// demonstrateClientPatterns shows net.Dial-like patterns  
func demonstrateClientPatterns() {
	fmt.Println("2. CLIENT PATTERNS (like net.Dial)")
	fmt.Println("==================================")
	
	ctx := context.Background()
	
	// Simple dial - just like net.Dial
	fmt.Println("✓ Simple dial (like net.Dial):")
	fmt.Println("    client, err := sdk2.Dial(ctx, \"stdio\", \"\")")
	
	// Advanced dial with options - like tls.Dial with Config
	fmt.Println("✓ Advanced dial with options (like grpc.Dial):")
	fmt.Println("    client, err := sdk2.DialConfig(ctx, \"tcp\", \"localhost:3000\",")
	fmt.Println("        sdk2.WithTimeout(30*time.Second),")
	fmt.Println("        sdk2.WithRetries(5, 2*time.Second),")
	fmt.Println("        sdk2.WithClientInfo(\"my-client\", \"2.0.0\"),")
	fmt.Println("    )")
	
	// High-level operations - like sql.DB
	fmt.Println("✓ High-level operations (like sql.DB):")
	fmt.Println("    tools, err := client.ListTools(ctx)")
	fmt.Println("    result, err := client.CallTool(ctx, \"echo\", args)")
	
	// Create a mock client for demonstration
	mockClient := createMockClient()
	
	// Demonstrate high-level operations
	tools, _ := mockClient.ListTools(ctx)
	fmt.Printf("✓ Mock client returned %d tools\n", len(tools))
	
	// Show enhanced content types
	fmt.Println("✓ Enhanced content types with validation:")
	text := sdk2.MustNewTextContent("Hello, stdlib!")
	image := sdk2.MustNewImageContent("base64data", "image/png")
	resource := sdk2.MustNewResourceReferenceContent("file://example.txt")
	fmt.Printf("    • Text: %s\n", text.ContentType())
	fmt.Printf("    • Image: %s\n", image.ContentType()) 
	fmt.Printf("    • Resource: %s\n", resource.ContentType())
	
	fmt.Println()
}

// demonstrateHandlerPatterns shows http.Handler-like patterns
func demonstrateHandlerPatterns() {
	fmt.Println("3. HANDLER PATTERNS (like net/http)")
	fmt.Println("===================================")
	
	// Function handlers - like http.HandleFunc
	fmt.Println("✓ Function handler (like http.HandleFunc):")
	handler1 := sdk2.HandlerFunc(func(w sdk2.ResponseWriter, r *sdk2.Request) {
		fmt.Fprintf(w, "Method: %s", r.Method)
	})
	fmt.Printf("    Handler type: %T\n", handler1)
	
	// Type-based handlers - like http.Handler interface
	fmt.Println("✓ Type-based handler (like http.Handler):")
	handler2 := &EchoHandler{}
	fmt.Printf("    Handler type: %T\n", handler2)
	
	// Middleware - using standard patterns
	fmt.Println("✓ Middleware (standard Go patterns):")
	wrapped := LoggingMiddleware(handler1)
	fmt.Printf("    Wrapped handler: %T\n", wrapped)
	
	// ServeMux - like http.ServeMux
	fmt.Println("✓ ServeMux (like http.ServeMux):")
	mux := sdk2.NewServeMux()
	mux.Handle("test/method", handler1)
	mux.HandleFunc("other/method", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		w.WriteHeader(sdk2.StatusOK)
	})
	fmt.Printf("    Mux type: %T\n", mux)
	fmt.Println()
}

// demonstrateFunctionalOptions shows options patterns
func demonstrateFunctionalOptions() {
	fmt.Println("4. FUNCTIONAL OPTIONS (like grpc, tls)")
	fmt.Println("======================================")
	
	// Client options
	fmt.Println("✓ Client options (like grpc.DialOption):")
	config := &sdk2.ClientConfig{}
	
	sdk2.WithTimeout(30 * time.Second)(config)
	sdk2.WithRetries(5, 2*time.Second)(config)
	sdk2.WithClientInfo("demo-client", "1.0.0")(config)
	
	fmt.Printf("    Timeout: %v\n", config.Timeout)
	fmt.Printf("    Max Retries: %d\n", config.MaxRetries)
	fmt.Printf("    Client Name: %s\n", config.ClientInfo.Name)
	
	// Server options work similarly
	fmt.Println("✓ Server options work the same way with sdk2.NewServer()")
	fmt.Println()
}

// demonstrateTypeSafety shows the strong typing features
func demonstrateTypeSafety() {
	fmt.Println("5. TYPE SAFETY (sealed interfaces)")
	fmt.Println("==================================")
	
	// Sealed Content interface
	fmt.Println("✓ Content types (sealed interface):")
	
	textContent := sdk2.TextContent{Text: "Hello, world!"}
	imageContent := sdk2.ImageContent{
		Data:     "base64data...",
		MimeType: "image/png",
	}
	
	// These are the only types that can implement Content
	contents := []sdk2.Content{textContent, imageContent}
	
	for i, content := range contents {
		fmt.Printf("    Content %d: Type=%s, ContentType=%s\n", 
			i+1, fmt.Sprintf("%T", content), content.ContentType())
		
		// Validate content
		if err := content.Valid(); err != nil {
			fmt.Printf("      Validation error: %v\n", err)
		} else {
			fmt.Printf("      ✓ Valid content\n")
		}
	}
	
	// RequestID flexibility
	fmt.Println("✓ RequestID (flexible JSON-RPC IDs):")
	ids := []sdk2.RequestID{
		{Value: "string-id"},
		{Value: int64(123)},
		{Value: float64(123.45)},
		{Value: nil},
	}
	
	for _, id := range ids {
		data, _ := json.Marshal(id)
		fmt.Printf("    ID %s -> JSON: %s\n", id.String(), string(data))
	}
	fmt.Println()
}

// demonstrateTransportAbstraction shows transport patterns
func demonstrateTransportAbstraction() {
	fmt.Println("6. TRANSPORT ABSTRACTION (like net.Conn)")
	fmt.Println("========================================")
	
	// Different transport types
	fmt.Println("✓ Available transports:")
	
	// Stdio transport
	stdio := transport.NewStdio()
	fmt.Printf("    Stdio: %T (for subprocess communication)\n", stdio)
	
	// ReadWriteCloser transport (can wrap anything)
	fmt.Println("    ReadWriteCloser: wraps any io.ReadWriteCloser")
	
	// These would be available in a full implementation:
	fmt.Println("    TCP: (like net.Dial)")
	fmt.Println("    WebSocket: (for web clients)")
	fmt.Println("    HTTP: (for REST-like usage)")
	
	fmt.Println("✓ All transports implement the same interface")
	fmt.Printf("    Transport interface: Dial(ctx) (Conn, error)\n")
	fmt.Printf("    Conn interface: like net.Conn but for MCP\n")
	fmt.Println()
}

// handleToolCall demonstrates tool execution
func handleToolCall(w sdk2.ResponseWriter, r *sdk2.Request) {
	var call sdk2.ToolCall
	if err := json.Unmarshal(r.Params, &call); err != nil {
		sdk2.Error(w, "Invalid tool call parameters", sdk2.StatusBadRequest)
		return
	}
	
	var result *sdk2.ToolResult
	
	switch call.Name {
	case "echo":
		message := call.Arguments["message"].(string)
		result = &sdk2.ToolResult{
			Content: []sdk2.Content{
				sdk2.TextContent{Text: fmt.Sprintf("Echo: %s", message)},
			},
		}
		
	case "math":
		operation := call.Arguments["operation"].(string)
		a := call.Arguments["a"].(float64)
		b := call.Arguments["b"].(float64)
		
		var value float64
		switch operation {
		case "add":
			value = a + b
		case "subtract":
			value = a - b
		case "multiply":
			value = a * b
		case "divide":
			if b == 0 {
				result = &sdk2.ToolResult{
					Content: []sdk2.Content{
						sdk2.TextContent{Text: "Error: Division by zero"},
					},
					IsError: true,
				}
			} else {
				value = a / b
			}
		default:
			result = &sdk2.ToolResult{
				Content: []sdk2.Content{
					sdk2.TextContent{Text: fmt.Sprintf("Unknown operation: %s", operation)},
				},
				IsError: true,
			}
		}
		
		if result == nil {
			result = &sdk2.ToolResult{
				Content: []sdk2.Content{
					sdk2.TextContent{Text: fmt.Sprintf("%.2f", value)},
				},
			}
		}
		
	default:
		sdk2.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sdk2.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// EchoHandler demonstrates type-based handler
type EchoHandler struct{}

func (h *EchoHandler) ServeRequest(w sdk2.ResponseWriter, r *sdk2.Request) {
	var call sdk2.ToolCall
	json.Unmarshal(r.Params, &call)
	
	message, ok := call.Arguments["message"].(string)
	if !ok {
		sdk2.Error(w, "message parameter required", sdk2.StatusBadRequest)
		return
	}
	
	result := &sdk2.ToolResult{
		Content: []sdk2.Content{
			sdk2.TextContent{Text: "Echo: " + message},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sdk2.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// LoggingMiddleware demonstrates middleware patterns
func LoggingMiddleware(next sdk2.Handler) sdk2.Handler {
	return sdk2.HandlerFunc(func(w sdk2.ResponseWriter, r *sdk2.Request) {
		fmt.Printf("-> %s\n", r.Method)
		next.ServeRequest(w, r)
		fmt.Printf("<- completed\n")
	})
}

// createMockClient creates a simple mock for demonstration
func createMockClient() sdk2.Client {
	return &mockClient{
		tools: []sdk2.Tool{
			{
				Name:        "echo",
				Description: "Echoes back the input",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`),
			},
			{
				Name:        "add",
				Description: "Adds two numbers",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}}}`),
			},
		},
	}
}

// mockClient implements Client for demonstration
type mockClient struct {
	tools []sdk2.Tool
}

func (c *mockClient) Do(req *sdk2.Request) (*sdk2.Response, error) {
	return nil, fmt.Errorf("not implemented in demo")
}

func (c *mockClient) ListTools(ctx context.Context) ([]sdk2.Tool, error) {
	return c.tools, nil
}

func (c *mockClient) CallTool(ctx context.Context, name string, args map[string]any) (*sdk2.ToolResult, error) {
	return &sdk2.ToolResult{
		Content: []sdk2.Content{
			sdk2.TextContent{Text: fmt.Sprintf("Mock result from %s", name)},
		},
	}, nil
}

func (c *mockClient) ListResources(ctx context.Context) ([]sdk2.Resource, error) {
	return nil, fmt.Errorf("not implemented in demo")
}

func (c *mockClient) ReadResource(ctx context.Context, uri string) (*sdk2.ResourceContent, error) {
	return nil, fmt.Errorf("not implemented in demo")
}

func (c *mockClient) ListPrompts(ctx context.Context) ([]sdk2.Prompt, error) {
	return nil, fmt.Errorf("not implemented in demo")
}

func (c *mockClient) GetPrompt(ctx context.Context, name string, args map[string]any) (*sdk2.PromptResult, error) {
	return nil, fmt.Errorf("not implemented in demo")
}

func (c *mockClient) Close() error {
	log.Println("Mock client closed")
	return nil
}

// Helper functions for schema generation
func echoSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Message to echo back",
			},
		},
		"required": []string{"message"},
	}
	data, _ := json.Marshal(schema)
	return json.RawMessage(data)
}

func mathSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Mathematical operation",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
			},
			"a": map[string]any{
				"type":        "number",
				"description": "First number",
			},
			"b": map[string]any{
				"type":        "number",
				"description": "Second number",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
	data, _ := json.Marshal(schema)
	return json.RawMessage(data)
}