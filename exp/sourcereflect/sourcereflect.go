// Package sourcereflect provides functionality to generate JSON schema from Go types
package sourcereflect

import (
	"fmt"
	"reflect"
	"strings"
)

// Schema represents a JSON Schema
type Schema struct {
	Type       string                 `json:"type,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Properties map[string]*Schema     `json:"properties,omitempty"`
	Items      *Schema                `json:"items,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Enum       []interface{}          `json:"enum,omitempty"`
	Format     string                 `json:"format,omitempty"`
	Pattern    string                 `json:"pattern,omitempty"`
	Reference  string                 `json:"$ref,omitempty"`
	Additional map[string]interface{} `json:"-"`
}

// TypeToSchema converts a Go type to a JSON schema
func TypeToSchema(t reflect.Type) (*Schema, error) {
	switch t.Kind() {
	case reflect.Struct:
		return structToSchema(t)
	case reflect.Slice, reflect.Array:
		return arrayToSchema(t)
	case reflect.Map:
		return mapToSchema(t)
	case reflect.Ptr:
		return TypeToSchema(t.Elem())
	case reflect.Interface:
		return &Schema{Type: "object"}, nil
	default:
		return primitiveToSchema(t)
	}
}

// FromValue generates a JSON schema from a runtime value
func FromValue(v interface{}) (*Schema, error) {
	if v == nil {
		return &Schema{Type: "null"}, nil
	}
	return TypeToSchema(reflect.TypeOf(v))
}

// FromType generates a JSON schema from a reflect.Type
func FromType(t reflect.Type) (*Schema, error) {
	return TypeToSchema(t)
}

func structToSchema(t reflect.Type) (*Schema, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %v", t.Kind())
	}

	schema := &Schema{
		Type:       "object",
		Title:      t.Name(),
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		isRequired := true

		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			// Check if field is omitempty
			for _, part := range parts[1:] {
				if part == "omitempty" {
					isRequired = false
					break
				}
			}
		}

		if isRequired {
			schema.Required = append(schema.Required, fieldName)
		}

		fieldSchema, err := TypeToSchema(field.Type)
		if err != nil {
			return nil, fmt.Errorf("error processing field %s: %w", field.Name, err)
		}

		// Add field documentation if available
		if docTag := field.Tag.Get("doc"); docTag != "" {
			if fieldSchema.Additional == nil {
				fieldSchema.Additional = make(map[string]interface{})
			}
			fieldSchema.Additional["description"] = docTag
		}

		schema.Properties[fieldName] = fieldSchema
	}

	return schema, nil
}

func arrayToSchema(t reflect.Type) (*Schema, error) {
	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected slice or array, got %v", t.Kind())
	}

	itemSchema, err := TypeToSchema(t.Elem())
	if err != nil {
		return nil, fmt.Errorf("error processing array element: %w", err)
	}

	return &Schema{
		Type:  "array",
		Items: itemSchema,
	}, nil
}

func mapToSchema(t reflect.Type) (*Schema, error) {
	if t.Kind() != reflect.Map {
		return nil, fmt.Errorf("expected map, got %v", t.Kind())
	}

	// JSON objects have string keys
	if t.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("map keys must be strings for JSON schema")
	}

	valueSchema, err := TypeToSchema(t.Elem())
	if err != nil {
		return nil, fmt.Errorf("error processing map value: %w", err)
	}

	return &Schema{
		Type: "object",
		Additional: map[string]interface{}{
			"additionalProperties": valueSchema,
		},
	}, nil
}

func primitiveToSchema(t reflect.Type) (*Schema, error) {
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer"}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}, nil
	case reflect.Bool:
		return &Schema{Type: "boolean"}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", t.Kind())
	}
}