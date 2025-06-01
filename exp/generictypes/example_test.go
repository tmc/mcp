package generictypes_test

import (
	"encoding/json"
	"fmt"
	"testing"
	
	"github.com/tmc/mcp/exp/generictypes"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func ExampleListResult() {
	// Create a list of resources using the generic ListResult
	cursor := modelcontextprotocol.Cursor("next-page")
	resources := generictypes.ListResult[modelcontextprotocol.Resource]{
		Items: []modelcontextprotocol.Resource{
			{URI: "file:///doc1.txt", Name: "Document 1"},
			{URI: "file:///doc2.txt", Name: "Document 2"},
		},
		NextCursor: &cursor,
	}
	
	// Filter resources
	filtered := generictypes.FilterList(resources, func(r modelcontextprotocol.Resource) bool {
		return r.Name == "Document 1"
	})
	
	fmt.Printf("Found %d resources\n", len(filtered.Items))
	// Output: Found 1 resources
}

func ExampleOptional() {
	// Create an optional string
	opt := generictypes.NewOptional("hello")
	
	// Check if value is present
	if opt.IsPresent() {
		fmt.Println(opt.MustGet())
	}
	
	// Create an empty optional
	empty := generictypes.Empty[string]()
	fmt.Println(empty.OrElse("default"))
	
	// Output:
	// hello
	// default
}

func ExampleBuilder() {
	// Build a Resource using the builder pattern
	resource, err := generictypes.NewResourceBuilder("file:///example.txt", "Example").
		WithDescription("An example file").
		WithMimeType("text/plain").
		WithSize(1024).
		Build()
		
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Built resource: %s\n", resource.Name)
	// Output: Built resource: Example
}

func TestOptionalJSON(t *testing.T) {
	// Test marshaling
	opt := generictypes.NewOptional("test")
	data, err := json.Marshal(opt)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	
	if string(data) != `"test"` {
		t.Errorf("Expected \"test\", got %s", string(data))
	}
	
	// Test unmarshaling
	var opt2 generictypes.Optional[string]
	err = json.Unmarshal([]byte(`"value"`), &opt2)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	
	if !opt2.IsPresent() || opt2.MustGet() != "value" {
		t.Errorf("Expected Optional with value 'value', got %v", opt2)
	}
	
	// Test null unmarshaling
	var opt3 generictypes.Optional[string]
	err = json.Unmarshal([]byte(`null`), &opt3)
	if err != nil {
		t.Fatalf("Unmarshal null error: %v", err)
	}
	
	if opt3.IsPresent() {
		t.Error("Expected empty Optional for null value")
	}
}

func TestListOperations(t *testing.T) {
	// Create a list of prompts
	prompts := generictypes.ListResult[modelcontextprotocol.Prompt]{
		Items: []modelcontextprotocol.Prompt{
			{Name: "greeting", Description: strPtr("Say hello")},
			{Name: "farewell", Description: strPtr("Say goodbye")},
			{Name: "question", Description: nil},
		},
	}
	
	// Map to get just names
	names := generictypes.MapList(prompts, func(p modelcontextprotocol.Prompt) string {
		return p.Name
	})
	
	if len(names.Items) != 3 {
		t.Errorf("Expected 3 names, got %d", len(names.Items))
	}
	
	// Filter prompts with descriptions
	withDesc := generictypes.FilterList(prompts, func(p modelcontextprotocol.Prompt) bool {
		return p.Description != nil
	})
	
	if len(withDesc.Items) != 2 {
		t.Errorf("Expected 2 prompts with descriptions, got %d", len(withDesc.Items))
	}
}

func TestTypedUnion(t *testing.T) {
	// Create a typed union for content types
	contentUnion := generictypes.NewTypedUnion[any]("type").
		Register("text", func(data json.RawMessage) (any, error) {
			var tc modelcontextprotocol.TextContent
			err := json.Unmarshal(data, &tc)
			return tc, err
		}).
		Register("image", func(data json.RawMessage) (any, error) {
			var ic modelcontextprotocol.ImageContent
			err := json.Unmarshal(data, &ic)
			return ic, err
		})
	
	// Test text content
	textData := json.RawMessage(`{"type": "text", "text": "Hello"}`)
	content, err := contentUnion.Unmarshal(textData)
	if err != nil {
		t.Fatalf("Failed to unmarshal text content: %v", err)
	}
	
	if tc, ok := content.(modelcontextprotocol.TextContent); ok {
		if tc.Text != "Hello" {
			t.Errorf("Expected text 'Hello', got '%s'", tc.Text)
		}
	} else {
		t.Error("Expected TextContent type")
	}
}

func TestRequestBuilder(t *testing.T) {
	// Build a request with metadata
	params := struct {
		URI string `json:"uri"`
	}{
		URI: "file:///test.txt",
	}
	
	request, err := generictypes.NewRequestBuilder(params).
		WithProgressToken("token-123").
		Build()
		
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}
	
	if request.Meta == nil || request.Meta.ProgressToken == nil {
		t.Error("Expected progress token in metadata")
	} else if *request.Meta.ProgressToken != "token-123" {
		t.Errorf("Expected token 'token-123', got '%v'", *request.Meta.ProgressToken)
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}