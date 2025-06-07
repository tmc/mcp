package sdk2_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tmc/mcp/exp/sdk2"
)

// TestContentTypeValidation ensures our content types work correctly
func TestContentTypeValidation(t *testing.T) {
	t.Run("TextContent", func(t *testing.T) {
		// Valid text content
		text, err := sdk2.NewTextContent("Hello, world!")
		if err != nil {
			t.Fatalf("Expected valid text content to succeed: %v", err)
		}
		if text.ContentType() != "text/plain" {
			t.Errorf("Expected text/plain, got %s", text.ContentType())
		}

		// Invalid empty text
		_, err = sdk2.NewTextContent("")
		if err == nil {
			t.Error("Expected empty text to fail validation")
		}

		// Must constructor should work for valid content
		validText := sdk2.MustNewTextContent("Valid text")
		if validText.Text != "Valid text" {
			t.Error("MustNewTextContent didn't set text correctly")
		}
	})

	t.Run("ImageContent", func(t *testing.T) {
		// Valid image content
		image, err := sdk2.NewImageContent("base64data", "image/png")
		if err != nil {
			t.Fatalf("Expected valid image content to succeed: %v", err)
		}
		if image.ContentType() != "image/png" {
			t.Errorf("Expected image/png, got %s", image.ContentType())
		}

		// Invalid empty data
		_, err = sdk2.NewImageContent("", "image/png")
		if err == nil {
			t.Error("Expected empty data to fail validation")
		}

		// Invalid empty mime type
		_, err = sdk2.NewImageContent("data", "")
		if err == nil {
			t.Error("Expected empty mime type to fail validation")
		}
	})

	t.Run("ResourceReferenceContent", func(t *testing.T) {
		// Valid resource content
		resource, err := sdk2.NewResourceReferenceContent("file://example.txt")
		if err != nil {
			t.Fatalf("Expected valid resource content to succeed: %v", err)
		}
		if resource.ContentType() != "text/plain" {
			t.Errorf("Expected text/plain default, got %s", resource.ContentType())
		}

		// Invalid empty URI
		_, err = sdk2.NewResourceReferenceContent("")
		if err == nil {
			t.Error("Expected empty URI to fail validation")
		}
	})
}

// TestToolCreation ensures tool constructors work correctly
func TestToolCreation(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string"},
		},
		"required": []string{"message"},
	}

	tool, err := sdk2.NewTool("echo", "Echo back input", schema)
	if err != nil {
		t.Fatalf("Expected tool creation to succeed: %v", err)
	}

	if tool.Name != "echo" {
		t.Errorf("Expected name 'echo', got %s", tool.Name)
	}
	if tool.Description != "Echo back input" {
		t.Errorf("Expected description 'Echo back input', got %s", tool.Description)
	}

	// Verify schema was marshaled correctly
	var unmarshaledSchema map[string]any
	if err := json.Unmarshal(tool.InputSchema, &unmarshaledSchema); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Must constructor should work
	mustTool := sdk2.MustNewTool("must-echo", "Must echo", schema)
	if mustTool.Name != "must-echo" {
		t.Error("MustNewTool didn't set name correctly")
	}
}

// TestClientOptions ensures functional options work correctly
func TestClientOptions(t *testing.T) {
	config := &sdk2.ClientConfig{}

	// Apply options
	sdk2.WithTimeout(30 * time.Second)(config)
	sdk2.WithRetries(5, 2*time.Second)(config)
	sdk2.WithClientInfo("test-client", "1.0.0")(config)

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", config.Timeout)
	}
	if config.MaxRetries != 5 {
		t.Errorf("Expected 5 retries, got %d", config.MaxRetries)
	}
	if config.RetryDelay != 2*time.Second {
		t.Errorf("Expected 2s retry delay, got %v", config.RetryDelay)
	}
	if config.ClientInfo.Name != "test-client" {
		t.Errorf("Expected client name 'test-client', got %s", config.ClientInfo.Name)
	}
	if config.ClientInfo.Version != "1.0.0" {
		t.Errorf("Expected client version '1.0.0', got %s", config.ClientInfo.Version)
	}
}

// TestRequestIDHandling ensures JSON marshaling/unmarshaling works
func TestRequestIDHandling(t *testing.T) {
	testCases := []struct {
		name     string
		value    any
		expected string
	}{
		{"string ID", "test-id", `"test-id"`},
		{"number ID", int64(123), `123`},
		{"float ID", float64(123.45), `123.45`},
		{"null ID", nil, `null`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := sdk2.RequestID{Value: tc.value}

			// Test marshaling
			data, err := json.Marshal(id)
			if err != nil {
				t.Fatalf("Failed to marshal ID: %v", err)
			}
			if string(data) != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, string(data))
			}

			// Test unmarshaling
			var unmarshaledID sdk2.RequestID
			if err := json.Unmarshal(data, &unmarshaledID); err != nil {
				t.Fatalf("Failed to unmarshal ID: %v", err)
			}

			// For float64, JSON unmarshaling converts all numbers to float64
			if tc.value != nil {
				switch v := tc.value.(type) {
				case int64:
					if unmarshaledID.Value != float64(v) {
						t.Errorf("Expected %v, got %v", float64(v), unmarshaledID.Value)
					}
				default:
					if unmarshaledID.Value != tc.value {
						t.Errorf("Expected %v, got %v", tc.value, unmarshaledID.Value)
					}
				}
			} else {
				if unmarshaledID.Value != nil {
					t.Errorf("Expected nil, got %v", unmarshaledID.Value)
				}
			}
		})
	}
}

// TestContentMarshaling ensures content types marshal/unmarshal correctly
func TestContentMarshaling(t *testing.T) {
	contents := []sdk2.Content{
		sdk2.MustNewTextContent("Hello, world!"),
		sdk2.MustNewImageContent("base64data", "image/png"),
		sdk2.MustNewResourceReferenceContent("file://example.txt"),
	}

	for i, content := range contents {
		t.Run(content.ContentType(), func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(content)
			if err != nil {
				t.Fatalf("Failed to marshal content %d: %v", i, err)
			}

			// Unmarshal
			unmarshaledContent, err := sdk2.UnmarshalContent(data)
			if err != nil {
				t.Fatalf("Failed to unmarshal content %d: %v", i, err)
			}

			// Verify type
			if unmarshaledContent.ContentType() != content.ContentType() {
				t.Errorf("Content type mismatch: expected %s, got %s",
					content.ContentType(), unmarshaledContent.ContentType())
			}

			// Verify it's valid
			if err := unmarshaledContent.Valid(); err != nil {
				t.Errorf("Unmarshaled content is invalid: %v", err)
			}
		})
	}
}

// TestServerCreation ensures server constructors work
func TestServerCreation(t *testing.T) {
	// Simple server
	server1 := &sdk2.Server{
		Addr:         ":stdio",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if server1.Addr != ":stdio" {
		t.Error("Server addr not set correctly")
	}

	// Server with options
	server2 := sdk2.NewServer(
		sdk2.WithServerInfo("test-server", "1.0.0"),
		sdk2.WithTimeouts(5*time.Second, 5*time.Second),
	)

	if server2.ReadTimeout != 5*time.Second {
		t.Error("Server timeout not set correctly")
	}

	// Should use DefaultServeMux by default
	if server2.Handler != sdk2.DefaultServeMux {
		t.Error("Server should use DefaultServeMux by default")
	}
}
