package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// Test types for comprehensive type-safe API testing
type CalculateArgs struct {
	Operation string  `json:"operation" validate:"required,nonempty"`
	A         float64 `json:"a" validate:"required"`
	B         float64 `json:"b" validate:"required"`
}

type CalculateResult struct {
	Result float64 `json:"result"`
	Error  string  `json:"error,omitempty"`
}

type UserProfile struct {
	ID       int               `json:"id" validate:"required"`
	Name     string            `json:"name" validate:"required,nonempty"`
	Email    string            `json:"email" validate:"required"`
	Metadata map[string]string `json:"metadata"`
	Tags     []string          `json:"tags"`
	Active   bool              `json:"active"`
}

type SearchQuery struct {
	Term    string            `json:"term" validate:"required,nonempty"`
	Filters map[string]string `json:"filters"`
	Limit   int               `json:"limit"`
}

type SearchResult struct {
	Results []UserProfile `json:"results"`
	Total   int           `json:"total"`
	Query   SearchQuery   `json:"query"`
}

// Test Server.RegisterTypedTool method
func TestServer_RegisterTypedTool(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Test successful registration with new function
	err := RegisterTypedToolWithServer(server, "calculate", "Perform arithmetic operations", 
		func(ctx context.Context, args CalculateArgs) (CalculateResult, error) {
			switch args.Operation {
			case "add":
				return CalculateResult{Result: args.A + args.B}, nil
			case "subtract":
				return CalculateResult{Result: args.A - args.B}, nil
			case "multiply":
				return CalculateResult{Result: args.A * args.B}, nil
			case "divide":
				if args.B == 0 {
					return CalculateResult{Error: "division by zero"}, nil
				}
				return CalculateResult{Result: args.A / args.B}, nil
			default:
				return CalculateResult{Error: "unsupported operation"}, nil
			}
		})

	if err != nil {
		t.Fatalf("Failed to register typed tool: %v", err)
	}

	// Verify tool was registered
	toolDef, exists := server.tools["calculate"]
	if !exists {
		t.Fatal("Tool 'calculate' was not registered")
	}

	if toolDef.tool.Name != "calculate" {
		t.Errorf("Expected tool name 'calculate', got %s", toolDef.tool.Name)
	}

	// Verify schema generation
	var schema map[string]any
	err = json.Unmarshal(toolDef.tool.InputSchema, &schema)
	if err != nil {
		t.Fatalf("Failed to unmarshal input schema: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties is not a map")
	}

	// Check expected properties
	expectedProps := map[string]string{
		"operation": "string",
		"a":         "number", 
		"b":         "number",
	}

	for propName, expectedType := range expectedProps {
		prop, exists := properties[propName].(map[string]any)
		if !exists {
			t.Errorf("Property '%s' not found in schema", propName)
			continue
		}

		if prop["type"] != expectedType {
			t.Errorf("Property '%s' expected type '%s', got %v", propName, expectedType, prop["type"])
		}
	}
}

// Test backward compatibility with global RegisterTypedTool function
func TestRegisterTypedTool_BackwardCompatibility(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Test that the global function still works
	err := RegisterTypedTool(server, "legacy_add", "Legacy add function",
		func(ctx context.Context, args CalculateArgs) (CalculateResult, error) {
			return CalculateResult{Result: args.A + args.B}, nil
		})

	if err != nil {
		t.Fatalf("Backward compatibility test failed: %v", err)
	}

	// Verify tool was registered
	_, exists := server.tools["legacy_add"]
	if !exists {
		t.Fatal("Legacy tool registration failed")
	}
}

// Test Client.CallToolTyped method
func TestClient_CallToolTyped(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create server with typed tool
	server := NewServer("calc-server", "1.0.0")
	err := RegisterTypedToolWithServer(server, "calculate", "Perform calculations",
		func(ctx context.Context, args CalculateArgs) (CalculateResult, error) {
			switch args.Operation {
			case "add":
				return CalculateResult{Result: args.A + args.B}, nil
			case "multiply":
				return CalculateResult{Result: args.A * args.B}, nil
			default:
				return CalculateResult{Error: "unsupported operation"}, nil
			}
		})
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Create in-memory transport pair
	serverTransport, clientTransport := createInMemoryTransportPair()

	// Start server
	serverCtx, serverCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer serverCancel()

	go func() {
		if err := server.Serve(serverCtx, serverTransport); err != nil && serverCtx.Err() == nil {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client
	client, err := NewClient(clientTransport)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = client.Initialize(ctx, InitializeRequest{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Test type-safe tool call
	args := CalculateArgs{
		Operation: "add",
		A:         10.5,
		B:         5.3,
	}

	result, err := CallToolTyped[CalculateArgs, CalculateResult](client, ctx, "calculate", args)
	if err != nil {
		t.Fatalf("Type-safe tool call failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	expected := 15.8
	if result.Result != expected {
		t.Errorf("Expected result %f, got %f", expected, result.Result)
	}

	// Test with multiplication
	args.Operation = "multiply"
	args.A = 4
	args.B = 3

	result, err = CallToolTyped[CalculateArgs, CalculateResult](client, ctx, "calculate", args)
	if err != nil {
		t.Fatalf("Type-safe multiply call failed: %v", err)
	}

	if result.Result != 12 {
		t.Errorf("Expected multiply result 12, got %f", result.Result)
	}
}

// Test Handler and HandlerChain
func TestHandlerChain(t *testing.T) {
	// Create a simple handler
	calculator := HandlerFunc[CalculateArgs, CalculateResult](
		func(ctx context.Context, req CalculateArgs) (CalculateResult, error) {
			switch req.Operation {
			case "add":
				return CalculateResult{Result: req.A + req.B}, nil
			default:
				return CalculateResult{Error: "unsupported operation"}, nil
			}
		})

	// Create validation function
	validator := ValidationFunc[CalculateArgs](func(ctx context.Context, value CalculateArgs) error {
		if value.Operation == "" {
			return fmt.Errorf("operation is required")
		}
		return nil
	})

	// Create handler chain
	chain := NewHandlerChain(calculator).WithValidation(validator)

	// Test valid request
	ctx := context.Background()
	validArgs := CalculateArgs{Operation: "add", A: 5, B: 3}

	result, err := chain.Handle(ctx, validArgs)
	if err != nil {
		t.Fatalf("Handler chain failed for valid request: %v", err)
	}

	if result.Result != 8 {
		t.Errorf("Expected result 8, got %f", result.Result)
	}

	// Test invalid request (missing operation)
	invalidArgs := CalculateArgs{A: 5, B: 3}

	_, err = chain.Handle(ctx, invalidArgs)
	if err == nil {
		t.Fatal("Expected validation error for invalid request")
	}

	if !typedContains(err.Error(), "operation is required") {
		t.Errorf("Expected validation error about operation, got: %v", err)
	}
}

// Test StructValidator
func TestStructValidator(t *testing.T) {
	validator := NewStructValidator()

	// Test with valid struct
	ctx := context.Background()
	validProfile := UserProfile{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
		Active: true,
	}

	err := validator.Validate(ctx, validProfile)
	if err != nil {
		t.Errorf("Validation should pass for valid struct: %v", err)
	}

	// Test with struct missing required field (using struct tags)
	invalidProfile := UserProfile{
		Name:  "", // Empty name should fail validation
		Email: "john@example.com",
	}

	err = validator.Validate(ctx, invalidProfile)
	if err == nil {
		t.Error("Expected validation error for invalid struct")
	}
}

// Test EnhancedSchemaGenerator
func TestEnhancedSchemaGenerator(t *testing.T) {
	generator := NewEnhancedSchemaGenerator()

	// Test schema generation
	schema, err := GenerateSchemaWithGenerator[CalculateArgs](generator)
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	var schemaMap map[string]any
	err = json.Unmarshal(schema, &schemaMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	if schemaMap["type"] != "object" {
		t.Errorf("Expected schema type 'object', got %v", schemaMap["type"])
	}

	// Test OpenAPI schema generation
	openAPISchema, err := GenerateOpenAPISchemaWithGenerator[CalculateArgs](generator)
	if err != nil {
		t.Fatalf("Failed to generate OpenAPI schema: %v", err)
	}

	if openAPISchema["$schema"] == nil {
		t.Error("Expected $schema field in OpenAPI schema")
	}

	// Test schema comparison
	schema2, err := GenerateSchemaWithGenerator[CalculateResult](generator)
	if err != nil {
		t.Fatalf("Failed to generate second schema: %v", err)
	}

	compatible, differences, err := generator.CompareSchemas(schema, schema2)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// These schemas should be different
	if compatible {
		t.Error("Expected schemas to be incompatible")
	}

	if len(differences) == 0 {
		t.Error("Expected differences between schemas")
	}
}

// Test global convenience functions
func TestGlobalConvenienceFunctions(t *testing.T) {
	// Test GenerateTypedSchema
	schema, err := GenerateTypedSchema[UserProfile]()
	if err != nil {
		t.Fatalf("Failed to generate typed schema: %v", err)
	}

	var schemaMap map[string]any
	err = json.Unmarshal(schema, &schemaMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	if schemaMap["type"] != "object" {
		t.Errorf("Expected schema type 'object', got %v", schemaMap["type"])
	}

	// Test GenerateOpenAPISchema
	openAPISchema, err := GenerateOpenAPISchema[UserProfile]()
	if err != nil {
		t.Fatalf("Failed to generate OpenAPI schema: %v", err)
	}

	if openAPISchema["$schema"] == nil {
		t.Error("Expected $schema field in OpenAPI schema")
	}
}

// Test complex type handling
func TestComplexTypeHandling(t *testing.T) {
	server := NewServer("search-server", "1.0.0")

	// Register tool with complex types
	err := RegisterTypedToolWithServer(server, "search", "Search for users",
		func(ctx context.Context, query SearchQuery) (SearchResult, error) {
			// Mock search implementation
			users := []UserProfile{
				{ID: 1, Name: "John Doe", Email: "john@example.com", Active: true},
				{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Active: true},
			}

			// Simple filtering
			var results []UserProfile
			for _, user := range users {
				if typedContains(user.Name, query.Term) || typedContains(user.Email, query.Term) {
					results = append(results, user)
				}
			}

			// Apply limit
			if query.Limit > 0 && len(results) > query.Limit {
				results = results[:query.Limit]
			}

			return SearchResult{
				Results: results,
				Total:   len(results),
				Query:   query,
			}, nil
		})

	if err != nil {
		t.Fatalf("Failed to register search tool: %v", err)
	}

	// Verify complex schema generation
	toolDef := server.tools["search"]
	var schema map[string]any
	err = json.Unmarshal(toolDef.tool.InputSchema, &schema)
	if err != nil {
		t.Fatalf("Failed to unmarshal complex schema: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties is not a map")
	}

	// Check nested object schema (filters)
	filtersProperty, exists := properties["filters"].(map[string]any)
	if !exists {
		t.Fatal("Property 'filters' not found")
	}

	if filtersProperty["type"] != "object" {
		t.Errorf("Expected 'filters' type 'object', got %v", filtersProperty["type"])
	}

	// Verify additionalProperties for map[string]string
	if filtersProperty["additionalProperties"] == nil {
		t.Error("Expected additionalProperties for map type")
	}
}

// Helper function to create in-memory transport pair for testing
func createInMemoryTransportPair() (serverTransport, clientTransport Transport) {
	// This is a simplified implementation for testing
	// In a real implementation, you'd use something like in-memory pipes
	return StdioTransport(), StdioTransport()
}

// Helper function for string contains check  
func typedContains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > 0 && len(substr) > 0 &&
			(str[:len(substr)] == substr ||
				str[len(str)-len(substr):] == substr ||
				typedContainsSubstring(str, substr))))
}

func typedContainsSubstring(str, substr string) bool {
	if len(substr) > len(str) {
		return false
	}
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}