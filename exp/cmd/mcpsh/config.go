package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// mcpConfig represents the structure of a .mcp.json file.
type mcpConfig struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

// mcpServerConfig represents a single server entry in .mcp.json.
type mcpServerConfig struct {
	Command  string            `json:"command"`
	Args     []string          `json:"args"`
	Env      map[string]string `json:"env,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
	URL      string            `json:"url,omitempty"`
}

// loadMCPConfig reads and parses a .mcp.json file.
func loadMCPConfig(path string) (*mcpConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg mcpConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// findMCPConfig walks up from the given directory looking for .mcp.json.
func findMCPConfig(dir string) string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".mcp.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// serverCommand builds a shell command string from a server config entry.
func serverCommand(cfg mcpServerConfig) string {
	parts := make([]string, 0, 1+len(cfg.Args))
	parts = append(parts, shellQuote(cfg.Command))
	for _, arg := range cfg.Args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

// enabledServerNames returns sorted names of non-disabled servers.
func enabledServerNames(cfg *mcpConfig) []string {
	var names []string
	for name, srv := range cfg.MCPServers {
		if !srv.Disabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// resolveConfig applies --config and --server flags to bootstrap options.
// With --server, it sets opts.Cmd/opts.HTTPURL for a single server.
// Without --server, it populates opts.configServers for multi-server mode.
func resolveConfig(opts *bootstrapOptions) error {
	if opts.ConfigFile == "" && opts.ServerName == "" {
		return nil
	}
	// If --server is given without --config, auto-discover.
	if opts.ConfigFile == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
		opts.ConfigFile = findMCPConfig(cwd)
		if opts.ConfigFile == "" {
			return fmt.Errorf("no .mcp.json found (looked upward from %s)", cwd)
		}
	}
	cfg, err := loadMCPConfig(opts.ConfigFile)
	if err != nil {
		return err
	}
	names := enabledServerNames(cfg)
	if len(names) == 0 {
		return fmt.Errorf("no enabled servers in %s", opts.ConfigFile)
	}

	// Single server: --server given, or only one in config.
	if opts.ServerName != "" || len(names) == 1 {
		name := opts.ServerName
		if name == "" {
			name = names[0]
		}
		return resolveSingleServer(opts, cfg, name)
	}

	// Multi-server: connect to all enabled servers.
	for _, name := range names {
		srv := cfg.MCPServers[name]
		entry := configServerEntry{name: name, cfg: srv}
		if srv.Command != "" {
			entry.command = envPrefix(srv) + serverCommand(srv)
		}
		opts.configServers = append(opts.configServers, entry)
	}
	return nil
}

func resolveSingleServer(opts *bootstrapOptions, cfg *mcpConfig, name string) error {
	names := enabledServerNames(cfg)
	srv, ok := cfg.MCPServers[name]
	if !ok {
		return fmt.Errorf("server %q not found in %s; available: %s",
			name, opts.ConfigFile, strings.Join(names, ", "))
	}
	if srv.Disabled {
		return fmt.Errorf("server %q is disabled in %s", name, opts.ConfigFile)
	}
	if srv.URL != "" {
		if srv.Command != "" {
			return fmt.Errorf("server %q has both command and url in %s", name, opts.ConfigFile)
		}
		opts.HTTPURL = srv.URL
		return nil
	}
	if srv.Command == "" {
		return fmt.Errorf("server %q has no command or url in %s", name, opts.ConfigFile)
	}
	opts.Cmd = envPrefix(srv) + serverCommand(srv)
	return nil
}

func envPrefix(srv mcpServerConfig) string {
	if len(srv.Env) == 0 {
		return ""
	}
	envKeys := make([]string, 0, len(srv.Env))
	for k := range srv.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	var parts []string
	for _, k := range envKeys {
		parts = append(parts, shellQuote(k+"="+srv.Env[k]))
	}
	return strings.Join(parts, " ") + " "
}
