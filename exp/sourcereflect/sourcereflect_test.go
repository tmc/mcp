package sourcereflect

import (
	"reflect"
	"testing"
)

type TestUser struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Age      int      `json:"age,omitempty"`
	Tags     []string `json:"tags"`
	Settings map[string]interface{} `json:"settings"`
}

func TestFromType(t *testing.T) {
	schema, err := FromType(reflect.TypeOf(TestUser{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if schema.Title != "TestUser" {
		t.Errorf("Expected title 'TestUser', got '%s'", schema.Title)
	}

	// Check properties
	if len(schema.Properties) != 6 {
		t.Errorf("Expected 6 properties, got %d", len(schema.Properties))
	}

	// Check ID property
	if idProp, ok := schema.Properties["id"]; ok {
		if idProp.Type != "integer" {
			t.Errorf("Expected id type 'integer', got '%s'", idProp.Type)
		}
	} else {
		t.Error("Property 'id' not found")
	}

	// Check required fields
	requiredMap := make(map[string]bool)
	for _, field := range schema.Required {
		requiredMap[field] = true
	}

	if !requiredMap["id"] {
		t.Error("Field 'id' should be required")
	}

	if requiredMap["age"] {
		t.Error("Field 'age' should not be required (has omitempty)")
	}
}

func TestFromValue(t *testing.T) {
	user := TestUser{
		ID:    1,
		Name:  "John",
		Email: "john@example.com",
		Tags:  []string{"admin", "user"},
	}

	schema, err := FromValue(user)
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}
}

func TestPrimitiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "hello", "string"},
		{"int", 42, "integer"},
		{"float", 3.14, "number"},
		{"bool", true, "boolean"},
		{"slice", []int{1, 2, 3}, "array"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := FromValue(tt.value)
			if err != nil {
				t.Fatalf("Failed to generate schema: %v", err)
			}

			if schema.Type != tt.expected {
				t.Errorf("Expected type '%s', got '%s'", tt.expected, schema.Type)
			}
		})
	}
}

func TestSchemaToJSON(t *testing.T) {
	schema, err := FromType(reflect.TypeOf(TestUser{}))
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	jsonStr, err := schema.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Test pretty print
	prettyJSON, err := schema.ToPrettyJSON()
	if err != nil {
		t.Fatalf("Failed to convert to pretty JSON: %v", err)
	}

	if len(prettyJSON) <= len(jsonStr) {
		t.Error("Pretty JSON should be longer than compact JSON")
	}
}

func TestSchemaFromCaller(t *testing.T) {
	schema, err := SchemaFromCaller(TestUser{})
	if err != nil {
		t.Fatalf("Failed to generate schema with caller info: %v", err)
	}

	if schema.Additional == nil {
		t.Fatal("Expected additional metadata")
	}

	if _, ok := schema.Additional["$sourceLocation"]; !ok {
		t.Error("Expected source location metadata")
	}
}

func TestSchemaBuilder(t *testing.T) {
	schema := NewSchemaBuilder().
		WithType("object").
		WithTitle("CustomSchema").
		WithProperty("name", &Schema{Type: "string"}).
		WithProperty("age", &Schema{Type: "integer"}).
		WithRequired("name").
		Build()

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if schema.Title != "CustomSchema" {
		t.Errorf("Expected title 'CustomSchema', got '%s'", schema.Title)
	}

	if len(schema.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(schema.Properties))
	}

	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Error("Expected 'name' to be required")
	}
}