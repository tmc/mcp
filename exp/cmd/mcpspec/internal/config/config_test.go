package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigSaveLoad(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-config-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config
	configPath := filepath.Join(tempDir, "config.json")
	testConfig := &Config{
		DefaultServer: "test-server",
		Servers: map[string]Server{
			"test-server": {
				Endpoint:  "http://localhost:9090",
				Transport: "http",
				Timeout:   60,
				TLSVerify: false,
				Environment: map[string]string{
					"DEBUG": "true",
				},
				InitParams: json.RawMessage(`{"capabilities": ["tool1", "tool2"]}`),
			},
		},
		LogLevel:       "debug",
		LogFile:        "/var/log/mcpspec.log",
		MaxMessageSize: 5 * 1024 * 1024,
	}

	// Save the config
	if err := SaveConfig(testConfig, configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Check that the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config file not created: %v", err)
	}

	// Load the config
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Compare the configs
	if loadedConfig.DefaultServer != testConfig.DefaultServer {
		t.Errorf("DefaultServer mismatch: got %q, want %q", loadedConfig.DefaultServer, testConfig.DefaultServer)
	}

	if loadedConfig.LogLevel != testConfig.LogLevel {
		t.Errorf("LogLevel mismatch: got %q, want %q", loadedConfig.LogLevel, testConfig.LogLevel)
	}

	if loadedConfig.LogFile != testConfig.LogFile {
		t.Errorf("LogFile mismatch: got %q, want %q", loadedConfig.LogFile, testConfig.LogFile)
	}

	if loadedConfig.MaxMessageSize != testConfig.MaxMessageSize {
		t.Errorf("MaxMessageSize mismatch: got %d, want %d", loadedConfig.MaxMessageSize, testConfig.MaxMessageSize)
	}

	// Check the server config
	testServer := testConfig.Servers["test-server"]
	loadedServer, ok := loadedConfig.Servers["test-server"]
	if !ok {
		t.Fatalf("test-server not found in loaded config")
	}

	if loadedServer.Endpoint != testServer.Endpoint {
		t.Errorf("Endpoint mismatch: got %q, want %q", loadedServer.Endpoint, testServer.Endpoint)
	}

	if loadedServer.Transport != testServer.Transport {
		t.Errorf("Transport mismatch: got %q, want %q", loadedServer.Transport, testServer.Transport)
	}

	if loadedServer.Timeout != testServer.Timeout {
		t.Errorf("Timeout mismatch: got %d, want %d", loadedServer.Timeout, testServer.Timeout)
	}

	if loadedServer.TLSVerify != testServer.TLSVerify {
		t.Errorf("TLSVerify mismatch: got %v, want %v", loadedServer.TLSVerify, testServer.TLSVerify)
	}

	// Compare the environment map
	if !reflect.DeepEqual(loadedServer.Environment, testServer.Environment) {
		t.Errorf("Environment mismatch: got %v, want %v", loadedServer.Environment, testServer.Environment)
	}

	// Compare the init params
	if string(loadedServer.InitParams) != string(testServer.InitParams) {
		t.Errorf("InitParams mismatch: got %s, want %s", loadedServer.InitParams, testServer.InitParams)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	if config.DefaultServer != "local" {
		t.Errorf("expected default server to be 'local', got %q", config.DefaultServer)
	}

	if len(config.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(config.Servers))
	}

	if _, ok := config.Servers["local"]; !ok {
		t.Errorf("expected 'local' server, not found")
	}

	if config.LogLevel != "info" {
		t.Errorf("expected log level to be 'info', got %q", config.LogLevel)
	}
}

func TestGetDefaultServer(t *testing.T) {
	config := &Config{
		DefaultServer: "test",
		Servers: map[string]Server{
			"test": {
				Endpoint: "http://localhost:8080",
			},
		},
	}

	server, err := config.GetDefaultServer()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if server.Endpoint != "http://localhost:8080" {
		t.Errorf("expected endpoint 'http://localhost:8080', got %q", server.Endpoint)
	}

	// Test missing default server
	config.DefaultServer = "nonexistent"
	_, err = config.GetDefaultServer()
	if err == nil {
		t.Error("expected error for nonexistent default server, got nil")
	}
}

func TestGetServer(t *testing.T) {
	config := &Config{
		Servers: map[string]Server{
			"test": {
				Endpoint: "http://localhost:8080",
			},
		},
	}

	server, err := config.GetServer("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if server.Endpoint != "http://localhost:8080" {
		t.Errorf("expected endpoint 'http://localhost:8080', got %q", server.Endpoint)
	}

	// Test nonexistent server
	_, err = config.GetServer("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent server, got nil")
	}
}

func TestAddServer(t *testing.T) {
	config := &Config{}

	// Add a server
	config.AddServer("test", Server{
		Endpoint: "http://localhost:8080",
	})

	if len(config.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(config.Servers))
	}

	server, ok := config.Servers["test"]
	if !ok {
		t.Fatal("expected 'test' server, not found")
	}

	if server.Endpoint != "http://localhost:8080" {
		t.Errorf("expected endpoint 'http://localhost:8080', got %q", server.Endpoint)
	}

	// Update the server
	config.AddServer("test", Server{
		Endpoint: "http://localhost:9090",
	})

	server = config.Servers["test"]
	if server.Endpoint != "http://localhost:9090" {
		t.Errorf("expected updated endpoint 'http://localhost:9090', got %q", server.Endpoint)
	}
}

func TestRemoveServer(t *testing.T) {
	config := &Config{
		DefaultServer: "test1",
		Servers: map[string]Server{
			"test1": {
				Endpoint: "http://localhost:8080",
			},
			"test2": {
				Endpoint: "http://localhost:9090",
			},
		},
	}

	// Remove a server
	removed := config.RemoveServer("test2")
	if !removed {
		t.Error("expected RemoveServer to return true, got false")
	}

	if len(config.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(config.Servers))
	}

	if _, ok := config.Servers["test2"]; ok {
		t.Error("expected 'test2' server to be removed, but it's still there")
	}

	// Test removing the default server
	removed = config.RemoveServer("test1")
	if !removed {
		t.Error("expected RemoveServer to return true, got false")
	}

	if len(config.Servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(config.Servers))
	}

	// Test removing nonexistent server
	removed = config.RemoveServer("nonexistent")
	if removed {
		t.Error("expected RemoveServer to return false, got true")
	}
}

func TestSetDefaultServer(t *testing.T) {
	config := &Config{
		DefaultServer: "test1",
		Servers: map[string]Server{
			"test1": {
				Endpoint: "http://localhost:8080",
			},
			"test2": {
				Endpoint: "http://localhost:9090",
			},
		},
	}

	// Set a valid default server
	err := config.SetDefaultServer("test2")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if config.DefaultServer != "test2" {
		t.Errorf("expected default server to be 'test2', got %q", config.DefaultServer)
	}

	// Set an invalid default server
	err = config.SetDefaultServer("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent server, got nil")
	}
}
