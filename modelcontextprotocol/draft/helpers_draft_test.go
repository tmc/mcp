package draft

import (
	"testing"

	base "github.com/tmc/mcp/modelcontextprotocol"
)

func TestCallToolResultHelpers(t *testing.T) {
	t.Run("NewCallToolResultStructured", func(t *testing.T) {
		sc := map[string]any{"key": "value"}
		result := NewCallToolResultStructured(sc)

		if !result.isStructuredResult {
			t.Error("Expected structured result")
		}
		if result.StructuredContent == nil {
			t.Error("Expected StructuredContent to be set")
		}
		if result.StructuredContent["key"] != "value" {
			t.Error("Expected structured content key to equal 'value'")
		}
		if result.Content != nil {
			t.Error("Expected Content to be nil for structured result without compatibility content")
		}
		if result.IsError != nil {
			t.Error("Expected IsError to be nil for non-error result")
		}
	})

	t.Run("NewCallToolResultStructured with compatibility content", func(t *testing.T) {
		sc := map[string]any{"key": "value"}
		textContent := base.NewTextContent("compatibility text")
		result := NewCallToolResultStructured(sc, textContent)

		if !result.isStructuredResult {
			t.Error("Expected structured result")
		}
		if result.StructuredContent == nil {
			t.Error("Expected StructuredContent to be set")
		}
		if result.Content == nil {
			t.Error("Expected Content to be set for structured result with compatibility content")
		}
		if len(*result.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(*result.Content))
		}
	})

	t.Run("NewCallToolResultUnstructured", func(t *testing.T) {
		textContent := base.NewTextContent("result text")
		result := NewCallToolResultUnstructured(textContent)

		if result.isStructuredResult {
			t.Error("Expected unstructured result")
		}
		if result.StructuredContent != nil {
			t.Error("Expected StructuredContent to be nil for unstructured result")
		}
		if result.Content == nil {
			t.Error("Expected Content to be set")
		}
		if len(*result.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(*result.Content))
		}
		if result.IsError != nil {
			t.Error("Expected IsError to be nil for non-error result")
		}
	})

	t.Run("NewCallToolResultError", func(t *testing.T) {
		textContent := base.NewTextContent("error message")
		result := NewCallToolResultError(textContent)

		if result.isStructuredResult {
			t.Error("Expected unstructured result for error")
		}
		if result.StructuredContent != nil {
			t.Error("Expected StructuredContent to be nil for error result")
		}
		if result.IsError == nil || !*result.IsError {
			t.Error("Expected IsError to be true")
		}
		if result.Content == nil {
			t.Error("Expected Content to be set")
		}
		if len(*result.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(*result.Content))
		}
	})

	t.Run("NewCallToolResultError without content", func(t *testing.T) {
		result := NewCallToolResultError()

		if result.isStructuredResult {
			t.Error("Expected unstructured result for error")
		}
		if result.IsError == nil || !*result.IsError {
			t.Error("Expected IsError to be true")
		}
		if result.Content != nil {
			t.Error("Expected Content to be nil for error without content")
		}
	})
}
