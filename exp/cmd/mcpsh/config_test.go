package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
)

func TestLoadMCPConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	os.WriteFile(path, []byte(`{
		"mcpServers": {
			"echo": {
				"command": "echo-server",
				"args": ["--stdio"]
			},
			"web": {
				"url": "http://localhost:8080/mcp"
			},
			"off": {
				"command": "disabled-server",
				"disabled": true
			}
		}
	}`), 0o644)

	cfg, err := loadMCPConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MCPServers) != 3 {
		t.Fatalf("got %d servers, want 3", len(cfg.MCPServers))
	}
	names := enabledServerNames(cfg)
	if len(names) != 2 {
		t.Fatalf("got %d enabled servers, want 2", len(names))
	}
	if names[0] != "echo" || names[1] != "web" {
		t.Fatalf("got names %v, want [echo web]", names)
	}
}

func TestFindMCPConfig(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(nested, 0o755)
	cfgPath := filepath.Join(dir, ".mcp.json")
	os.WriteFile(cfgPath, []byte(`{"mcpServers":{}}`), 0o644)

	got := findMCPConfig(nested)
	if got != cfgPath {
		t.Fatalf("findMCPConfig(%s) = %s, want %s", nested, got, cfgPath)
	}
}

func TestFindMCPConfig_notFound(t *testing.T) {
	dir := t.TempDir()
	got := findMCPConfig(dir)
	if got != "" {
		t.Fatalf("findMCPConfig(%s) = %s, want empty", dir, got)
	}
}

func TestServerCommand(t *testing.T) {
	cfg := mcpServerConfig{
		Command: "/usr/bin/echo",
		Args:    []string{"--stdio", "hello world"},
	}
	got := serverCommand(cfg)
	want := "/usr/bin/echo --stdio 'hello world'"
	if got != want {
		t.Fatalf("serverCommand = %q, want %q", got, want)
	}
}

func TestResolveConfig_singleServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	os.WriteFile(path, []byte(`{
		"mcpServers": {
			"test": {
				"command": "test-server",
				"args": ["--stdio"]
			}
		}
	}`), 0o644)

	opts := bootstrapOptions{ConfigFile: path}
	if err := resolveConfig(&opts); err != nil {
		t.Fatal(err)
	}
	if opts.Cmd == "" {
		t.Fatal("expected Cmd to be set")
	}
	if opts.Cmd != "test-server --stdio" {
		t.Fatalf("Cmd = %q, want %q", opts.Cmd, "test-server --stdio")
	}
}

func TestResolveConfig_multipleServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	os.WriteFile(path, []byte(`{
		"mcpServers": {
			"a": {"command": "a-server"},
			"b": {"command": "b-server"}
		}
	}`), 0o644)

	opts := bootstrapOptions{ConfigFile: path}
	if err := resolveConfig(&opts); err != nil {
		t.Fatal(err)
	}
	if len(opts.configServers) != 2 {
		t.Fatalf("got %d config servers, want 2", len(opts.configServers))
	}
	if opts.configServers[0].name != "a" || opts.configServers[1].name != "b" {
		t.Fatalf("got names %q %q, want a b", opts.configServers[0].name, opts.configServers[1].name)
	}
	if opts.Cmd != "" {
		t.Fatal("Cmd should not be set in multi-server mode")
	}
}

func TestResolveConfig_namedServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	os.WriteFile(path, []byte(`{
		"mcpServers": {
			"a": {"command": "a-server", "args": ["-v"]},
			"b": {"command": "b-server"}
		}
	}`), 0o644)

	opts := bootstrapOptions{ConfigFile: path, ServerName: "a"}
	if err := resolveConfig(&opts); err != nil {
		t.Fatal(err)
	}
	if opts.Cmd != "a-server -v" {
		t.Fatalf("Cmd = %q, want %q", opts.Cmd, "a-server -v")
	}
}

func TestResolveConfig_urlServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	os.WriteFile(path, []byte(`{
		"mcpServers": {
			"web": {"url": "http://localhost:8080/mcp"}
		}
	}`), 0o644)

	opts := bootstrapOptions{ConfigFile: path}
	if err := resolveConfig(&opts); err != nil {
		t.Fatal(err)
	}
	if opts.HTTPURL != "http://localhost:8080/mcp" {
		t.Fatalf("HTTPURL = %q, want %q", opts.HTTPURL, "http://localhost:8080/mcp")
	}
}

func TestResolveConfig_withEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	os.WriteFile(path, []byte(`{
		"mcpServers": {
			"test": {
				"command": "server",
				"env": {"FOO": "bar", "BAZ": "qux"}
			}
		}
	}`), 0o644)

	opts := bootstrapOptions{ConfigFile: path}
	if err := resolveConfig(&opts); err != nil {
		t.Fatal(err)
	}
	want := "BAZ=qux FOO=bar server"
	if opts.Cmd != want {
		t.Fatalf("Cmd = %q, want %q", opts.Cmd, want)
	}
}

func TestResolveConfig_noFlags(t *testing.T) {
	opts := bootstrapOptions{}
	if err := resolveConfig(&opts); err != nil {
		t.Fatal(err)
	}
	if opts.Cmd != "" {
		t.Fatal("expected no Cmd when no config flags given")
	}
}

func TestMultiServerAllTools(t *testing.T) {
	a := &app{
		servers: []*serverConn{
			{
				name: "srv-a",
				tools: []mcp.Tool{
					{Name: "alpha"},
					{Name: "beta"},
				},
			},
			{
				name: "srv-b",
				tools: []mcp.Tool{
					{Name: "gamma"},
				},
			},
		},
	}
	tools := a.allTools()
	if len(tools) != 3 {
		t.Fatalf("got %d tools, want 3", len(tools))
	}
	// Multi-server should prefix with server name.
	names := make([]string, len(tools))
	for i, nt := range tools {
		names[i] = nt.displayName()
	}
	// Sorted: srv-a/alpha, srv-a/beta, srv-b/gamma
	if names[0] != "srv-a/alpha" || names[1] != "srv-a/beta" || names[2] != "srv-b/gamma" {
		t.Fatalf("tool names = %v", names)
	}
}

func TestSingleServerNoPrefix(t *testing.T) {
	a := &app{
		servers: []*serverConn{
			{
				name:  "only",
				tools: []mcp.Tool{{Name: "alpha"}},
			},
		},
	}
	tools := a.allTools()
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(tools))
	}
	if tools[0].displayName() != "alpha" {
		t.Fatalf("displayName = %q, want %q", tools[0].displayName(), "alpha")
	}
}

func TestMultiServerToolCommands(t *testing.T) {
	backendA := &fakeBackend{result: &mcp.CallToolResult{
		Content: []any{map[string]any{"type": "text", "text": "from-a"}},
	}}
	backendB := &fakeBackend{result: &mcp.CallToolResult{
		Content: []any{map[string]any{"type": "text", "text": "from-b"}},
	}}
	a := &app{
		servers: []*serverConn{
			{
				name:    "srv-a",
				backend: backendA,
				tools: []mcp.Tool{
					{Name: "echo", Description: "Echo from A", InputSchema: json.RawMessage(`{"type":"object","properties":{"msg":{"type":"string"}},"required":["msg"]}`)},
				},
			},
			{
				name:    "srv-b",
				backend: backendB,
				tools: []mcp.Tool{
					{Name: "ping", Description: "Ping from B"},
				},
			},
		},
		opts: bootstrapOptions{Timeout: time.Second},
	}
	root := &cobra.Command{Use: toolName}
	root.AddGroup(&cobra.Group{ID: groupTools, Title: "Discovered Tools"})
	addToolCommands(root, a)

	// srv-a echo should work
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"srv-a", "echo", "--msg", "hello"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if backendA.call.Name != "echo" {
		t.Fatalf("expected call to echo, got %q", backendA.call.Name)
	}
	if !strings.Contains(out.String(), "from-a") {
		t.Fatalf("output = %q", out.String())
	}

	// srv-b ping should work
	out.Reset()
	root.SetArgs([]string{"srv-b", "ping"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if backendB.call.Name != "ping" {
		t.Fatalf("expected call to ping, got %q", backendB.call.Name)
	}
}
