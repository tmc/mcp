package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestServer(t *testing.T) {
	srv := NewServer("test", "1.0.0")

	// Register a test tool
	err := srv.RegisterTool(NewTool("echo", func(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
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
	msg := `{"method":"echo","params":"hello world"}`
	resp, err := srv.Handle(context.Background(), []byte(msg))
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var result struct {
		Content []Content `json:"content"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(result.Content) != 1 || result.Content[0].Text != "hello world" {
		t.Errorf("unexpected response: %s", resp)
	}
}
