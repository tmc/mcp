package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
)

type fakeBackend struct {
	call     mcp.CallToolRequest
	result   *mcp.CallToolResult
	tools    []mcp.Tool
	init     *mcp.InitializeResult
	notified []string
}

func (f *fakeBackend) Initialize(context.Context, mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	if f.init != nil {
		return f.init, nil
	}
	return &mcp.InitializeResult{}, nil
}

func (f *fakeBackend) ListTools(context.Context, mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{Tools: f.tools}, nil
}

func (f *fakeBackend) CallTool(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	f.call = req
	if f.result != nil {
		return f.result, nil
	}
	return &mcp.CallToolResult{}, nil
}

func (f *fakeBackend) Notify(_ context.Context, method string, _ any) error {
	f.notified = append(f.notified, method)
	return nil
}

func (f *fakeBackend) Close() error { return nil }

func TestParseBootstrapArgs(t *testing.T) {
	opts, err := parseBootstrapArgs([]string{"echo", "--cmd=server --flag", "--timeout", "2s", "--raw"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Cmd != "server --flag" {
		t.Fatalf("cmd=%q", opts.Cmd)
	}
	if opts.Timeout != 2*time.Second {
		t.Fatalf("timeout=%v", opts.Timeout)
	}
	if !opts.Raw {
		t.Fatal("raw flag not set")
	}
	if opts.ServerStderr {
		t.Fatal("server stderr should default to false")
	}

	opts, err = parseBootstrapArgs([]string{"tools", "--cmd", "server", "--server-stderr"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ServerStderr {
		t.Fatal("server stderr flag not set")
	}

	opts, err = parseBootstrapArgs([]string{"tools", "--cmd", "server", "--spy-record", "session.mcp", "--spy-ui", "--spy-pretty", "--spy-spec-file", "server.mcpspec"})
	if err == nil || !strings.Contains(err.Error(), "--spy-pretty is not supported") {
		t.Fatalf("err=%v", err)
	}

	opts, err = parseBootstrapArgs([]string{"tools", "--http", "http://localhost:8080/mcp", "--spy-record", "session.mcp", "--spy-ui", "--spy-pretty", "--spy-spec-file", "server.mcpspec"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.SpyRecord != "session.mcp" || !opts.SpyUI || !opts.SpyPretty || opts.SpySpecFile != "server.mcpspec" {
		t.Fatalf("unexpected spy opts: %+v", opts)
	}
}

func TestParseBootstrapArgsRejectsSpyOpenWithoutUI(t *testing.T) {
	_, err := parseBootstrapArgs([]string{"tools", "--cmd", "server", "--spy-open"})
	if err == nil || !strings.Contains(err.Error(), "--spy-open requires --spy-ui") {
		t.Fatalf("err=%v", err)
	}
}

func TestDynamicToolCommandExecutes(t *testing.T) {
	backend := &fakeBackend{
		result: &mcp.CallToolResult{
			Content: []any{map[string]any{"type": "text", "text": "done"}},
		},
	}
	app := &app{
		servers: []*serverConn{{
			name:    "fake",
			backend: backend,
			info:    &mcp.InitializeResult{ServerInfo: mcp.Implementation{Name: "fake", Version: "1.0.0"}},
			tools: []mcp.Tool{
				{
					Name:        "echo",
					Description: "Echo a message",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"message"},"mode":{"type":"string","enum":["loud","quiet"]}},"required":["message"]}`),
				},
			},
		}},
		opts: bootstrapOptions{Timeout: time.Second},
	}
	root := &cobra.Command{Use: toolName}
	root.AddGroup(&cobra.Group{ID: groupTools, Title: "Discovered Tools"})
	root.SetOut(new(bytes.Buffer))
	addToolCommands(root, app)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"echo", "--message", "hello", "--mode", "loud"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if backend.call.Name != "echo" {
		t.Fatalf("name=%q", backend.call.Name)
	}
	var args map[string]any
	if err := json.Unmarshal(backend.call.Arguments, &args); err != nil {
		t.Fatal(err)
	}
	if args["message"] != "hello" || args["mode"] != "loud" {
		t.Fatalf("args=%v", args)
	}
	if strings.TrimSpace(out.String()) != "done" {
		t.Fatalf("out=%q", out.String())
	}
}

func TestDynamicToolCommandCompletion(t *testing.T) {
	backend := &fakeBackend{}
	app := &app{
		servers: []*serverConn{{
			name:    "fake",
			backend: backend,
			tools: []mcp.Tool{
				{
					Name:        "echo",
					Description: "Echo a message",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"mode":{"type":"string","enum":["loud","quiet"]}}}`),
				},
			},
		}},
		opts: bootstrapOptions{Timeout: time.Second},
	}
	root := &cobra.Command{Use: toolName}
	root.AddGroup(&cobra.Group{ID: groupTools, Title: "Discovered Tools"})
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddGroup(&cobra.Group{ID: groupMeta, Title: "Support Commands"})
	root.AddCommand(newCompletionCommand())
	addToolCommands(root, app)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"__complete", "echo", "--mode", ""})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "loud") || !strings.Contains(got, "quiet") {
		t.Fatalf("completion=%q", got)
	}
}

func TestToolsCommandListsDiscoveredTools(t *testing.T) {
	app := &app{
		servers: []*serverConn{{
			name: "test",
			tools: []mcp.Tool{
				{Name: "zeta", Description: "last"},
				{Name: "alpha", Description: "first"},
			},
		}},
		opts: bootstrapOptions{Timeout: time.Second},
	}
	root := &cobra.Command{Use: toolName}
	root.AddGroup(&cobra.Group{ID: groupMeta, Title: "Support Commands"})
	root.AddCommand(newToolsCommand(app))

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"tools"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "zeta") {
		t.Fatalf("tools=%q", got)
	}
}

func TestServerStderr(t *testing.T) {
	if got := serverStderr(bootstrapOptions{}); got != io.Discard {
		t.Fatalf("default stderr = %T, want io.Discard", got)
	}
	if got := serverStderr(bootstrapOptions{ServerStderr: true}); got != os.Stderr {
		t.Fatalf("forwarded stderr = %T, want os.Stderr", got)
	}
	if got := serverStderr(bootstrapOptions{SpyUI: true}); got != os.Stderr {
		t.Fatalf("spy ui stderr = %T, want os.Stderr", got)
	}
}

func TestWrappedCommandWithoutSpy(t *testing.T) {
	opts := bootstrapOptions{Cmd: "server --stdio"}
	if got := wrappedCommand(opts); got != opts.Cmd {
		t.Fatalf("wrapped=%q", got)
	}
}

func TestWrappedCommandWithSpy(t *testing.T) {
	opts := bootstrapOptions{
		Cmd:          "server --stdio",
		SpyRecord:    "session.mcp",
		SpyUI:        true,
		SpyPretty:    true,
		SpySpecFile:  "server.mcpspec",
		ServerStderr: false,
	}
	got := wrappedCommand(opts)
	for _, want := range []string{
		"mcpspy",
		"-f",
		"session.mcp",
		"-l",
		"--spec-file",
		"server.mcpspec",
		"--pass-through",
		"-no-stderr",
		"sh",
		"-lc",
		"server --stdio",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("wrapped=%q missing %q", got, want)
		}
	}
}

func TestNameNormalization(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		fn   func(string) string
	}{
		{name: "camel command", in: "ProjectListWindows", want: "project-list-windows", fn: cobraName},
		{name: "initialism command", in: "ProjectLS", want: "project-ls", fn: cobraName},
		{name: "camel flag", in: "tabIdentifier", want: "tab-identifier", fn: flagName},
		{name: "path flag", in: "sourceFilePath", want: "source-file-path", fn: flagName},
	}
	for _, tt := range tests {
		if got := tt.fn(tt.in); got != tt.want {
			t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestJSONFlagMergesWithExplicitFlags(t *testing.T) {
	backend := &fakeBackend{result: &mcp.CallToolResult{}}
	app := &app{
		servers: []*serverConn{{
			name:    "fake",
			backend: backend,
			tools: []mcp.Tool{
				{
					Name:        "compose",
					Description: "Compose an object",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"config":{"type":"object"}},"required":["name","config"]}`),
				},
			},
		}},
		opts: bootstrapOptions{Timeout: time.Second},
	}
	root := &cobra.Command{Use: toolName}
	root.AddGroup(&cobra.Group{ID: groupTools, Title: "Discovered Tools"})
	addToolCommands(root, app)
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"compose", "--json", `{"config":{"enabled":true}}`, "--name", "demo"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var args map[string]any
	if err := json.Unmarshal(backend.call.Arguments, &args); err != nil {
		t.Fatal(err)
	}
	if args["name"] != "demo" {
		t.Fatalf("args=%v", args)
	}
	config, ok := args["config"].(map[string]any)
	if !ok || config["enabled"] != true {
		t.Fatalf("args=%v", args)
	}
}
