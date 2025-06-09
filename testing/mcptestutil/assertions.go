// Package mcptestutil provides assertion helpers for testing MCP implementations.
package mcptestutil

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp"
)

// JSONEqual compares two values after JSON marshaling/unmarshaling to handle
// differences in field ordering and whitespace.
func JSONEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()
	
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("Failed to marshal actual value: %v", err)
	}
	
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("Failed to marshal expected value: %v", err)
	}
	
	var actualNormalized, expectedNormalized interface{}
	
	if err := json.Unmarshal(actualJSON, &actualNormalized); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}
	
	if err := json.Unmarshal(expectedJSON, &expectedNormalized); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}
	
	if !reflect.DeepEqual(actualNormalized, expectedNormalized) {
		t.Fatalf("JSON values not equal:\nActual:   %s\nExpected: %s", 
			string(actualJSON), string(expectedJSON))
	}
}

// AssertToolEqual compares two Tool objects for equality.
func AssertToolEqual(t *testing.T, actual, expected *mcp.Tool) {
	t.Helper()
	
	if actual == nil && expected == nil {
		return
	}
	
	if actual == nil || expected == nil {
		t.Fatalf("Tool mismatch: one is nil\nActual: %v\nExpected: %v", actual, expected)
	}
	
	if actual.Name != expected.Name {
		t.Errorf("Tool name mismatch: got %q, want %q", actual.Name, expected.Name)
	}
	
	if actual.Description != expected.Description {
		t.Errorf("Tool description mismatch: got %q, want %q", actual.Description, expected.Description)
	}
	
	JSONEqual(t, actual.InputSchema, expected.InputSchema)
}

// AssertResourceEqual compares two Resource objects for equality.
func AssertResourceEqual(t *testing.T, actual, expected *mcp.Resource) {
	t.Helper()
	
	if actual == nil && expected == nil {
		return
	}
	
	if actual == nil || expected == nil {
		t.Fatalf("Resource mismatch: one is nil\nActual: %v\nExpected: %v", actual, expected)
	}
	
	if actual.URI != expected.URI {
		t.Errorf("Resource URI mismatch: got %q, want %q", actual.URI, expected.URI)
	}
	
	
	if actual.Description != expected.Description {
		t.Errorf("Resource description mismatch: got %q, want %q", actual.Description, expected.Description)
	}
	
	if actual.MimeType != expected.MimeType {
		t.Errorf("Resource MIME type mismatch: got %q, want %q", actual.MimeType, expected.MimeType)
	}
}

// AssertPromptEqual compares two Prompt objects for equality.
func AssertPromptEqual(t *testing.T, actual, expected *mcp.Prompt) {
	t.Helper()
	
	if actual == nil && expected == nil {
		return
	}
	
	if actual == nil || expected == nil {
		t.Fatalf("Prompt mismatch: one is nil\nActual: %v\nExpected: %v", actual, expected)
	}
	
	if actual.Name != expected.Name {
		t.Errorf("Prompt name mismatch: got %q, want %q", actual.Name, expected.Name)
	}
	
	if actual.Description != expected.Description {
		t.Errorf("Prompt description mismatch: got %q, want %q", actual.Description, expected.Description)
	}
	
	if len(actual.Arguments) != len(expected.Arguments) {
		t.Errorf("Prompt arguments length mismatch: got %d, want %d", 
			len(actual.Arguments), len(expected.Arguments))
		return
	}
	
	for i, arg := range actual.Arguments {
		expectedArg := expected.Arguments[i]
		if arg.Name != expectedArg.Name {
			t.Errorf("Prompt argument[%d] name mismatch: got %q, want %q", 
				i, arg.Name, expectedArg.Name)
		}
		if arg.Description != expectedArg.Description {
			t.Errorf("Prompt argument[%d] description mismatch: got %q, want %q", 
				i, arg.Description, expectedArg.Description)
		}
		if arg.Required != expectedArg.Required {
			t.Errorf("Prompt argument[%d] required mismatch: got %t, want %t", 
				i, arg.Required, expectedArg.Required)
		}
	}
}

// AssertContentEqual compares two Content arrays for equality.
func AssertContentEqual(t *testing.T, actual, expected []mcp.Content) {
	t.Helper()
	
	if len(actual) != len(expected) {
		t.Fatalf("Content length mismatch: got %d, want %d", len(actual), len(expected))
	}
	
	for i, content := range actual {
		expectedContent := expected[i]
		
		// Compare the content using JSON marshaling since Content is an interface
		JSONEqual(t, content, expectedContent)
	}
}

// AssertTimeWithin asserts that a time is within a certain duration of another time.
func AssertTimeWithin(t *testing.T, actual, expected time.Time, delta time.Duration) {
	t.Helper()
	
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	
	if diff > delta {
		t.Errorf("Time difference too large: got %v, want within %v of %v", 
			actual, delta, expected)
	}
}

// AssertStringSliceEqual compares two string slices for equality.
func AssertStringSliceEqual(t *testing.T, actual, expected []string) {
	t.Helper()
	
	if len(actual) != len(expected) {
		t.Fatalf("String slice length mismatch: got %d, want %d\nActual: %v\nExpected: %v", 
			len(actual), len(expected), actual, expected)
	}
	
	for i, s := range actual {
		if s != expected[i] {
			t.Errorf("String slice[%d] mismatch: got %q, want %q", i, s, expected[i])
		}
	}
}

// AssertMapStringEqual compares two string maps for equality.
func AssertMapStringEqual(t *testing.T, actual, expected map[string]string) {
	t.Helper()
	
	if len(actual) != len(expected) {
		t.Fatalf("Map length mismatch: got %d, want %d\nActual: %v\nExpected: %v", 
			len(actual), len(expected), actual, expected)
	}
	
	for k, v := range expected {
		if actualV, exists := actual[k]; !exists {
			t.Errorf("Missing key %q in actual map", k)
		} else if actualV != v {
			t.Errorf("Map[%q] mismatch: got %q, want %q", k, actualV, v)
		}
	}
	
	for k := range actual {
		if _, exists := expected[k]; !exists {
			t.Errorf("Unexpected key %q in actual map", k)
		}
	}
}

// AssertErrorContains asserts that an error contains a specific substring.
func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("Error %q does not contain %q", err.Error(), substr)
	}
}

// AssertErrorType asserts that an error is of a specific type.
func AssertErrorType[T error](t *testing.T, err error) T {
	t.Helper()
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	typedErr, ok := err.(T)
	if !ok {
		t.Fatalf("Error is not of expected type: got %T, want %T", err, typedErr)
	}
	
	return typedErr
}

// AssertPanic asserts that a function panics.
func AssertPanic(t *testing.T, f func()) {
	t.Helper()
	
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected function to panic, but it didn't")
		}
	}()
	
	f()
}

// AssertNoPanic asserts that a function does not panic.
func AssertNoPanic(t *testing.T, f func()) {
	t.Helper()
	
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Function panicked unexpectedly: %v", r)
		}
	}()
	
	f()
}

// AssertImplementsInterface asserts that a value implements an interface.
func AssertImplementsInterface[T any](t *testing.T, value interface{}) {
	t.Helper()
	
	var zero T
	interfaceType := reflect.TypeOf(&zero).Elem()
	valueType := reflect.TypeOf(value)
	
	if !valueType.Implements(interfaceType) {
		t.Fatalf("Type %v does not implement interface %v", valueType, interfaceType)
	}
}

// AssertDeepEqual performs a deep equality check with helpful error messages.
func AssertDeepEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()
	
	if !reflect.DeepEqual(actual, expected) {
		actualJSON, _ := json.MarshalIndent(actual, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		
		t.Fatalf("Values not deeply equal:\nActual:\n%s\n\nExpected:\n%s", 
			string(actualJSON), string(expectedJSON))
	}
}