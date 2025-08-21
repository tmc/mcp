// Package transport provides enhanced transport abstraction layer v2 with plugin support
// for the MCP command toolkit. It follows the Russ Cox coding style and provides
// a unified interface for all transport types with extensible plugin architecture.
package transport

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"
)

// Transport represents the core transport interface for MCP communication.
// This interface abstracts the underlying transport mechanism and provides
// a unified way to establish connections for MCP protocol communication.
type Transport interface {
	// Dial establishes a connection to the MCP server
	Dial(ctx context.Context) (io.ReadWriteCloser, error)

	// Name returns the transport name
	Name() string

	// Type returns the transport type (stdio, http, websocket, etc.)
	Type() string

	// Config returns transport-specific configuration
	Config() Config

	// Health checks the transport health
	Health(ctx context.Context) error

	// Close closes the transport and releases resources
	Close() error
}

// Config represents transport configuration.
type Config struct {
	// Transport type (stdio, http, websocket, tcp, etc.)
	Type string `json:"type" yaml:"type"`

	// Transport-specific parameters
	Parameters map[string]interface{} `json:"parameters" yaml:"parameters"`

	// Connection timeout
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// Maximum number of retries
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// Enable connection pooling
	EnablePooling bool `json:"enable_pooling" yaml:"enable_pooling"`

	// Pool configuration
	Pool PoolConfig `json:"pool" yaml:"pool"`

	// Security configuration
	Security SecurityConfig `json:"security" yaml:"security"`

	// Middleware configuration
	Middleware []MiddlewareConfig `json:"middleware" yaml:"middleware"`
}

// PoolConfig represents connection pool configuration.
type PoolConfig struct {
	// Maximum number of connections in the pool
	MaxSize int `json:"max_size" yaml:"max_size"`

	// Minimum number of connections to maintain
	MinSize int `json:"min_size" yaml:"min_size"`

	// Maximum time a connection can be idle
	MaxIdleTime time.Duration `json:"max_idle_time" yaml:"max_idle_time"`

	// Maximum lifetime of a connection
	MaxLifetime time.Duration `json:"max_lifetime" yaml:"max_lifetime"`

	// Enable connection validation
	EnableValidation bool `json:"enable_validation" yaml:"enable_validation"`
}

// SecurityConfig represents security configuration.
type SecurityConfig struct {
	// Enable TLS
	EnableTLS bool `json:"enable_tls" yaml:"enable_tls"`

	// TLS configuration
	TLS TLSConfig `json:"tls" yaml:"tls"`

	// Authentication configuration
	Auth AuthConfig `json:"auth" yaml:"auth"`
}

// TLSConfig represents TLS configuration.
type TLSConfig struct {
	// Certificate file path
	CertFile string `json:"cert_file" yaml:"cert_file"`

	// Private key file path
	KeyFile string `json:"key_file" yaml:"key_file"`

	// CA certificate file path
	CAFile string `json:"ca_file" yaml:"ca_file"`

	// Skip certificate verification
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	// Authentication type (bearer, basic, oauth2, etc.)
	Type string `json:"type" yaml:"type"`

	// Authentication credentials
	Credentials map[string]string `json:"credentials" yaml:"credentials"`
}

// MiddlewareConfig represents middleware configuration.
type MiddlewareConfig struct {
	// Middleware name
	Name string `json:"name" yaml:"name"`

	// Middleware enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Middleware parameters
	Parameters map[string]interface{} `json:"parameters" yaml:"parameters"`
}

// Factory creates transport instances.
type Factory interface {
	// Create creates a new transport instance
	Create(config Config) (Transport, error)

	// Type returns the transport type this factory creates
	Type() string

	// ValidateConfig validates transport configuration
	ValidateConfig(config Config) error
}

// Plugin represents a transport plugin.
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Factory returns the transport factory
	Factory() Factory

	// Load loads the plugin with the given configuration
	Load(config map[string]interface{}) error

	// Unload unloads the plugin
	Unload() error
}

// Manager manages transport instances and plugins.
type Manager struct {
	mu         sync.RWMutex
	factories  map[string]Factory
	plugins    map[string]Plugin
	transports map[string]Transport
	config     ManagerConfig
}

// ManagerConfig represents transport manager configuration.
type ManagerConfig struct {
	// Default transport type
	DefaultType string `json:"default_type" yaml:"default_type"`

	// Plugin directory
	PluginDir string `json:"plugin_dir" yaml:"plugin_dir"`

	// Enable plugin loading
	EnablePlugins bool `json:"enable_plugins" yaml:"enable_plugins"`

	// Transport configurations
	Transports map[string]Config `json:"transports" yaml:"transports"`
}

// NewManager creates a new transport manager.
func NewManager(config ManagerConfig) (*Manager, error) {
	m := &Manager{
		factories:  make(map[string]Factory),
		plugins:    make(map[string]Plugin),
		transports: make(map[string]Transport),
		config:     config,
	}

	// Register built-in transport factories
	if err := m.registerBuiltinFactories(); err != nil {
		return nil, fmt.Errorf("failed to register builtin factories: %w", err)
	}

	// Load plugins if enabled
	if config.EnablePlugins {
		if err := m.loadPlugins(); err != nil {
			return nil, fmt.Errorf("failed to load plugins: %w", err)
		}
	}

	return m, nil
}

// RegisterFactory registers a transport factory.
func (m *Manager) RegisterFactory(factory Factory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	transportType := factory.Type()
	if _, exists := m.factories[transportType]; exists {
		return fmt.Errorf("factory for transport type %s already registered", transportType)
	}

	m.factories[transportType] = factory
	return nil
}

// GetFactory returns a transport factory by type.
func (m *Manager) GetFactory(transportType string) (Factory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	factory, exists := m.factories[transportType]
	if !exists {
		return nil, fmt.Errorf("no factory registered for transport type: %s", transportType)
	}

	return factory, nil
}

// Create creates a new transport instance.
func (m *Manager) Create(config Config) (Transport, error) {
	factory, err := m.GetFactory(config.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get factory: %w", err)
	}

	// Validate configuration
	if err := factory.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create transport
	transport, err := factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	return transport, nil
}

// CreateFromURL creates a transport from a URL.
func (m *Manager) CreateFromURL(rawURL string) (Transport, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Map URL scheme to transport type
	transportType := m.mapSchemeToType(parsedURL.Scheme)

	// Create configuration from URL
	config := Config{
		Type:       transportType,
		Parameters: make(map[string]interface{}),
	}

	// Extract parameters from URL
	if err := m.extractURLParameters(parsedURL, &config); err != nil {
		return nil, fmt.Errorf("failed to extract URL parameters: %w", err)
	}

	return m.Create(config)
}

// GetTransport returns a cached transport instance.
func (m *Manager) GetTransport(name string) (Transport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	transport, exists := m.transports[name]
	if !exists {
		return nil, fmt.Errorf("transport %s not found", name)
	}

	return transport, nil
}

// LoadPlugin loads a transport plugin.
func (m *Manager) LoadPlugin(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if err := plugin.Load(config); err != nil {
		return fmt.Errorf("failed to load plugin %s: %w", name, err)
	}

	// Register plugin factory
	factory := plugin.Factory()
	m.factories[factory.Type()] = factory

	return nil
}

// UnloadPlugin unloads a transport plugin.
func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if err := plugin.Unload(); err != nil {
		return fmt.Errorf("failed to unload plugin %s: %w", name, err)
	}

	// Remove plugin factory
	factory := plugin.Factory()
	delete(m.factories, factory.Type())

	return nil
}

// ListFactories returns all registered transport factories.
func (m *Manager) ListFactories() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var types []string
	for transportType := range m.factories {
		types = append(types, transportType)
	}

	return types
}

// ListPlugins returns all loaded plugins.
func (m *Manager) ListPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name := range m.plugins {
		names = append(names, name)
	}

	return names
}

// Close closes all transports and unloads plugins.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all transports
	for name, transport := range m.transports {
		if err := transport.Close(); err != nil {
			return fmt.Errorf("failed to close transport %s: %w", name, err)
		}
	}

	// Unload all plugins
	for name, plugin := range m.plugins {
		if err := plugin.Unload(); err != nil {
			return fmt.Errorf("failed to unload plugin %s: %w", name, err)
		}
	}

	return nil
}

// registerBuiltinFactories registers built-in transport factories.
func (m *Manager) registerBuiltinFactories() error {
	builtins := []Factory{
		&StdioFactory{},
		&HTTPFactory{},
		&WebSocketFactory{},
		&TCPFactory{},
		&UnixFactory{},
	}

	for _, factory := range builtins {
		if err := m.RegisterFactory(factory); err != nil {
			return fmt.Errorf("failed to register builtin factory %s: %w", factory.Type(), err)
		}
	}

	return nil
}

// loadPlugins loads transport plugins from the plugin directory.
func (m *Manager) loadPlugins() error {
	if m.config.PluginDir == "" {
		return nil
	}

	// Plugin loading would be implemented here
	// For now, return nil as plugin loading is complex
	return nil
}

// mapSchemeToType maps URL scheme to transport type.
func (m *Manager) mapSchemeToType(scheme string) string {
	switch scheme {
	case "http", "https":
		return "http"
	case "ws", "wss":
		return "websocket"
	case "tcp", "tcp4", "tcp6":
		return "tcp"
	case "unix":
		return "unix"
	case "stdio":
		return "stdio"
	default:
		return m.config.DefaultType
	}
}

// extractURLParameters extracts parameters from URL and sets them in config.
func (m *Manager) extractURLParameters(parsedURL *url.URL, config *Config) error {
	switch config.Type {
	case "http":
		config.Parameters["url"] = parsedURL.String()
		config.Parameters["method"] = "POST"
		if parsedURL.Scheme == "https" {
			config.Security.EnableTLS = true
		}
	case "websocket":
		config.Parameters["url"] = parsedURL.String()
		if parsedURL.Scheme == "wss" {
			config.Security.EnableTLS = true
		}
	case "tcp":
		config.Parameters["address"] = parsedURL.Host
		config.Parameters["network"] = "tcp"
	case "unix":
		config.Parameters["path"] = parsedURL.Path
	case "stdio":
		if parsedURL.Host != "" {
			config.Parameters["command"] = parsedURL.Host
		}
		if parsedURL.Path != "" {
			config.Parameters["args"] = parsedURL.Path
		}
	}

	// Extract query parameters
	query := parsedURL.Query()
	for key, values := range query {
		if len(values) > 0 {
			config.Parameters[key] = values[0]
		}
	}

	return nil
}

// Middleware represents transport middleware.
type Middleware interface {
	// Name returns the middleware name
	Name() string

	// Apply applies the middleware to the transport
	Apply(transport Transport) Transport
}

// MiddlewareChain represents a chain of middleware.
type MiddlewareChain struct {
	middleware []Middleware
}

// NewMiddlewareChain creates a new middleware chain.
func NewMiddlewareChain(middleware ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{middleware: middleware}
}

// Apply applies all middleware in the chain to the transport.
func (c *MiddlewareChain) Apply(transport Transport) Transport {
	result := transport
	for _, middleware := range c.middleware {
		result = middleware.Apply(result)
	}
	return result
}

// Add adds middleware to the chain.
func (c *MiddlewareChain) Add(middleware Middleware) {
	c.middleware = append(c.middleware, middleware)
}

// Default manager instance
var defaultManager *Manager

// Default returns the default transport manager.
func Default() *Manager {
	if defaultManager == nil {
		var err error
		defaultManager, err = NewManager(ManagerConfig{
			DefaultType:   "stdio",
			EnablePlugins: true,
		})
		if err != nil {
			panic(fmt.Sprintf("failed to create default transport manager: %v", err))
		}
	}
	return defaultManager
}

// Create creates a transport using the default manager.
func Create(config Config) (Transport, error) {
	return Default().Create(config)
}

// CreateFromURL creates a transport from URL using the default manager.
func CreateFromURL(rawURL string) (Transport, error) {
	return Default().CreateFromURL(rawURL)
}
