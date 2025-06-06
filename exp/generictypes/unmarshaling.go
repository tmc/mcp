package generictypes

import (
	"encoding/json"
	"fmt"
)

// UnionType represents a type that can be unmarshaled from a discriminated union.
type UnionType interface {
	TypeField() string
}

// TypedUnion handles unmarshaling of discriminated unions based on a type field.
type TypedUnion[T any] struct {
	typeField string
	handlers  map[string]UnmarshalFunc[T]
}

// UnmarshalFunc is a function that unmarshals JSON data into a specific type.
type UnmarshalFunc[T any] func(data json.RawMessage) (T, error)

// NewTypedUnion creates a new TypedUnion unmarshaler.
func NewTypedUnion[T any](typeField string) *TypedUnion[T] {
	return &TypedUnion[T]{
		typeField: typeField,
		handlers:  make(map[string]UnmarshalFunc[T]),
	}
}

// Register adds a handler for a specific type value.
func (tu *TypedUnion[T]) Register(typeValue string, handler UnmarshalFunc[T]) *TypedUnion[T] {
	tu.handlers[typeValue] = handler
	return tu
}

// Unmarshal unmarshals JSON data using the appropriate handler based on the type field.
func (tu *TypedUnion[T]) Unmarshal(data json.RawMessage) (T, error) {
	var zero T

	if string(data) == "null" || len(data) == 0 {
		return zero, nil
	}

	// Probe the type field
	probe := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &probe); err != nil {
		return zero, fmt.Errorf("probing %s field: %w", tu.typeField, err)
	}

	typeRaw, ok := probe[tu.typeField]
	if !ok {
		return zero, fmt.Errorf("missing %s field in JSON", tu.typeField)
	}

	var typeValue string
	if err := json.Unmarshal(typeRaw, &typeValue); err != nil {
		return zero, fmt.Errorf("unmarshaling %s field: %w", tu.typeField, err)
	}

	handler, ok := tu.handlers[typeValue]
	if !ok {
		return zero, fmt.Errorf("unknown %s value: '%s'", tu.typeField, typeValue)
	}

	return handler(data)
}

// --- Generic content unmarshaler ---

// ContentUnmarshaler demonstrates how we could simplify content unmarshaling.
var ContentUnmarshaler = NewTypedUnion[any]("type").
	Register("text", func(data json.RawMessage) (any, error) {
		var tc struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		err := json.Unmarshal(data, &tc)
		return tc, err
	}).
	Register("image", func(data json.RawMessage) (any, error) {
		var ic struct {
			Type     string `json:"type"`
			Data     string `json:"data"`
			MimeType string `json:"mimeType"`
		}
		err := json.Unmarshal(data, &ic)
		return ic, err
	}).
	Register("audio", func(data json.RawMessage) (any, error) {
		var ac struct {
			Type     string `json:"type"`
			Data     string `json:"data"`
			MimeType string `json:"mimeType"`
		}
		err := json.Unmarshal(data, &ac)
		return ac, err
	})

// UnmarshalList is a generic helper for unmarshaling lists of items.
func UnmarshalList[T any](data []json.RawMessage, unmarshalFunc func(json.RawMessage) (T, error)) ([]T, error) {
	result := make([]T, len(data))
	for i, raw := range data {
		item, err := unmarshalFunc(raw)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling item %d: %w", i, err)
		}
		result[i] = item
	}
	return result, nil
}

// FieldUnmarshaler helps unmarshal specific fields with custom logic.
type FieldUnmarshaler[T any] struct {
	unmarshalFunc func(json.RawMessage) (T, error)
}

// NewFieldUnmarshaler creates a new field unmarshaler.
func NewFieldUnmarshaler[T any](unmarshalFunc func(json.RawMessage) (T, error)) *FieldUnmarshaler[T] {
	return &FieldUnmarshaler[T]{unmarshalFunc: unmarshalFunc}
}

// Unmarshal unmarshals a field using the custom function.
func (fu *FieldUnmarshaler[T]) Unmarshal(data json.RawMessage) (T, error) {
	var zero T

	if string(data) == "null" || len(data) == 0 {
		return zero, nil
	}

	return fu.unmarshalFunc(data)
}

// UnmarshalField is a helper method for unmarshaling a specific field from a struct.
func UnmarshalField[T any](data []byte, fieldName string, unmarshalFunc func(json.RawMessage) (T, error)) (T, error) {
	var zero T

	fields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &fields); err != nil {
		return zero, fmt.Errorf("unmarshaling to map: %w", err)
	}

	fieldData, ok := fields[fieldName]
	if !ok {
		return zero, nil // Field not present
	}

	return unmarshalFunc(fieldData)
}

// --- Example: How this could simplify EmbeddedResource unmarshaling ---

func UnmarshalEmbeddedResourceGeneric(data []byte) error {
	// This demonstrates how the generic unmarshaler could work
	type EmbeddedResource struct {
		Type        string `json:"type"`
		Resource    any    `json:"resource"`
		Annotations any    `json:"annotations,omitempty"`
	}

	resourceUnmarshaler := NewFieldUnmarshaler(func(data json.RawMessage) (any, error) {
		// Simplified resource contents unmarshaling
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(data, &probe); err != nil {
			return nil, err
		}

		if _, hasText := probe["text"]; hasText {
			var tc struct {
				URI      string  `json:"uri"`
				Text     string  `json:"text"`
				MimeType *string `json:"mimeType,omitempty"`
			}
			err := json.Unmarshal(data, &tc)
			return tc, err
		}

		if _, hasBlob := probe["blob"]; hasBlob {
			var bc struct {
				URI      string  `json:"uri"`
				Blob     string  `json:"blob"`
				MimeType *string `json:"mimeType,omitempty"`
			}
			err := json.Unmarshal(data, &bc)
			return bc, err
		}

		return nil, fmt.Errorf("resource contents has neither text nor blob field")
	})

	resource, err := UnmarshalField(data, "resource", resourceUnmarshaler.Unmarshal)
	if err != nil {
		return fmt.Errorf("unmarshaling resource field: %w", err)
	}

	_ = resource // Use the unmarshaled resource
	return nil
}
