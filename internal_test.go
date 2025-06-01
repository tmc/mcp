package mcp

import (
	"context"
	"testing"
)

// TestRegisterTypedToolInternal tests the RegisterTypedTool function
func TestRegisterTypedToolInternal(t *testing.T) {
	type TestInput struct {
		Message string `json:"message" description:"Test message"`
	}

	type TestOutput struct {
		Reply string `json:"reply"`
	}

	server := NewServer("test", "1.0")

	// Test the type-safe tool registration
	err := RegisterTypedTool[TestInput, TestOutput](server, "typed-test", "A typed test tool",
		func(ctx context.Context, input TestInput) (TestOutput, error) {
			return TestOutput{Reply: "Got: " + input.Message}, nil
		})

	if err != nil {
		t.Fatalf("RegisterTypedTool() error = %v", err)
	}
}
