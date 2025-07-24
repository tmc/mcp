package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestManagerCreation(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	if manager == nil {
		t.Fatal("Manager is nil")
	}
	
	// Test that defaults are applied
	config := manager.Get()
	if config.Global.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", config.Global.LogLevel)
	}
	
	if config.Global.OutputFormat != "json" {
		t.Errorf("Expected default output format 'json', got %s", config.Global.OutputFormat)
	}
}

func TestConfigLoading(t *testing.T) {
	// Create temporary config file
	configContent := `
global:
  log_level: debug
  output_format: yaml
  no_color: true
  transport:
    default_type: http
    timeout: 60s
  performance:
    max_concurrency: 20
    buffer_size: 16384
tools:
  test-tool:
    enabled: true
    config:
      setting1: value1
      setting2: 42
`
	
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Create manager with config file
	manager, err := NewManager(WithConfigFile(configPath))
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	// Load configuration
	ctx := context.Background()
	if err := manager.Load(ctx); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	config := manager.Get()
	
	// Test global config
	if config.Global.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got %s", config.Global.LogLevel)
	}
	
	if config.Global.OutputFormat != "yaml" {
		t.Errorf("Expected output format 'yaml', got %s", config.Global.OutputFormat)
	}
	
	if !config.Global.NoColor {
		t.Error("Expected no_color to be true")
	}
	
	if config.Global.Transport.DefaultType != "http" {
		t.Errorf("Expected transport type 'http', got %s", config.Global.Transport.DefaultType)
	}
	
	if config.Global.Transport.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.Global.Transport.Timeout)
	}
	
	if config.Global.Performance.MaxConcurrency != 20 {
		t.Errorf("Expected max concurrency 20, got %d", config.Global.Performance.MaxConcurrency)
	}
	
	if config.Global.Performance.BufferSize != 16384 {
		t.Errorf("Expected buffer size 16384, got %d", config.Global.Performance.BufferSize)
	}
	
	// Test tool config
	var toolConfig map[string]interface{}
	if err := manager.GetTool("test-tool", &toolConfig); err != nil {
		t.Fatalf("Failed to get tool config: %v", err)
	}
	
	if enabled, ok := toolConfig["enabled"].(bool); !ok || !enabled {
		t.Error("Expected tool to be enabled")
	}
	
	if config, ok := toolConfig["config"].(map[string]interface{}); ok {
		if setting1, ok := config["setting1"].(string); !ok || setting1 != "value1" {
			t.Errorf("Expected setting1 'value1', got %v", setting1)
		}
		
		if setting2, ok := config["setting2"].(float64); !ok || setting2 != 42 {
			t.Errorf("Expected setting2 42, got %v", setting2)
		}
	} else {
		t.Error("Expected tool config to be present")
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("MCP_LOG_LEVEL", "warn")
	os.Setenv("MCP_OUTPUT_FORMAT", "table")
	os.Setenv("MCP_TRANSPORT_TIMEOUT", "45s")
	os.Setenv("MCP_TOOL_MYTOOL_ENABLED", "true")
	os.Setenv("MCP_TOOL_MYTOOL_CONFIG_HOST", "localhost")
	os.Setenv("MCP_TOOL_MYTOOL_CONFIG_PORT", "8080")
	
	defer func() {
		os.Unsetenv("MCP_LOG_LEVEL")
		os.Unsetenv("MCP_OUTPUT_FORMAT")
		os.Unsetenv("MCP_TRANSPORT_TIMEOUT")
		os.Unsetenv("MCP_TOOL_MYTOOL_ENABLED")
		os.Unsetenv("MCP_TOOL_MYTOOL_CONFIG_HOST")
		os.Unsetenv("MCP_TOOL_MYTOOL_CONFIG_PORT")
	}()
	
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	ctx := context.Background()
	if err := manager.Load(ctx); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	config := manager.Get()
	
	// Test global config from env
	if config.Global.LogLevel != "warn" {
		t.Errorf("Expected log level 'warn', got %s", config.Global.LogLevel)
	}
	
	if config.Global.OutputFormat != "table" {
		t.Errorf("Expected output format 'table', got %s", config.Global.OutputFormat)
	}
	
	if config.Global.Transport.Timeout != 45*time.Second {
		t.Errorf("Expected timeout 45s, got %v", config.Global.Transport.Timeout)
	}
	
	// Test tool config from env
	var toolConfig map[string]interface{}
	if err := manager.GetTool("mytool", &toolConfig); err != nil {
		t.Fatalf("Failed to get tool config: %v", err)
	}
	
	if enabled, ok := toolConfig["enabled"].(bool); !ok || !enabled {
		t.Error("Expected tool to be enabled")
	}
	
	if config, ok := toolConfig["config"].(map[string]interface{}); ok {
		if host, ok := config["host"].(string); !ok || host != "localhost" {
			t.Errorf("Expected host 'localhost', got %v", host)
		}
		
		if port, ok := config["port"].(string); !ok || port != "8080" {
			t.Errorf("Expected port '8080', got %v", port)
		}
	} else {
		t.Error("Expected tool config to be present")
	}
}

func TestConfigValidation(t *testing.T) {
	// Test validator function
	validateGlobal := func(config interface{}) error {
		if cfg, ok := config.(*GlobalConfig); ok {
			if cfg.LogLevel != "info" && cfg.LogLevel != "debug" && cfg.LogLevel != "warn" && cfg.LogLevel != "error" {
				return errors.New("invalid log level")
			}
		}
		return nil
	}
	
	manager, err := NewManager(WithValidator("global", validateGlobal))
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	// Set invalid log level
	os.Setenv("MCP_LOG_LEVEL", "invalid")
	defer os.Unsetenv("MCP_LOG_LEVEL")
	
	ctx := context.Background()
	if err := manager.Load(ctx); err == nil {
		t.Error("Expected validation error for invalid log level")
	}
}

func TestConfigChange(t *testing.T) {
	changeCount := 0
	listener := func(old, new *Config) error {
		changeCount++
		return nil
	}
	
	manager, err := NewManager(WithChangeListener(listener))
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	ctx := context.Background()
	if err := manager.Load(ctx); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if changeCount != 1 {
		t.Errorf("Expected change count 1, got %d", changeCount)
	}
}

func TestFileSource(t *testing.T) {
	// Test YAML file
	yamlContent := `
global:
  log_level: debug
tools:
  test:
    enabled: true
`
	
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}
	
	source := &FileSource{Path: yamlPath}
	config, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}
	
	if config.Global.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got %s", config.Global.LogLevel)
	}
	
	// Test JSON file
	jsonContent := `{
		"global": {
			"log_level": "info",
			"output_format": "json"
		},
		"tools": {
			"test": {
				"enabled": false
			}
		}
	}`
	
	jsonPath := filepath.Join(tmpDir, "config.json")
	
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}
	
	source = &FileSource{Path: jsonPath}
	config, err = source.Load(context.Background())
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}
	
	if config.Global.LogLevel != "info" {
		t.Errorf("Expected log level 'info', got %s", config.Global.LogLevel)
	}
	
	if config.Global.OutputFormat != "json" {
		t.Errorf("Expected output format 'json', got %s", config.Global.OutputFormat)
	}
}

func TestEnvSource(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_LOG_LEVEL", "debug")
	os.Setenv("TEST_OUTPUT_FORMAT", "yaml")
	os.Setenv("TEST_TRANSPORT_TIMEOUT", "30s")
	
	defer func() {
		os.Unsetenv("TEST_LOG_LEVEL")
		os.Unsetenv("TEST_OUTPUT_FORMAT")
		os.Unsetenv("TEST_TRANSPORT_TIMEOUT")
	}()
	
	source := &EnvSource{Prefix: "TEST_"}
	config, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("Failed to load env config: %v", err)
	}
	
	if config.Global.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got %s", config.Global.LogLevel)
	}
	
	if config.Global.OutputFormat != "yaml" {
		t.Errorf("Expected output format 'yaml', got %s", config.Global.OutputFormat)
	}
	
	if config.Global.Transport.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Global.Transport.Timeout)
	}
}

func TestMultiSource(t *testing.T) {
	// Create file source
	configContent := `
global:
  log_level: debug
  output_format: yaml
`
	
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	fileSource := &FileSource{Path: configPath}
	
	// Set environment variables (higher priority)
	os.Setenv("TEST_LOG_LEVEL", "warn")
	defer os.Unsetenv("TEST_LOG_LEVEL")
	
	envSource := &EnvSource{Prefix: "TEST_"}
	
	// Create multi source
	multiSource := &MultiSource{
		Sources: []Source{envSource, fileSource}, // env has higher priority
	}
	
	config, err := multiSource.Load(context.Background())
	if err != nil {
		t.Fatalf("Failed to load multi config: %v", err)
	}
	
	// Environment should override file
	if config.Global.LogLevel != "warn" {
		t.Errorf("Expected log level 'warn' (from env), got %s", config.Global.LogLevel)
	}
	
	// File should provide fallback
	if config.Global.OutputFormat != "yaml" {
		t.Errorf("Expected output format 'yaml' (from file), got %s", config.Global.OutputFormat)
	}
}

func TestApplyDefaults(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	config := &Config{
		Global: GlobalConfig{},
		Tools:  make(map[string]json.RawMessage),
	}
	
	if err := manager.applyDefaults(config); err != nil {
		t.Fatalf("Failed to apply defaults: %v", err)
	}
	
	// Check defaults
	if config.Global.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", config.Global.LogLevel)
	}
	
	if config.Global.LogFormat != "json" {
		t.Errorf("Expected default log format 'json', got %s", config.Global.LogFormat)
	}
	
	if config.Global.OutputFormat != "json" {
		t.Errorf("Expected default output format 'json', got %s", config.Global.OutputFormat)
	}
	
	if config.Global.Transport.DefaultType != "stdio" {
		t.Errorf("Expected default transport type 'stdio', got %s", config.Global.Transport.DefaultType)
	}
	
	if config.Global.Performance.MaxConcurrency != 10 {
		t.Errorf("Expected default max concurrency 10, got %d", config.Global.Performance.MaxConcurrency)
	}
	
	if config.Global.Performance.BufferSize != 8192 {
		t.Errorf("Expected default buffer size 8192, got %d", config.Global.Performance.BufferSize)
	}
	
	if config.Global.Performance.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.Global.Performance.Timeout)
	}
}

func TestToolConfig(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	
	// Set tool config
	toolConfig := map[string]interface{}{
		"enabled": true,
		"host":    "localhost",
		"port":    8080,
		"timeout": "30s",
	}
	
	if err := manager.SetTool("test-tool", toolConfig); err != nil {
		t.Fatalf("Failed to set tool config: %v", err)
	}
	
	// Get tool config
	var retrievedConfig map[string]interface{}
	if err := manager.GetTool("test-tool", &retrievedConfig); err != nil {
		t.Fatalf("Failed to get tool config: %v", err)
	}
	
	if !reflect.DeepEqual(toolConfig, retrievedConfig) {
		t.Errorf("Expected config %+v, got %+v", toolConfig, retrievedConfig)
	}
	
	// Test non-existent tool
	if err := manager.GetTool("non-existent", &retrievedConfig); err == nil {
		t.Error("Expected error for non-existent tool")
	}
}

func TestStructDefaults(t *testing.T) {
	type TestStruct struct {
		StringField  string        `default:"test"`
		IntField     int           `default:"42"`
		BoolField    bool          `default:"true"`
		DurationField time.Duration `default:"30s"`
	}
	
	var s TestStruct
	v := reflect.ValueOf(&s).Elem()
	
	if err := applyStructDefaults(v); err != nil {
		t.Fatalf("Failed to apply struct defaults: %v", err)
	}
	
	if s.StringField != "test" {
		t.Errorf("Expected string field 'test', got %s", s.StringField)
	}
	
	if s.IntField != 42 {
		t.Errorf("Expected int field 42, got %d", s.IntField)
	}
	
	if !s.BoolField {
		t.Error("Expected bool field true")
	}
	
	if s.DurationField != 30*time.Second {
		t.Errorf("Expected duration field 30s, got %v", s.DurationField)
	}
}

func TestNumberParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"42", 42, false},
		{"-42", -42, false},
		{"123", 123, false},
		{"abc", 0, true},
		{"12a", 0, true},
		{"1.5", 0, true},
	}
	
	for _, test := range tests {
		result, err := parseNumber(test.input)
		
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", test.input, err)
			}
			
			if result != test.expected {
				t.Errorf("Expected %d for input %s, got %d", test.expected, test.input, result)
			}
		}
	}
}

func BenchmarkConfigLoad(b *testing.B) {
	manager, err := NewManager()
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := manager.Load(ctx); err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

func BenchmarkToolConfigGet(b *testing.B) {
	manager, err := NewManager()
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}
	
	// Set tool config
	toolConfig := map[string]interface{}{
		"enabled": true,
		"host":    "localhost",
		"port":    8080,
	}
	
	if err := manager.SetTool("test-tool", toolConfig); err != nil {
		b.Fatalf("Failed to set tool config: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var config map[string]interface{}
		if err := manager.GetTool("test-tool", &config); err != nil {
			b.Fatalf("Failed to get tool config: %v", err)
		}
	}
}