package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamableHTTPInitialPostCreatesSession(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "echo"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: "ok"}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()

	sessionID, got := postStreamable(t, httpServer.URL+"/mcp", "", `{
		"jsonrpc":"2.0",
		"id":"init",
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-11-25",
			"capabilities":{},
			"clientInfo":{"name":"streamable-test-client","version":"0.0.0"}
		}
	}`)
	if sessionID == "" {
		t.Fatal("initial POST did not return Mcp-Session-Id")
	}
	if len(got) != 1 {
		t.Fatalf("initial POST returned %d messages, want 1", len(got))
	}
	if got[0].ID != "init" {
		t.Fatalf("initialize response ID = %v, want init", got[0].ID)
	}
	if got[0].Result == nil {
		t.Fatalf("initialize response missing result: %+v", got[0])
	}

	_, got = postStreamable(t, httpServer.URL+"/mcp", sessionID, `{
		"jsonrpc":"2.0",
		"id":"tools",
		"method":"tools/list"
	}`)
	if len(got) != 1 {
		t.Fatalf("tools/list returned %d messages, want 1", len(got))
	}
	if got[0].ID != "tools" {
		t.Fatalf("tools/list response ID = %v, want tools", got[0].ID)
	}
	result, ok := got[0].Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/list result has type %T, want object", got[0].Result)
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools/list tools = %#v, want one tool", result["tools"])
	}

	req, err := http.NewRequest(http.MethodDelete, httpServer.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("NewRequest DELETE: %v", err)
	}
	req.Header.Set(streamableSessionHeader, sessionID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE session: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestStreamableHTTPRejectsNonLocalhostHostHeader(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/mcp", bytes.NewBufferString(`{
		"jsonrpc":"2.0",
		"id":"init",
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-11-25",
			"capabilities":{},
			"clientInfo":{"name":"streamable-test-client","version":"0.0.0"}
		}
	}`))
	if err != nil {
		t.Fatalf("NewRequest POST: %v", err)
	}
	req.Host = "evil.example.com"
	req.Header.Set("Origin", "http://evil.example.com")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST invalid host: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST invalid host status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func postStreamable(t *testing.T, url, sessionID, body string) (string, []JSONRPCMessage) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("NewRequest POST: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set(streamableSessionHeader, sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST status = %d, want %d: %s", resp.StatusCode, http.StatusOK, data)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	var messages []JSONRPCMessage
	for evt, err := range scanEvents(resp.Body) {
		if err != nil {
			t.Fatalf("scan SSE: %v", err)
		}
		if len(evt.data) == 0 {
			continue
		}
		var msg JSONRPCMessage
		if err := json.Unmarshal(evt.data, &msg); err != nil {
			t.Fatalf("decode SSE data %q: %v", evt.data, err)
		}
		messages = append(messages, msg)
	}
	return resp.Header.Get(streamableSessionHeader), messages
}
