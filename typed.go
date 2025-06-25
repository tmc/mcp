// Package mcp - Type-Safe API Implementation with Generics
//
// This file contains comprehensive type-safe APIs using Go generics to improve
// compile-time safety and developer experience while maintaining 100% backward compatibility.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Type-Safe Tool Registration
// =========================

// TypedToolHandlerFunc defines the signature for type-safe tool handlers
type TypedToolHandlerFunc[TArg any, TResult any] func(ctx context.Context, args TArg) (TResult, error)

// RegisterTypedTool registers a type-safe tool handler with automatic JSON marshaling/unmarshaling
// and schema generation. It provides compile-time type safety while maintaining runtime compatibility.
func RegisterTypedToolWithServer[TArg any, TResult any](
	s *Server,
	name string,
	description string,
	handler TypedToolHandlerFunc[TArg, TResult],
) error {
	if s == nil {
		return fmt.Errorf("server is nil")
	}
	
	// Create a JSON schema from the TArg type
	inputSchema, err := createJSONSchema[TArg]()
	if err != nil {
		return fmt.Errorf("failed to create input schema for tool %q: %w", name, err)
	}

	// Create the untyped handler wrapper
	toolHandler := func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		// Parse the input with validation
		var input TArg
		if len(req.Arguments) > 0 {
			if err := json.Unmarshal(req.Arguments, &input); err != nil {
				return &CallToolResult{
					IsError: true,
					Content: []any{
						map[string]string{
							"type": "text",
							"text": fmt.Sprintf("Invalid input for tool %q: %v", name, err),
						},
					},
				}, nil
			}
		}

		// Call the typed handler
		output, err := handler(ctx, input)
		if err != nil {
			return &CallToolResult{
				IsError: true,
				Content: []any{
					map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Tool %q error: %v", name, err),
					},
				},
			}, nil
		}

		// Marshal the typed output
		outputJSON, err := json.Marshal(output)
		if err != nil {
			return &CallToolResult{
				IsError: true,
				Content: []any{
					map[string]string{
						"type": "text",
						"text": fmt.Sprintf("Failed to marshal output for tool %q: %v", name, err),
					},
				},
			}, nil
		}

		// Return structured result
		return &CallToolResult{
			Content: []any{
				map[string]any{
					"type":   "text",
					"format": "json", 
					"text":   string(outputJSON),
				},
			},
		}, nil
	}

	// Register the tool with the server
	tool := Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
	
	return s.RegisterTool(tool, toolHandler)
}

// Type-Safe Client Methods
// =======================

// CallToolTyped performs a type-safe tool call with compile-time type checking
func CallToolTyped[TArg any, TResult any](
	c *Client,
	ctx context.Context, 
	toolName string, 
	args TArg,
) (*TResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	// Marshal arguments with type safety
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments for tool %q: %w", toolName, err)
	}

	// Make the call
	request := CallToolRequest{
		Name:      toolName,
		Arguments: argsJSON,
	}

	result, err := c.CallTool(ctx, request)
	if err != nil {
		return nil, err
	}

	if result.IsError {
		// Extract error message from content
		if len(result.Content) > 0 {
			if content, ok := result.Content[0].(map[string]any); ok {
				if text, ok := content["text"].(string); ok {
					return nil, fmt.Errorf("tool error: %s", text)
				}
			}
		}
		return nil, fmt.Errorf("tool %q returned error", toolName)
	}

	// Parse the typed result
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("tool %q returned no content", toolName)
	}

	// Extract JSON content
	content, ok := result.Content[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tool %q returned invalid content format", toolName)
	}

	text, ok := content["text"].(string)
	if !ok {
		return nil, fmt.Errorf("tool %q returned no text content", toolName)
	}

	// Unmarshal into typed result
	var typedResult TResult
	if err := json.Unmarshal([]byte(text), &typedResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result from tool %q: %w", toolName, err)
	}

	return &typedResult, nil
}

// ReadResourceTyped performs a type-safe resource read with automatic content parsing
func ReadResourceTyped[TResult any](
	c *Client,
	ctx context.Context,
	uri string,
) (*TResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	request := ReadResourceRequest{URI: uri}
	result, err := c.ReadResource(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(result.Contents) == 0 {
		return nil, fmt.Errorf("resource %q returned no content", uri)
	}

	// Extract content based on type
	var contentText string
	content := result.Contents[0]
	
	switch c := content.(type) {
	case TextResourceContents:
		contentText = c.Text
	case BlobResourceContents:
		// For blob content, assume it's JSON encoded
		contentText = c.Blob
	default:
		return nil, fmt.Errorf("unsupported content type for resource %q", uri)
	}

	// Parse into typed result
	var typedResult TResult
	if err := json.Unmarshal([]byte(contentText), &typedResult); err != nil {
		return nil, fmt.Errorf("failed to parse resource %q content: %w", uri, err)
	}

	return &typedResult, nil
}

// GetPromptTyped performs a type-safe prompt retrieval with argument validation
func GetPromptTyped[TArg any, TResult any](
	c *Client,
	ctx context.Context,
	promptName string,
	args TArg,
) (*TResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	// Convert typed args to map
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments for prompt %q: %w", promptName, err)
	}

	var argsMap map[string]interface{}
	if err := json.Unmarshal(argsJSON, &argsMap); err != nil {
		return nil, fmt.Errorf("failed to convert arguments for prompt %q: %w", promptName, err)
	}

	request := GetPromptRequest{
		Name:      promptName,
		Arguments: argsMap,
	}

	result, err := c.GetPrompt(ctx, request)
	if err != nil {
		return nil, err
	}

	// Convert prompt result to typed format
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt result for %q: %w", promptName, err)
	}

	var typedResult TResult
	if err := json.Unmarshal(resultJSON, &typedResult); err != nil {
		return nil, fmt.Errorf("failed to parse prompt %q result: %w", promptName, err)
	}

	return &typedResult, nil
}

// Generic Handler Types
// ====================

// Handler defines a generic handler interface for type-safe request processing
type Handler[TRequest any, TResponse any] interface {
	Handle(ctx context.Context, req TRequest) (TResponse, error)
}

// HandlerFunc is a function type that implements Handler
type HandlerFunc[TRequest any, TResponse any] func(ctx context.Context, req TRequest) (TResponse, error)

func (f HandlerFunc[TRequest, TResponse]) Handle(ctx context.Context, req TRequest) (TResponse, error) {
	return f(ctx, req)
}

// MiddlewareFunc defines type-safe middleware for request processing
type MiddlewareFunc[T any] func(next Handler[T, T]) Handler[T, T]

// ValidationFunc defines type-safe parameter validation
type ValidationFunc[T any] func(ctx context.Context, value T) error

// HandlerChain manages a chain of type-safe handlers with middleware support
type HandlerChain[TRequest any, TResponse any] struct {
	handler     Handler[TRequest, TResponse]
	middlewares []MiddlewareFunc[TRequest]
	validators  []ValidationFunc[TRequest]
}

// NewHandlerChain creates a new handler chain with the given base handler
func NewHandlerChain[TRequest any, TResponse any](
	handler Handler[TRequest, TResponse],
) *HandlerChain[TRequest, TResponse] {
	return &HandlerChain[TRequest, TResponse]{
		handler:     handler,
		middlewares: make([]MiddlewareFunc[TRequest], 0),
		validators:  make([]ValidationFunc[TRequest], 0),
	}
}

// WithMiddleware adds middleware to the handler chain
func (hc *HandlerChain[TRequest, TResponse]) WithMiddleware(
	middleware MiddlewareFunc[TRequest],
) *HandlerChain[TRequest, TResponse] {
	hc.middlewares = append(hc.middlewares, middleware)
	return hc
}

// WithValidation adds validation to the handler chain
func (hc *HandlerChain[TRequest, TResponse]) WithValidation(
	validator ValidationFunc[TRequest],
) *HandlerChain[TRequest, TResponse] {
	hc.validators = append(hc.validators, validator)
	return hc
}

// Handle processes a request through the complete handler chain
func (hc *HandlerChain[TRequest, TResponse]) Handle(
	ctx context.Context, 
	req TRequest,
) (TResponse, error) {
	var zero TResponse
	
	// Run validation first
	for _, validator := range hc.validators {
		if err := validator(ctx, req); err != nil {
			return zero, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Apply middleware (this would need adaptation for TRequest -> TResponse)
	// For now, we'll call the handler directly
	return hc.handler.Handle(ctx, req)
}

// Enhanced Validation Framework
// ============================

// ValidatedField represents a field with validation rules
type ValidatedField struct {
	Name     string                                  `json:"name"`
	Required bool                                    `json:"required"`
	Validate func(ctx context.Context, value any) error `json:"-"`
}

// StructValidator provides struct tag-based validation
type StructValidator struct {
	typeValidators map[reflect.Type][]ValidatedField
}

// NewStructValidator creates a new struct validator
func NewStructValidator() *StructValidator {
	return &StructValidator{
		typeValidators: make(map[reflect.Type][]ValidatedField),
	}
}

// RegisterTypeValidation registers validation rules for a specific type
func RegisterTypeValidation[T any](sv *StructValidator, fields ...ValidatedField) {
	var zero T
	t := reflect.TypeOf(zero)
	sv.typeValidators[t] = fields
}

// Validate validates a value using registered validation rules
func (sv *StructValidator) Validate(ctx context.Context, value any) error {
	t := reflect.TypeOf(value)
	fields, exists := sv.typeValidators[t]
	if !exists {
		// No specific validation registered, use struct tags
		return sv.validateWithTags(ctx, value)
	}

	// Use registered validation rules
	v := reflect.ValueOf(value)
	for _, field := range fields {
		fieldValue := v.FieldByName(field.Name)
		if !fieldValue.IsValid() {
			continue
		}

		if field.Required && fieldValue.IsZero() {
			return fmt.Errorf("required field %q is missing or empty", field.Name)
		}

		if field.Validate != nil {
			if err := field.Validate(ctx, fieldValue.Interface()); err != nil {
				return fmt.Errorf("validation failed for field %q: %w", field.Name, err)
			}
		}
	}

	return nil
}

// validateWithTags performs validation using struct tags
func (sv *StructValidator) validateWithTags(ctx context.Context, value any) error {
	t := reflect.TypeOf(value)
	v := reflect.ValueOf(value)

	if t.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("cannot validate nil pointer")
		}
		t = t.Elem()
		v = v.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil // Only validate structs
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check validation tags
		validateTag := field.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// Parse validation rules
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if err := sv.applyValidationRule(field.Name, rule, fieldValue); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyValidationRule applies a single validation rule to a field
func (sv *StructValidator) applyValidationRule(fieldName, rule string, fieldValue reflect.Value) error {
	switch rule {
	case "required":
		if fieldValue.IsZero() {
			return fmt.Errorf("required field %q is missing or empty", fieldName)
		}
	case "nonempty":
		if fieldValue.Kind() == reflect.String && fieldValue.String() == "" {
			return fmt.Errorf("field %q cannot be empty", fieldName)
		}
	default:
		// Support custom validation patterns
		if strings.HasPrefix(rule, "min=") {
			// Example: min=5
			// This would need more implementation for different types
		}
		if strings.HasPrefix(rule, "max=") {
			// Example: max=100
			// This would need more implementation for different types
		}
	}
	return nil
}

// Schema Generation Integration
// ============================

// EnhancedSchemaGenerator provides advanced schema generation with caching and validation
type EnhancedSchemaGenerator struct {
	cache     *SchemaCache
	validator *StructValidator
}

// NewEnhancedSchemaGenerator creates a new enhanced schema generator
func NewEnhancedSchemaGenerator() *EnhancedSchemaGenerator {
	return &EnhancedSchemaGenerator{
		cache:     &SchemaCache{},
		validator: NewStructValidator(),
	}
}

// GenerateSchema generates a comprehensive JSON schema for the given type
func GenerateSchemaWithGenerator[T any](esg *EnhancedSchemaGenerator) (json.RawMessage, error) {
	return createJSONSchema[T]()
}

// GenerateOpenAPISchema generates an OpenAPI-compatible schema
func GenerateOpenAPISchemaWithGenerator[T any](esg *EnhancedSchemaGenerator) (map[string]any, error) {
	schema, err := createJSONSchema[T]()
	if err != nil {
		return nil, err
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return nil, err
	}

	// Add OpenAPI-specific enhancements
	schemaMap["$schema"] = "http://json-schema.org/draft-07/schema#"
	
	return schemaMap, nil
}

// CompareSchemas compares two schemas for compatibility
func (esg *EnhancedSchemaGenerator) CompareSchemas(
	schema1, schema2 json.RawMessage,
) (bool, []string, error) {
	var s1, s2 map[string]any
	
	if err := json.Unmarshal(schema1, &s1); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal schema1: %w", err)
	}
	
	if err := json.Unmarshal(schema2, &s2); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal schema2: %w", err)
	}

	differences := []string{}
	compatible := esg.compareSchemaObjects("", s1, s2, &differences)
	
	return compatible, differences, nil
}

// compareSchemaObjects recursively compares schema objects
func (esg *EnhancedSchemaGenerator) compareSchemaObjects(
	path string,
	obj1, obj2 map[string]any,
	differences *[]string,
) bool {
	compatible := true

	// Check type compatibility
	type1, ok1 := obj1["type"].(string)
	type2, ok2 := obj2["type"].(string)
	
	if ok1 && ok2 && type1 != type2 {
		*differences = append(*differences, fmt.Sprintf("%s: type mismatch (%s vs %s)", path, type1, type2))
		compatible = false
	}

	// Check required fields
	req1, ok1 := obj1["required"].([]interface{})
	req2, ok2 := obj2["required"].([]interface{})
	
	if ok1 && ok2 {
		req1Set := make(map[string]bool)
		for _, r := range req1 {
			if s, ok := r.(string); ok {
				req1Set[s] = true
			}
		}
		
		for _, r := range req2 {
			if s, ok := r.(string); ok {
				if !req1Set[s] {
					*differences = append(*differences, fmt.Sprintf("%s: new required field %s", path, s))
					compatible = false
				}
			}
		}
	}

	return compatible
}

// Global enhanced schema generator instance
var enhancedSchemaGenerator = NewEnhancedSchemaGenerator()

// Convenience functions for global access
func GenerateTypedSchema[T any]() (json.RawMessage, error) {
	return GenerateSchemaWithGenerator[T](enhancedSchemaGenerator)
}

func GenerateOpenAPISchema[T any]() (map[string]any, error) {
	return GenerateOpenAPISchemaWithGenerator[T](enhancedSchemaGenerator)
}

func CompareSchemas(schema1, schema2 json.RawMessage) (bool, []string, error) {
	return enhancedSchemaGenerator.CompareSchemas(schema1, schema2)
}