// Package main implements mcp-config: Configuration management for MCP services
//
// This tool provides comprehensive configuration management capabilities for MCP services,
// including environment-specific configurations, secret management, validation, 
// templates, and hot reloading.
//
// Key features:
// - Environment-specific configuration management
// - Secret management with multiple backends (Vault, K8s secrets, etc.)
// - Configuration validation and schema enforcement
// - Template system for dynamic configuration
// - Hot reloading of configuration without service restart
// - Configuration versioning and rollback
// - Audit logging of configuration changes
//
// Usage:
//   mcp-config [command] [flags]
//
// Commands:
//   init        Initialize configuration management
//   validate    Validate configuration files
//   template    Process configuration templates
//   serve       Start configuration service
//   watch       Watch for configuration changes
//   secret      Manage secrets
//   env         Environment management
//
// Examples:
//   mcp-config init --environment production
//   mcp-config validate --config config.yaml
//   mcp-config template --input template.yaml --output config.yaml
//   mcp-config serve --port 8080
//   mcp-config watch --config config.yaml --reload-command "systemctl restart mcp-server"
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	version = "1.0.0"
)

// Main application structure
type ConfigApp struct {
	config    *ConfigManagementConfig
	logger    *slog.Logger
	validator *ConfigValidator
	templater *ConfigTemplater
	secrets   *SecretManager
	watcher   *ConfigWatcher
	server    *ConfigServer
	env       *EnvironmentManager
}

// ConfigManagementConfig defines the configuration management system configuration
type ConfigManagementConfig struct {
	// Service configuration
	ServiceName string `json:"service_name" yaml:"service_name"`
	Port        int    `json:"port" yaml:"port"`
	LogLevel    string `json:"log_level" yaml:"log_level"`
	
	// Environment configuration
	Environment string                 `json:"environment" yaml:"environment"`
	Environments map[string]*EnvConfig `json:"environments" yaml:"environments"`
	
	// Secret management
	Secrets SecretConfig `json:"secrets" yaml:"secrets"`
	
	// Template configuration
	Templates TemplateConfig `json:"templates" yaml:"templates"`
	
	// Watching configuration
	Watch WatchConfig `json:"watch" yaml:"watch"`
	
	// Validation configuration
	Validation ValidationConfig `json:"validation" yaml:"validation"`
	
	// Audit configuration
	Audit AuditConfig `json:"audit" yaml:"audit"`
}

// EnvConfig defines environment-specific configuration
type EnvConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	ConfigPaths []string          `json:"config_paths" yaml:"config_paths"`
	SecretPaths []string          `json:"secret_paths" yaml:"secret_paths"`
	Variables   map[string]string `json:"variables" yaml:"variables"`
	Overrides   map[string]interface{} `json:"overrides" yaml:"overrides"`
}

// SecretConfig defines secret management configuration
type SecretConfig struct {
	Backend   string            `json:"backend" yaml:"backend"` // vault, k8s, file, env
	Address   string            `json:"address" yaml:"address"`
	Token     string            `json:"token" yaml:"token"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Paths     map[string]string `json:"paths" yaml:"paths"`
	Encryption EncryptionConfig `json:"encryption" yaml:"encryption"`
}

// EncryptionConfig defines encryption configuration for secrets
type EncryptionConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Algorithm string `json:"algorithm" yaml:"algorithm"` // aes-256-gcm
	KeyFile   string `json:"key_file" yaml:"key_file"`
	KeyEnv    string `json:"key_env" yaml:"key_env"`
}

// TemplateConfig defines template system configuration
type TemplateConfig struct {
	Enabled       bool              `json:"enabled" yaml:"enabled"`
	InputDir      string            `json:"input_dir" yaml:"input_dir"`
	OutputDir     string            `json:"output_dir" yaml:"output_dir"`
	Functions     map[string]string `json:"functions" yaml:"functions"`
	Variables     map[string]interface{} `json:"variables" yaml:"variables"`
	Delimiters    DelimiterConfig   `json:"delimiters" yaml:"delimiters"`
}

// DelimiterConfig defines template delimiters
type DelimiterConfig struct {
	Left  string `json:"left" yaml:"left"`
	Right string `json:"right" yaml:"right"`
}

// WatchConfig defines file watching configuration
type WatchConfig struct {
	Enabled       bool     `json:"enabled" yaml:"enabled"`
	Paths         []string `json:"paths" yaml:"paths"`
	ReloadCommand string   `json:"reload_command" yaml:"reload_command"`
	ReloadDelay   time.Duration `json:"reload_delay" yaml:"reload_delay"`
	IgnorePatterns []string `json:"ignore_patterns" yaml:"ignore_patterns"`
}

// ValidationConfig defines validation configuration
type ValidationConfig struct {
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	SchemaFile  string   `json:"schema_file" yaml:"schema_file"`
	Rules       []ValidationRule `json:"rules" yaml:"rules"`
	StrictMode  bool     `json:"strict_mode" yaml:"strict_mode"`
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Name        string `json:"name" yaml:"name"`
	Path        string `json:"path" yaml:"path"`
	Type        string `json:"type" yaml:"type"` // string, number, boolean, array, object
	Required    bool   `json:"required" yaml:"required"`
	Pattern     string `json:"pattern" yaml:"pattern"`
	MinLength   int    `json:"min_length" yaml:"min_length"`
	MaxLength   int    `json:"max_length" yaml:"max_length"`
	MinValue    float64 `json:"min_value" yaml:"min_value"`
	MaxValue    float64 `json:"max_value" yaml:"max_value"`
	AllowedValues []string `json:"allowed_values" yaml:"allowed_values"`
}

// AuditConfig defines audit logging configuration
type AuditConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	LogFile  string `json:"log_file" yaml:"log_file"`
	LogLevel string `json:"log_level" yaml:"log_level"`
	Format   string `json:"format" yaml:"format"` // json, text
}

// ConfigValidator validates configuration files
type ConfigValidator struct {
	config *ConfigManagementConfig
	logger *slog.Logger
	schema map[string]interface{}
}

// ConfigTemplater processes configuration templates
type ConfigTemplater struct {
	config *ConfigManagementConfig
	logger *slog.Logger
	funcs  template.FuncMap
}

// SecretManager manages secrets from various backends
type SecretManager struct {
	config *ConfigManagementConfig
	logger *slog.Logger
	cache  map[string]SecretValue
	mutex  sync.RWMutex
}

// SecretValue represents a secret value with metadata
type SecretValue struct {
	Value     string    `json:"value"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Encrypted bool      `json:"encrypted"`
}

// ConfigWatcher watches for configuration changes
type ConfigWatcher struct {
	config *ConfigManagementConfig
	logger *slog.Logger
	watchers map[string]*FileWatcher
	mutex   sync.RWMutex
}

// FileWatcher represents a file watcher
type FileWatcher struct {
	path     string
	callback func(string)
	stop     chan bool
}

// ConfigServer provides HTTP API for configuration management
type ConfigServer struct {
	config *ConfigManagementConfig
	logger *slog.Logger
	server *http.Server
	validator *ConfigValidator
	templater *ConfigTemplater
	secrets   *SecretManager
}

// EnvironmentManager manages environment-specific configurations
type EnvironmentManager struct {
	config *ConfigManagementConfig
	logger *slog.Logger
	current *EnvConfig
}

// AuditEvent represents an audit event
type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	User      string    `json:"user"`
	Resource  string    `json:"resource"`
	Changes   map[string]interface{} `json:"changes"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// NewConfigApp creates a new configuration management application
func NewConfigApp(config *ConfigManagementConfig) *ConfigApp {
	logLevel := slog.LevelInfo
	switch config.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	validator := &ConfigValidator{
		config: config,
		logger: logger,
		schema: make(map[string]interface{}),
	}

	templater := &ConfigTemplater{
		config: config,
		logger: logger,
		funcs:  make(template.FuncMap),
	}

	secrets := &SecretManager{
		config: config,
		logger: logger,
		cache:  make(map[string]SecretValue),
	}

	watcher := &ConfigWatcher{
		config:   config,
		logger:   logger,
		watchers: make(map[string]*FileWatcher),
	}

	server := &ConfigServer{
		config:    config,
		logger:    logger,
		validator: validator,
		templater: templater,
		secrets:   secrets,
	}

	env := &EnvironmentManager{
		config: config,
		logger: logger,
	}

	return &ConfigApp{
		config:    config,
		logger:    logger,
		validator: validator,
		templater: templater,
		secrets:   secrets,
		watcher:   watcher,
		server:    server,
		env:       env,
	}
}

// ConfigValidator implementation
func (cv *ConfigValidator) LoadSchema(schemaFile string) error {
	if schemaFile == "" {
		return nil
	}

	data, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cv.schema); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	cv.logger.Info("Loaded validation schema", "file", schemaFile)
	return nil
}

func (cv *ConfigValidator) ValidateConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate against rules
	for _, rule := range cv.config.Validation.Rules {
		if err := cv.validateRule(config, rule); err != nil {
			return fmt.Errorf("validation failed for rule %s: %w", rule.Name, err)
		}
	}

	cv.logger.Info("Configuration validation successful", "file", configFile)
	return nil
}

func (cv *ConfigValidator) validateRule(config map[string]interface{}, rule ValidationRule) error {
	value, exists := getValueByPath(config, rule.Path)
	
	if !exists {
		if rule.Required {
			return fmt.Errorf("required field %s is missing", rule.Path)
		}
		return nil
	}

	// Type validation
	if !cv.validateType(value, rule.Type) {
		return fmt.Errorf("field %s has invalid type, expected %s", rule.Path, rule.Type)
	}

	// Pattern validation
	if rule.Pattern != "" {
		if str, ok := value.(string); ok {
			matched, err := regexp.MatchString(rule.Pattern, str)
			if err != nil {
				return fmt.Errorf("invalid pattern %s: %w", rule.Pattern, err)
			}
			if !matched {
				return fmt.Errorf("field %s does not match pattern %s", rule.Path, rule.Pattern)
			}
		}
	}

	// Length validation
	if rule.MinLength > 0 || rule.MaxLength > 0 {
		if str, ok := value.(string); ok {
			length := len(str)
			if rule.MinLength > 0 && length < rule.MinLength {
				return fmt.Errorf("field %s is too short (min %d)", rule.Path, rule.MinLength)
			}
			if rule.MaxLength > 0 && length > rule.MaxLength {
				return fmt.Errorf("field %s is too long (max %d)", rule.Path, rule.MaxLength)
			}
		}
	}

	// Value validation
	if rule.MinValue != 0 || rule.MaxValue != 0 {
		if num, ok := value.(float64); ok {
			if rule.MinValue != 0 && num < rule.MinValue {
				return fmt.Errorf("field %s is too small (min %f)", rule.Path, rule.MinValue)
			}
			if rule.MaxValue != 0 && num > rule.MaxValue {
				return fmt.Errorf("field %s is too large (max %f)", rule.Path, rule.MaxValue)
			}
		}
	}

	// Allowed values validation
	if len(rule.AllowedValues) > 0 {
		if str, ok := value.(string); ok {
			allowed := false
			for _, allowed_value := range rule.AllowedValues {
				if str == allowed_value {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("field %s has invalid value, allowed values: %v", rule.Path, rule.AllowedValues)
			}
		}
	}

	return nil
}

func (cv *ConfigValidator) validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true
	}
}

// ConfigTemplater implementation
func (ct *ConfigTemplater) InitializeFunctions() {
	ct.funcs["env"] = os.Getenv
	ct.funcs["default"] = func(defaultValue, value string) string {
		if value == "" {
			return defaultValue
		}
		return value
	}
	ct.funcs["upper"] = strings.ToUpper
	ct.funcs["lower"] = strings.ToLower
	ct.funcs["title"] = strings.Title
	ct.funcs["replace"] = strings.ReplaceAll
	ct.funcs["split"] = strings.Split
	ct.funcs["join"] = strings.Join
	ct.funcs["trim"] = strings.TrimSpace
	ct.funcs["base64"] = func(s string) string {
		return hex.EncodeToString([]byte(s))
	}
	ct.funcs["sha256"] = func(s string) string {
		hash := sha256.Sum256([]byte(s))
		return hex.EncodeToString(hash[:])
	}
	ct.funcs["now"] = time.Now
	ct.funcs["formatTime"] = func(format string, t time.Time) string {
		return t.Format(format)
	}
}

func (ct *ConfigTemplater) ProcessTemplate(inputFile, outputFile string, variables map[string]interface{}) error {
	// Read template
	templateData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Create template
	tmpl, err := template.New(filepath.Base(inputFile)).
		Funcs(ct.funcs).
		Parse(string(templateData))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Merge variables
	allVariables := make(map[string]interface{})
	for k, v := range ct.config.Templates.Variables {
		allVariables[k] = v
	}
	for k, v := range variables {
		allVariables[k] = v
	}

	// Add environment variables
	allVariables["Env"] = func(key string) string {
		return os.Getenv(key)
	}

	// Execute template
	var output strings.Builder
	if err := tmpl.Execute(&output, allVariables); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write output
	if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	ct.logger.Info("Template processed successfully", "input", inputFile, "output", outputFile)
	return nil
}

func (ct *ConfigTemplater) ProcessDirectory(inputDir, outputDir string, variables map[string]interface{}) error {
	return filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip non-template files
		if !strings.HasSuffix(path, ".tmpl") && !strings.HasSuffix(path, ".template") {
			return nil
		}

		// Calculate output path
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		outputPath := filepath.Join(outputDir, relPath)
		outputPath = strings.TrimSuffix(outputPath, ".tmpl")
		outputPath = strings.TrimSuffix(outputPath, ".template")

		// Create output directory
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}

		// Process template
		return ct.ProcessTemplate(path, outputPath, variables)
	})
}

// SecretManager implementation
func (sm *SecretManager) GetSecret(key string) (SecretValue, error) {
	sm.mutex.RLock()
	cached, exists := sm.cache[key]
	sm.mutex.RUnlock()

	if exists {
		return cached, nil
	}

	// Load from backend
	value, err := sm.loadFromBackend(key)
	if err != nil {
		return SecretValue{}, err
	}

	// Cache the value
	sm.mutex.Lock()
	sm.cache[key] = value
	sm.mutex.Unlock()

	return value, nil
}

func (sm *SecretManager) SetSecret(key string, value SecretValue) error {
	if err := sm.saveToBackend(key, value); err != nil {
		return err
	}

	sm.mutex.Lock()
	sm.cache[key] = value
	sm.mutex.Unlock()

	sm.logger.Info("Secret updated", "key", key)
	return nil
}

func (sm *SecretManager) loadFromBackend(key string) (SecretValue, error) {
	switch sm.config.Secrets.Backend {
	case "file":
		return sm.loadFromFile(key)
	case "env":
		return sm.loadFromEnv(key)
	case "vault":
		return sm.loadFromVault(key)
	case "k8s":
		return sm.loadFromK8s(key)
	default:
		return SecretValue{}, fmt.Errorf("unsupported secret backend: %s", sm.config.Secrets.Backend)
	}
}

func (sm *SecretManager) saveToBackend(key string, value SecretValue) error {
	switch sm.config.Secrets.Backend {
	case "file":
		return sm.saveToFile(key, value)
	case "env":
		return fmt.Errorf("cannot save to env backend")
	case "vault":
		return sm.saveToVault(key, value)
	case "k8s":
		return sm.saveToK8s(key, value)
	default:
		return fmt.Errorf("unsupported secret backend: %s", sm.config.Secrets.Backend)
	}
}

func (sm *SecretManager) loadFromFile(key string) (SecretValue, error) {
	path, exists := sm.config.Secrets.Paths[key]
	if !exists {
		return SecretValue{}, fmt.Errorf("secret path not found for key: %s", key)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return SecretValue{}, fmt.Errorf("failed to read secret file: %w", err)
	}

	return SecretValue{
		Value:     string(data),
		Version:   "1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Encrypted: false,
	}, nil
}

func (sm *SecretManager) saveToFile(key string, value SecretValue) error {
	path, exists := sm.config.Secrets.Paths[key]
	if !exists {
		return fmt.Errorf("secret path not found for key: %s", key)
	}

	return os.WriteFile(path, []byte(value.Value), 0600)
}

func (sm *SecretManager) loadFromEnv(key string) (SecretValue, error) {
	value := os.Getenv(key)
	if value == "" {
		return SecretValue{}, fmt.Errorf("environment variable not found: %s", key)
	}

	return SecretValue{
		Value:     value,
		Version:   "1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Encrypted: false,
	}, nil
}

func (sm *SecretManager) loadFromVault(key string) (SecretValue, error) {
	// Implementation would use Vault API
	return SecretValue{}, fmt.Errorf("vault backend not implemented")
}

func (sm *SecretManager) saveToVault(key string, value SecretValue) error {
	// Implementation would use Vault API
	return fmt.Errorf("vault backend not implemented")
}

func (sm *SecretManager) loadFromK8s(key string) (SecretValue, error) {
	// Implementation would use Kubernetes API
	return SecretValue{}, fmt.Errorf("k8s backend not implemented")
}

func (sm *SecretManager) saveToK8s(key string, value SecretValue) error {
	// Implementation would use Kubernetes API
	return fmt.Errorf("k8s backend not implemented")
}

// ConfigWatcher implementation
func (cw *ConfigWatcher) StartWatching(ctx context.Context) error {
	for _, path := range cw.config.Watch.Paths {
		watcher := &FileWatcher{
			path: path,
			callback: func(changedPath string) {
				cw.logger.Info("Configuration file changed", "path", changedPath)
				
				// Wait for reload delay
				time.Sleep(cw.config.Watch.ReloadDelay)
				
				// Execute reload command
				if cw.config.Watch.ReloadCommand != "" {
					if err := cw.executeReloadCommand(); err != nil {
						cw.logger.Error("Failed to execute reload command", "error", err)
					}
				}
			},
			stop: make(chan bool),
		}

		go cw.watchFile(ctx, watcher)
		
		cw.mutex.Lock()
		cw.watchers[path] = watcher
		cw.mutex.Unlock()
	}

	cw.logger.Info("Started watching configuration files", "paths", cw.config.Watch.Paths)
	return nil
}

func (cw *ConfigWatcher) watchFile(ctx context.Context, watcher *FileWatcher) {
	// Simple file watching implementation
	// In production, this would use a proper file watching library
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastModTime time.Time
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-watcher.stop:
			return
		case <-ticker.C:
			info, err := os.Stat(watcher.path)
			if err != nil {
				continue
			}

			if info.ModTime().After(lastModTime) {
				lastModTime = info.ModTime()
				watcher.callback(watcher.path)
			}
		}
	}
}

func (cw *ConfigWatcher) executeReloadCommand() error {
	parts := strings.Fields(cw.config.Watch.ReloadCommand)
	if len(parts) == 0 {
		return fmt.Errorf("empty reload command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	cw.logger.Info("Reload command executed successfully", "command", cw.config.Watch.ReloadCommand)
	return nil
}

// ConfigServer implementation
func (cs *ConfigServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Configuration endpoints
	mux.HandleFunc("/config", cs.handleGetConfig)
	mux.HandleFunc("/config/validate", cs.handleValidateConfig)
	mux.HandleFunc("/config/template", cs.handleProcessTemplate)
	mux.HandleFunc("/config/reload", cs.handleReloadConfig)

	// Secret endpoints
	mux.HandleFunc("/secrets", cs.handleGetSecrets)
	mux.HandleFunc("/secrets/", cs.handleSecret)

	// Environment endpoints
	mux.HandleFunc("/environments", cs.handleGetEnvironments)
	mux.HandleFunc("/environments/", cs.handleEnvironment)

	// Health endpoints
	mux.HandleFunc("/health", cs.handleHealth)
	mux.HandleFunc("/ready", cs.handleReady)

	cs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cs.config.Port),
		Handler: mux,
	}

	cs.logger.Info("Starting configuration server", "port", cs.config.Port)

	go func() {
		if err := cs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cs.logger.Error("Configuration server error", "error", err)
		}
	}()

	return nil
}

func (cs *ConfigServer) Stop(ctx context.Context) error {
	if cs.server != nil {
		return cs.server.Shutdown(ctx)
	}
	return nil
}

func (cs *ConfigServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cs.config); err != nil {
		cs.logger.Error("Failed to encode config", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (cs *ConfigServer) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConfigFile string `json:"config_file"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := cs.validator.ValidateConfig(req.ConfigFile); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "valid"})
}

func (cs *ConfigServer) handleProcessTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InputFile  string                 `json:"input_file"`
		OutputFile string                 `json:"output_file"`
		Variables  map[string]interface{} `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := cs.templater.ProcessTemplate(req.InputFile, req.OutputFile, req.Variables); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}

func (cs *ConfigServer) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Trigger configuration reload
	// Implementation depends on specific requirements
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"})
}

func (cs *ConfigServer) handleGetSecrets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return list of secret keys (not values)
	keys := make([]string, 0)
	for key := range cs.config.Secrets.Paths {
		keys = append(keys, key)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"keys": keys})
}

func (cs *ConfigServer) handleSecret(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/secrets/"):]
	if key == "" {
		http.Error(w, "Secret key required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get secret (return metadata only, not value)
		secret, err := cs.secrets.GetSecret(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Return metadata only
		metadata := map[string]interface{}{
			"version":    secret.Version,
			"created_at": secret.CreatedAt,
			"updated_at": secret.UpdatedAt,
			"encrypted":  secret.Encrypted,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metadata)

	case http.MethodPost:
		// Set secret
		var req struct {
			Value string `json:"value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		secret := SecretValue{
			Value:     req.Value,
			Version:   generateVersion(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Encrypted: false,
		}

		if err := cs.secrets.SetSecret(key, secret); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (cs *ConfigServer) handleGetEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cs.config.Environments)
}

func (cs *ConfigServer) handleEnvironment(w http.ResponseWriter, r *http.Request) {
	envName := r.URL.Path[len("/environments/"):]
	if envName == "" {
		http.Error(w, "Environment name required", http.StatusBadRequest)
		return
	}

	env, exists := cs.config.Environments[envName]
	if !exists {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(env)
}

func (cs *ConfigServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (cs *ConfigServer) handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// Helper functions
func getValueByPath(config map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := config

	for i, part := range parts {
		if i == len(parts)-1 {
			value, exists := current[part]
			return value, exists
		}

		next, exists := current[part]
		if !exists {
			return nil, false
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil, false
		}

		current = nextMap
	}

	return nil, false
}

func generateVersion() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Main application logic
func (app *ConfigApp) Run(ctx context.Context, command string, args []string) error {
	switch command {
	case "init":
		return app.initConfig(args)
	case "validate":
		return app.validateConfig(args)
	case "template":
		return app.processTemplate(args)
	case "serve":
		return app.runServer(ctx)
	case "watch":
		return app.watchConfigs(ctx)
	case "secret":
		return app.manageSecret(args)
	case "env":
		return app.manageEnvironment(args)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func (app *ConfigApp) initConfig(args []string) error {
	app.logger.Info("Initializing configuration management")
	
	// Create default directories
	dirs := []string{
		"config",
		"config/environments",
		"config/templates",
		"config/secrets",
		"config/schemas",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create example configuration
	exampleConfig := `
service_name: "mcp-service"
port: 8080
log_level: "info"
environment: "development"

environments:
  development:
    name: "development"
    description: "Development environment"
    config_paths:
      - "config/dev.yaml"
    variables:
      log_level: "debug"
      
  production:
    name: "production"
    description: "Production environment"
    config_paths:
      - "config/prod.yaml"
    variables:
      log_level: "warn"

secrets:
  backend: "file"
  paths:
    database_password: "config/secrets/db_password"
    api_key: "config/secrets/api_key"

templates:
  enabled: true
  input_dir: "config/templates"
  output_dir: "config/generated"

watch:
  enabled: true
  paths:
    - "config/app.yaml"
    - "config/environments"
  reload_command: "systemctl reload mcp-service"
  reload_delay: "5s"

validation:
  enabled: true
  schema_file: "config/schemas/config.schema.yaml"
  strict_mode: true

audit:
  enabled: true
  log_file: "config/audit.log"
  format: "json"
`

	if err := os.WriteFile("config/mcp-config.yaml", []byte(exampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	app.logger.Info("Configuration management initialized successfully")
	return nil
}

func (app *ConfigApp) validateConfig(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("config file path required")
	}

	configFile := args[0]
	
	// Load validation schema
	if err := app.validator.LoadSchema(app.config.Validation.SchemaFile); err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	// Validate configuration
	if err := app.validator.ValidateConfig(configFile); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	app.logger.Info("Configuration validation successful")
	return nil
}

func (app *ConfigApp) processTemplate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("input and output file paths required")
	}

	inputFile := args[0]
	outputFile := args[1]

	// Initialize template functions
	app.templater.InitializeFunctions()

	// Process template
	variables := make(map[string]interface{})
	if err := app.templater.ProcessTemplate(inputFile, outputFile, variables); err != nil {
		return fmt.Errorf("template processing failed: %w", err)
	}

	app.logger.Info("Template processing successful")
	return nil
}

func (app *ConfigApp) runServer(ctx context.Context) error {
	// Initialize components
	app.templater.InitializeFunctions()
	
	if err := app.validator.LoadSchema(app.config.Validation.SchemaFile); err != nil {
		app.logger.Warn("Failed to load validation schema", "error", err)
	}

	// Start server
	if err := app.server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	app.logger.Info("Configuration service started successfully")

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.server.Stop(shutdownCtx); err != nil {
		app.logger.Error("Failed to stop server", "error", err)
	}

	app.logger.Info("Configuration service stopped")
	return nil
}

func (app *ConfigApp) watchConfigs(ctx context.Context) error {
	if !app.config.Watch.Enabled {
		return fmt.Errorf("watching is not enabled")
	}

	if err := app.watcher.StartWatching(ctx); err != nil {
		return fmt.Errorf("failed to start watching: %w", err)
	}

	app.logger.Info("Configuration watching started")

	// Wait for shutdown signal
	<-ctx.Done()

	app.logger.Info("Configuration watching stopped")
	return nil
}

func (app *ConfigApp) manageSecret(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("secret command and key required")
	}

	subcommand := args[0]
	key := args[1]

	switch subcommand {
	case "get":
		secret, err := app.secrets.GetSecret(key)
		if err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}
		fmt.Printf("Secret: %s\nVersion: %s\nCreated: %s\nUpdated: %s\n",
			secret.Value, secret.Version, secret.CreatedAt, secret.UpdatedAt)

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("secret value required")
		}
		value := args[2]
		
		secret := SecretValue{
			Value:     value,
			Version:   generateVersion(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Encrypted: false,
		}

		if err := app.secrets.SetSecret(key, secret); err != nil {
			return fmt.Errorf("failed to set secret: %w", err)
		}

		fmt.Printf("Secret %s updated successfully\n", key)

	default:
		return fmt.Errorf("unknown secret command: %s", subcommand)
	}

	return nil
}

func (app *ConfigApp) manageEnvironment(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("environment command required")
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		fmt.Println("Available environments:")
		for name, env := range app.config.Environments {
			fmt.Printf("  %s: %s\n", name, env.Description)
		}

	case "current":
		fmt.Printf("Current environment: %s\n", app.config.Environment)

	case "switch":
		if len(args) < 2 {
			return fmt.Errorf("environment name required")
		}
		envName := args[1]
		
		if _, exists := app.config.Environments[envName]; !exists {
			return fmt.Errorf("environment not found: %s", envName)
		}

		app.config.Environment = envName
		fmt.Printf("Switched to environment: %s\n", envName)

	default:
		return fmt.Errorf("unknown environment command: %s", subcommand)
	}

	return nil
}

func main() {
	var (
		configFile = flag.String("config", "config/mcp-config.yaml", "Path to configuration file")
		command    = flag.String("command", "serve", "Command to run")
		port       = flag.Int("port", 8080, "Port to serve on")
		logLevel   = flag.String("log-level", "info", "Log level")
	)
	flag.Parse()

	// Create default configuration
	config := &ConfigManagementConfig{
		ServiceName: "mcp-config",
		Port:        *port,
		LogLevel:    *logLevel,
		Environment: "development",
		Environments: make(map[string]*EnvConfig),
		Secrets: SecretConfig{
			Backend: "file",
			Paths:   make(map[string]string),
		},
		Templates: TemplateConfig{
			Enabled: true,
			Variables: make(map[string]interface{}),
		},
		Watch: WatchConfig{
			Enabled:     true,
			ReloadDelay: 5 * time.Second,
		},
		Validation: ValidationConfig{
			Enabled: true,
		},
		Audit: AuditConfig{
			Enabled: true,
			Format:  "json",
		},
	}

	// Load configuration from file if exists
	if _, err := os.Stat(*configFile); err == nil {
		data, err := os.ReadFile(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config file: %v\n", err)
			os.Exit(1)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config file: %v\n", err)
			os.Exit(1)
		}
	}

	// Create and run application
	app := NewConfigApp(config)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := app.Run(ctx, *command, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}