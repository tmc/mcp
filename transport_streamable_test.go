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

func TestStreamableHTTPInlineProgressNotification(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "progress"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		total := 100.0
		if err := server.NotifyProgress(ctx, req.ProgressToken(), 50, &total); err != nil {
			return nil, err
		}
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: "ok"}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()

	sessionID, _ := postStreamable(t, httpServer.URL+"/mcp", "", `{
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

	_, got := postStreamable(t, httpServer.URL+"/mcp", sessionID, `{
		"jsonrpc":"2.0",
		"id":"call",
		"method":"tools/call",
		"params":{
			"name":"progress",
			"arguments":{},
			"_meta":{"progressToken":"progress-test-1"}
		}
	}`)

	var sawProgress, sawResponse bool
	for _, msg := range got {
		if msg.Method == string(MethodProgress) {
			sawProgress = true
			params, ok := msg.Params.(map[string]any)
			if !ok {
				t.Fatalf("progress params have type %T, want object", msg.Params)
			}
			if params["progressToken"] != "progress-test-1" {
				t.Fatalf("progressToken = %v, want progress-test-1", params["progressToken"])
			}
			if params["progress"] != float64(50) {
				t.Fatalf("progress = %v, want 50", params["progress"])
			}
		}
		if msg.ID == "call" && msg.Method == "" {
			sawResponse = true
		}
	}
	if !sawProgress {
		t.Fatalf("tools/call returned %#v, want progress notification", got)
	}
	if !sawResponse {
		t.Fatalf("tools/call returned %#v, want call response", got)
	}
}

func TestStreamableHTTPInlineSamplingRequest(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "sample"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		result, err := server.CreateMessage(ctx, CreateMessageRequest{
			Messages: []SamplingMessage{
				{Role: RoleUser, Content: TextContent{Type: "text", Text: "sample this"}},
			},
			MaxTokens: 100,
		})
		if err != nil {
			return nil, err
		}
		text, _ := result.Content.(TextContent)
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: text.Text}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()

	sessionID, _ := postStreamable(t, httpServer.URL+"/mcp", "", `{
		"jsonrpc":"2.0",
		"id":"init",
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-11-25",
			"capabilities":{"sampling":{}},
			"clientInfo":{"name":"streamable-test-client","version":"0.0.0"}
		}
	}`)
	if sessionID == "" {
		t.Fatal("initial POST did not return Mcp-Session-Id")
	}

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/mcp", bytes.NewBufferString(`{
		"jsonrpc":"2.0",
		"id":"call",
		"method":"tools/call",
		"params":{"name":"sample","arguments":{}}
	}`))
	if err != nil {
		t.Fatalf("NewRequest POST: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(streamableSessionHeader, sessionID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST tools/call: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST status = %d, want %d: %s", resp.StatusCode, http.StatusOK, data)
	}

	var sawSampling, sawResponse bool
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
		if msg.Method == string(MethodSamplingCreateMessage) {
			sawSampling = true
			reply := JSONRPCMessage{
				JSONRPC: JSONRPC_VERSION,
				ID:      msg.ID,
				Result: map[string]any{
					"role":  string(RoleAssistant),
					"model": "test-model",
					"content": map[string]any{
						"type": "text",
						"text": "sampled response",
					},
				},
			}
			data, err := json.Marshal(reply)
			if err != nil {
				t.Fatalf("marshal sampling reply: %v", err)
			}
			postStreamableAccepted(t, httpServer.URL+"/mcp", sessionID, data)
		}
		if msg.ID == "call" && msg.Method == "" {
			sawResponse = true
			result, ok := msg.Result.(map[string]any)
			if !ok {
				t.Fatalf("call result has type %T, want object", msg.Result)
			}
			content, ok := result["content"].([]any)
			if !ok || len(content) != 1 {
				t.Fatalf("call content = %#v, want one item", result["content"])
			}
			text, ok := content[0].(map[string]any)
			if !ok || text["text"] != "sampled response" {
				t.Fatalf("call content = %#v, want sampled response", content[0])
			}
			break
		}
	}
	if !sawSampling {
		t.Fatal("tools/call did not receive sampling request")
	}
	if !sawResponse {
		t.Fatal("tools/call did not receive final response")
	}
}

func TestStreamableHTTPInlineElicitationRequest(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "elicit"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		result, err := server.Elicit(ctx, ElicitRequest{
			Message: "need input",
			RequestedSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"username": map[string]any{"type": "string"},
				},
				"required": []string{"username"},
			},
		})
		if err != nil {
			return nil, err
		}
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: result.Content["username"].(string)}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()

	sessionID, _ := postStreamable(t, httpServer.URL+"/mcp", "", `{
		"jsonrpc":"2.0",
		"id":"init",
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-11-25",
			"capabilities":{"elicitation":{"form":{}}},
			"clientInfo":{"name":"streamable-test-client","version":"0.0.0"}
		}
	}`)
	if sessionID == "" {
		t.Fatal("initial POST did not return Mcp-Session-Id")
	}

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/mcp", bytes.NewBufferString(`{
		"jsonrpc":"2.0",
		"id":"call",
		"method":"tools/call",
		"params":{"name":"elicit","arguments":{}}
	}`))
	if err != nil {
		t.Fatalf("NewRequest POST: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(streamableSessionHeader, sessionID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST tools/call: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST status = %d, want %d: %s", resp.StatusCode, http.StatusOK, data)
	}

	var sawElicitation, sawResponse bool
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
		if msg.Method == string(MethodElicitationCreate) {
			sawElicitation = true
			reply := JSONRPCMessage{
				JSONRPC: JSONRPC_VERSION,
				ID:      msg.ID,
				Result: map[string]any{
					"action": "accept",
					"content": map[string]any{
						"username": "alice",
					},
				},
			}
			data, err := json.Marshal(reply)
			if err != nil {
				t.Fatalf("marshal elicitation reply: %v", err)
			}
			postStreamableAccepted(t, httpServer.URL+"/mcp", sessionID, data)
		}
		if msg.ID == "call" && msg.Method == "" {
			sawResponse = true
			result, ok := msg.Result.(map[string]any)
			if !ok {
				t.Fatalf("call result has type %T, want object", msg.Result)
			}
			content, ok := result["content"].([]any)
			if !ok || len(content) != 1 {
				t.Fatalf("call content = %#v, want one item", result["content"])
			}
			text, ok := content[0].(map[string]any)
			if !ok || text["text"] != "alice" {
				t.Fatalf("call content = %#v, want alice", content[0])
			}
			break
		}
	}
	if !sawElicitation {
		t.Fatal("tools/call did not receive elicitation request")
	}
	if !sawResponse {
		t.Fatal("tools/call did not receive final response")
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

func postStreamableAccepted(t *testing.T, url, sessionID string, body []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest POST response: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(streamableSessionHeader, sessionID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST response status = %d, want %d: %s", resp.StatusCode, http.StatusAccepted, data)
	}
}
