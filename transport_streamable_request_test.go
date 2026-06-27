// Copyright 2025 The MCP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// startStreamableSession spins up a streamable HTTP server for the given MCP
// server and initializes a session with the supplied client capabilities JSON.
// It returns the base URL and the session id.
func startStreamableSession(t *testing.T, server *Server, capabilities string) (string, string) {
	t.Helper()
	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	t.Cleanup(httpServer.Close)
	url := httpServer.URL + "/mcp"
	sessionID, _ := postStreamable(t, url, "", `{
		"jsonrpc":"2.0",
		"id":"init",
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-11-25",
			"capabilities":`+capabilities+`,
			"clientInfo":{"name":"streamable-test-client","version":"0.0.0"}
		}
	}`)
	if sessionID == "" {
		t.Fatal("initial POST did not return Mcp-Session-Id")
	}
	return url, sessionID
}

// callToolStreaming POSTs a tools/call and drives the resulting SSE stream,
// answering any server-initiated request matching method via answer (which
// returns the JSON-RPC result object). It returns the final tool result object.
func callToolStreaming(t *testing.T, url, sessionID, tool, method string, answer func(req JSONRPCMessage) any) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{
		"jsonrpc":"2.0",
		"id":"call",
		"method":"tools/call",
		"params":{"name":"`+tool+`","arguments":{}}
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

	var sawRequest bool
	var result map[string]any
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
		if method != "" && msg.Method == method {
			sawRequest = true
			reply := JSONRPCMessage{JSONRPC: JSONRPC_VERSION, ID: msg.ID, Result: answer(msg)}
			data, err := json.Marshal(reply)
			if err != nil {
				t.Fatalf("marshal reply: %v", err)
			}
			postStreamableAccepted(t, url, sessionID, data)
		}
		if msg.ID == "call" && msg.Method == "" {
			r, _ := msg.Result.(map[string]any)
			result = r
			break
		}
	}
	if method != "" && !sawRequest {
		t.Fatalf("tools/call did not receive %s request", method)
	}
	if result == nil {
		t.Fatal("tools/call did not receive final response")
	}
	return result
}

// TestStreamableHTTPInlineListRootsRequest exercises Server.ListRoots end to end:
// a tool requests the client's roots, the client answers over SSE, and the result
// flows back through the tool result.
func TestStreamableHTTPInlineListRootsRequest(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "roots"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		result, err := server.ListRoots(ctx)
		if err != nil {
			return nil, err
		}
		name := ""
		if len(result.Roots) > 0 {
			name = result.Roots[0].Name
		}
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: name}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	url, sessionID := startStreamableSession(t, server, `{"roots":{"listChanged":true}}`)
	result := callToolStreaming(t, url, sessionID, "roots", string(MethodRootsList), func(JSONRPCMessage) any {
		return map[string]any{"roots": []map[string]any{{"uri": "file:///work", "name": "work"}}}
	})

	content, ok := result["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("call content = %#v, want one item", result["content"])
	}
	text, ok := content[0].(map[string]any)
	if !ok || text["text"] != "work" {
		t.Fatalf("call content = %#v, want roots name work", content[0])
	}
}

// TestStreamableHTTPInlineElicitationURLMode exercises URL-mode elicitation end
// to end against a client that advertises only the url elicitation capability.
func TestStreamableHTTPInlineElicitationURLMode(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "elicit-url"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		result, err := server.Elicit(ctx, ElicitRequest{
			Message: "open this",
			URL:     "https://example.com/auth",
		})
		if err != nil {
			return nil, err
		}
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: result.Action}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	url, sessionID := startStreamableSession(t, server, `{"elicitation":{"url":{}}}`)
	var gotMode, gotURL string
	result := callToolStreaming(t, url, sessionID, "elicit-url", string(MethodElicitationCreate), func(req JSONRPCMessage) any {
		if params, ok := req.Params.(map[string]any); ok {
			gotMode, _ = params["mode"].(string)
			gotURL, _ = params["url"].(string)
		}
		return map[string]any{"action": "accept"}
	})

	if gotMode != "url" {
		t.Errorf("elicitation mode = %q, want url", gotMode)
	}
	if gotURL != "https://example.com/auth" {
		t.Errorf("elicitation url = %q, want example url", gotURL)
	}
	content, _ := result["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("call content = %#v, want one item", result["content"])
	}
	if text, _ := content[0].(map[string]any); text["text"] != "accept" {
		t.Errorf("call content = %#v, want accept", content[0])
	}
}

// TestStreamableHTTPElicitationDecline verifies a declined elicitation flows back
// to the tool handler without error.
func TestStreamableHTTPElicitationDecline(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "elicit"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		result, err := server.Elicit(ctx, ElicitRequest{Message: "need input"})
		if err != nil {
			return nil, err
		}
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: result.Action}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	url, sessionID := startStreamableSession(t, server, `{"elicitation":{"form":{}}}`)
	result := callToolStreaming(t, url, sessionID, "elicit", string(MethodElicitationCreate), func(JSONRPCMessage) any {
		return map[string]any{"action": "decline"}
	})
	content, _ := result["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("call content = %#v, want one item", result["content"])
	}
	if text, _ := content[0].(map[string]any); text["text"] != "decline" {
		t.Errorf("call content = %#v, want decline", content[0])
	}
}

// TestStreamableHTTPElicitationCompleteNotification verifies that
// Server.NotifyElicitationComplete reaches the client over the active SSE stream.
func TestStreamableHTTPElicitationCompleteNotification(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterTool(Tool{Name: "complete"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		if err := server.NotifyElicitationComplete(ctx, "elicit-42"); err != nil {
			return nil, err
		}
		return &CallToolResult{Content: []any{TextContent{Type: "text", Text: "done"}}}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	url, sessionID := startStreamableSession(t, server, `{}`)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{
		"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"complete","arguments":{}}
	}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(streamableSessionHeader, sessionID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	var sawComplete, sawResponse bool
	for evt, err := range scanEvents(resp.Body) {
		if err != nil {
			t.Fatalf("scan SSE: %v", err)
		}
		if len(evt.data) == 0 {
			continue
		}
		var msg JSONRPCMessage
		if err := json.Unmarshal(evt.data, &msg); err != nil {
			t.Fatalf("decode SSE: %v", err)
		}
		if msg.Method == string(MethodElicitationComplete) {
			sawComplete = true
			params, _ := msg.Params.(map[string]any)
			if params["elicitationId"] != "elicit-42" {
				t.Errorf("elicitationId = %v, want elicit-42", params["elicitationId"])
			}
		}
		if msg.ID == "call" && msg.Method == "" {
			sawResponse = true
			break
		}
	}
	if !sawComplete {
		t.Error("did not observe elicitation complete notification")
	}
	if !sawResponse {
		t.Error("did not observe tool response")
	}
}

// TestStreamableHTTPResourceUpdatedNotification verifies that a resources/updated
// notification reaches a subscribed client over the standalone GET SSE stream.
func TestStreamableHTTPResourceUpdatedNotification(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterResource(Resource{URI: "test://doc", Name: "doc"}, func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{TextResourceContents{URI: "test://doc", Text: "v1"}}, nil
	}); err != nil {
		t.Fatalf("RegisterResource: %v", err)
	}

	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()
	url := httpServer.URL + "/mcp"

	sessionID, _ := postStreamable(t, url, "", `{
		"jsonrpc":"2.0","id":"init","method":"initialize",
		"params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"c","version":"0"}}
	}`)
	if sessionID == "" {
		t.Fatal("no session id")
	}

	// Subscribe to the resource.
	postStreamable(t, url, sessionID, `{"jsonrpc":"2.0","id":"sub","method":"resources/subscribe","params":{"uri":"test://doc"}}`)

	// Open the standalone GET stream where out-of-band notifications are routed.
	getReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest GET: %v", err)
	}
	getReq.Header.Set("Accept", "text/event-stream")
	getReq.Header.Set(streamableSessionHeader, sessionID)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer getResp.Body.Close()

	// Trigger the update from another goroutine once the GET stream is reading.
	go func() {
		_ = server.ResourceUpdated(context.Background(), ResourceUpdatedNotificationParams{URI: "test://doc"})
	}()

	var sawUpdate bool
	for evt, err := range scanEvents(getResp.Body) {
		if err != nil {
			t.Fatalf("scan SSE: %v", err)
		}
		if len(evt.data) == 0 {
			continue
		}
		var msg JSONRPCMessage
		if err := json.Unmarshal(evt.data, &msg); err != nil {
			continue
		}
		if msg.Method == string(MethodResourceUpdated) {
			params, _ := msg.Params.(map[string]any)
			if params["uri"] == "test://doc" {
				sawUpdate = true
				break
			}
		}
	}
	if !sawUpdate {
		t.Error("did not observe resources/updated notification on GET stream")
	}
}

// TestStreamableHTTPToolListChangedNotification verifies that registering a tool
// after a client has connected emits a tools/list_changed notification over the
// wire (and not merely to in-process dispatcher subscribers).
func TestStreamableHTTPToolListChangedNotification(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")

	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()
	url := httpServer.URL + "/mcp"

	sessionID, _ := postStreamable(t, url, "", `{
		"jsonrpc":"2.0","id":"init","method":"initialize",
		"params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"c","version":"0"}}
	}`)
	if sessionID == "" {
		t.Fatal("no session id")
	}

	getReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest GET: %v", err)
	}
	getReq.Header.Set("Accept", "text/event-stream")
	getReq.Header.Set(streamableSessionHeader, sessionID)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer getResp.Body.Close()

	go func() {
		_ = server.RegisterTool(Tool{Name: "late"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
			return &CallToolResult{}, nil
		})
	}()

	var sawListChanged bool
	for evt, err := range scanEvents(getResp.Body) {
		if err != nil {
			t.Fatalf("scan SSE: %v", err)
		}
		if len(evt.data) == 0 {
			continue
		}
		var msg JSONRPCMessage
		if err := json.Unmarshal(evt.data, &msg); err != nil {
			continue
		}
		if msg.Method == string(MethodToolListChanged) {
			sawListChanged = true
			break
		}
	}
	if !sawListChanged {
		t.Error("did not observe tools/list_changed notification on GET stream")
	}
}

// TestStreamableHTTPServerRequestTimeout verifies that a server-initiated request
// fails with an error when the client never answers and the configured
// serverRequestTimeout elapses.
func TestStreamableHTTPServerRequestTimeout(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0", WithServerRequestTimeout(150*time.Millisecond))
	toolErr := make(chan error, 1)
	if err := server.RegisterTool(Tool{Name: "slow"}, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		_, err := server.Elicit(ctx, ElicitRequest{Message: "answer me"})
		toolErr <- err
		if err != nil {
			return nil, err
		}
		return &CallToolResult{}, nil
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	url, sessionID := startStreamableSession(t, server, `{"elicitation":{"form":{}}}`)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{
		"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"slow","arguments":{}}
	}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(streamableSessionHeader, sessionID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	// Drain the stream until the tool error response arrives. The client never
	// answers the elicitation request, so the timeout must fire.
	for evt, err := range scanEvents(resp.Body) {
		if err != nil {
			break
		}
		if len(evt.data) == 0 {
			continue
		}
		var msg JSONRPCMessage
		if err := json.Unmarshal(evt.data, &msg); err != nil {
			continue
		}
		if msg.ID == "call" && msg.Method == "" {
			break
		}
	}

	select {
	case err := <-toolErr:
		if err == nil {
			t.Fatal("Elicit returned no error despite client never answering")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("tool handler did not return after server request timeout")
	}
}

// TestStreamableTransportCorrelationCleanup drives the transport directly to
// verify that completing a client request frees its stream correlation and any
// server-request correlations pinned to it (including ones abandoned on
// timeout), and that client and server request ids in the same numeric space do
// not collide.
func TestStreamableTransportCorrelationCleanup(t *testing.T) {
	tr := newStreamableServerTransport("sess", nil)

	clientSid := streamID(7)
	// Inbound client request id=1 arrives on stream 7.
	if err := tr.receive(context.Background(), JSONRPCMessage{ID: float64(1), Method: "tools/call", JSONRPC: "2.0"}, clientSid); err != nil {
		t.Fatalf("receive client request: %v", err)
	}
	// Drain it so the buffered incoming channel does not block subsequent ops.
	if _, err := tr.Read(context.Background()); err != nil {
		t.Fatalf("read client request: %v", err)
	}

	// Server-initiated request id=1 (same numeric id as the client request) is
	// emitted while handling the client request: it must route to stream 7 and
	// be tracked separately from the client request.
	if err := tr.Write(context.Background(), JSONRPCMessage{ID: float64(1), Method: "sampling/createMessage", JSONRPC: "2.0"}); err != nil {
		t.Fatalf("write server request: %v", err)
	}
	tr.mu.RLock()
	srvSid, ok := tr.serverRequestStreams[float64(1)]
	cliSid := tr.clientRequestStreams[float64(1)]
	tr.mu.RUnlock()
	if !ok || srvSid != clientSid {
		t.Fatalf("server request routed to stream %v (ok=%v), want %v", srvSid, ok, clientSid)
	}
	if cliSid != clientSid {
		t.Fatalf("client request correlation overwritten: got %v, want %v", cliSid, clientSid)
	}

	// The client never answers the server request (timeout). The tool returns
	// and the server writes the client response, which must route to stream 7
	// (not the default), and must free the whole stream including the orphaned
	// server-request entry.
	tr.mu.Lock()
	sid := tr.getStreamID(JSONRPCMessage{ID: float64(1), JSONRPC: "2.0"})
	tr.mu.Unlock()
	if sid != clientSid {
		t.Fatalf("client response routed to stream %v, want %v", sid, clientSid)
	}
	if err := tr.Write(context.Background(), JSONRPCMessage{ID: float64(1), Result: map[string]any{}, JSONRPC: "2.0"}); err != nil {
		t.Fatalf("write client response: %v", err)
	}

	tr.mu.RLock()
	defer tr.mu.RUnlock()
	if len(tr.clientRequestStreams) != 0 {
		t.Errorf("clientRequestStreams not cleaned: %v", tr.clientRequestStreams)
	}
	if len(tr.serverRequestStreams) != 0 {
		t.Errorf("serverRequestStreams not cleaned (orphaned server request leaked): %v", tr.serverRequestStreams)
	}
	if tr.lastRequestStream != 0 {
		t.Errorf("lastRequestStream not reset: %v", tr.lastRequestStream)
	}
}

// TestStreamableHTTPOutOfBandBeforeGET verifies that an out-of-band server
// notification emitted before any GET stream connects is still delivered once a
// fresh GET (no Last-Event-ID) opens.
func TestStreamableHTTPOutOfBandBeforeGET(t *testing.T) {
	server := NewServer("streamable-test", "0.0.0")
	if err := server.RegisterResource(Resource{URI: "test://doc", Name: "doc"}, func(ctx context.Context, req ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{TextResourceContents{URI: "test://doc", Text: "v1"}}, nil
	}); err != nil {
		t.Fatalf("RegisterResource: %v", err)
	}

	httpServer := httptest.NewServer(NewStreamableHTTPHandler(func(*http.Request) *Server {
		return server
	}, nil))
	defer httpServer.Close()
	url := httpServer.URL + "/mcp"

	sessionID, _ := postStreamable(t, url, "", `{
		"jsonrpc":"2.0","id":"init","method":"initialize",
		"params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"c","version":"0"}}
	}`)
	if sessionID == "" {
		t.Fatal("no session id")
	}
	postStreamable(t, url, sessionID, `{"jsonrpc":"2.0","id":"sub","method":"resources/subscribe","params":{"uri":"test://doc"}}`)

	// Emit the out-of-band notification BEFORE any GET stream is open. It should
	// be buffered on the standalone GET stream and replayed when the GET arrives.
	if err := server.ResourceUpdated(context.Background(), ResourceUpdatedNotificationParams{URI: "test://doc"}); err != nil {
		t.Fatalf("ResourceUpdated: %v", err)
	}

	getReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest GET: %v", err)
	}
	getReq.Header.Set("Accept", "text/event-stream")
	getReq.Header.Set(streamableSessionHeader, sessionID)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer getResp.Body.Close()

	done := make(chan bool, 1)
	go func() {
		for evt, err := range scanEvents(getResp.Body) {
			if err != nil || len(evt.data) == 0 {
				continue
			}
			var msg JSONRPCMessage
			if err := json.Unmarshal(evt.data, &msg); err != nil {
				continue
			}
			if msg.Method == string(MethodResourceUpdated) {
				done <- true
				return
			}
		}
		done <- false
	}()

	select {
	case ok := <-done:
		if !ok {
			t.Error("out-of-band notification emitted before GET was not delivered to the fresh GET stream")
		}
	case <-time.After(5 * time.Second):
		t.Error("timed out waiting for buffered out-of-band notification")
	}
}
