// Package mcp provides a comprehensive Go implementation of the Model Context Protocol (MCP).
//
// MCP is a protocol for enabling AI assistants to securely access external data sources
// and tools. This package includes client and server implementations, transport abstractions,
// and utilities for building MCP-enabled applications.
//
// Key features:
//   - Full MCP protocol implementation with type safety
//   - Multiple transport options (stdio, SSE, WebSocket)
//   - Automatic JSON-RPC handling with proper error propagation
//   - Context-aware request cancellation
//   - Extensible tool, prompt, and resource registration
//   - Comprehensive testing and debugging utilities
//
// Example server:
//
//	server := mcp.NewServer("my-server", "1.0.0")
//	server.RegisterTool(mcp.Tool{
//		Name: "echo",
//		Description: "Echo back the input",
//	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//		return &mcp.CallToolResult{
//			Content: []any{map[string]string{"type": "text", "text": string(req.Arguments)}},
//		}, nil
//	})
//	server.Serve(context.Background(), mcp.StdioTransport())
//
// Example client:
//
//	client, _ := mcp.NewClient(mcp.StdioTransport())
//	client.Initialize(ctx, mcp.InitializeRequest{
//		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
//		ClientInfo: mcp.Implementation{Name: "my-client", Version: "1.0.0"},
//	})
//	tools, _ := client.ListTools(ctx, mcp.ListToolsRequest{})
//
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Support for creating typed tool handlers that automatically handle JSON serialization/deserialization.
// This allows for a more idiomatic Go API when registering tools.

// RegisterTypedTool registers a type-safe tool handler with automatic JSON marshaling/unmarshaling.
// Input is the Go type for the tool's input, and Output is the Go type for the tool's output.
// 
// Deprecated: Use server.RegisterTypedTool() method instead for better encapsulation.
// This function is maintained for backward compatibility.
func RegisterTypedTool[Input any, Output any](
	server *Server,
	name string,
	description string,
	handler func(context.Context, Input) (Output, error),
) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}
	
	// Use the new typed tool registration function
	return RegisterTypedToolWithServer(server, name, description, handler)
}

// SchemaCache provides thread-safe caching for JSON schemas to avoid regenerating
// the same schema multiple times. This significantly improves performance when
// registering multiple tools with the same input types.
type SchemaCache struct {
	mu     sync.RWMutex
	schemas map[string]json.RawMessage
}

// GetOrCreate retrieves a cached schema or generates a new one using the provided generator
func (c *SchemaCache) GetOrCreate(typeKey string, generator func() (json.RawMessage, error)) (json.RawMessage, error) {
	// Check cache first (read lock)
	c.mu.RLock()
	if schema, exists := c.schemas[typeKey]; exists {
		c.mu.RUnlock()
		return schema, nil
	}
	c.mu.RUnlock()
	
	// Generate new schema (write lock)
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check pattern to avoid race conditions
	if schema, exists := c.schemas[typeKey]; exists {
		return schema, nil
	}
	
	// Generate and cache the schema
	schema, err := generator()
	if err != nil {
		return nil, err
	}
	
	if c.schemas == nil {
		c.schemas = make(map[string]json.RawMessage)
	}
	c.schemas[typeKey] = schema
	
	return schema, nil
}

// Global schema cache instance
var schemaCache = &SchemaCache{}

// createJSONSchema generates a JSON schema representation for the given Go type.
// This function provides automatic schema generation for tool input validation,
// supporting primitive types (string, number, boolean) and complex objects.
// Optimized version that avoids marshal→unmarshal roundtrips and uses reflection
// for better performance and type introspection.
func createJSONSchema[T any]() (json.RawMessage, error) {
	var example T
	typeKey := fmt.Sprintf("%T", example)
	
	return schemaCache.GetOrCreate(typeKey, func() (json.RawMessage, error) {
		return generateJSONSchemaReflection[T]()
	})
}

// generateJSONSchemaReflection creates a JSON schema using reflection instead of
// marshal→unmarshal roundtrips for better performance and accuracy.
func generateJSONSchemaReflection[T any]() (json.RawMessage, error) {
	var example T
	t := reflect.TypeOf(example)
	
	schema := generateSchemaForType(t)
	return json.Marshal(schema)
}

// generateSchemaForType recursively generates schema for a reflect.Type
func generateSchemaForType(t reflect.Type) map[string]any {
	// Handle pointers by dereferencing
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		 reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Array, reflect.Slice:
		return map[string]any{
			"type": "array",
			"items": generateSchemaForType(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type": "object",
			"additionalProperties": generateSchemaForType(t.Elem()),
		}
	case reflect.Struct:
		return generateStructSchema(t)
	case reflect.Interface:
		// For interface{} types, we can't determine the schema statically
		return map[string]any{"type": "object"}
	default:
		// Default to object for unknown types
		return map[string]any{"type": "object"}
	}
}

// generateStructSchema generates a JSON schema for a struct type using field introspection
func generateStructSchema(t reflect.Type) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	
	properties := schema["properties"].(map[string]any)
	required := []string{}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		
		// Get JSON tag name or use field name
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		
		if jsonTag != "" {
			// Parse json tag (e.g., "name,omitempty")
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			} else if parts[0] == "-" {
				// Skip fields marked with json:"-"
				continue
			}
			
			// Check if field is required (no "omitempty")
			isRequired := true
			for _, part := range parts[1:] {
				if part == "omitempty" {
					isRequired = false
					break
				}
			}
			
			if isRequired {
				required = append(required, fieldName)
			}
		} else {
			// No json tag, assume required
			required = append(required, fieldName)
		}
		
		// Generate schema for field type
		fieldSchema := generateSchemaForType(field.Type)
		
		// Add description from field comments if available
		// This could be enhanced with custom tags like `description:"..."`
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema["description"] = desc
		}
		
		properties[fieldName] = fieldSchema
	}
	
	if len(required) > 0 {
		schema["required"] = required
	}
	
	return schema
}
