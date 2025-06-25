// Package mcp - Type-Safe API Examples
//
// This file demonstrates the comprehensive type-safe APIs for MCP with Go generics.
// These examples show how to use the new type-safe features while maintaining
// full backward compatibility with existing APIs.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Example types for demonstration
type MathOperation struct {
	Operation string  `json:"operation" validate:"required,nonempty" description:"The mathematical operation to perform"`
	A         float64 `json:"a" validate:"required" description:"First operand"`
	B         float64 `json:"b" validate:"required" description:"Second operand"`
}

type MathResult struct {
	Result float64 `json:"result" description:"The result of the operation"`
	Error  string  `json:"error,omitempty" description:"Error message if operation failed"`
}

type UserData struct {
	ID       int               `json:"id" validate:"required"`
	Name     string            `json:"name" validate:"required,nonempty"`
	Email    string            `json:"email" validate:"required"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
	Active   bool              `json:"active"`
}

type SearchRequest struct {
	Query   string            `json:"query" validate:"required,nonempty"`
	Filters map[string]string `json:"filters"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}

type SearchResponse struct {
	Results []UserData `json:"results"`
	Total   int        `json:"total"`
	HasMore bool       `json:"hasMore"`
}

// ExampleTypedToolServer demonstrates creating a server with type-safe tools
func ExampleTypedToolServer() {
	// Create a new server
	server := NewServer("math-server", "1.0.0", 
		WithServerName("Enhanced Math Server"),
		WithServerInstructions("A demonstration of type-safe MCP tools"))

	// Register a type-safe math tool with compile-time type checking
	err := RegisterTypedToolWithServer(server, "calculate", "Perform mathematical calculations",
		func(ctx context.Context, req MathOperation) (MathResult, error) {
			switch req.Operation {
			case "add":
				return MathResult{Result: req.A + req.B}, nil
			case "subtract":
				return MathResult{Result: req.A - req.B}, nil
			case "multiply":
				return MathResult{Result: req.A * req.B}, nil
			case "divide":
				if req.B == 0 {
					return MathResult{Error: "division by zero"}, nil
				}
				return MathResult{Result: req.A / req.B}, nil
			case "power":
				// Simple power implementation for positive integer exponents
				result := 1.0
				for i := 0; i < int(req.B); i++ {
					result *= req.A
				}
				return MathResult{Result: result}, nil
			default:
				return MathResult{Error: fmt.Sprintf("unsupported operation: %s", req.Operation)}, nil
			}
		})

	if err != nil {
		log.Fatalf("Failed to register math tool: %v", err)
	}

	// Register a complex search tool demonstrating nested types
	err = RegisterTypedToolWithServer(server, "search_users", "Search for users with filters",
		func(ctx context.Context, req SearchRequest) (SearchResponse, error) {
			// Mock database of users
			users := []UserData{
				{ID: 1, Name: "Alice Johnson", Email: "alice@example.com", Tags: []string{"admin", "developer"}, Active: true},
				{ID: 2, Name: "Bob Smith", Email: "bob@example.com", Tags: []string{"user"}, Active: true},
				{ID: 3, Name: "Charlie Brown", Email: "charlie@example.com", Tags: []string{"manager"}, Active: false},
				{ID: 4, Name: "Diana Prince", Email: "diana@example.com", Tags: []string{"admin"}, Active: true},
			}

			var results []UserData
			for _, user := range users {
				// Simple search logic
				if containsIgnoreCase(user.Name, req.Query) || 
				   containsIgnoreCase(user.Email, req.Query) {
					// Apply filters
					matches := true
					for key, value := range req.Filters {
						switch key {
						case "active":
							if (value == "true") != user.Active {
								matches = false
							}
						case "tag":
							if !containsTag(user.Tags, value) {
								matches = false
							}
						}
					}
					if matches {
						results = append(results, user)
					}
				}
			}

			// Apply pagination
			start := req.Offset
			if start > len(results) {
				start = len(results)
			}

			end := start + req.Limit
			if req.Limit <= 0 || end > len(results) {
				end = len(results)
			}

			paginatedResults := results[start:end]
			hasMore := end < len(results)

			return SearchResponse{
				Results: paginatedResults,
				Total:   len(results),
				HasMore: hasMore,
			}, nil
		})

	if err != nil {
		log.Fatalf("Failed to register search tool: %v", err)
	}

	// Start the server
	ctx := context.Background()
	if err := server.Serve(ctx, StdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ExampleTypedToolClient demonstrates using type-safe client methods
func ExampleTypedToolClient() {
	// Create a client (assuming server is running)
	client, err := NewClient(StdioTransport())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize the client
	ctx := context.Background()
	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "typed-client", Version: "1.0.0"},
	})
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Type-safe tool call with compile-time checking
	mathReq := MathOperation{
		Operation: "multiply",
		A:         12.5,
		B:         4.0,
	}

	result, err := CallToolTyped[MathOperation, MathResult](client, ctx, "calculate", mathReq)
	if err != nil {
		log.Fatalf("Math tool call failed: %v", err)
	}

	fmt.Printf("Math result: %.2f\n", result.Result)

	// Complex search with type safety
	searchReq := SearchRequest{
		Query:   "alice",
		Filters: map[string]string{"active": "true"},
		Limit:   10,
		Offset:  0,
	}

	searchResult, err := CallToolTyped[SearchRequest, SearchResponse](client, ctx, "search_users", searchReq)
	if err != nil {
		log.Fatalf("Search tool call failed: %v", err)
	}

	fmt.Printf("Found %d users:\n", len(searchResult.Results))
	for _, user := range searchResult.Results {
		fmt.Printf("- %s (%s)\n", user.Name, user.Email)
	}
}

// ExampleHandlerChainWithValidation demonstrates advanced handler chaining
func ExampleHandlerChainWithValidation() {
	// Create a validation function for math operations
	mathValidator := ValidationFunc[MathOperation](func(ctx context.Context, req MathOperation) error {
		validOps := map[string]bool{
			"add": true, "subtract": true, "multiply": true, "divide": true, "power": true,
		}
		
		if !validOps[req.Operation] {
			return fmt.Errorf("invalid operation: %s", req.Operation)
		}
		
		if req.Operation == "divide" && req.B == 0 {
			return fmt.Errorf("division by zero not allowed")
		}
		
		if req.Operation == "power" && req.B < 0 {
			return fmt.Errorf("negative exponents not supported")
		}
		
		return nil
	})

	// Create a math handler
	mathHandler := HandlerFunc[MathOperation, MathResult](func(ctx context.Context, req MathOperation) (MathResult, error) {
		switch req.Operation {
		case "add":
			return MathResult{Result: req.A + req.B}, nil
		case "subtract":
			return MathResult{Result: req.A - req.B}, nil
		case "multiply":
			return MathResult{Result: req.A * req.B}, nil
		case "divide":
			return MathResult{Result: req.A / req.B}, nil
		case "power":
			result := 1.0
			for i := 0; i < int(req.B); i++ {
				result *= req.A
			}
			return MathResult{Result: result}, nil
		default:
			return MathResult{Error: "unsupported operation"}, nil
		}
	})

	// Create a handler chain with validation
	chain := NewHandlerChain(mathHandler).WithValidation(mathValidator)

	// Test the chain
	ctx := context.Background()
	
	// Valid request
	validReq := MathOperation{Operation: "multiply", A: 6, B: 7}
	result, err := chain.Handle(ctx, validReq)
	if err != nil {
		log.Printf("Validation failed: %v", err)
	} else {
		fmt.Printf("Valid result: %.2f\n", result.Result)
	}

	// Invalid request
	invalidReq := MathOperation{Operation: "invalid", A: 5, B: 3}
	_, err = chain.Handle(ctx, invalidReq)
	if err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}
}

// ExampleStructValidation demonstrates struct tag-based validation
func ExampleStructValidation() {
	validator := NewStructValidator()

	// Valid user data
	validUser := UserData{
		ID:     1,
		Name:   "John Doe",
		Email:  "john@example.com",
		Active: true,
		Tags:   []string{"user", "verified"},
	}

	ctx := context.Background()
	if err := validator.Validate(ctx, validUser); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("User data is valid")
	}

	// Invalid user data (missing required field)
	invalidUser := UserData{
		Name:  "", // Empty name should fail
		Email: "john@example.com",
	}

	if err := validator.Validate(ctx, invalidUser); err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}
}

// ExampleSchemaGeneration demonstrates automatic schema generation
func ExampleSchemaGeneration() {
	// Generate JSON schema for a complex type
	schema, err := GenerateTypedSchema[SearchRequest]()
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}

	// Pretty print the schema
	var schemaMap map[string]any
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		log.Fatalf("Failed to unmarshal schema: %v", err)
	}

	prettySchema, err := json.MarshalIndent(schemaMap, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal pretty schema: %v", err)
	}

	fmt.Println("Generated JSON Schema:")
	fmt.Println(string(prettySchema))

	// Generate OpenAPI-compatible schema
	openAPISchema, err := GenerateOpenAPISchema[SearchRequest]()
	if err != nil {
		log.Fatalf("Failed to generate OpenAPI schema: %v", err)
	}

	prettyOpenAPI, err := json.MarshalIndent(openAPISchema, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal OpenAPI schema: %v", err)
	}

	fmt.Println("\nGenerated OpenAPI Schema:")
	fmt.Println(string(prettyOpenAPI))
}

// ExampleBackwardCompatibility demonstrates maintaining 100% backward compatibility
func ExampleBackwardCompatibility() {
	server := NewServer("compat-server", "1.0.0")

	// Old way still works (backward compatibility)
	err := RegisterTypedTool(server, "legacy_add", "Legacy addition tool",
		func(ctx context.Context, req MathOperation) (MathResult, error) {
			return MathResult{Result: req.A + req.B}, nil
		})
	if err != nil {
		log.Fatalf("Legacy registration failed: %v", err)
	}

	// New way for better encapsulation
	err = RegisterTypedToolWithServer(server, "modern_multiply", "Modern multiplication tool",
		func(ctx context.Context, req MathOperation) (MathResult, error) {
			return MathResult{Result: req.A * req.B}, nil
		})
	if err != nil {
		log.Fatalf("Modern registration failed: %v", err)
	}

	// Both tools are registered and work identically
	fmt.Println("Both legacy and modern APIs work seamlessly")
}

// Helper functions for the examples

func containsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) && 
		   len(substr) > 0 && 
		   containsIgnoreCaseImpl(str, substr)
}

func containsIgnoreCaseImpl(str, substr string) bool {
	str = toLower(str)
	substr = toLower(substr)
	
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i, c := range []byte(s) {
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ExampleMinimalSetup shows the simplest way to get started with type-safe MCP
func ExampleMinimalSetup() {
	// Create server
	server := NewServer("my-server", "1.0.0")

	// Register a simple typed tool
	RegisterTypedToolWithServer(server, "hello", "Say hello",
		func(ctx context.Context, name string) (string, error) {
			return fmt.Sprintf("Hello, %s!", name), nil
		})

	// Start server
	if err := server.Serve(context.Background(), StdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// RunExamples runs all the examples for demonstration
func RunExamples() {
	fmt.Println("=== MCP Type-Safe API Examples ===")
	fmt.Println()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "server":
			fmt.Println("Starting typed tool server...")
			ExampleTypedToolServer()
		case "client":
			fmt.Println("Running typed tool client...")
			ExampleTypedToolClient()
		case "validation":
			fmt.Println("Demonstrating handler chain validation...")
			ExampleHandlerChainWithValidation()
		case "struct-validation":
			fmt.Println("Demonstrating struct validation...")
			ExampleStructValidation()
		case "schema":
			fmt.Println("Demonstrating schema generation...")
			ExampleSchemaGeneration()
		case "compatibility":
			fmt.Println("Demonstrating backward compatibility...")
			ExampleBackwardCompatibility()
		case "minimal":
			fmt.Println("Running minimal setup example...")
			ExampleMinimalSetup()
		default:
			fmt.Printf("Unknown example: %s\n", os.Args[1])
			fmt.Println("Available examples: server, client, validation, struct-validation, schema, compatibility, minimal")
		}
	} else {
		fmt.Println("Usage: go run examples_typed.go <example>")
		fmt.Println("Available examples: server, client, validation, struct-validation, schema, compatibility, minimal")
	}
}