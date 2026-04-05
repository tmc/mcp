package main

import (
	"os"
	"path/filepath"
	"testing"
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
	err := resolveConfig(&opts)
	if err == nil {
		t.Fatal("expected error for multiple servers without --server")
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
