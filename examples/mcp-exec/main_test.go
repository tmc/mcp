package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/internal/mcptest"
)

func TestExecServer(t *testing.T) {
	// Build test server
	tmpDir := t.TempDir()
	serverPath := filepath.Join(tmpDir, "server")
	if err := exec.Command("go", "build", "-o", serverPath, ".").Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
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

	if reply.Name != "exec-server" {
		t.Errorf("got server name %q, want %q", reply.Name, "exec-server")
	}

	// Test tool listing
	result, err := server.Call("listTools", nil)
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

	// Verify exec tool exists
	var hasExec bool
	for _, tool := range toolsReply.Tools {
		if tool.Name == "exec" {
			hasExec = true
			break
		}
	}
	if !hasExec {
		t.Error("exec tool not found")
	}

	// Test executing echo command
	args, _ := json.Marshal(map[string]interface{}{
		"command": "echo",
		"args":    []string{"hello world"},
	})
	result, err = server.Call("exec", args)
	if err != nil {
		t.Fatalf("Failed to call exec: %v", err)
	}

	var execReply struct {
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			IsError bool   `json:"isError,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &execReply); err != nil {
		t.Fatalf("Failed to parse exec reply: %v", err)
	}

	if len(execReply.Content) != 1 || execReply.Content[0].Text != "hello world\n" {
		t.Errorf("got content %+v, want 'hello world\\n'", execReply.Content)
	}

	// Test executing with timeout
	timeout := 0.1 // 100ms
	args, _ = json.Marshal(map[string]interface{}{
		"command": "sleep",
		"args":    []string{"1"},
		"timeout": timeout,
	})
	result, err = server.Call("exec", args)
	if err != nil {
		t.Fatalf("Failed to call exec: %v", err)
	}

	if err := json.Unmarshal(result, &execReply); err != nil {
		t.Fatalf("Failed to parse exec reply: %v", err)
	}

	if len(execReply.Content) != 1 || !execReply.Content[0].IsError {
		t.Error("expected timeout error")
	}

	// Print debug log on failure
	if t.Failed() {
		t.Logf("Debug log:\n%s", debugLog.String())
	}
}
