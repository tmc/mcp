package sourcereflect

import (
	"encoding/json"
	"fmt"
)

// ToJSON converts a schema to JSON string
func (s *Schema) ToJSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}
	return string(data), nil
}

// ToPrettyJSON converts a schema to pretty-printed JSON string
func (s *Schema) ToPrettyJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}
	return string(data), nil
}

// MarshalJSON implements custom JSON marshaling for Schema
func (s *Schema) MarshalJSON() ([]byte, error) {
	// Create a temporary struct with all fields
	type Alias Schema
	temp := &struct {
		*Alias
		AdditionalProperties interface{} `json:"additionalProperties,omitempty"`
		Description          string      `json:"description,omitempty"`
		SourceLocation       interface{} `json:"$sourceLocation,omitempty"`
	}{
		Alias: (*Alias)(s),
	}

	// Extract special fields from Additional map
	if s.Additional != nil {
		if ap, ok := s.Additional["additionalProperties"]; ok {
			temp.AdditionalProperties = ap
		}
		if desc, ok := s.Additional["description"]; ok {
			if descStr, ok := desc.(string); ok {
				temp.Description = descStr
			}
		}
		if loc, ok := s.Additional["$sourceLocation"]; ok {
			temp.SourceLocation = loc
		}
	}

	return json.Marshal(temp)
}

// Clone creates a deep copy of the schema
func (s *Schema) Clone() *Schema {
	if s == nil {
		return nil
	}

	clone := &Schema{
		Type:      s.Type,
		Title:     s.Title,
		Format:    s.Format,
		Pattern:   s.Pattern,
		Reference: s.Reference,
	}

	if s.Properties != nil {
		clone.Properties = make(map[string]*Schema)
		for k, v := range s.Properties {
			clone.Properties[k] = v.Clone()
		}
	}

	if s.Items != nil {
		clone.Items = s.Items.Clone()
	}

	if s.Required != nil {
		clone.Required = make([]string, len(s.Required))
		copy(clone.Required, s.Required)
	}

	if s.Enum != nil {
		clone.Enum = make([]interface{}, len(s.Enum))
		copy(clone.Enum, s.Enum)
	}

	if s.Additional != nil {
		clone.Additional = make(map[string]interface{})
		for k, v := range s.Additional {
			clone.Additional[k] = v
		}
	}

	return clone
}

// SchemaBuilder provides a fluent interface for building schemas
type SchemaBuilder struct {
	schema *Schema
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		schema: &Schema{},
	}
}

// WithType sets the type
func (b *SchemaBuilder) WithType(t string) *SchemaBuilder {
	b.schema.Type = t
	return b
}

// WithTitle sets the title
func (b *SchemaBuilder) WithTitle(title string) *SchemaBuilder {
	b.schema.Title = title
	return b
}

// WithProperty adds a property
func (b *SchemaBuilder) WithProperty(name string, schema *Schema) *SchemaBuilder {
	if b.schema.Properties == nil {
		b.schema.Properties = make(map[string]*Schema)
	}
	b.schema.Properties[name] = schema
	return b
}

// WithRequired adds required fields
func (b *SchemaBuilder) WithRequired(fields ...string) *SchemaBuilder {
	b.schema.Required = append(b.schema.Required, fields...)
	return b
}

// Build returns the built schema
func (b *SchemaBuilder) Build() *Schema {
	return b.schema
}
