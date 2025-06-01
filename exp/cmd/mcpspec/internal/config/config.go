package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the global configuration for the mcpspec tools.
type Config struct {
	DefaultServer  string            `json:"default_server"`
	Servers        map[string]Server `json:"servers"`
	LogLevel       string            `json:"log_level"`
	LogFile        string            `json:"log_file"`
	MaxMessageSize int               `json:"max_message_size"`
}

// Server represents the configuration for an MCP server.
type Server struct {
	Endpoint    string            `json:"endpoint"`
	Transport   string            `json:"transport"` // "http", "stdio", "websocket"
	Environment map[string]string `json:"environment"`
	Timeout     int               `json:"timeout"` // timeout in seconds
	TLSVerify   bool              `json:"tls_verify"`
	InitParams  json.RawMessage   `json:"init_params"`
}

// LoadConfig loads the configuration from the specified file or from the default locations.
func LoadConfig(configPath string) (*Config, error) {
	if configPath != "" {
		return loadConfigFromPath(configPath)
	}

	// Try to load from default locations
	locations := []string{
		".mcpspec.json",
		filepath.Join(homeDir(), ".mcpspec", "config.json"),
		"/etc/mcpspec/config.json",
	}

	var config *Config
	var err error
	for _, location := range locations {
		config, err = loadConfigFromPath(location)
		if err == nil {
			return config, nil
		}
	}

	// If no config found, return a default config
	return defaultConfig(), nil
}

// loadConfigFromPath loads the configuration from the specified file path.
func loadConfigFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to the specified file.
func SaveConfig(config *Config, path string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal the config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write the config to the file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	return nil
}

// defaultConfig returns a default configuration.
func defaultConfig() *Config {
	return &Config{
		DefaultServer: "local",
		Servers: map[string]Server{
			"local": {
				Endpoint:  "http://localhost:8080",
				Transport: "http",
				Timeout:   30,
				TLSVerify: true,
			},
		},
		LogLevel:       "info",
		MaxMessageSize: 10 * 1024 * 1024, // 10MB
	}
}

// homeDir returns the user's home directory.
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// GetDefaultServer returns the default server configuration.
func (c *Config) GetDefaultServer() (*Server, error) {
	if c.DefaultServer == "" {
		return nil, fmt.Errorf("no default server specified in config")
	}

	server, ok := c.Servers[c.DefaultServer]
	if !ok {
		return nil, fmt.Errorf("default server %q not found in config", c.DefaultServer)
	}

	return &server, nil
}

// GetServer returns the specified server configuration.
func (c *Config) GetServer(name string) (*Server, error) {
	server, ok := c.Servers[name]
	if !ok {
		return nil, fmt.Errorf("server %q not found in config", name)
	}

	return &server, nil
}

// AddServer adds or updates a server configuration.
func (c *Config) AddServer(name string, server Server) {
	if c.Servers == nil {
		c.Servers = make(map[string]Server)
	}
	c.Servers[name] = server
}

// RemoveServer removes a server configuration.
func (c *Config) RemoveServer(name string) bool {
	if _, ok := c.Servers[name]; !ok {
		return false
	}
	delete(c.Servers, name)

	// If the default server was removed, update it
	if c.DefaultServer == name && len(c.Servers) > 0 {
		for k := range c.Servers {
			c.DefaultServer = k
			break
		}
	}

	return true
}

// SetDefaultServer sets the default server.
func (c *Config) SetDefaultServer(name string) error {
	if _, ok := c.Servers[name]; !ok {
		return fmt.Errorf("server %q not found in config", name)
	}
	c.DefaultServer = name
	return nil
}
