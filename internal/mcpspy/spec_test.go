package mcpspy

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSpecTrackerBuildsSpecFromTraffic(t *testing.T) {
	rec := New(io.Discard, Options{})
	specPath := filepath.Join(t.TempDir(), "trace.mcpspec")
	tracker := NewSpecTracker(rec, SpecOptions{
		Path: specPath,
		Name: "fallback-name",
	})
	defer tracker.Close()

	send := func(dir, line string) {
		t.Helper()
		if _, err := rec.Writer(dir, io.Discard).Write([]byte(line + "\n")); err != nil {
			t.Fatal(err)
		}
	}

	send("recv", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`)
	send("send", `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","serverInfo":{"name":"weather","version":"1.2.3"},"instructions":"Weather server"}}`)
	send("recv", `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	send("send", `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"get_weather","description":"Get weather","inputSchema":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}]}}`)
	send("recv", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_weather","arguments":{"city":"Paris","units":"metric"}}}`)
	send("send", `{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"sunny"}]}}`)
	send("recv", `{"jsonrpc":"2.0","id":4,"method":"prompts/get","params":{"name":"weather_report","arguments":{"city":"Paris"}}}`)
	send("recv", `{"jsonrpc":"2.0","id":5,"method":"resources/list","params":{}}`)
	send("send", `{"jsonrpc":"2.0","id":5,"result":{"resources":[{"uri":"weather://Paris/current","name":"Current Weather","mimeType":"application/json","description":"Current conditions"}]}}`)

	snapshot := waitForSpec(t, tracker, func(snapshot SpecSnapshot) bool {
		return snapshot.Spec.Server.Name == "weather" &&
			len(snapshot.Spec.Tools) == 1 &&
			len(snapshot.Spec.Prompts) == 1 &&
			len(snapshot.Spec.Resources) == 1
	})

	if snapshot.Path != specPath {
		t.Fatalf("path=%q, want %q", snapshot.Path, specPath)
	}
	if snapshot.Spec.Server.Name != "weather" {
		t.Fatalf("server name=%q", snapshot.Spec.Server.Name)
	}
	if snapshot.Spec.Server.Version != "1.2.3" {
		t.Fatalf("server version=%q", snapshot.Spec.Server.Version)
	}
	if len(snapshot.Spec.Tools) != 1 {
		t.Fatalf("tools len=%d, want 1", len(snapshot.Spec.Tools))
	}
	tool := snapshot.Spec.Tools[0]
	if tool.Name != "get_weather" {
		t.Fatalf("tool name=%q", tool.Name)
	}
	if !strings.Contains(string(tool.InputSchema), `"city"`) {
		t.Fatalf("inputSchema=%s", tool.InputSchema)
	}
	if !strings.Contains(string(tool.ReturnType), `"content"`) {
		t.Fatalf("returnType=%s", tool.ReturnType)
	}
	if len(snapshot.Spec.Prompts) != 1 || len(snapshot.Spec.Prompts[0].Arguments) != 1 {
		t.Fatalf("prompts=%+v", snapshot.Spec.Prompts)
	}
	if snapshot.Spec.Prompts[0].Arguments[0].Name != "city" || !snapshot.Spec.Prompts[0].Arguments[0].Required {
		t.Fatalf("prompt args=%+v", snapshot.Spec.Prompts[0].Arguments)
	}
	if len(snapshot.Spec.Resources) != 1 || snapshot.Spec.Resources[0].MimeType != "application/json" {
		t.Fatalf("resources=%+v", snapshot.Spec.Resources)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	var onDisk SpecDocument
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("unmarshal disk spec: %v", err)
	}
	if onDisk.Server.Name != "weather" {
		t.Fatalf("disk server=%q", onDisk.Server.Name)
	}
}

func TestSpecFilenameFor(t *testing.T) {
	t.Setenv("HOME", "/tmp/mcpspy-home")
	tests := []struct {
		in   string
		want string
	}{
		{"", "/tmp/mcpspy-home/.mcpspy/specs/stdin.mcpspec"},
		{"trace.mcp", "/tmp/mcpspy-home/.mcpspy/specs/trace.mcpspec"},
		{"/tmp/trace.json", "/tmp/mcpspy-home/.mcpspy/specs/trace.mcpspec"},
		{"trace", "/tmp/mcpspy-home/.mcpspy/specs/trace.mcpspec"},
		{"mcp server", "/tmp/mcpspy-home/.mcpspy/specs/mcp-server.mcpspec"},
	}
	for _, tt := range tests {
		if got := SpecFilenameFor(tt.in); got != tt.want {
			t.Fatalf("SpecFilenameFor(%q)=%q, want %q", tt.in, got, tt.want)
		}
	}
}

func waitForSpec(t *testing.T, tracker *SpecTracker, ready func(SpecSnapshot) bool) SpecSnapshot {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := tracker.Snapshot()
		if ready(snapshot) {
			return snapshot
		}
		time.Sleep(20 * time.Millisecond)
	}
	snapshot := tracker.Snapshot()
	t.Fatalf("timed out waiting for spec: %+v", snapshot)
	return SpecSnapshot{}
}
