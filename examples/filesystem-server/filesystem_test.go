package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/internal/mcptest"
)

func TestFilesystemServer(t *testing.T) {
	// Build test server
	tmpDir := t.TempDir()
	serverPath := filepath.Join(tmpDir, "server")
	if err := exec.Command("go", "build", "-o", serverPath, ".").Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}

	// Create test files
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Start server with debug logging
	debugLog := &bytes.Buffer{}
	server := mcptest.NewTestServer(t, serverPath, mcptest.WithDebugLog(debugLog))
	defer server.Close()

	// Initialize server
	ctx := context.Background()
	reply, err := server.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	if reply.ServerInfo.Name != "filesystem" {
		t.Errorf("got server name %q, want %q", reply.ServerInfo.Name, "filesystem")
	}

	// Test tool listing
	result, err := server.Call("tools/list", nil)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	var toolsReply struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsReply); err != nil {
		t.Fatalf("Failed to parse tools reply: %v", err)
	}

	// Verify read_file tool exists
	var hasReadFile bool
	for _, tool := range toolsReply.Tools {
		if tool.Name == "read_file" {
			hasReadFile = true
			break
		}
	}
	if !hasReadFile {
		t.Error("read_file tool not found")
	}

	// Test reading file
	result, err = server.Call("tools/call", map[string]interface{}{
		"name": "read_file",
		"arguments": map[string]interface{}{
			"path": testFile,
		},
	})
	if err != nil {
		t.Fatalf("Failed to call read_file: %v", err)
	}

	var readReply struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &readReply); err != nil {
		t.Fatalf("Failed to parse read reply: %v", err)
	}

	if len(readReply.Content) != 1 || readReply.Content[0].Text != "hello world" {
		t.Errorf("got content %+v, want 'hello world'", readReply.Content)
	}

	// Print debug log on failure
	if t.Failed() {
		t.Logf("Debug log:\n%s", debugLog.String())
	}
}
