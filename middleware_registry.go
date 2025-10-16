// Package mcp - Middleware Registry and Configuration System
//
// This file implements the middleware registry, configuration system,
// and dynamic middleware management for the comprehensive middleware system.

package mcp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// Middleware Configuration System
// ===============================

// MiddlewareRegistry manages middleware registration, discovery, and configuration
type MiddlewareRegistry struct {
	mu          sync.RWMutex
	middlewares map[string]MiddlewareFactory
	configs     map[string]interface{}
	instances   map[string]Middleware
	groups      map[string]*MiddlewareGroup
	logger      *slog.Logger
}

// MiddlewareFactory creates middleware instances from configuration
type MiddlewareFactory interface {
	Create(config interface{}) (Middleware, error)
	ConfigType() interface{} // Returns a zero value of the expected config type
	Name() string
	Description() string
}

// MiddlewareConfig provides configuration for the entire middleware system
type MiddlewareConfig struct {
	// Global settings
	Enabled        bool          `json:"enabled" yaml:"enabled"`
	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout"`
	MaxConcurrency int           `json:"max_concurrency" yaml:"max_concurrency"`

	// Individual middleware configurations
	Logging        *LoggingConfig              `json:"logging,omitempty" yaml:"logging,omitempty"`
	Authentication *AuthConfig                 `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	RateLimit      *RateLimitConfig            `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
	Timeout        *TimeoutConfig              `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Recovery       *RecoveryConfig             `json:"recovery,omitempty" yaml:"recovery,omitempty"`
	Metrics        *MetricsConfig              `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	CORS           *CORSConfig                 `json:"cors,omitempty" yaml:"cors,omitempty"`
	Compression    *CompressionConfig          `json:"compression,omitempty" yaml:"compression,omitempty"`
	Validation     *MiddlewareValidationConfig `json:"validation,omitempty" yaml:"validation,omitempty"`
	Caching        *CachingConfig              `json:"caching,omitempty" yaml:"caching,omitempty"`

	// Transport-specific configurations
	TransportConfigs map[string]*TransportMiddlewareConfig `json:"transport_configs,omitempty" yaml:"transport_configs,omitempty"`

	// Method-specific configurations
	MethodConfigs map[string]*MethodMiddlewareConfig `json:"method_configs,omitempty" yaml:"method_configs,omitempty"`

	// Conditional middleware
	ConditionalMiddleware []*ConditionalMiddlewareConfig `json:"conditional_middleware,omitempty" yaml:"conditional_middleware,omitempty"`

	// Custom middleware configurations
	Custom map[string]interface{} `json:"custom,omitempty" yaml:"custom,omitempty"`
}

// TransportMiddlewareConfig configures middleware for specific transports
type TransportMiddlewareConfig struct {
	Transport    string                 `json:"transport" yaml:"transport"`
	EnabledOnly  []string               `json:"enabled_only,omitempty" yaml:"enabled_only,omitempty"`
	DisabledOnly []string               `json:"disabled_only,omitempty" yaml:"disabled_only,omitempty"`
	CustomConfig map[string]interface{} `json:"custom_config,omitempty" yaml:"custom_config,omitempty"`
}

// MethodMiddlewareConfig configures middleware for specific methods
type MethodMiddlewareConfig struct {
	Method        string                 `json:"method" yaml:"method"`
	EnabledOnly   []string               `json:"enabled_only,omitempty" yaml:"enabled_only,omitempty"`
	DisabledOnly  []string               `json:"disabled_only,omitempty" yaml:"disabled_only,omitempty"`
	CustomTimeout *time.Duration         `json:"custom_timeout,omitempty" yaml:"custom_timeout,omitempty"`
	CustomConfig  map[string]interface{} `json:"custom_config,omitempty" yaml:"custom_config,omitempty"`
}

// ConditionalMiddlewareConfig configures conditional middleware application
type ConditionalMiddlewareConfig struct {
	Name      string      `json:"name" yaml:"name"`
	Condition string      `json:"condition" yaml:"condition"` // Expression or rule
	Config    interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Priority  int         `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// NewMiddlewareRegistry creates a new middleware registry
func NewMiddlewareRegistry(logger *slog.Logger) *MiddlewareRegistry {
	if logger == nil {
		logger = slog.Default()
	}

	registry := &MiddlewareRegistry{
		middlewares: make(map[string]MiddlewareFactory),
		configs:     make(map[string]interface{}),
		instances:   make(map[string]Middleware),
		groups:      make(map[string]*MiddlewareGroup),
		logger:      logger,
	}

	// Register built-in middleware factories
	registry.registerBuiltinFactories()

	return registry
}

// RegisterFactory registers a middleware factory
func (r *MiddlewareRegistry) RegisterFactory(factory MiddlewareFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := factory.Name()
	if _, exists := r.middlewares[name]; exists {
		return fmt.Errorf("middleware factory %q already registered", name)
	}

	r.middlewares[name] = factory
	r.logger.Debug("Middleware factory registered", "name", name)

	return nil
}

// GetFactory retrieves a middleware factory by name
func (r *MiddlewareRegistry) GetFactory(name string) (MiddlewareFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.middlewares[name]
	return factory, exists
}

// CreateMiddleware creates a middleware instance from configuration
func (r *MiddlewareRegistry) CreateMiddleware(name string, config interface{}) (Middleware, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	factory, exists := r.middlewares[name]
	if !exists {
		return nil, fmt.Errorf("middleware factory %q not found", name)
	}

	instance, err := factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create middleware %q: %w", name, err)
	}

	r.instances[name] = instance
	r.logger.Debug("Middleware instance created", "name", name)

	return instance, nil
}

// GetMiddleware retrieves a middleware instance by name
func (r *MiddlewareRegistry) GetMiddleware(name string) (Middleware, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, exists := r.instances[name]
	return instance, exists
}

// ListFactories returns all registered middleware factories
func (r *MiddlewareRegistry) ListFactories() []MiddlewareFactory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factories := make([]MiddlewareFactory, 0, len(r.middlewares))
	for _, factory := range r.middlewares {
		factories = append(factories, factory)
	}

	return factories
}

// MiddlewareGroup manages a group of middleware with common configuration
type MiddlewareGroup struct {
	name        string
	middlewares []Middleware
	config      *MiddlewareConfig
	registry    *MiddlewareRegistry
	enabled     bool
}

// NewMiddlewareGroup creates a new middleware group
func NewMiddlewareGroup(name string, registry *MiddlewareRegistry) *MiddlewareGroup {
	return &MiddlewareGroup{
		name:        name,
		middlewares: make([]Middleware, 0),
		registry:    registry,
		enabled:     true,
	}
}

// Add adds middleware to the group
func (g *MiddlewareGroup) Add(middleware Middleware) *MiddlewareGroup {
	g.middlewares = append(g.middlewares, middleware)
	return g
}

// AddByName adds middleware to the group by factory name and config
func (g *MiddlewareGroup) AddByName(name string, config interface{}) error {
	middleware, err := g.registry.CreateMiddleware(name, config)
	if err != nil {
		return err
	}

	g.middlewares = append(g.middlewares, middleware)
	return nil
}

// Apply applies all middleware in the group to a handler
func (g *MiddlewareGroup) Apply(handler MCPHandler) MCPHandler {
	if !g.enabled {
		return handler
	}

	// Sort middleware by priority (highest first)
	sorted := make([]Middleware, len(g.middlewares))
	copy(sorted, g.middlewares)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() > sorted[j].Priority()
	})

	// Apply middleware in priority order
	current := handler
	for _, middleware := range sorted {
		current = middleware.Apply(current)
	}

	return current
}

// Enable enables the middleware group
func (g *MiddlewareGroup) Enable() {
	g.enabled = true
}

// Disable disables the middleware group
func (g *MiddlewareGroup) Disable() {
	g.enabled = false
}

// IsEnabled returns whether the group is enabled
func (g *MiddlewareGroup) IsEnabled() bool {
	return g.enabled
}

// Built-in Middleware Factories
// =============================

// registerBuiltinFactories registers all built-in middleware factories
func (r *MiddlewareRegistry) registerBuiltinFactories() {
	factories := []MiddlewareFactory{
		&LoggingMiddlewareFactory{},
		&AuthenticationMiddlewareFactory{},
		&RateLimitMiddlewareFactory{},
		&TimeoutMiddlewareFactory{},
		&RecoveryMiddlewareFactory{},
		&MetricsMiddlewareFactory{},
		&CORSMiddlewareFactory{},
		&CompressionMiddlewareFactory{},
		&ValidationMiddlewareFactory{},
		&CachingMiddlewareFactory{},
	}

	for _, factory := range factories {
		if err := r.RegisterFactory(factory); err != nil {
			r.logger.Error("Failed to register built-in factory", "name", factory.Name(), "error", err)
		}
	}
}

// LoggingMiddlewareFactory creates logging middleware instances
type LoggingMiddlewareFactory struct{}

func (f *LoggingMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	var loggingConfig LoggingConfig
	if config != nil {
		if lc, ok := config.(*LoggingConfig); ok {
			loggingConfig = *lc
		} else if lc, ok := config.(LoggingConfig); ok {
			loggingConfig = lc
		} else {
			// Try to unmarshal from map or JSON
			if err := mapToStruct(config, &loggingConfig); err != nil {
				return nil, fmt.Errorf("invalid logging config: %w", err)
			}
		}
	}

	return NewLoggingMiddleware(loggingConfig), nil
}

func (f *LoggingMiddlewareFactory) ConfigType() interface{} {
	return LoggingConfig{}
}

func (f *LoggingMiddlewareFactory) Name() string {
	return "logging"
}

func (f *LoggingMiddlewareFactory) Description() string {
	return "Provides comprehensive request/response logging"
}

// AuthenticationMiddlewareFactory creates authentication middleware instances
type AuthenticationMiddlewareFactory struct{}

func (f *AuthenticationMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	var authConfig AuthConfig
	if config != nil {
		if ac, ok := config.(*AuthConfig); ok {
			authConfig = *ac
		} else if ac, ok := config.(AuthConfig); ok {
			authConfig = ac
		} else {
			if err := mapToStruct(config, &authConfig); err != nil {
				return nil, fmt.Errorf("invalid auth config: %w", err)
			}
		}
	}

	if authConfig.Provider == nil {
		// Create default memory provider for testing
		authConfig.Provider = NewMemoryOAuthProvider()
	}

	return NewAuthenticationMiddleware(authConfig), nil
}

func (f *AuthenticationMiddlewareFactory) ConfigType() interface{} {
	return AuthConfig{}
}

func (f *AuthenticationMiddlewareFactory) Name() string {
	return "authentication"
}

func (f *AuthenticationMiddlewareFactory) Description() string {
	return "Provides OAuth2 token-based authentication"
}

// RateLimitMiddlewareFactory creates rate limiting middleware instances
type RateLimitMiddlewareFactory struct{}

func (f *RateLimitMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	var rlConfig RateLimitConfig
	if config != nil {
		if rc, ok := config.(*RateLimitConfig); ok {
			rlConfig = *rc
		} else if rc, ok := config.(RateLimitConfig); ok {
			rlConfig = rc
		} else {
			if err := mapToStruct(config, &rlConfig); err != nil {
				return nil, fmt.Errorf("invalid rate limit config: %w", err)
			}
		}
	}

	return NewRateLimitMiddleware(rlConfig), nil
}

func (f *RateLimitMiddlewareFactory) ConfigType() interface{} {
	return RateLimitConfig{}
}

func (f *RateLimitMiddlewareFactory) Name() string {
	return "rate_limit"
}

func (f *RateLimitMiddlewareFactory) Description() string {
	return "Provides request rate limiting with configurable strategies"
}

// TimeoutMiddlewareFactory creates timeout middleware instances
type TimeoutMiddlewareFactory struct{}

// TimeoutConfig configures timeout middleware
type TimeoutConfig struct {
	Timeout         time.Duration `json:"timeout" yaml:"timeout"`
	TimeoutResponse *string       `json:"timeout_response,omitempty" yaml:"timeout_response,omitempty"`
}

func (f *TimeoutMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	var timeoutConfig TimeoutConfig
	if config != nil {
		if tc, ok := config.(*TimeoutConfig); ok {
			timeoutConfig = *tc
		} else if tc, ok := config.(TimeoutConfig); ok {
			timeoutConfig = tc
		} else {
			if err := mapToStruct(config, &timeoutConfig); err != nil {
				return nil, fmt.Errorf("invalid timeout config: %w", err)
			}
		}
	}

	if timeoutConfig.Timeout == 0 {
		timeoutConfig.Timeout = 30 * time.Second
	}

	return NewTimeoutMiddleware(timeoutConfig.Timeout), nil
}

func (f *TimeoutMiddlewareFactory) ConfigType() interface{} {
	return TimeoutConfig{}
}

func (f *TimeoutMiddlewareFactory) Name() string {
	return "timeout"
}

func (f *TimeoutMiddlewareFactory) Description() string {
	return "Provides request timeout handling with graceful degradation"
}

// RecoveryMiddlewareFactory creates recovery middleware instances
type RecoveryMiddlewareFactory struct{}

// RecoveryConfig configures recovery middleware
type RecoveryConfig struct {
	IncludeStack bool         `json:"include_stack" yaml:"include_stack"`
	Logger       *slog.Logger `json:"-" yaml:"-"`
}

func (f *RecoveryMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	var recoveryConfig RecoveryConfig
	if config != nil {
		if rc, ok := config.(*RecoveryConfig); ok {
			recoveryConfig = *rc
		} else if rc, ok := config.(RecoveryConfig); ok {
			recoveryConfig = rc
		} else {
			if err := mapToStruct(config, &recoveryConfig); err != nil {
				return nil, fmt.Errorf("invalid recovery config: %w", err)
			}
		}
	}

	return NewRecoveryMiddleware(recoveryConfig.Logger, recoveryConfig.IncludeStack), nil
}

func (f *RecoveryMiddlewareFactory) ConfigType() interface{} {
	return RecoveryConfig{}
}

func (f *RecoveryMiddlewareFactory) Name() string {
	return "recovery"
}

func (f *RecoveryMiddlewareFactory) Description() string {
	return "Provides panic recovery with structured error responses"
}

// MetricsMiddlewareFactory creates metrics middleware instances
type MetricsMiddlewareFactory struct{}

// MetricsConfig configures metrics middleware
type MetricsConfig struct {
	Registry MetricsRegistry `json:"-" yaml:"-"`
	Labels   []string        `json:"labels,omitempty" yaml:"labels,omitempty"`
	Buckets  []float64       `json:"buckets,omitempty" yaml:"buckets,omitempty"`
}

func (f *MetricsMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	var metricsConfig MetricsConfig
	if config != nil {
		if mc, ok := config.(*MetricsConfig); ok {
			metricsConfig = *mc
		} else if mc, ok := config.(MetricsConfig); ok {
			metricsConfig = mc
		} else {
			if err := mapToStruct(config, &metricsConfig); err != nil {
				return nil, fmt.Errorf("invalid metrics config: %w", err)
			}
		}
	}

	if metricsConfig.Registry == nil {
		// Create default in-memory registry
		metricsConfig.Registry = &InMemoryMetricsRegistry{}
	}

	return NewMetricsMiddleware(metricsConfig.Registry), nil
}

func (f *MetricsMiddlewareFactory) ConfigType() interface{} {
	return MetricsConfig{}
}

func (f *MetricsMiddlewareFactory) Name() string {
	return "metrics"
}

func (f *MetricsMiddlewareFactory) Description() string {
	return "Provides comprehensive metrics collection and monitoring"
}

// Placeholder factories for additional middleware
// (These would be implemented with full functionality)

type CORSMiddlewareFactory struct{}

func (f *CORSMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	var corsConfig CORSConfig
	if config != nil {
		if cc, ok := config.(*CORSConfig); ok {
			corsConfig = *cc
		} else if cc, ok := config.(CORSConfig); ok {
			corsConfig = cc
		} else {
			if err := mapToStruct(config, &corsConfig); err != nil {
				return nil, fmt.Errorf("invalid CORS config: %w", err)
			}
		}
	}

	return NewCORSMiddleware(corsConfig), nil
}

func (f *CORSMiddlewareFactory) ConfigType() interface{} {
	return CORSConfig{}
}

func (f *CORSMiddlewareFactory) Name() string {
	return "cors"
}

func (f *CORSMiddlewareFactory) Description() string {
	return "Provides Cross-Origin Resource Sharing (CORS) handling"
}

type CompressionMiddlewareFactory struct{}

func (f *CompressionMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	// Implementation would go here
	return &NoOpMiddleware{name: "compression"}, nil
}

func (f *CompressionMiddlewareFactory) ConfigType() interface{} {
	return CompressionConfig{}
}

func (f *CompressionMiddlewareFactory) Name() string {
	return "compression"
}

func (f *CompressionMiddlewareFactory) Description() string {
	return "Provides request/response compression (gzip, brotli)"
}

type ValidationMiddlewareFactory struct{}

func (f *ValidationMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	// Implementation would go here
	return &NoOpMiddleware{name: "validation"}, nil
}

func (f *ValidationMiddlewareFactory) ConfigType() interface{} {
	return MiddlewareValidationConfig{}
}

func (f *ValidationMiddlewareFactory) Name() string {
	return "validation"
}

func (f *ValidationMiddlewareFactory) Description() string {
	return "Provides request/response validation using JSON schemas"
}

type CachingMiddlewareFactory struct{}

func (f *CachingMiddlewareFactory) Create(config interface{}) (Middleware, error) {
	// Implementation would go here
	return &NoOpMiddleware{name: "caching"}, nil
}

func (f *CachingMiddlewareFactory) ConfigType() interface{} {
	return CachingConfig{}
}

func (f *CachingMiddlewareFactory) Name() string {
	return "caching"
}

func (f *CachingMiddlewareFactory) Description() string {
	return "Provides response caching with TTL and cache keys"
}

// Configuration Types for Placeholder Middleware
// ==============================================

type CORSConfig struct {
	AllowOrigins []string `json:"allow_origins" yaml:"allow_origins"`
	AllowMethods []string `json:"allow_methods" yaml:"allow_methods"`
	AllowHeaders []string `json:"allow_headers" yaml:"allow_headers"`
	MaxAge       int      `json:"max_age" yaml:"max_age"`
}

type CompressionConfig struct {
	Algorithms []string `json:"algorithms" yaml:"algorithms"`
	MinSize    int      `json:"min_size" yaml:"min_size"`
	Level      int      `json:"level" yaml:"level"`
}

type MiddlewareValidationConfig struct {
	SchemaRegistry string                 `json:"schema_registry" yaml:"schema_registry"`
	Schemas        map[string]interface{} `json:"schemas" yaml:"schemas"`
	StrictMode     bool                   `json:"strict_mode" yaml:"strict_mode"`
}

type CachingConfig struct {
	TTL         time.Duration `json:"ttl" yaml:"ttl"`
	MaxSize     int64         `json:"max_size" yaml:"max_size"`
	KeyStrategy string        `json:"key_strategy" yaml:"key_strategy"`
}

// Helper Types and Functions
// ==========================

// NoOpMiddleware is a placeholder middleware that does nothing
type NoOpMiddleware struct {
	name string
}

func (m *NoOpMiddleware) Apply(next MCPHandler) MCPHandler {
	return next
}

func (m *NoOpMiddleware) Name() string {
	return m.name
}

func (m *NoOpMiddleware) Priority() int {
	return 0
}

// InMemoryMetricsRegistry provides a simple in-memory metrics registry
type InMemoryMetricsRegistry struct {
	mu      sync.RWMutex
	metrics map[string]interface{}
}

func (r *InMemoryMetricsRegistry) RecordRequest(method string, duration time.Duration, statusCode int, labels map[string]string) {
	// Simple implementation for demonstration
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.metrics == nil {
		r.metrics = make(map[string]interface{})
	}

	key := fmt.Sprintf("request_%s_%d", method, statusCode)
	r.metrics[key] = map[string]interface{}{
		"method":      method,
		"duration":    duration,
		"status_code": statusCode,
		"labels":      labels,
		"timestamp":   time.Now(),
	}
}

func (r *InMemoryMetricsRegistry) RecordActiveRequests(count int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.metrics == nil {
		r.metrics = make(map[string]interface{})
	}

	r.metrics["active_requests"] = count
}

func (r *InMemoryMetricsRegistry) RecordError(method string, errorType string, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.metrics == nil {
		r.metrics = make(map[string]interface{})
	}

	key := fmt.Sprintf("error_%s_%s", method, errorType)
	r.metrics[key] = map[string]interface{}{
		"method":     method,
		"error_type": errorType,
		"labels":     labels,
		"timestamp":  time.Now(),
	}
}

// mapToStruct converts a map or other structure to a target struct using JSON marshaling
func mapToStruct(source interface{}, target interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}
