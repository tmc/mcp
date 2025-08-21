// Package plugin provides a plugin architecture for extensible tool functionality
// in the MCP command toolkit. It follows the Russ Cox coding style and provides
// a flexible plugin system with lifecycle management, dependency resolution,
// and safe loading/unloading of plugins.
package plugin

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tmc/mcp/internal/foundation/config"
	"github.com/tmc/mcp/internal/foundation/errors"
)

// Plugin represents a plugin interface that all plugins must implement.
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Description returns the plugin description
	Description() string

	// Dependencies returns the plugin dependencies
	Dependencies() []Dependency

	// Initialize initializes the plugin with the given context and configuration
	Initialize(ctx context.Context, config Config) error

	// Start starts the plugin
	Start(ctx context.Context) error

	// Stop stops the plugin
	Stop(ctx context.Context) error

	// Cleanup cleans up plugin resources
	Cleanup() error

	// Health returns the plugin health status
	Health() HealthStatus
}

// Dependency represents a plugin dependency.
type Dependency struct {
	// Plugin name
	Name string `json:"name"`

	// Version constraint
	Version string `json:"version"`

	// Whether the dependency is optional
	Optional bool `json:"optional"`
}

// Config represents plugin configuration.
type Config struct {
	// Plugin name
	Name string `json:"name"`

	// Plugin enabled
	Enabled bool `json:"enabled"`

	// Plugin configuration
	Config map[string]interface{} `json:"config"`

	// Plugin settings
	Settings map[string]interface{} `json:"settings"`

	// Plugin timeout
	Timeout time.Duration `json:"timeout"`

	// Plugin priority
	Priority int `json:"priority"`
}

// HealthStatus represents plugin health status.
type HealthStatus struct {
	// Plugin name
	Name string `json:"name"`

	// Health status
	Status Status `json:"status"`

	// Health message
	Message string `json:"message"`

	// Last check time
	LastCheck time.Time `json:"last_check"`

	// Health details
	Details map[string]interface{} `json:"details,omitempty"`
}

// Status represents plugin status.
type Status string

const (
	StatusUnknown     Status = "unknown"
	StatusLoading     Status = "loading"
	StatusInitialized Status = "initialized"
	StatusStarting    Status = "starting"
	StatusRunning     Status = "running"
	StatusStopping    Status = "stopping"
	StatusStopped     Status = "stopped"
	StatusError       Status = "error"
	StatusUnloaded    Status = "unloaded"
)

// Registry manages plugin registration and lifecycle.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]*PluginInfo
	hooks   map[string][]Hook
	config  RegistryConfig
}

// PluginInfo contains information about a registered plugin.
type PluginInfo struct {
	// Plugin instance
	Plugin Plugin `json:"-"`

	// Plugin metadata
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	LoadTime    time.Time `json:"load_time"`

	// Plugin configuration
	Config Config `json:"config"`

	// Plugin status
	Status Status `json:"status"`

	// Plugin dependencies
	Dependencies []Dependency `json:"dependencies"`

	// Plugin health
	Health HealthStatus `json:"health"`

	// Plugin context
	Context context.Context    `json:"-"`
	Cancel  context.CancelFunc `json:"-"`
}

// RegistryConfig represents registry configuration.
type RegistryConfig struct {
	// Plugin directory
	PluginDir string `json:"plugin_dir"`

	// Enable hot reload
	HotReload bool `json:"hot_reload"`

	// Plugin timeout
	Timeout time.Duration `json:"timeout"`

	// Maximum concurrent plugins
	MaxConcurrent int `json:"max_concurrent"`

	// Health check interval
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// Hook represents a plugin hook.
type Hook interface {
	// Name returns the hook name
	Name() string

	// Execute executes the hook
	Execute(ctx context.Context, data interface{}) error
}

// HookFunc is a function that implements Hook.
type HookFunc func(ctx context.Context, data interface{}) error

// Name returns the hook name.
func (f HookFunc) Name() string {
	return "anonymous"
}

// Execute executes the hook.
func (f HookFunc) Execute(ctx context.Context, data interface{}) error {
	return f(ctx, data)
}

// NewRegistry creates a new plugin registry.
func NewRegistry(config RegistryConfig) (*Registry, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 10
	}

	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}

	r := &Registry{
		plugins: make(map[string]*PluginInfo),
		hooks:   make(map[string][]Hook),
		config:  config,
	}

	return r, nil
}

// Register registers a plugin with the registry.
func (r *Registry) Register(plugin Plugin, config Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if name == "" {
		return errors.InvalidArgument("plugin name cannot be empty")
	}

	// Check if plugin already registered
	if _, exists := r.plugins[name]; exists {
		return errors.AlreadyExists(fmt.Sprintf("plugin %s already registered", name))
	}

	// Create plugin context
	ctx, cancel := context.WithCancel(context.Background())

	// Create plugin info
	info := &PluginInfo{
		Plugin:       plugin,
		Name:         name,
		Version:      plugin.Version(),
		Description:  plugin.Description(),
		LoadTime:     time.Now(),
		Config:       config,
		Status:       StatusLoading,
		Dependencies: plugin.Dependencies(),
		Health: HealthStatus{
			Name:      name,
			Status:    StatusUnknown,
			Message:   "Plugin not initialized",
			LastCheck: time.Now(),
		},
		Context: ctx,
		Cancel:  cancel,
	}

	// Validate dependencies
	if err := r.validateDependencies(info); err != nil {
		cancel()
		return errors.Wrapf(err, errors.CodeValidation, "dependency validation failed for plugin %s", name)
	}

	// Register plugin
	r.plugins[name] = info

	// Execute pre-register hooks
	if err := r.executeHooks(ctx, "pre-register", info); err != nil {
		delete(r.plugins, name)
		cancel()
		return errors.Wrapf(err, errors.CodePlugin, "pre-register hook failed for plugin %s", name)
	}

	// Execute post-register hooks
	if err := r.executeHooks(ctx, "post-register", info); err != nil {
		delete(r.plugins, name)
		cancel()
		return errors.Wrapf(err, errors.CodePlugin, "post-register hook failed for plugin %s", name)
	}

	return nil
}

// Unregister unregisters a plugin from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.plugins[name]
	if !exists {
		return errors.NotFound(fmt.Sprintf("plugin %s not found", name))
	}

	// Stop plugin if running
	if info.Status == StatusRunning {
		if err := r.stopPlugin(info); err != nil {
			return errors.Wrapf(err, errors.CodePlugin, "failed to stop plugin %s", name)
		}
	}

	// Execute pre-unregister hooks
	if err := r.executeHooks(info.Context, "pre-unregister", info); err != nil {
		return errors.Wrapf(err, errors.CodePlugin, "pre-unregister hook failed for plugin %s", name)
	}

	// Cleanup plugin
	if err := info.Plugin.Cleanup(); err != nil {
		return errors.Wrapf(err, errors.CodePlugin, "cleanup failed for plugin %s", name)
	}

	// Cancel context
	info.Cancel()

	// Remove from registry
	delete(r.plugins, name)

	// Execute post-unregister hooks
	if err := r.executeHooks(context.Background(), "post-unregister", info); err != nil {
		return errors.Wrapf(err, errors.CodePlugin, "post-unregister hook failed for plugin %s", name)
	}

	return nil
}

// Get returns a plugin by name.
func (r *Registry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.plugins[name]
	if !exists {
		return nil, errors.NotFound(fmt.Sprintf("plugin %s not found", name))
	}

	return info.Plugin, nil
}

// List returns all registered plugins.
func (r *Registry) List() map[string]*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*PluginInfo)
	for name, info := range r.plugins {
		result[name] = info
	}

	return result
}

// Initialize initializes a plugin.
func (r *Registry) Initialize(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.plugins[name]
	if !exists {
		return errors.NotFound(fmt.Sprintf("plugin %s not found", name))
	}

	if info.Status != StatusLoading {
		return errors.InvalidArgument(fmt.Sprintf("plugin %s is not in loading state", name))
	}

	// Initialize plugin
	if err := info.Plugin.Initialize(info.Context, info.Config); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "initialization failed for plugin %s", name)
	}

	info.Status = StatusInitialized

	// Execute post-initialize hooks
	if err := r.executeHooks(info.Context, "post-initialize", info); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "post-initialize hook failed for plugin %s", name)
	}

	return nil
}

// Start starts a plugin.
func (r *Registry) Start(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.plugins[name]
	if !exists {
		return errors.NotFound(fmt.Sprintf("plugin %s not found", name))
	}

	if info.Status != StatusInitialized {
		return errors.InvalidArgument(fmt.Sprintf("plugin %s is not initialized", name))
	}

	return r.startPlugin(info)
}

// Stop stops a plugin.
func (r *Registry) Stop(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.plugins[name]
	if !exists {
		return errors.NotFound(fmt.Sprintf("plugin %s not found", name))
	}

	if info.Status != StatusRunning {
		return errors.InvalidArgument(fmt.Sprintf("plugin %s is not running", name))
	}

	return r.stopPlugin(info)
}

// StartAll starts all registered plugins.
func (r *Registry) StartAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Sort plugins by priority and dependencies
	plugins := r.sortPlugins()

	for _, info := range plugins {
		if info.Status == StatusInitialized {
			if err := r.startPlugin(info); err != nil {
				return errors.Wrapf(err, errors.CodePlugin, "failed to start plugin %s", info.Name)
			}
		}
	}

	return nil
}

// StopAll stops all running plugins.
func (r *Registry) StopAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop plugins in reverse order
	plugins := r.sortPlugins()
	for i := len(plugins) - 1; i >= 0; i-- {
		info := plugins[i]
		if info.Status == StatusRunning {
			if err := r.stopPlugin(info); err != nil {
				return errors.Wrapf(err, errors.CodePlugin, "failed to stop plugin %s", info.Name)
			}
		}
	}

	return nil
}

// Health returns the health status of all plugins.
func (r *Registry) Health() map[string]HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]HealthStatus)
	for name, info := range r.plugins {
		result[name] = info.Plugin.Health()
	}

	return result
}

// AddHook adds a hook to the registry.
func (r *Registry) AddHook(name string, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hooks[name] = append(r.hooks[name], hook)
}

// RemoveHook removes a hook from the registry.
func (r *Registry) RemoveHook(name string, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()

	hooks := r.hooks[name]
	for i, h := range hooks {
		if h == hook {
			r.hooks[name] = append(hooks[:i], hooks[i+1:]...)
			break
		}
	}
}

// Close closes the registry and cleans up all plugins.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop all plugins
	for _, info := range r.plugins {
		if info.Status == StatusRunning {
			if err := r.stopPlugin(info); err != nil {
				return errors.Wrapf(err, errors.CodePlugin, "failed to stop plugin %s", info.Name)
			}
		}
	}

	// Cleanup all plugins
	for name, info := range r.plugins {
		if err := info.Plugin.Cleanup(); err != nil {
			return errors.Wrapf(err, errors.CodePlugin, "cleanup failed for plugin %s", name)
		}
		info.Cancel()
	}

	// Clear registry
	r.plugins = make(map[string]*PluginInfo)
	r.hooks = make(map[string][]Hook)

	return nil
}

// validateDependencies validates plugin dependencies.
func (r *Registry) validateDependencies(info *PluginInfo) error {
	for _, dep := range info.Dependencies {
		depInfo, exists := r.plugins[dep.Name]
		if !exists {
			if !dep.Optional {
				return fmt.Errorf("required dependency %s not found", dep.Name)
			}
			continue
		}

		// Check version compatibility
		if dep.Version != "" && !isVersionCompatible(depInfo.Version, dep.Version) {
			return fmt.Errorf("dependency %s version %s is not compatible with required %s",
				dep.Name, depInfo.Version, dep.Version)
		}
	}

	return nil
}

// startPlugin starts a plugin.
func (r *Registry) startPlugin(info *PluginInfo) error {
	info.Status = StatusStarting

	// Execute pre-start hooks
	if err := r.executeHooks(info.Context, "pre-start", info); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "pre-start hook failed for plugin %s", info.Name)
	}

	// Start plugin
	if err := info.Plugin.Start(info.Context); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "start failed for plugin %s", info.Name)
	}

	info.Status = StatusRunning

	// Execute post-start hooks
	if err := r.executeHooks(info.Context, "post-start", info); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "post-start hook failed for plugin %s", info.Name)
	}

	return nil
}

// stopPlugin stops a plugin.
func (r *Registry) stopPlugin(info *PluginInfo) error {
	info.Status = StatusStopping

	// Execute pre-stop hooks
	if err := r.executeHooks(info.Context, "pre-stop", info); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "pre-stop hook failed for plugin %s", info.Name)
	}

	// Stop plugin
	if err := info.Plugin.Stop(info.Context); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "stop failed for plugin %s", info.Name)
	}

	info.Status = StatusStopped

	// Execute post-stop hooks
	if err := r.executeHooks(info.Context, "post-stop", info); err != nil {
		info.Status = StatusError
		return errors.Wrapf(err, errors.CodePlugin, "post-stop hook failed for plugin %s", info.Name)
	}

	return nil
}

// executeHooks executes all hooks for a given event.
func (r *Registry) executeHooks(ctx context.Context, event string, data interface{}) error {
	hooks := r.hooks[event]

	for _, hook := range hooks {
		if err := hook.Execute(ctx, data); err != nil {
			return errors.Wrapf(err, errors.CodePlugin, "hook %s failed", hook.Name())
		}
	}

	return nil
}

// sortPlugins sorts plugins by priority and dependencies.
func (r *Registry) sortPlugins() []*PluginInfo {
	var plugins []*PluginInfo
	for _, info := range r.plugins {
		plugins = append(plugins, info)
	}

	// Sort by priority (higher priority first)
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Config.Priority > plugins[j].Config.Priority
	})

	// TODO: Implement topological sort for dependency order

	return plugins
}

// isVersionCompatible checks if two versions are compatible.
func isVersionCompatible(version, constraint string) bool {
	// Simple version compatibility check
	// In a real implementation, you'd use a proper semver library
	return version == constraint || constraint == "*"
}

// Loader loads plugins from various sources.
type Loader interface {
	// Load loads a plugin from a source
	Load(source string) (Plugin, error)

	// SupportsSource returns whether the loader supports the given source
	SupportsSource(source string) bool
}

// LoaderRegistry manages plugin loaders.
type LoaderRegistry struct {
	loaders []Loader
}

// NewLoaderRegistry creates a new loader registry.
func NewLoaderRegistry() *LoaderRegistry {
	return &LoaderRegistry{}
}

// AddLoader adds a loader to the registry.
func (r *LoaderRegistry) AddLoader(loader Loader) {
	r.loaders = append(r.loaders, loader)
}

// Load loads a plugin from a source.
func (r *LoaderRegistry) Load(source string) (Plugin, error) {
	for _, loader := range r.loaders {
		if loader.SupportsSource(source) {
			return loader.Load(source)
		}
	}

	return nil, errors.NotFound(fmt.Sprintf("no loader found for source: %s", source))
}

// GoPluginLoader loads Go plugins.
type GoPluginLoader struct{}

// Load loads a Go plugin.
func (l *GoPluginLoader) Load(source string) (Plugin, error) {
	// This would implement Go plugin loading
	// For now, return an error as Go plugins are complex
	return nil, errors.Unimplemented("Go plugin loading not implemented")
}

// SupportsSource returns whether the loader supports the given source.
func (l *GoPluginLoader) SupportsSource(source string) bool {
	return strings.HasSuffix(source, ".so") || strings.HasSuffix(source, ".dll")
}

// BuiltinLoader loads built-in plugins.
type BuiltinLoader struct {
	plugins map[string]Plugin
}

// NewBuiltinLoader creates a new built-in loader.
func NewBuiltinLoader() *BuiltinLoader {
	return &BuiltinLoader{
		plugins: make(map[string]Plugin),
	}
}

// Register registers a built-in plugin.
func (l *BuiltinLoader) Register(name string, plugin Plugin) {
	l.plugins[name] = plugin
}

// Load loads a built-in plugin.
func (l *BuiltinLoader) Load(source string) (Plugin, error) {
	plugin, exists := l.plugins[source]
	if !exists {
		return nil, errors.NotFound(fmt.Sprintf("built-in plugin %s not found", source))
	}

	return plugin, nil
}

// SupportsSource returns whether the loader supports the given source.
func (l *BuiltinLoader) SupportsSource(source string) bool {
	_, exists := l.plugins[source]
	return exists
}

// Manager manages the entire plugin system.
type Manager struct {
	registry *Registry
	loaders  *LoaderRegistry
	config   config.Config
}

// NewManager creates a new plugin manager.
func NewManager(cfg config.Config) (*Manager, error) {
	registryConfig := RegistryConfig{
		PluginDir:           cfg.Global.Transport.DefaultType, // This should be plugin dir
		HotReload:           false,
		Timeout:             30 * time.Second,
		MaxConcurrent:       10,
		HealthCheckInterval: 30 * time.Second,
	}

	registry, err := NewRegistry(registryConfig)
	if err != nil {
		return nil, errors.Wrapf(err, errors.CodePlugin, "failed to create plugin registry")
	}

	loaders := NewLoaderRegistry()

	// Add built-in loaders
	loaders.AddLoader(NewBuiltinLoader())
	loaders.AddLoader(&GoPluginLoader{})

	return &Manager{
		registry: registry,
		loaders:  loaders,
		config:   cfg,
	}, nil
}

// LoadPlugin loads a plugin from a source.
func (m *Manager) LoadPlugin(source string, config Config) error {
	plugin, err := m.loaders.Load(source)
	if err != nil {
		return errors.Wrapf(err, errors.CodePlugin, "failed to load plugin from %s", source)
	}

	if err := m.registry.Register(plugin, config); err != nil {
		return errors.Wrapf(err, errors.CodePlugin, "failed to register plugin %s", plugin.Name())
	}

	return nil
}

// LoadPluginsFromDir loads plugins from a directory.
func (m *Manager) LoadPluginsFromDir(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return errors.Wrapf(err, errors.CodeFileSystem, "failed to glob plugins in %s", dir)
	}

	for _, match := range matches {
		config := Config{
			Enabled:  true,
			Config:   make(map[string]interface{}),
			Settings: make(map[string]interface{}),
			Timeout:  30 * time.Second,
			Priority: 0,
		}

		if err := m.LoadPlugin(match, config); err != nil {
			return errors.Wrapf(err, errors.CodePlugin, "failed to load plugin %s", match)
		}
	}

	return nil
}

// GetRegistry returns the plugin registry.
func (m *Manager) GetRegistry() *Registry {
	return m.registry
}

// GetLoaders returns the loader registry.
func (m *Manager) GetLoaders() *LoaderRegistry {
	return m.loaders
}

// Close closes the plugin manager.
func (m *Manager) Close() error {
	return m.registry.Close()
}

// Default plugin manager instance
var defaultManager *Manager

// DefaultManager returns the default plugin manager.
func DefaultManager() *Manager {
	if defaultManager == nil {
		cfg := config.Config{} // Empty config for default
		var err error
		defaultManager, err = NewManager(cfg)
		if err != nil {
			panic(fmt.Sprintf("failed to create default plugin manager: %v", err))
		}
	}
	return defaultManager
}

// BasePlugin provides a base implementation for plugins.
type BasePlugin struct {
	name        string
	version     string
	description string
	status      Status
	health      HealthStatus
}

// NewBasePlugin creates a new base plugin.
func NewBasePlugin(name, version, description string) *BasePlugin {
	return &BasePlugin{
		name:        name,
		version:     version,
		description: description,
		status:      StatusUnknown,
		health: HealthStatus{
			Name:      name,
			Status:    StatusUnknown,
			Message:   "Plugin not initialized",
			LastCheck: time.Now(),
		},
	}
}

// Name returns the plugin name.
func (p *BasePlugin) Name() string {
	return p.name
}

// Version returns the plugin version.
func (p *BasePlugin) Version() string {
	return p.version
}

// Description returns the plugin description.
func (p *BasePlugin) Description() string {
	return p.description
}

// Dependencies returns the plugin dependencies.
func (p *BasePlugin) Dependencies() []Dependency {
	return nil
}

// Initialize initializes the plugin.
func (p *BasePlugin) Initialize(ctx context.Context, config Config) error {
	p.status = StatusInitialized
	p.health.Status = StatusInitialized
	p.health.Message = "Plugin initialized"
	p.health.LastCheck = time.Now()
	return nil
}

// Start starts the plugin.
func (p *BasePlugin) Start(ctx context.Context) error {
	p.status = StatusRunning
	p.health.Status = StatusRunning
	p.health.Message = "Plugin running"
	p.health.LastCheck = time.Now()
	return nil
}

// Stop stops the plugin.
func (p *BasePlugin) Stop(ctx context.Context) error {
	p.status = StatusStopped
	p.health.Status = StatusStopped
	p.health.Message = "Plugin stopped"
	p.health.LastCheck = time.Now()
	return nil
}

// Cleanup cleans up plugin resources.
func (p *BasePlugin) Cleanup() error {
	p.status = StatusUnloaded
	p.health.Status = StatusUnloaded
	p.health.Message = "Plugin unloaded"
	p.health.LastCheck = time.Now()
	return nil
}

// Health returns the plugin health status.
func (p *BasePlugin) Health() HealthStatus {
	p.health.LastCheck = time.Now()
	return p.health
}

// Reflection utilities for plugin introspection

// GetPluginInfo extracts plugin information using reflection.
func GetPluginInfo(plugin Plugin) map[string]interface{} {
	v := reflect.ValueOf(plugin)
	t := reflect.TypeOf(plugin)

	info := make(map[string]interface{})
	info["name"] = plugin.Name()
	info["version"] = plugin.Version()
	info["description"] = plugin.Description()
	info["type"] = t.String()
	info["methods"] = getMethodNames(t)

	return info
}

// getMethodNames returns method names of a type.
func getMethodNames(t reflect.Type) []string {
	var methods []string

	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if method.IsExported() {
			methods = append(methods, method.Name)
		}
	}

	return methods
}
