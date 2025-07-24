// Package config provides a unified configuration management system for all MCP tools.
// It follows the Russ Cox coding style and provides thread-safe configuration loading,
// validation, and management with support for multiple configuration sources.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the unified configuration structure for all MCP tools.
// It uses a hierarchical structure with tool-specific sections.
type Config struct {
	// Global configuration that applies to all tools
	Global GlobalConfig `json:"global" yaml:"global"`
	
	// Tool-specific configurations
	Tools map[string]json.RawMessage `json:"tools" yaml:"tools"`
	
	// Runtime configuration
	Runtime RuntimeConfig `json:"runtime" yaml:"runtime"`
	
	// Loaded configuration metadata
	meta ConfigMetadata `json:"-" yaml:"-"`
}

// GlobalConfig contains settings that apply across all MCP tools.
type GlobalConfig struct {
	// Logging configuration
	LogLevel  string `json:"log_level" yaml:"log_level" default:"info"`
	LogFormat string `json:"log_format" yaml:"log_format" default:"json"`
	
	// Output configuration
	OutputFormat string `json:"output_format" yaml:"output_format" default:"json"`
	NoColor      bool   `json:"no_color" yaml:"no_color" default:"false"`
	
	// Transport configuration
	Transport TransportConfig `json:"transport" yaml:"transport"`
	
	// Performance configuration
	Performance PerformanceConfig `json:"performance" yaml:"performance"`
	
	// Security configuration
	Security SecurityConfig `json:"security" yaml:"security"`
}

// TransportConfig contains default transport settings.
type TransportConfig struct {
	DefaultType string        `json:"default_type" yaml:"default_type" default:"stdio"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout" default:"30s"`
	
	// Plugin configuration
	Plugins []PluginConfig `json:"plugins" yaml:"plugins"`
}

// PluginConfig defines configuration for transport plugins.
type PluginConfig struct {
	Name    string            `json:"name" yaml:"name"`
	Type    string            `json:"type" yaml:"type"`
	Config  map[string]any    `json:"config" yaml:"config"`
	Enabled bool              `json:"enabled" yaml:"enabled" default:"true"`
}

// PerformanceConfig contains performance-related settings.
type PerformanceConfig struct {
	MaxConcurrency int           `json:"max_concurrency" yaml:"max_concurrency" default:"10"`
	BufferSize     int           `json:"buffer_size" yaml:"buffer_size" default:"8192"`
	Timeout        time.Duration `json:"timeout" yaml:"timeout" default:"30s"`
	EnableMetrics  bool          `json:"enable_metrics" yaml:"enable_metrics" default:"false"`
}

// SecurityConfig contains security-related settings.
type SecurityConfig struct {
	EnableAuth    bool     `json:"enable_auth" yaml:"enable_auth" default:"false"`
	AllowedHosts  []string `json:"allowed_hosts" yaml:"allowed_hosts"`
	RequireHTTPS  bool     `json:"require_https" yaml:"require_https" default:"true"`
	EnableRateLimit bool   `json:"enable_rate_limit" yaml:"enable_rate_limit" default:"true"`
}

// RuntimeConfig contains runtime-specific settings.
type RuntimeConfig struct {
	ConfigPath    string    `json:"config_path" yaml:"config_path"`
	WorkingDir    string    `json:"working_dir" yaml:"working_dir"`
	LoadTime      time.Time `json:"load_time" yaml:"load_time"`
	ReloadEnabled bool      `json:"reload_enabled" yaml:"reload_enabled" default:"false"`
}

// ConfigMetadata contains metadata about the loaded configuration.
type ConfigMetadata struct {
	Sources   []string          `json:"sources"`
	Overrides map[string]string `json:"overrides"`
	Errors    []string          `json:"errors"`
	Version   string            `json:"version"`
}

// Manager handles configuration loading, validation, and management.
type Manager struct {
	mu     sync.RWMutex
	config *Config
	
	// Configuration sources in order of precedence
	sources []Source
	
	// Validation functions
	validators map[string]ValidatorFunc
	
	// Change listeners
	listeners []ChangeListener
	
	// Environment variable prefix
	envPrefix string
}

// Source represents a configuration source.
type Source interface {
	Load(ctx context.Context) (*Config, error)
	Name() string
	Priority() int
}

// ValidatorFunc validates a configuration section.
type ValidatorFunc func(interface{}) error

// ChangeListener is called when configuration changes.
type ChangeListener func(old, new *Config) error

// NewManager creates a new configuration manager.
func NewManager(opts ...Option) (*Manager, error) {
	m := &Manager{
		config:     &Config{},
		sources:    []Source{},
		validators: make(map[string]ValidatorFunc),
		listeners:  []ChangeListener{},
		envPrefix:  "MCP_",
	}
	
	// Apply options
	for _, opt := range opts {
		if err := opt(m); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}
	
	// Add default sources if none configured
	if len(m.sources) == 0 {
		m.addDefaultSources()
	}
	
	return m, nil
}

// Option configures a Manager.
type Option func(*Manager) error

// WithConfigFile adds a file-based configuration source.
func WithConfigFile(path string) Option {
	return func(m *Manager) error {
		m.sources = append(m.sources, &FileSource{Path: path})
		return nil
	}
}

// WithEnvPrefix sets the environment variable prefix.
func WithEnvPrefix(prefix string) Option {
	return func(m *Manager) error {
		m.envPrefix = prefix
		return nil
	}
}

// WithValidator adds a configuration validator.
func WithValidator(section string, validator ValidatorFunc) Option {
	return func(m *Manager) error {
		m.validators[section] = validator
		return nil
	}
}

// WithChangeListener adds a configuration change listener.
func WithChangeListener(listener ChangeListener) Option {
	return func(m *Manager) error {
		m.listeners = append(m.listeners, listener)
		return nil
	}
}

// Load loads configuration from all sources.
func (m *Manager) Load(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	oldConfig := m.config
	
	// Create new config with defaults
	newConfig := &Config{
		Global: GlobalConfig{},
		Tools:  make(map[string]json.RawMessage),
		Runtime: RuntimeConfig{
			LoadTime: time.Now(),
		},
	}
	
	// Apply defaults
	if err := m.applyDefaults(newConfig); err != nil {
		return fmt.Errorf("failed to apply defaults: %w", err)
	}
	
	// Load from sources in order of precedence
	for _, source := range m.sources {
		sourceConfig, err := source.Load(ctx)
		if err != nil {
			// Log error but continue with other sources
			newConfig.meta.Errors = append(newConfig.meta.Errors, 
				fmt.Sprintf("source %s: %v", source.Name(), err))
			continue
		}
		
		if sourceConfig != nil {
			if err := m.mergeConfig(newConfig, sourceConfig); err != nil {
				return fmt.Errorf("failed to merge config from %s: %w", source.Name(), err)
			}
			newConfig.meta.Sources = append(newConfig.meta.Sources, source.Name())
		}
	}
	
	// Validate configuration
	if err := m.validate(newConfig); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	m.config = newConfig
	
	// Notify listeners
	for _, listener := range m.listeners {
		if err := listener(oldConfig, newConfig); err != nil {
			return fmt.Errorf("change listener failed: %w", err)
		}
	}
	
	return nil
}

// Get returns the current configuration.
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetTool returns tool-specific configuration.
func (m *Manager) GetTool(toolName string, target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	raw, exists := m.config.Tools[toolName]
	if !exists {
		return fmt.Errorf("no configuration found for tool: %s", toolName)
	}
	
	return json.Unmarshal(raw, target)
}

// SetTool sets tool-specific configuration.
func (m *Manager) SetTool(toolName string, config interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal tool config: %w", err)
	}
	
	if m.config.Tools == nil {
		m.config.Tools = make(map[string]json.RawMessage)
	}
	
	m.config.Tools[toolName] = raw
	return nil
}

// addDefaultSources adds default configuration sources.
func (m *Manager) addDefaultSources() {
	// Environment variables (highest priority)
	m.sources = append(m.sources, &EnvSource{Prefix: m.envPrefix})
	
	// User config file
	if home, err := os.UserHomeDir(); err == nil {
		userConfig := filepath.Join(home, ".mcp", "config.yaml")
		m.sources = append(m.sources, &FileSource{Path: userConfig})
	}
	
	// System config file
	m.sources = append(m.sources, &FileSource{Path: "/etc/mcp/config.yaml"})
	
	// Working directory config
	m.sources = append(m.sources, &FileSource{Path: "mcp.yaml"})
}

// applyDefaults applies default values to configuration.
func (m *Manager) applyDefaults(config *Config) error {
	return applyStructDefaults(reflect.ValueOf(config).Elem())
}

// applyStructDefaults applies default values using reflection.
func applyStructDefaults(v reflect.Value) error {
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// Skip unexported fields
		if !field.CanSet() {
			continue
		}
		
		// Get default value from tag
		defaultValue := fieldType.Tag.Get("default")
		if defaultValue == "" {
			// Recursively apply defaults to nested structs
			if field.Kind() == reflect.Struct {
				if err := applyStructDefaults(field); err != nil {
					return err
				}
			}
			continue
		}
		
		// Apply default value
		if err := setFieldDefault(field, defaultValue); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}
	}
	
	return nil
}

// setFieldDefault sets a default value for a field.
func setFieldDefault(field reflect.Value, defaultValue string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Bool:
		val := strings.ToLower(defaultValue) == "true"
		field.SetBool(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// Parse duration
			duration, err := time.ParseDuration(defaultValue)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			field.SetInt(int64(duration))
		} else {
			// Parse integer
			val, err := parseInt(defaultValue)
			if err != nil {
				return fmt.Errorf("invalid integer: %w", err)
			}
			field.SetInt(val)
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	
	return nil
}

// mergeConfig merges source configuration into target.
func (m *Manager) mergeConfig(target, source *Config) error {
	// Merge global config
	if err := mergeStruct(&target.Global, &source.Global); err != nil {
		return fmt.Errorf("failed to merge global config: %w", err)
	}
	
	// Merge tool configs
	for toolName, toolConfig := range source.Tools {
		target.Tools[toolName] = toolConfig
	}
	
	// Merge runtime config
	if err := mergeStruct(&target.Runtime, &source.Runtime); err != nil {
		return fmt.Errorf("failed to merge runtime config: %w", err)
	}
	
	return nil
}

// validate validates the configuration.
func (m *Manager) validate(config *Config) error {
	// Validate global config
	if validator, exists := m.validators["global"]; exists {
		if err := validator(&config.Global); err != nil {
			return fmt.Errorf("global config validation failed: %w", err)
		}
	}
	
	// Validate tool configs
	for toolName, toolConfig := range config.Tools {
		if validator, exists := m.validators[toolName]; exists {
			var parsed interface{}
			if err := json.Unmarshal(toolConfig, &parsed); err != nil {
				return fmt.Errorf("failed to parse tool config for %s: %w", toolName, err)
			}
			if err := validator(parsed); err != nil {
				return fmt.Errorf("tool config validation failed for %s: %w", toolName, err)
			}
		}
	}
	
	return nil
}

// Helper functions
func parseInt(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return parseNumber(s)
}

func parseNumber(s string) (int64, error) {
	// Simple integer parsing
	var result int64
	negative := false
	
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}
	
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		result = result*10 + int64(c-'0')
	}
	
	if negative {
		result = -result
	}
	
	return result, nil
}

func mergeStruct(target, source interface{}) error {
	targetVal := reflect.ValueOf(target).Elem()
	sourceVal := reflect.ValueOf(source).Elem()
	
	for i := 0; i < sourceVal.NumField(); i++ {
		sourceField := sourceVal.Field(i)
		targetField := targetVal.Field(i)
		
		if !sourceField.IsZero() && targetField.CanSet() {
			targetField.Set(sourceField)
		}
	}
	
	return nil
}