package generictypes

import (
	"encoding/json"
	"fmt"
)

// Optional represents a value that may or may not be present.
// It's similar to a nullable type but provides more explicit handling.
type Optional[T any] struct {
	value T
	valid bool
}

// NewOptional creates a new Optional with a value.
func NewOptional[T any](value T) Optional[T] {
	return Optional[T]{value: value, valid: true}
}

// NewOptionalPtr creates an Optional from a pointer.
// If the pointer is nil, the Optional is empty.
func NewOptionalPtr[T any](ptr *T) Optional[T] {
	if ptr == nil {
		return Optional[T]{}
	}
	return Optional[T]{value: *ptr, valid: true}
}

// Empty returns an empty Optional.
func Empty[T any]() Optional[T] {
	return Optional[T]{}
}

// Get returns the value and whether it's valid.
func (o Optional[T]) Get() (T, bool) {
	return o.value, o.valid
}

// MustGet returns the value or panics if not valid.
func (o Optional[T]) MustGet() T {
	if !o.valid {
		panic("Optional.MustGet called on empty Optional")
	}
	return o.value
}

// OrElse returns the value if present, otherwise returns the default value.
func (o Optional[T]) OrElse(defaultValue T) T {
	if o.valid {
		return o.value
	}
	return defaultValue
}

// IsPresent returns true if the Optional contains a value.
func (o Optional[T]) IsPresent() bool {
	return o.valid
}

// IfPresent executes the function if a value is present.
func (o Optional[T]) IfPresent(fn func(T)) {
	if o.valid {
		fn(o.value)
	}
}

// Map transforms the value if present.
func (o Optional[T]) Map(fn func(T) T) Optional[T] {
	if !o.valid {
		return o
	}
	return NewOptional(fn(o.value))
}

// FlatMap transforms the Optional value into another Optional.
func (o Optional[T]) FlatMap(fn func(T) Optional[T]) Optional[T] {
	if !o.valid {
		return Optional[T]{}
	}
	return fn(o.value)
}

// ToPtr returns a pointer to the value, or nil if empty.
func (o Optional[T]) ToPtr() *T {
	if !o.valid {
		return nil
	}
	return &o.value
}

// MarshalJSON implements json.Marshaler.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = Optional[T]{}
		return nil
	}
	
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	
	*o = NewOptional(value)
	return nil
}

// String implements fmt.Stringer.
func (o Optional[T]) String() string {
	if !o.valid {
		return "<empty>"
	}
	return fmt.Sprintf("%v", o.value)
}

// --- Helper functions for common optional operations ---

// MapOptional transforms an optional value using a mapping function.
func MapOptional[T, U any](opt Optional[T], fn func(T) U) Optional[U] {
	if !opt.valid {
		return Empty[U]()
	}
	return NewOptional(fn(opt.value))
}

// FilterOptional returns the Optional if the predicate is true, otherwise returns empty.
func FilterOptional[T any](opt Optional[T], predicate func(T) bool) Optional[T] {
	if !opt.valid || !predicate(opt.value) {
		return Empty[T]()
	}
	return opt
}

// OptionalFrom creates an Optional from a value and a boolean indicating validity.
func OptionalFrom[T any](value T, valid bool) Optional[T] {
	if !valid {
		return Empty[T]()
	}
	return NewOptional(value)
}

// --- Example usage in MCP types ---

// These demonstrate how Optional could replace pointer fields:

type ToolWithOptional struct {
	Name        string
	Description Optional[string]          // Instead of *string
	InputSchema json.RawMessage
	Annotations Optional[ToolAnnotations] // Instead of *ToolAnnotations
}

type ResourceWithOptional struct {
	URI         string
	Name        string
	Description Optional[string] // Instead of *string
	MimeType    Optional[string] // Instead of *string
	Size        Optional[int64]  // Instead of *int64
}

// Helper to convert existing pointer-based types
func FromPtr[T any](ptr *T) Optional[T] {
	return NewOptionalPtr(ptr)
}

// Helper to convert to pointer for compatibility
func ToPtr[T any](opt Optional[T]) *T {
	return opt.ToPtr()
}

// ToolAnnotations example for the above
type ToolAnnotations struct {
	Title           Optional[string]
	ReadOnlyHint    Optional[bool]
	DestructiveHint Optional[bool]
	IdempotentHint  Optional[bool]
	OpenWorldHint   Optional[bool]
}