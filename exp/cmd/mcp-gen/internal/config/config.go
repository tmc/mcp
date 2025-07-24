// Package config provides configuration management for mcp-gen
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Config represents the configuration for mcp-gen
type Config struct {
	// Language settings
	Language string `json:"language"`
	
	// Output settings
	Output  string `json:"output"`
	Package string `json:"package"`
	
	// Generation settings
	TypeSafe       bool     `json:"type_safe"`
	Middleware     bool     `json:"middleware"`
	Documentation  bool     `json:"documentation"`
	Tests          bool     `json:"tests"`
	Examples       bool     `json:"examples"`
	Dependencies   []string `json:"dependencies"`
	
	// Language-specific settings
	Go         GoConfig         `json:"go"`
	TypeScript TypeScriptConfig `json:"typescript"`
	Python     PythonConfig     `json:"python"`
	Rust       RustConfig       `json:"rust"`
	Java       JavaConfig       `json:"java"`
	
	// Template settings
	Templates TemplateConfig `json:"templates"`
	
	// Plugin settings
	Plugins []PluginConfig `json:"plugins"`
	
	// Runtime settings (not serialized)
	Verbose bool `json:"-"`
	DryRun  bool `json:"-"`
}

// GoConfig contains Go-specific configuration
type GoConfig struct {
	ModulePath      string            `json:"module_path"`
	GoVersion       string            `json:"go_version"`
	ImportPaths     map[string]string `json:"import_paths"`
	GenerateGoMod   bool              `json:"generate_go_mod"`
	UseGenerics     bool              `json:"use_generics"`
	UseContexts     bool              `json:"use_contexts"`
	UseMiddleware   bool              `json:"use_middleware"`
	BuildTags       []string          `json:"build_tags"`
	LintingConfig   LintingConfig     `json:"linting"`
}

// TypeScriptConfig contains TypeScript-specific configuration
type TypeScriptConfig struct {
	ModuleName      string            `json:"module_name"`
	TypeScriptVersion string          `json:"typescript_version"`
	OutputFormat    string            `json:"output_format"` // "esm", "cjs", "umd"
	Declaration     bool              `json:"declaration"`
	SourceMap       bool              `json:"source_map"`
	StrictMode      bool              `json:"strict_mode"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"dev_dependencies"`
}

// PythonConfig contains Python-specific configuration
type PythonConfig struct {
	PackageName     string            `json:"package_name"`
	PythonVersion   string            `json:"python_version"`
	UseDataclasses  bool              `json:"use_dataclasses"`
	UsePydantic     bool              `json:"use_pydantic"`
	UseAsyncio      bool              `json:"use_asyncio"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"dev_dependencies"`
}

// RustConfig contains Rust-specific configuration
type RustConfig struct {
	CrateName     string            `json:"crate_name"`
	RustVersion   string            `json:"rust_version"`
	Edition       string            `json:"edition"`
	UseSerde      bool              `json:"use_serde"`
	UseTokio      bool              `json:"use_tokio"`
	UseClap       bool              `json:"use_clap"`
	Dependencies  map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"dev_dependencies"`
}

// JavaConfig contains Java-specific configuration
type JavaConfig struct {
	PackageName   string            `json:"package_name"`
	JavaVersion   string            `json:"java_version"`
	UseLombok     bool              `json:"use_lombok"`
	UseJackson    bool              `json:"use_jackson"`
	UseSpring     bool              `json:"use_spring"`
	Dependencies  map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"dev_dependencies"`
}

// TemplateConfig contains template-specific configuration
type TemplateConfig struct {
	Directory  string            `json:"directory"`
	CustomVars map[string]string `json:"custom_vars"`
}

// PluginConfig contains plugin-specific configuration
type PluginConfig struct {
	Name    string                 `json:"name"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// LintingConfig contains linting configuration
type LintingConfig struct {
	Enabled bool     `json:"enabled"`
	Rules   []string `json:"rules"`
}

// Default returns a configuration with sensible defaults
func Default() *Config {
	return &Config{
		Language:      "go",
		Output:        ".",
		TypeSafe:      true,
		Middleware:    true,
		Documentation: true,
		Tests:         true,
		Examples:      false,
		
		Go: GoConfig{
			GoVersion:     "1.21",
			GenerateGoMod: true,
			UseGenerics:   true,
			UseContexts:   true,
			UseMiddleware: true,
			LintingConfig: LintingConfig{
				Enabled: true,
				Rules:   []string{"go vet", "golint", "ineffassign"},
			},
		},
		
		TypeScript: TypeScriptConfig{
			TypeScriptVersion: "5.0",
			OutputFormat:      "esm",
			Declaration:       true,
			SourceMap:         true,
			StrictMode:        true,
		},
		
		Python: PythonConfig{
			PythonVersion:  "3.9",
			UseDataclasses: true,
			UsePydantic:    true,
			UseAsyncio:     true,
		},
		
		Rust: RustConfig{
			RustVersion: "1.70",
			Edition:     "2021",
			UseSerde:    true,
			UseTokio:    true,
			UseClap:     true,
		},
		
		Java: JavaConfig{
			JavaVersion: "17",
			UseLombok:   true,
			UseJackson:  true,
			UseSpring:   false,
		},
		
		Templates: TemplateConfig{
			Directory:  "",
			CustomVars: make(map[string]string),
		},
		
		Plugins: []PluginConfig{},
	}
}

// Load loads configuration from file or returns default if file doesn't exist
func Load(configPath string) (*Config, error) {
	cfg := Default()
	
	if configPath == "" {
		// Try to find config file in current directory
		candidates := []string{
			"mcp-gen.json",
			".mcp-gen.json",
			"mcp-gen.config.json",
		}
		
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				configPath = candidate
				break
			}
		}
	}
	
	if configPath == "" {
		return cfg, nil
	}
	
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return cfg, nil
}

// Save saves configuration to file
func (c *Config) Save(configPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	supportedLanguages := []string{"go", "typescript", "python", "rust", "java"}
	
	found := false
	for _, lang := range supportedLanguages {
		if c.Language == lang {
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("unsupported language: %s (supported: %v)", c.Language, supportedLanguages)
	}
	
	if c.Output == "" {
		return fmt.Errorf("output directory is required")
	}
	
	// Language-specific validation
	switch c.Language {
	case "go":
		if c.Go.ModulePath == "" && c.Package == "" {
			return fmt.Errorf("Go module path or package name is required")
		}
	case "typescript":
		if c.TypeScript.ModuleName == "" && c.Package == "" {
			return fmt.Errorf("TypeScript module name or package name is required")
		}
	case "python":
		if c.Python.PackageName == "" && c.Package == "" {
			return fmt.Errorf("Python package name is required")
		}
	case "rust":
		if c.Rust.CrateName == "" && c.Package == "" {
			return fmt.Errorf("Rust crate name or package name is required")
		}
	case "java":
		if c.Java.PackageName == "" && c.Package == "" {
			return fmt.Errorf("Java package name is required")
		}
	}
	
	return nil
}

// GetPackageName returns the appropriate package name for the target language
func (c *Config) GetPackageName() string {
	if c.Package != "" {
		return c.Package
	}
	
	switch c.Language {
	case "go":
		return c.Go.ModulePath
	case "typescript":
		return c.TypeScript.ModuleName
	case "python":
		return c.Python.PackageName
	case "rust":
		return c.Rust.CrateName
	case "java":
		return c.Java.PackageName
	}
	
	return ""
}