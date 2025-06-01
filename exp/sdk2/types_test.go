package sdk2

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTextContent_MarshalJSON(t *testing.T) {
	content := TextContent{Text: "Hello, world!"}
	
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal TextContent: %v", err)
	}
	
	expected := `{"type":"text","text":"Hello, world!"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestImageContent_MarshalJSON(t *testing.T) {
	content := ImageContent{
		Data:     "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
		MimeType: "image/png",
	}
	
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal ImageContent: %v", err)
	}
	
	expected := `{"type":"image","data":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==","mimeType":"image/png"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestClientOptions(t *testing.T) {
	config := &ClientConfig{}
	
	// Test WithTimeout
	WithTimeout(10 * time.Second)(config)
	if config.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", config.Timeout)
	}
	
	// Test WithRetries
	WithRetries(5, 2*time.Second)(config)
	if config.MaxRetries != 5 {
		t.Errorf("Expected max retries 5, got %v", config.MaxRetries)
	}
	if config.RetryDelay != 2*time.Second {
		t.Errorf("Expected retry delay 2s, got %v", config.RetryDelay)
	}
	
	// Test WithClientInfo
	WithClientInfo("test-client", "1.0.0")(config)
	if config.ClientInfo.Name != "test-client" {
		t.Errorf("Expected client name 'test-client', got %s", config.ClientInfo.Name)
	}
	if config.ClientInfo.Version != "1.0.0" {
		t.Errorf("Expected client version '1.0.0', got %s", config.ClientInfo.Version)
	}
}