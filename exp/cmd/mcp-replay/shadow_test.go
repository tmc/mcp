package main

import (
	"os"
	"testing"
)

func TestParseShadowRecording(t *testing.T) {
	// Create a test file with shadow responses
	content := `# mcptrace:v1
mcp-recv {"jsonrpc":"2.0","method":"init","id":1} # 1683000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":"primary"} # 1683000001.000
mcp-send-shadow {"jsonrpc":"2.0","id":1,"result":"shadow"} # 1683000001.100
`

	// Write to temp file
	tmpfile := t.TempDir() + "/test-shadow.mcp"
	err := os.WriteFile(tmpfile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the recording
	recording, err := ParseRecording(tmpfile)
	if err != nil {
		t.Fatalf("Failed to parse recording: %v", err)
	}

	// Check that we have the correct counts
	if len(recording.Requests) != 1 {
		t.Errorf("Expected 1 request, got %d", len(recording.Requests))
	}

	if len(recording.Responses) != 1 {
		t.Errorf("Expected 1 primary response, got %d", len(recording.Responses))
	}

	if len(recording.ShadowResponses) != 1 {
		t.Errorf("Expected 1 shadow response, got %d", len(recording.ShadowResponses))
	}

	// Check shadow response content
	if len(recording.ShadowResponses) > 0 {
		shadow := recording.ShadowResponses[0]
		if shadow.Content != `{"jsonrpc":"2.0","id":1,"result":"shadow"}` {
			t.Errorf("Unexpected shadow content: %s", shadow.Content)
		}
		if shadow.ID != "1" {
			t.Errorf("Expected shadow ID 1, got %s", shadow.ID)
		}
	}

	// Check shadow request map
	if shadowResponses, ok := recording.ShadowRequestMap["1"]; !ok {
		t.Errorf("Expected shadow response for ID 1 in map")
	} else if len(shadowResponses) != 1 {
		t.Errorf("Expected 1 shadow response in map, got %d", len(shadowResponses))
	}
}
