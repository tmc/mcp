package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestServer(t *testing.T) {
	srv := NewServer("test", "1.0.0")

	// Register a test tool
	err := srv.RegisterTool(NewTool("echo", "Echo back the input", func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: string(args),
			}},
		}, nil
	}))
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	// Test tool execution
	msg := `{"jsonrpc":"2.0","id":1,"method":"echo","params":"hello world"}`
	resp, err := srv.Handle(context.Background(), []byte(msg))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var result struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Content []Content `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.JSONRPC != JSONRPCVersion {
		t.Errorf("got JSONRPC version %q, want %q", result.JSONRPC, JSONRPCVersion)
	}

	if result.ID != 1 {
		t.Errorf("got ID %d, want 1", result.ID)
	}

	if len(result.Result.Content) != 1 || result.Result.Content[0].Text != "hello world" {
		t.Errorf("unexpected response content: %+v", result.Result.Content)
	}
}
