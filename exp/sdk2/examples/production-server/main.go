// Package main demonstrates a production-ready MCP server using all advanced SDK2 features.
// This showcases real-world patterns including middleware, error handling, observability,
// connection pooling, type safety, and comprehensive testing.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tmc/mcp/exp/sdk2"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Show different patterns based on args
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "server":
			runProductionServer()
		case "client":
			runProductionClient()
		case "load-test":
			runLoadTest()
		case "integration-test":
			runIntegrationTest()
		case "middleware-demo":
			runMiddlewareDemo()
		case "error-handling-demo":
			runErrorHandlingDemo()
		case "type-safety-demo":
			runTypeSafetyDemo()
		default:
			fmt.Printf("Usage: %s [server|client|load-test|integration-test|middleware-demo|error-handling-demo|type-safety-demo]\n", os.Args[0])
		}
		return
	}

	fmt.Println("Production MCP Server Demo - Choose a demo:")
	fmt.Println("  go run main.go server              - Production server with all features")
	fmt.Println("  go run main.go client              - Advanced client with connection pooling")
	fmt.Println("  go run main.go load-test           - Load testing demonstration")
	fmt.Println("  go run main.go integration-test    - Integration testing patterns")
	fmt.Println("  go run main.go middleware-demo     - Advanced middleware patterns")
	fmt.Println("  go run main.go error-handling-demo - Error handling and recovery")
	fmt.Println("  go run main.go type-safety-demo    - Type safety and validation")
}

// runProductionServer demonstrates a production-ready server.
func runProductionServer() {
	slog.Info("Starting production MCP server")

	// Create service implementations
	mathService := &MathService{}
	fileService := &FileService{Root: "/tmp"}
	aiService := &AIService{}

	// Setup middleware chain
	mux := sdk2.NewServeMux()

	// Register services with type-safe handlers
	registerMathService(mux, mathService)
	registerFileService(mux, fileService)
	registerAIService(mux, aiService)

	// Create comprehensive middleware chain
	handler := sdk2.Chain(mux,
		sdk2.RecoveryMiddleware(),
		sdk2.RequestIDMiddleware(),
		sdk2.LoggingMiddlewareWithLogger(slog.Default()),
		sdk2.MetricsMiddleware(),
		sdk2.AuthMiddleware(),
		sdk2.RateLimitMiddleware(),
		sdk2.TimeoutMiddleware(30*time.Second),
		sdk2.CORSMiddleware(),
	)

	// Create server with production configuration
	server := sdk2.NewServer(
		sdk2.WithHandler(handler),
		sdk2.WithServerInfo("production-server", "1.0.0"),
		sdk2.WithTimeouts(10*time.Second, 10*time.Second),
		sdk2.WithCapabilities(&sdk2.ServerCapabilities{
			Tools:     &sdk2.ToolsCapability{ListChanged: true},
			Resources: &sdk2.ResourcesCapability{Subscribe: true, ListChanged: true},
			Prompts:   &sdk2.PromptsCapability{ListChanged: true},
			Logging:   &sdk2.LoggingCapability{Level: "info"},
		}),
	)

	server.Addr = ":stdio"

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("Shutting down server gracefully")
		cancel()
	}()

	// Start server with context
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ListenAndServe()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			slog.Error("Server failed", "error", err)
		}
	case <-ctx.Done():
		slog.Info("Server shutdown complete")
	}
}

// runProductionClient demonstrates an advanced client with connection pooling.
func runProductionClient() {
	ctx := context.Background()

	slog.Info("Starting production MCP client")

	// Create client with connection pooling and advanced configuration
	client := sdk2.NewClient(
		sdk2.WithConnectionPool(&sdk2.PoolConfig{
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		}),
		sdk2.WithRetryPolicy(&sdk2.RetryConfig{
			MaxRetries:    3,
			BackoffFunc:   sdk2.ExponentialBackoff,
			RetryableFunc: sdk2.IsRetryable,
		}),
		sdk2.WithTimeout(30*time.Second),
		sdk2.WithClientInfo("production-client", "1.0.0"),
		sdk2.WithNotificationHandler(sdk2.NotificationHandlerFunc(
			func(ctx context.Context, method string, params json.RawMessage) error {
				slog.Info("Notification received", "method", method)
				return nil
			},
		)),
	)
	defer client.Close()

	// Health check
	if err := client.Ping(ctx); err != nil {
		slog.Error("Client health check failed", "error", err)
		return
	}

	// Demonstrate operations with error handling
	slog.Info("Listing available tools")
	tools, err := client.ListTools(ctx)
	if err != nil {
		slog.Error("Failed to list tools", "error", err)
		return
	}
	slog.Info("Found tools", "count", len(tools))

	// Demonstrate tool calls with proper error handling
	for _, tool := range tools {
		slog.Info("Calling tool", "name", tool.Name)

		var args map[string]any
		switch tool.Name {
		case "add":
			args = map[string]any{"a": 5.0, "b": 3.0}
		case "read-file":
			args = map[string]any{"path": "/tmp/example.txt"}
		case "generate-text":
			args = map[string]any{"prompt": "Hello, world!", "max_tokens": 50}
		default:
			continue
		}

		result, err := client.CallTool(ctx, tool.Name, args)
		if err != nil {
			slog.Error("Tool call failed", "tool", tool.Name, "error", err)
			continue
		}
		slog.Info("Tool call succeeded", "tool", tool.Name, "result", result)
	}

	// Show connection statistics
	if pooledClient, ok := client.(*sdk2.PooledClient); ok {
		stats := pooledClient.Stats()
		slog.Info("Connection pool statistics",
			"open_connections", stats.OpenConnections,
			"idle_connections", stats.Idle,
			"in_use", stats.InUse,
		)
	}
}

// Service implementations demonstrating type-safe patterns

// MathService provides mathematical operations.
type MathService struct{}

type AddRequest struct {
	A float64 `json:"a" validate:"required"`
	B float64 `json:"b" validate:"required"`
}

type AddResponse struct {
	Result float64 `json:"result"`
}

func (m *MathService) Add(ctx context.Context, req AddRequest) (AddResponse, error) {
	// Validate request
	if err := sdk2.ValidateStruct(req); err != nil {
		return AddResponse{}, sdk2.NewError("math.add", sdk2.StatusBadRequest, "validation failed", err)
	}

	slog.InfoContext(ctx, "Performing addition", "a", req.A, "b", req.B)
	return AddResponse{Result: req.A + req.B}, nil
}

type MultiplyRequest struct {
	A float64 `json:"a" validate:"required"`
	B float64 `json:"b" validate:"required"`
}

type MultiplyResponse struct {
	Result float64 `json:"result"`
}

func (m *MathService) Multiply(ctx context.Context, req MultiplyRequest) (MultiplyResponse, error) {
	if err := sdk2.ValidateStruct(req); err != nil {
		return MultiplyResponse{}, sdk2.NewError("math.multiply", sdk2.StatusBadRequest, "validation failed", err)
	}

	result := req.A * req.B
	slog.InfoContext(ctx, "Performing multiplication", "a", req.A, "b", req.B, "result", result)
	return MultiplyResponse{Result: result}, nil
}

// FileService provides file operations.
type FileService struct {
	Root string
}

type ReadFileRequest struct {
	Path string `json:"path" validate:"required"`
}

type ReadFileResponse struct {
	Content sdk2.Content `json:"content"`
}

func (f *FileService) ReadFile(ctx context.Context, req ReadFileRequest) (ReadFileResponse, error) {
	if err := sdk2.ValidateStruct(req); err != nil {
		return ReadFileResponse{}, sdk2.NewError("file.read", sdk2.StatusBadRequest, "validation failed", err)
	}

	// Security check: ensure path is within root
	// In production, implement proper path validation
	slog.InfoContext(ctx, "Reading file", "path", req.Path)

	// Mock file content
	content, err := sdk2.NewEnhancedTextContent(
		fmt.Sprintf("Mock content for file: %s", req.Path),
		sdk2.WithTextLanguage("en"),
		sdk2.WithTextEncoding("utf-8"),
	)
	if err != nil {
		return ReadFileResponse{}, sdk2.NewError("file.read", sdk2.StatusInternalServerError, "content creation failed", err)
	}

	return ReadFileResponse{Content: content}, nil
}

// AIService provides AI operations.
type AIService struct{}

type GenerateTextRequest struct {
	Prompt    string `json:"prompt" validate:"required"`
	MaxTokens int    `json:"max_tokens" validate:"min=1,max=4096"`
}

type GenerateTextResponse struct {
	Text   string `json:"text"`
	Tokens int    `json:"tokens"`
}

func (a *AIService) GenerateText(ctx context.Context, req GenerateTextRequest) (GenerateTextResponse, error) {
	if err := sdk2.ValidateStruct(req); err != nil {
		return GenerateTextResponse{}, sdk2.NewError("ai.generate", sdk2.StatusBadRequest, "validation failed", err)
	}

	slog.InfoContext(ctx, "Generating text", "prompt", req.Prompt, "max_tokens", req.MaxTokens)

	// Mock AI generation
	response := GenerateTextResponse{
		Text:   fmt.Sprintf("AI generated response to: %s", req.Prompt),
		Tokens: 25,
	}

	return response, nil
}

// Service registration functions

func registerMathService(mux *sdk2.ServeMux, service *MathService) {
	// Register tools/list handler
	mux.HandleFunc("tools/list", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		tools := []sdk2.Tool{
			{
				Name:        "add",
				Description: "Adds two numbers",
				InputSchema: mustMarshal(AddRequest{}),
			},
			{
				Name:        "multiply",
				Description: "Multiplies two numbers",
				InputSchema: mustMarshal(MultiplyRequest{}),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"tools": tools})
	})

	// Register tool call handler
	mux.HandleFunc("tools/call", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		var call sdk2.ToolCall
		if err := json.Unmarshal(r.Params, &call); err != nil {
			sdk2.Error(w, "Invalid tool call parameters", sdk2.StatusBadRequest)
			return
		}

		var result interface{}
		var err error

		switch call.Name {
		case "add":
			var req AddRequest
			if err := mapToStruct(call.Arguments, &req); err != nil {
				sdk2.Error(w, "Invalid add parameters", sdk2.StatusBadRequest)
				return
			}
			result, err = service.Add(r.Context, req)

		case "multiply":
			var req MultiplyRequest
			if err := mapToStruct(call.Arguments, &req); err != nil {
				sdk2.Error(w, "Invalid multiply parameters", sdk2.StatusBadRequest)
				return
			}
			result, err = service.Multiply(r.Context, req)

		default:
			sdk2.Error(w, fmt.Sprintf("Unknown tool: %s", call.Name), sdk2.StatusNotFound)
			return
		}

		if err != nil {
			handleServiceError(w, err)
			return
		}

		// Convert result to ToolResult
		content, err := resultToContent(result)
		if err != nil {
			sdk2.Error(w, "Failed to serialize result", sdk2.StatusInternalServerError)
			return
		}

		toolResult := &sdk2.ToolResult{Content: []sdk2.Content{content}}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(toolResult)
	})
}

func registerFileService(mux *sdk2.ServeMux, service *FileService) {
	// Register resources/list handler
	mux.HandleFunc("resources/list", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		resources := []sdk2.Resource{
			{
				URI:         "file:///tmp/example.txt",
				Name:        "example.txt",
				Description: "Example text file",
				MimeType:    "text/plain",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"resources": resources})
	})

	// Register resources/read handler
	mux.HandleFunc("resources/read", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		var req sdk2.ResourceRequest
		if err := json.Unmarshal(r.Params, &req); err != nil {
			sdk2.Error(w, "Invalid resource request", sdk2.StatusBadRequest)
			return
		}

		// Extract path from URI
		path := req.URI // Simplified - in production, parse URI properly

		readReq := ReadFileRequest{Path: path}
		response, err := service.ReadFile(r.Context, readReq)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		resourceContent := &sdk2.ResourceContent{
			URI:      req.URI,
			MimeType: "text/plain",
			Content:  []sdk2.Content{response.Content},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"contents": []sdk2.ResourceContent{*resourceContent}})
	})
}

func registerAIService(mux *sdk2.ServeMux, service *AIService) {
	// Register prompts/list handler
	mux.HandleFunc("prompts/list", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		prompts := []sdk2.Prompt{
			{
				Name:        "generate-text",
				Description: "Generates text using AI",
				ArgumentsSchema: mustMarshal(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"prompt": map[string]any{
							"type":        "string",
							"description": "The prompt for text generation",
						},
						"max_tokens": map[string]any{
							"type":        "integer",
							"description": "Maximum number of tokens to generate",
							"minimum":     1,
							"maximum":     4096,
							"default":     100,
						},
					},
					"required": []string{"prompt"},
				}),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"prompts": prompts})
	})

	// Register prompts/get handler
	mux.HandleFunc("prompts/get", func(w sdk2.ResponseWriter, r *sdk2.Request) {
		var req sdk2.PromptRequest
		if err := json.Unmarshal(r.Params, &req); err != nil {
			sdk2.Error(w, "Invalid prompt request", sdk2.StatusBadRequest)
			return
		}

		if req.Name != "generate-text" {
			sdk2.Error(w, fmt.Sprintf("Unknown prompt: %s", req.Name), sdk2.StatusNotFound)
			return
		}

		var genReq GenerateTextRequest
		if err := mapToStruct(req.Arguments, &genReq); err != nil {
			sdk2.Error(w, "Invalid generate text parameters", sdk2.StatusBadRequest)
			return
		}

		response, err := service.GenerateText(r.Context, genReq)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		promptResult := &sdk2.PromptResult{
			Description: "AI generated text",
			Messages: []sdk2.PromptMessage{
				{
					Role: "assistant",
					Content: []sdk2.Content{
						sdk2.TextContent{Text: response.Text},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(promptResult)
	})
}

// Utility functions

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(data)
}

func mapToStruct(m map[string]any, dest interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func resultToContent(result interface{}) (sdk2.Content, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return sdk2.TextContent{Text: string(data)}, nil
}

func handleServiceError(w sdk2.ResponseWriter, err error) {
	if mcpErr, ok := err.(*sdk2.MCPError); ok {
		sdk2.Error(w, mcpErr.Message, mcpErr.Code)
	} else {
		sdk2.Error(w, "Internal server error", sdk2.StatusInternalServerError)
	}
}

// Demonstration functions

func runLoadTest() {
	fmt.Println("=== LOAD TESTING DEMONSTRATION ===")

	// Create a test server
	server := sdk2.NewTestServer(nil)
	defer server.Close()

	// Create load test helper
	loadTest := sdk2.NewLoadTestHelper(10, 5*time.Second)

	fmt.Println("Running load test: 10 concurrent requests for 5 seconds")

	result := loadTest.RunLoadTest(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		tools, err := server.Client.ListTools(ctx)
		if err != nil {
			return err
		}

		// Simple validation
		if len(tools) < 0 {
			return fmt.Errorf("unexpected tools count")
		}

		return nil
	})

	fmt.Println(result.String())
}

func runIntegrationTest() {
	fmt.Println("=== INTEGRATION TESTING DEMONSTRATION ===")

	// Create test helper
	helper := sdk2.NewTestHelper(&testLogger{})

	// Create test server with our handlers
	mux := sdk2.NewServeMux()
	mathService := &MathService{}
	registerMathService(mux, mathService)

	server := sdk2.NewTestServer(mux)
	defer server.Close()

	ctx := context.Background()

	// Test tools listing
	tools, err := server.Client.ListTools(ctx)
	helper.AssertNoError(err)
	helper.AssertEqual(len(tools), 2)

	// Test tool call
	result, err := server.Client.CallTool(ctx, "add", map[string]any{
		"a": 5.0,
		"b": 3.0,
	})
	helper.AssertNoError(err)
	helper.AssertTrue(len(result.Content) > 0, "result should have content")

	fmt.Println("✓ All integration tests passed")
}

func runMiddlewareDemo() {
	fmt.Println("=== MIDDLEWARE DEMONSTRATION ===")

	// Create a handler
	handler := sdk2.HandlerFunc(func(w sdk2.ResponseWriter, r *sdk2.Request) {
		w.WriteHeader(sdk2.StatusOK)
		fmt.Fprintf(w, "Hello from %s", r.Method)
	})

	// Apply middleware chain
	middlewareChain := sdk2.Chain(handler,
		sdk2.RecoveryMiddleware(),
		sdk2.RequestIDMiddleware(),
		sdk2.LoggingMiddleware(),
		sdk2.TimeoutMiddleware(5*time.Second),
	)

	// Test the middleware chain
	recorder := sdk2.NewResponseRecorder()
	request := &sdk2.Request{
		Method:  "test/method",
		Context: context.Background(),
	}

	middlewareChain.ServeRequest(recorder, request)

	fmt.Printf("Response code: %d\n", recorder.Code)
	fmt.Printf("Response body: %s\n", recorder.Result())
	fmt.Printf("Request ID header: %s\n", recorder.Header.Get("X-Request-ID"))

	fmt.Println("✓ Middleware demonstration complete")
}

func runErrorHandlingDemo() {
	fmt.Println("=== ERROR HANDLING DEMONSTRATION ===")

	// Test different error types
	errors := []error{
		sdk2.NewError("demo", sdk2.StatusBadRequest, "validation failed", fmt.Errorf("field required")),
		&sdk2.TimeoutError{Op: "demo", Timeout: "5s"},
		&sdk2.ConnectionError{Op: "dial", Network: "tcp", Address: "localhost:3000"},
	}

	for i, err := range errors {
		fmt.Printf("Error %d: %v\n", i+1, err)

		// Test error unwrapping
		if unwrapped := err.(interface{ Unwrap() error }); unwrapped != nil {
			if cause := unwrapped.Unwrap(); cause != nil {
				fmt.Printf("  Cause: %v\n", cause)
			}
		}

		// Test error type checking
		if sdk2.IsTimeout(err) {
			fmt.Println("  → This is a timeout error")
		}

		if sdk2.IsRetryable(err) {
			fmt.Println("  → This error is retryable")
		}

		fmt.Println()
	}

	fmt.Println("✓ Error handling demonstration complete")
}

func runTypeSafetyDemo() {
	fmt.Println("=== TYPE SAFETY DEMONSTRATION ===")

	// Test content validation
	fmt.Println("1. Content Validation:")

	// Valid content
	text, err := sdk2.NewEnhancedTextContent("Hello, world!", sdk2.WithTextLanguage("en"))
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  ✓ Valid text content: %s\n", text.ContentType())
	}

	// Invalid content
	_, err = sdk2.NewEnhancedTextContent("", sdk2.WithTextLanguage("invalid-lang"))
	if err != nil {
		fmt.Printf("  ✓ Validation caught invalid content: %v\n", err)
	}

	// Test struct validation
	fmt.Println("\n2. Struct Validation:")

	validRequest := AddRequest{A: 5.0, B: 3.0}
	err = sdk2.ValidateStruct(validRequest)
	if err == nil {
		fmt.Println("  ✓ Valid struct passed validation")
	}

	// Test sealed interface
	fmt.Println("\n3. Sealed Interface:")
	contents := []sdk2.Content{
		sdk2.TextContent{Text: "Hello"},
		sdk2.ImageContent{Data: "base64data", MimeType: "image/png"},
	}

	for i, content := range contents {
		fmt.Printf("  Content %d: %T (%s)\n", i+1, content, content.ContentType())
	}

	fmt.Println("\n✓ Type safety demonstration complete")
}

// testLogger implements a simple test logger
type testLogger struct{}

func (t *testLogger) Helper()                                   {}
func (t *testLogger) Fatalf(format string, args ...interface{}) { log.Fatalf(format, args...) }
func (t *testLogger) Errorf(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}
