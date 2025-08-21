// Package mcp - Middleware Integration with Transport Layer and Server
//
// This file implements the integration between the comprehensive middleware system
// and the MCP transport layer, providing transport-specific middleware support
// and enhanced server functionality.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/jsonrpc2"
)

// Transport Integration
// ====================

// TransportMiddlewareManager manages middleware for different transport types
type TransportMiddlewareManager struct {
	mu              sync.RWMutex
	transportChains map[string]*MiddlewareChain
	globalChain     *MiddlewareChain
	methodChains    map[string]*MiddlewareChain
	registry        *MiddlewareRegistry
	logger          *slog.Logger
}

// NewTransportMiddlewareManager creates a new transport middleware manager
func NewTransportMiddlewareManager(registry *MiddlewareRegistry, logger *slog.Logger) *TransportMiddlewareManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &TransportMiddlewareManager{
		transportChains: make(map[string]*MiddlewareChain),
		methodChains:    make(map[string]*MiddlewareChain),
		registry:        registry,
		logger:          logger,
	}
}

// SetGlobalChain sets the global middleware chain
func (tm *TransportMiddlewareManager) SetGlobalChain(chain *MiddlewareChain) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.globalChain = chain
}

// SetTransportChain sets the middleware chain for a specific transport
func (tm *TransportMiddlewareManager) SetTransportChain(transport string, chain *MiddlewareChain) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.transportChains[transport] = chain
}

// SetMethodChain sets the middleware chain for a specific method
func (tm *TransportMiddlewareManager) SetMethodChain(method string, chain *MiddlewareChain) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.methodChains[method] = chain
}

// GetChainForRequest returns the appropriate middleware chain for a request
func (tm *TransportMiddlewareManager) GetChainForRequest(transport, method string) *MiddlewareChain {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Method-specific chain takes precedence
	if chain, exists := tm.methodChains[method]; exists {
		return chain
	}

	// Transport-specific chain
	if chain, exists := tm.transportChains[transport]; exists {
		return chain
	}

	// Global chain
	return tm.globalChain
}

// Enhanced Server with Middleware Support
// ======================================

// ServerMiddlewareConfig configures middleware for the enhanced server
type ServerMiddlewareConfig struct {
	GlobalConfig     *MiddlewareConfig            `json:"global_config,omitempty"`
	TransportConfigs map[string]*MiddlewareConfig `json:"transport_configs,omitempty"`
	MethodConfigs    map[string]*MiddlewareConfig `json:"method_configs,omitempty"`
	EnableMetrics    bool                         `json:"enable_metrics"`
	MetricsRegistry  MetricsRegistry              `json:"-"`
	Logger           *slog.Logger                 `json:"-"`
}

// EnhancedServer extends the base Server with comprehensive middleware support
type EnhancedServer struct {
	*Server
	middlewareManager *TransportMiddlewareManager
	registry          *MiddlewareRegistry
	config            *ServerMiddlewareConfig
	handler           MCPHandler
	logger            *slog.Logger
}

// NewEnhancedServer creates a new enhanced server with middleware support
func NewEnhancedServer(opts ...ServerOption) *EnhancedServer {
	return NewEnhancedServerWithName("enhanced-mcp-server", "1.0.0", opts...)
}

// NewEnhancedServerWithName creates a new enhanced server with custom name and version
func NewEnhancedServerWithName(name, version string, opts ...ServerOption) *EnhancedServer {
	baseServer := NewServer(name, version, opts...)
	logger := slog.Default()

	registry := NewMiddlewareRegistry(logger)
	manager := NewTransportMiddlewareManager(registry, logger)

	enhanced := &EnhancedServer{
		Server:            baseServer,
		middlewareManager: manager,
		registry:          registry,
		logger:            logger,
	}

	// Set up the unified handler
	enhanced.handler = enhanced.createUnifiedHandler()

	return enhanced
}

// SetMiddlewareConfig configures middleware for the server
func (s *EnhancedServer) SetMiddlewareConfig(config *ServerMiddlewareConfig) error {
	s.config = config

	// Configure global middleware
	if config.GlobalConfig != nil {
		chain, err := s.createChainFromConfig(config.GlobalConfig)
		if err != nil {
			return fmt.Errorf("failed to create global middleware chain: %w", err)
		}
		s.middlewareManager.SetGlobalChain(chain)
	}

	// Configure transport-specific middleware
	for transport, transportConfig := range config.TransportConfigs {
		chain, err := s.createChainFromConfig(transportConfig)
		if err != nil {
			return fmt.Errorf("failed to create middleware chain for transport %s: %w", transport, err)
		}
		s.middlewareManager.SetTransportChain(transport, chain)
	}

	// Configure method-specific middleware
	for method, methodConfig := range config.MethodConfigs {
		chain, err := s.createChainFromConfig(methodConfig)
		if err != nil {
			return fmt.Errorf("failed to create middleware chain for method %s: %w", method, err)
		}
		s.middlewareManager.SetMethodChain(method, chain)
	}

	return nil
}

// UseMiddleware adds middleware to the global chain
func (s *EnhancedServer) UseMiddleware(middleware Middleware) {
	if s.middlewareManager.globalChain == nil {
		s.middlewareManager.globalChain = &MiddlewareChain{}
	}
	s.middlewareManager.globalChain.middlewares = append(s.middlewareManager.globalChain.middlewares, middleware)
}

// UseMiddlewareForTransport adds middleware for a specific transport
func (s *EnhancedServer) UseMiddlewareForTransport(transport string, middleware Middleware) {
	chain := s.middlewareManager.transportChains[transport]
	if chain == nil {
		chain = &MiddlewareChain{}
		s.middlewareManager.transportChains[transport] = chain
	}
	chain.middlewares = append(chain.middlewares, middleware)
}

// UseMiddlewareForMethod adds middleware for a specific method
func (s *EnhancedServer) UseMiddlewareForMethod(method string, middleware Middleware) {
	chain := s.middlewareManager.methodChains[method]
	if chain == nil {
		chain = &MiddlewareChain{}
		s.middlewareManager.methodChains[method] = chain
	}
	chain.middlewares = append(chain.middlewares, middleware)
}

// createChainFromConfig creates a middleware chain from configuration
func (s *EnhancedServer) createChainFromConfig(config *MiddlewareConfig) (*MiddlewareChain, error) {
	chain := &MiddlewareChain{
		registry: s.registry,
		config:   config,
		metrics:  &MiddlewareMetrics{},
	}

	if !config.Enabled {
		return chain, nil
	}

	// Create middleware instances from config
	if config.Logging != nil {
		middleware, err := s.registry.CreateMiddleware("logging", config.Logging)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.Authentication != nil {
		middleware, err := s.registry.CreateMiddleware("authentication", config.Authentication)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.RateLimit != nil {
		middleware, err := s.registry.CreateMiddleware("rate_limit", config.RateLimit)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.Timeout != nil {
		middleware, err := s.registry.CreateMiddleware("timeout", config.Timeout)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.Recovery != nil {
		middleware, err := s.registry.CreateMiddleware("recovery", config.Recovery)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.Metrics != nil {
		middleware, err := s.registry.CreateMiddleware("metrics", config.Metrics)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.CORS != nil {
		middleware, err := s.registry.CreateMiddleware("cors", config.CORS)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.Compression != nil {
		middleware, err := s.registry.CreateMiddleware("compression", config.Compression)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.Validation != nil {
		middleware, err := s.registry.CreateMiddleware("validation", config.Validation)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	if config.Caching != nil {
		middleware, err := s.registry.CreateMiddleware("caching", config.Caching)
		if err != nil {
			return nil, err
		}
		chain.middlewares = append(chain.middlewares, middleware)
	}

	// Add conditional middleware
	for _, conditionalConfig := range config.ConditionalMiddleware {
		condition, err := s.createCondition(conditionalConfig.Condition)
		if err != nil {
			return nil, fmt.Errorf("failed to create condition for %s: %w", conditionalConfig.Name, err)
		}

		baseMiddleware, err := s.registry.CreateMiddleware(conditionalConfig.Name, conditionalConfig.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create conditional middleware %s: %w", conditionalConfig.Name, err)
		}

		conditional := NewConditionalMiddleware(condition, baseMiddleware, s.logger)
		chain.middlewares = append(chain.middlewares, conditional)
	}

	return chain, nil
}

// createCondition creates a condition evaluator from a condition string
func (s *EnhancedServer) createCondition(conditionStr string) (ConditionEvaluator, error) {
	// Parse condition string (simplified implementation)
	parts := strings.Split(conditionStr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid condition format: %s", conditionStr)
	}

	conditionType := parts[0]
	value := parts[1]

	switch conditionType {
	case "method":
		return NewMethodCondition(strings.Split(value, ",")), nil
	case "client":
		return NewClientCondition(strings.Split(value, ",")), nil
	case "regex":
		return NewRegexCondition(value, "method")
	default:
		return nil, fmt.Errorf("unknown condition type: %s", conditionType)
	}
}

// createUnifiedHandler creates a unified handler that bridges legacy and new systems
func (s *EnhancedServer) createUnifiedHandler() MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Extract transport and method information
		transport := s.getTransportFromContext(ctx)
		method := req.GetMethod()

		// Get appropriate middleware chain
		chain := s.middlewareManager.GetChainForRequest(transport, method)

		// Create the base handler that delegates to the original server
		baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
			return s.delegateToOriginalServer(ctx, req)
		})

		// Apply middleware chain if available
		if chain != nil {
			finalHandler := chain.Apply(baseHandler)
			return finalHandler.Handle(ctx, req)
		}

		return baseHandler.Handle(ctx, req)
	})
}

// delegateToOriginalServer delegates to the original server's handler system
func (s *EnhancedServer) delegateToOriginalServer(ctx context.Context, req MCPRequest) (MCPResponse, error) {
	// Convert unified request back to JSON-RPC format
	jsonRPCReq, err := s.requestToJSONRPC(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request to JSON-RPC: %w", err)
	}

	// Call the original server's handleRequest method
	// This accesses the embedded *Server's request handling
	if s.Server == nil {
		return nil, fmt.Errorf("original server is nil")
	}

	// Use the original server's handleRequest method directly
	result, err := s.Server.handleRequest(ctx, jsonRPCReq)
	if err != nil {
		return &ErrorResponseImpl{
			Error: &ResponseError{
				Code:    -32603,
				Message: err.Error(),
			},
		}, nil
	}

	// Wrap the result in a success response
	return &SuccessResponseImpl{
		Result: result,
	}, nil
}

// Request/Response Adapters
// ========================

// UnifiedRequest implements the Request interface for the middleware system
type UnifiedRequest struct {
	method string
	id     interface{}
	params json.RawMessage
	ctx    context.Context
}

func (r *UnifiedRequest) GetMethod() string {
	return r.method
}

func (r *UnifiedRequest) GetID() interface{} {
	return r.id
}

func (r *UnifiedRequest) GetParams() json.RawMessage {
	return r.params
}

func (r *UnifiedRequest) GetContext() context.Context {
	return r.ctx
}

func (r *UnifiedRequest) WithContext(ctx context.Context) MCPRequest {
	return &UnifiedRequest{
		method: r.method,
		id:     r.id,
		params: r.params,
		ctx:    ctx,
	}
}

// SuccessResponseImpl implements Response for successful responses
type SuccessResponseImpl struct {
	Result interface{} `json:"result"`
}

func (r *SuccessResponseImpl) GetResult() interface{} {
	return r.Result
}

func (r *SuccessResponseImpl) GetError() *ResponseError {
	return nil
}

func (r *SuccessResponseImpl) IsError() bool {
	return false
}

// Helper methods
func (s *EnhancedServer) getTransportFromContext(ctx context.Context) string {
	if transport, ok := ctx.Value("transport").(string); ok {
		return transport
	}
	return "unknown"
}

func (s *EnhancedServer) requestToJSONRPC(req MCPRequest) (*jsonrpc2.Request, error) {
	var id jsonrpc2.ID
	if reqID := req.GetID(); reqID != nil {
		if idStr, ok := reqID.(string); ok {
			id = jsonrpc2.StringID(idStr)
		} else if idNum, ok := reqID.(int); ok {
			id = jsonrpc2.Int64ID(int64(idNum))
		} else if idNum, ok := reqID.(int64); ok {
			id = jsonrpc2.Int64ID(idNum)
		}
	}

	return &jsonrpc2.Request{
		Method: req.GetMethod(),
		ID:     id,
		Params: req.GetParams(),
	}, nil
}

// Middleware Metrics and Monitoring
// =================================

// MiddlewareMetrics provides metrics collection for middleware performance
type MiddlewareMetrics struct {
	mu               sync.RWMutex
	requestCounts    map[string]int64
	errorCounts      map[string]int64
	latencies        map[string]time.Duration
	activeMiddleware map[string]bool
}

// RecordRequest records a request processed by middleware
func (m *MiddlewareMetrics) RecordRequest(middlewareName string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.requestCounts == nil {
		m.requestCounts = make(map[string]int64)
		m.errorCounts = make(map[string]int64)
		m.latencies = make(map[string]time.Duration)
		m.activeMiddleware = make(map[string]bool)
	}

	m.requestCounts[middlewareName]++
	m.latencies[middlewareName] = duration
	m.activeMiddleware[middlewareName] = true

	if !success {
		m.errorCounts[middlewareName]++
	}
}

// GetMetrics returns current middleware metrics
func (m *MiddlewareMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]interface{})

	for name := range m.activeMiddleware {
		metrics[name] = map[string]interface{}{
			"requests":     m.requestCounts[name],
			"errors":       m.errorCounts[name],
			"last_latency": m.latencies[name],
			"error_rate":   float64(m.errorCounts[name]) / float64(m.requestCounts[name]),
		}
	}

	return metrics
}

// Apply method for MiddlewareChain (enhanced version)
func (mc *MiddlewareChain) Apply(handler MCPHandler) MCPHandler {
	if len(mc.middlewares) == 0 {
		return handler
	}

	// Sort middleware by priority
	sortedMiddleware := make([]Middleware, len(mc.middlewares))
	copy(sortedMiddleware, mc.middlewares)

	// Simple bubble sort by priority (descending)
	for i := 0; i < len(sortedMiddleware)-1; i++ {
		for j := 0; j < len(sortedMiddleware)-i-1; j++ {
			if sortedMiddleware[j].Priority() < sortedMiddleware[j+1].Priority() {
				sortedMiddleware[j], sortedMiddleware[j+1] = sortedMiddleware[j+1], sortedMiddleware[j]
			}
		}
	}

	// Apply middleware in reverse priority order (higher priority middleware wrap lower priority ones)
	current := handler
	for i := len(sortedMiddleware) - 1; i >= 0; i-- {
		current = sortedMiddleware[i].Apply(current)
	}

	return current
}

// Configuration Loading Utilities
// ===============================

// LoadMiddlewareConfigFromJSON loads middleware configuration from JSON
func LoadMiddlewareConfigFromJSON(data []byte) (*MiddlewareConfig, error) {
	var config MiddlewareConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal middleware config: %w", err)
	}
	return &config, nil
}

// LoadServerMiddlewareConfigFromJSON loads server middleware configuration from JSON
func LoadServerMiddlewareConfigFromJSON(data []byte) (*ServerMiddlewareConfig, error) {
	var config ServerMiddlewareConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server middleware config: %w", err)
	}
	return &config, nil
}

// ValidateMiddlewareConfig validates a middleware configuration
func ValidateMiddlewareConfig(config *MiddlewareConfig) error {
	if config == nil {
		return fmt.Errorf("middleware config cannot be nil")
	}

	// Validate timeout values
	if config.DefaultTimeout < 0 {
		return fmt.Errorf("default timeout cannot be negative")
	}

	// Validate concurrency limits
	if config.MaxConcurrency < 0 {
		return fmt.Errorf("max concurrency cannot be negative")
	}

	// Add more validation as needed
	return nil
}

// GetMiddlewareConfigSchema returns a JSON schema for middleware configuration
func GetMiddlewareConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"enabled": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether middleware is enabled",
				"default":     true,
			},
			"default_timeout": map[string]interface{}{
				"type":        "string",
				"description": "Default timeout duration (e.g., '30s')",
				"default":     "30s",
			},
			"max_concurrency": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum concurrent requests",
				"minimum":     1,
				"default":     100,
			},
			"logging": map[string]interface{}{
				"$ref": "#/definitions/LoggingConfig",
			},
			"authentication": map[string]interface{}{
				"$ref": "#/definitions/AuthConfig",
			},
			"rate_limit": map[string]interface{}{
				"$ref": "#/definitions/RateLimitConfig",
			},
		},
		"definitions": map[string]interface{}{
			"LoggingConfig": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"level": map[string]interface{}{
						"type": "string",
						"enum": []string{"debug", "info", "warn", "error"},
					},
					"include_request": map[string]interface{}{
						"type": "boolean",
					},
					"include_response": map[string]interface{}{
						"type": "boolean",
					},
				},
			},
		},
	}
}
