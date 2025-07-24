package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FileSource loads configuration from a file.
type FileSource struct {
	Path string
}

// Load loads configuration from a file.
func (s *FileSource) Load(ctx context.Context) (*Config, error) {
	if s.Path == "" {
		return nil, fmt.Errorf("file path is empty")
	}
	
	file, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, skip
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	
	// Determine format based on file extension
	ext := strings.ToLower(filepath.Ext(s.Path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &config); err != nil {
			if err := json.Unmarshal(data, &config); err != nil {
				return nil, fmt.Errorf("failed to parse config file (tried YAML and JSON): %w", err)
			}
		}
	}
	
	// Set runtime config
	config.Runtime.ConfigPath = s.Path
	if wd, err := os.Getwd(); err == nil {
		config.Runtime.WorkingDir = wd
	}
	
	return &config, nil
}

// Name returns the source name.
func (s *FileSource) Name() string {
	return fmt.Sprintf("file:%s", s.Path)
}

// Priority returns the source priority (lower = higher priority).
func (s *FileSource) Priority() int {
	return 100
}

// EnvSource loads configuration from environment variables.
type EnvSource struct {
	Prefix string
}

// Load loads configuration from environment variables.
func (s *EnvSource) Load(ctx context.Context) (*Config, error) {
	if s.Prefix == "" {
		s.Prefix = "MCP_"
	}
	
	config := &Config{
		Global: GlobalConfig{},
		Tools:  make(map[string]json.RawMessage),
	}
	
	// Get all environment variables with our prefix
	vars := s.getEnvVars()
	if len(vars) == 0 {
		return nil, nil // No environment variables found
	}
	
	// Parse environment variables
	for key, value := range vars {
		if err := s.parseEnvVar(config, key, value); err != nil {
			return nil, fmt.Errorf("failed to parse env var %s: %w", key, err)
		}
	}
	
	return config, nil
}

// getEnvVars returns all environment variables with the configured prefix.
func (s *EnvSource) getEnvVars() map[string]string {
	vars := make(map[string]string)
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := parts[0]
		value := parts[1]
		
		if strings.HasPrefix(key, s.Prefix) {
			// Remove prefix and convert to lowercase
			configKey := strings.ToLower(key[len(s.Prefix):])
			vars[configKey] = value
		}
	}
	
	return vars
}

// parseEnvVar parses a single environment variable and applies it to config.
func (s *EnvSource) parseEnvVar(config *Config, key, value string) error {
	parts := strings.Split(key, "_")
	if len(parts) == 0 {
		return nil
	}
	
	// Handle global configuration
	if parts[0] == "global" || parts[0] == "log" || parts[0] == "output" || parts[0] == "transport" {
		return s.parseGlobalEnvVar(config, parts, value)
	}
	
	// Handle tool-specific configuration
	if len(parts) >= 2 && parts[0] == "tool" {
		return s.parseToolEnvVar(config, parts[1:], value)
	}
	
	// Handle runtime configuration
	if parts[0] == "runtime" || parts[0] == "working" || parts[0] == "config" {
		return s.parseRuntimeEnvVar(config, parts, value)
	}
	
	return nil
}

// parseGlobalEnvVar parses global environment variables.
func (s *EnvSource) parseGlobalEnvVar(config *Config, parts []string, value string) error {
	switch {
	case matchesPath(parts, "log", "level"):
		config.Global.LogLevel = value
	case matchesPath(parts, "log", "format"):
		config.Global.LogFormat = value
	case matchesPath(parts, "output", "format"):
		config.Global.OutputFormat = value
	case matchesPath(parts, "output", "no", "color"):
		config.Global.NoColor = parseBool(value)
	case matchesPath(parts, "transport", "default", "type"):
		config.Global.Transport.DefaultType = value
	case matchesPath(parts, "transport", "timeout"):
		if duration, err := time.ParseDuration(value); err == nil {
			config.Global.Transport.Timeout = duration
		}
	case matchesPath(parts, "performance", "max", "concurrency"):
		if val, err := strconv.Atoi(value); err == nil {
			config.Global.Performance.MaxConcurrency = val
		}
	case matchesPath(parts, "performance", "buffer", "size"):
		if val, err := strconv.Atoi(value); err == nil {
			config.Global.Performance.BufferSize = val
		}
	case matchesPath(parts, "performance", "timeout"):
		if duration, err := time.ParseDuration(value); err == nil {
			config.Global.Performance.Timeout = duration
		}
	case matchesPath(parts, "performance", "enable", "metrics"):
		config.Global.Performance.EnableMetrics = parseBool(value)
	case matchesPath(parts, "security", "enable", "auth"):
		config.Global.Security.EnableAuth = parseBool(value)
	case matchesPath(parts, "security", "require", "https"):
		config.Global.Security.RequireHTTPS = parseBool(value)
	case matchesPath(parts, "security", "enable", "rate", "limit"):
		config.Global.Security.EnableRateLimit = parseBool(value)
	case matchesPath(parts, "security", "allowed", "hosts"):
		config.Global.Security.AllowedHosts = strings.Split(value, ",")
	}
	
	return nil
}

// parseToolEnvVar parses tool-specific environment variables.
func (s *EnvSource) parseToolEnvVar(config *Config, parts []string, value string) error {
	if len(parts) < 2 {
		return nil
	}
	
	toolName := parts[0]
	configPath := strings.Join(parts[1:], ".")
	
	// Get existing tool config or create new
	var toolConfig map[string]interface{}
	if existing, exists := config.Tools[toolName]; exists {
		if err := json.Unmarshal(existing, &toolConfig); err != nil {
			return fmt.Errorf("failed to unmarshal existing tool config: %w", err)
		}
	} else {
		toolConfig = make(map[string]interface{})
	}
	
	// Set nested value
	setNestedValue(toolConfig, configPath, value)
	
	// Marshal back to JSON
	data, err := json.Marshal(toolConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal tool config: %w", err)
	}
	
	config.Tools[toolName] = data
	return nil
}

// parseRuntimeEnvVar parses runtime environment variables.
func (s *EnvSource) parseRuntimeEnvVar(config *Config, parts []string, value string) error {
	switch {
	case matchesPath(parts, "config", "path"):
		config.Runtime.ConfigPath = value
	case matchesPath(parts, "working", "dir"):
		config.Runtime.WorkingDir = value
	case matchesPath(parts, "reload", "enabled"):
		config.Runtime.ReloadEnabled = parseBool(value)
	}
	
	return nil
}

// Name returns the source name.
func (s *EnvSource) Name() string {
	return fmt.Sprintf("env:%s", s.Prefix)
}

// Priority returns the source priority (lower = higher priority).
func (s *EnvSource) Priority() int {
	return 10 // Environment variables have high priority
}

// MultiSource combines multiple configuration sources.
type MultiSource struct {
	Sources []Source
}

// Load loads configuration from multiple sources.
func (s *MultiSource) Load(ctx context.Context) (*Config, error) {
	if len(s.Sources) == 0 {
		return nil, nil
	}
	
	// Sort sources by priority
	sources := make([]Source, len(s.Sources))
	copy(sources, s.Sources)
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Priority() < sources[j].Priority()
	})
	
	// Load from first source
	config, err := sources[0].Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load from primary source: %w", err)
	}
	
	if config == nil {
		config = &Config{
			Global: GlobalConfig{},
			Tools:  make(map[string]json.RawMessage),
		}
	}
	
	// Merge remaining sources
	for i := 1; i < len(sources); i++ {
		sourceConfig, err := sources[i].Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load from source %s: %w", sources[i].Name(), err)
		}
		
		if sourceConfig != nil {
			if err := mergeConfigs(config, sourceConfig); err != nil {
				return nil, fmt.Errorf("failed to merge source %s: %w", sources[i].Name(), err)
			}
		}
	}
	
	return config, nil
}

// Name returns the source name.
func (s *MultiSource) Name() string {
	names := make([]string, len(s.Sources))
	for i, source := range s.Sources {
		names[i] = source.Name()
	}
	return fmt.Sprintf("multi:[%s]", strings.Join(names, ","))
}

// Priority returns the source priority.
func (s *MultiSource) Priority() int {
	if len(s.Sources) == 0 {
		return 1000
	}
	return s.Sources[0].Priority()
}

// Helper functions

// matchesPath checks if parts match the given path.
func matchesPath(parts []string, path ...string) bool {
	if len(parts) != len(path) {
		return false
	}
	
	for i, part := range parts {
		if part != path[i] {
			return false
		}
	}
	
	return true
}

// parseBool parses a boolean value from string.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// setNestedValue sets a nested value in a map using dot notation.
func setNestedValue(m map[string]interface{}, path, value string) {
	parts := strings.Split(path, ".")
	current := m
	
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, set the value
			current[part] = value
		} else {
			// Intermediate part, create nested map if needed
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			if nested, ok := current[part].(map[string]interface{}); ok {
				current = nested
			}
		}
	}
}

// mergeConfigs merges two configurations.
func mergeConfigs(target, source *Config) error {
	// Merge global config
	if err := mergeStruct(&target.Global, &source.Global); err != nil {
		return fmt.Errorf("failed to merge global config: %w", err)
	}
	
	// Merge tools
	for toolName, toolConfig := range source.Tools {
		target.Tools[toolName] = toolConfig
	}
	
	// Merge runtime config
	if err := mergeStruct(&target.Runtime, &source.Runtime); err != nil {
		return fmt.Errorf("failed to merge runtime config: %w", err)
	}
	
	return nil
}