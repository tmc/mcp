// Package mcp - Comprehensive Middleware System Implementation
//
// This file implements Phase 2B of the MCP Go comprehensive roadmap:
// A production-ready middleware system that integrates with the type-safe APIs
// from Phase 2A and provides extensive middleware capabilities.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Core Middleware Infrastructure
// ==============================

// MiddlewareChain manages middleware for MCP handlers with type safety and performance optimization
type MiddlewareChain struct {
	mu          sync.RWMutex
	middlewares []Middleware
	registry    *MiddlewareRegistry
	config      *MiddlewareConfig
	metrics     *MiddlewareMetrics
}

// Middleware defines the core middleware interface compatible with both
// type-safe and legacy handler patterns
type Middleware interface {
	// Apply wraps a handler with middleware functionality
	Apply(next MCPHandler) MCPHandler

	// Name returns the middleware name for debugging and metrics
	Name() string

	// Priority returns the middleware priority for ordering (higher = earlier)
	Priority() int
}

// MCPHandler represents a unified handler interface that can process any MCP request
type MCPHandler interface {
	Handle(ctx context.Context, req MCPRequest) (MCPResponse, error)
}

// MCPRequest represents a unified request interface for all MCP operations
type MCPRequest interface {
	GetMethod() string
	GetID() interface{}
	GetParams() json.RawMessage
	GetContext() context.Context
	WithContext(ctx context.Context) MCPRequest
}

// MCPResponse represents a unified response interface for all MCP operations
type MCPResponse interface {
	GetResult() interface{}
	GetError() *ResponseError
	IsError() bool
}

// MCPHandlerFunc is an adapter to allow regular functions to be used as MCPHandlers
type MCPHandlerFunc func(ctx context.Context, req MCPRequest) (MCPResponse, error)

func (f MCPHandlerFunc) Handle(ctx context.Context, req MCPRequest) (MCPResponse, error) {
	return f(ctx, req)
}

// Core Middleware Components
// ==========================

// LoggingMiddleware provides comprehensive request/response logging
type LoggingMiddleware struct {
	logger         *slog.Logger
	level          slog.Level
	requestFields  []string
	responseFields []string
	sanitizer      func(interface{}) interface{}
}

// LoggingConfig configures the logging middleware
type LoggingConfig struct {
	Logger          *slog.Logger
	Level           slog.Level
	IncludeRequest  bool
	IncludeResponse bool
	IncludeParams   bool
	SanitizeFunc    func(interface{}) interface{}
	RequestFields   []string
	ResponseFields  []string
}

// NewLoggingMiddleware creates a new logging middleware with configuration
func NewLoggingMiddleware(config LoggingConfig) *LoggingMiddleware {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Level == 0 {
		config.Level = slog.LevelInfo
	}

	return &LoggingMiddleware{
		logger:         config.Logger,
		level:          config.Level,
		requestFields:  config.RequestFields,
		responseFields: config.ResponseFields,
		sanitizer:      config.SanitizeFunc,
	}
}

func (m *LoggingMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		start := time.Now()
		requestID := getOrGenerateRequestID(ctx)

		// Log request start
		logFields := []interface{}{
			"request_id", requestID,
			"method", req.GetMethod(),
			"start_time", start,
		}

		if len(m.requestFields) > 0 {
			params := req.GetParams()
			if params != nil && m.sanitizer != nil {
				var paramsMap map[string]interface{}
				if err := json.Unmarshal(params, &paramsMap); err == nil {
					sanitized := m.sanitizer(paramsMap)
					logFields = append(logFields, "params", sanitized)
				}
			}
		}

		m.logger.Log(ctx, m.level, "MCP request started", logFields...)

		// Process request
		resp, err := next.Handle(ctx, req)

		duration := time.Since(start)

		// Log response
		responseFields := []interface{}{
			"request_id", requestID,
			"method", req.GetMethod(),
			"duration", duration,
			"has_error", err != nil || (resp != nil && resp.IsError()),
		}

		if resp != nil && len(m.responseFields) > 0 {
			if result := resp.GetResult(); result != nil && m.sanitizer != nil {
				sanitized := m.sanitizer(result)
				responseFields = append(responseFields, "result", sanitized)
			}
		}

		if err != nil {
			responseFields = append(responseFields, "error", err.Error())
		}

		m.logger.Log(ctx, m.level, "MCP request completed", responseFields...)

		return resp, err
	})
}

func (m *LoggingMiddleware) Name() string {
	return "logging"
}

func (m *LoggingMiddleware) Priority() int {
	return 1000 // High priority for early logging
}

// AuthenticationMiddleware provides token-based authentication
type AuthenticationMiddleware struct {
	provider     OAuthProvider
	skipMethods  map[string]bool
	tokenCache   sync.Map
	cacheTimeout time.Duration
}

// AuthConfig configures the authentication middleware
type AuthConfig struct {
	Provider       OAuthProvider
	SkipMethods    []string
	CacheTimeout   time.Duration
	TokenExtractor func(ctx context.Context, req MCPRequest) (string, error)
}

// NewAuthenticationMiddleware creates a new authentication middleware
func NewAuthenticationMiddleware(config AuthConfig) *AuthenticationMiddleware {
	skipMethods := make(map[string]bool)

	// Default skip methods for MCP protocol
	defaultSkip := []string{"initialize", "initialized", "ping"}
	for _, method := range defaultSkip {
		skipMethods[method] = true
	}

	// Add user-defined skip methods
	for _, method := range config.SkipMethods {
		skipMethods[method] = true
	}

	if config.CacheTimeout == 0 {
		config.CacheTimeout = 5 * time.Minute
	}

	return &AuthenticationMiddleware{
		provider:     config.Provider,
		skipMethods:  skipMethods,
		cacheTimeout: config.CacheTimeout,
	}
}

func (m *AuthenticationMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Check if method should skip authentication
		if m.skipMethods[req.GetMethod()] {
			return next.Handle(ctx, req)
		}

		// Extract token from context or request
		token, err := m.extractToken(ctx, req)
		if err != nil {
			return nil, NewAuthError("Authentication required", ErrorInvalidRequest)
		}

		// Validate token (with caching)
		accessToken, err := m.validateTokenWithCache(ctx, token)
		if err != nil {
			return nil, NewAuthError("Invalid authentication", ErrorInvalidClient)
		}

		// Add authentication context
		authCtx := WithAuthContext(ctx, &AuthContext{
			AccessToken: accessToken,
			ClientID:    accessToken.ClientID,
			Scopes:      accessToken.Scopes,
		})

		return next.Handle(authCtx, req.WithContext(authCtx))
	})
}

func (m *AuthenticationMiddleware) extractToken(ctx context.Context, req MCPRequest) (string, error) {
	// Try to extract from Authorization header in context
	if authHeader, ok := ctx.Value("Authorization").(string); ok {
		return ParseAuthorizationHeader(authHeader)
	}

	// Try to extract from request parameters
	params := req.GetParams()
	if params != nil {
		var paramsMap map[string]interface{}
		if err := json.Unmarshal(params, &paramsMap); err == nil {
			if token, ok := paramsMap["auth_token"].(string); ok {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("no authentication token found")
}

func (m *AuthenticationMiddleware) validateTokenWithCache(ctx context.Context, token string) (*AccessToken, error) {
	// Check cache first
	if cached, ok := m.tokenCache.Load(token); ok {
		if entry, ok := cached.(*cachedToken); ok {
			if time.Now().Before(entry.ExpiresAt) {
				return entry.Token, nil
			}
			// Remove expired entry
			m.tokenCache.Delete(token)
		}
	}

	// Validate with provider
	accessToken, err := m.provider.ValidateAccessToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Cache valid token
	m.tokenCache.Store(token, &cachedToken{
		Token:     accessToken,
		ExpiresAt: time.Now().Add(m.cacheTimeout),
	})

	return accessToken, nil
}

func (m *AuthenticationMiddleware) Name() string {
	return "authentication"
}

func (m *AuthenticationMiddleware) Priority() int {
	return 900 // High priority, but after logging
}

// RateLimitMiddleware provides sophisticated rate limiting
type RateLimitMiddleware struct {
	limiters     sync.Map
	config       RateLimitConfig
	cleanupTimer *time.Timer
}

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
	KeyExtractor      func(ctx context.Context, req MCPRequest) string
	SkipMethods       []string
	WindowSize        time.Duration
	CleanupInterval   time.Duration
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(config RateLimitConfig) *RateLimitMiddleware {
	if config.RequestsPerSecond == 0 {
		config.RequestsPerSecond = 100
	}
	if config.BurstSize == 0 {
		config.BurstSize = 10
	}
	if config.KeyExtractor == nil {
		config.KeyExtractor = func(ctx context.Context, req MCPRequest) string {
			// Default: global rate limiting
			return "global"
		}
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 10 * time.Minute
	}

	rl := &RateLimitMiddleware{
		config: config,
	}

	// Start cleanup timer
	rl.cleanupTimer = time.AfterFunc(config.CleanupInterval, rl.cleanup)

	return rl
}

func (m *RateLimitMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Check if method should skip rate limiting
		for _, method := range m.config.SkipMethods {
			if method == req.GetMethod() {
				return next.Handle(ctx, req)
			}
		}

		// Get rate limit key
		key := m.config.KeyExtractor(ctx, req)

		// Get or create limiter
		limiterInterface, _ := m.limiters.LoadOrStore(key, &rateLimiterEntry{
			limiter:  rate.NewLimiter(rate.Limit(m.config.RequestsPerSecond), m.config.BurstSize),
			lastUsed: time.Now(),
		})

		entry := limiterInterface.(*rateLimiterEntry)
		entry.lastUsed = time.Now()

		// Check rate limit
		if !entry.limiter.Allow() {
			return NewRateLimitError("Rate limit exceeded"), nil
		}

		return next.Handle(ctx, req)
	})
}

func (m *RateLimitMiddleware) cleanup() {
	// Remove unused limiters
	cutoff := time.Now().Add(-m.config.CleanupInterval)

	m.limiters.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*rateLimiterEntry); ok {
			if entry.lastUsed.Before(cutoff) {
				m.limiters.Delete(key)
			}
		}
		return true
	})

	// Schedule next cleanup
	m.cleanupTimer.Reset(m.config.CleanupInterval)
}

func (m *RateLimitMiddleware) Name() string {
	return "rate_limit"
}

func (m *RateLimitMiddleware) Priority() int {
	return 800 // After auth, before business logic
}

// TimeoutMiddleware provides request timeout handling
type TimeoutMiddleware struct {
	timeout         time.Duration
	timeoutResponse func() MCPResponse
}

// NewTimeoutMiddleware creates a new timeout middleware
func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		timeout: timeout,
		timeoutResponse: func() MCPResponse {
			return NewErrorResponse("Request timeout", -32000) // Custom MCP error code
		},
	}
}

func (m *TimeoutMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Create timeout context
		timeoutCtx, cancel := context.WithTimeout(ctx, m.timeout)
		defer cancel()

		// Channel for result
		type result struct {
			resp MCPResponse
			err  error
		}
		resultChan := make(chan result, 1)

		// Execute handler in goroutine
		go func() {
			resp, err := next.Handle(timeoutCtx, req.WithContext(timeoutCtx))
			resultChan <- result{resp: resp, err: err}
		}()

		// Wait for result or timeout
		select {
		case res := <-resultChan:
			return res.resp, res.err
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return m.timeoutResponse(), nil
			}
			return nil, timeoutCtx.Err()
		}
	})
}

func (m *TimeoutMiddleware) Name() string {
	return "timeout"
}

func (m *TimeoutMiddleware) Priority() int {
	return 700 // After rate limiting
}

// RecoveryMiddleware provides panic recovery with structured error responses
type RecoveryMiddleware struct {
	logger       *slog.Logger
	includeStack bool
	recoverFunc  func(interface{}) MCPResponse
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(logger *slog.Logger, includeStack bool) *RecoveryMiddleware {
	if logger == nil {
		logger = slog.Default()
	}

	return &RecoveryMiddleware{
		logger:       logger,
		includeStack: includeStack,
		recoverFunc: func(err interface{}) MCPResponse {
			return NewErrorResponse("Internal server error", -32603) // JSON-RPC internal error
		},
	}
}

func (m *RecoveryMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (resp MCPResponse, err error) {
		defer func() {
			if r := recover(); r != nil {
				requestID := getOrGenerateRequestID(ctx)

				logFields := []interface{}{
					"request_id", requestID,
					"method", req.GetMethod(),
					"panic", r,
				}

				if m.includeStack {
					logFields = append(logFields, "stack", string(debug.Stack()))
				}

				m.logger.Error("Handler panic recovered", logFields...)

				resp = m.recoverFunc(r)
				err = nil // Don't propagate panic as error
			}
		}()

		return next.Handle(ctx, req)
	})
}

func (m *RecoveryMiddleware) Name() string {
	return "recovery"
}

func (m *RecoveryMiddleware) Priority() int {
	return 1100 // Highest priority to catch all panics
}

// MetricsMiddleware provides comprehensive metrics collection
type MetricsMiddleware struct {
	registry   MetricsRegistry
	labels     []string
	buckets    []float64
	activeReqs int64
}

// MetricsRegistry defines the interface for metrics collection
type MetricsRegistry interface {
	RecordRequest(method string, duration time.Duration, statusCode int, labels map[string]string)
	RecordActiveRequests(count int64)
	RecordError(method string, errorType string, labels map[string]string)
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(registry MetricsRegistry) *MetricsMiddleware {
	return &MetricsMiddleware{
		registry: registry,
		labels:   []string{"method", "client_id", "transport"},
		buckets:  []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
	}
}

func (m *MetricsMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		start := time.Now()

		// Increment active requests
		activeCount := atomic.AddInt64(&m.activeReqs, 1)
		m.registry.RecordActiveRequests(activeCount)
		defer func() {
			activeCount := atomic.AddInt64(&m.activeReqs, -1)
			m.registry.RecordActiveRequests(activeCount)
		}()

		// Extract labels
		labels := m.extractLabels(ctx, req)

		// Process request
		resp, err := next.Handle(ctx, req)

		duration := time.Since(start)

		// Determine status code
		statusCode := 200
		if err != nil || (resp != nil && resp.IsError()) {
			statusCode = 500
			if err != nil {
				m.registry.RecordError(req.GetMethod(), "handler_error", labels)
			} else {
				m.registry.RecordError(req.GetMethod(), "response_error", labels)
			}
		}

		// Record metrics
		m.registry.RecordRequest(req.GetMethod(), duration, statusCode, labels)

		return resp, err
	})
}

func (m *MetricsMiddleware) extractLabels(ctx context.Context, req MCPRequest) map[string]string {
	labels := make(map[string]string)
	labels["method"] = req.GetMethod()

	// Extract client ID from auth context if available
	if authCtx := GetAuthContext(ctx); authCtx != nil {
		labels["client_id"] = authCtx.ClientID
	} else {
		labels["client_id"] = "anonymous"
	}

	// Extract transport type from context
	if transport, ok := ctx.Value("transport").(string); ok {
		labels["transport"] = transport
	} else {
		labels["transport"] = "unknown"
	}

	return labels
}

func (m *MetricsMiddleware) Name() string {
	return "metrics"
}

func (m *MetricsMiddleware) Priority() int {
	return 600 // After core middleware, before business logic
}

// Supporting Types and Utilities
// ===============================

// cachedToken represents a cached authentication token
type cachedToken struct {
	Token     *AccessToken
	ExpiresAt time.Time
}

// rateLimiterEntry represents a rate limiter with usage tracking
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

// AuthContext contains authentication information
type AuthContext struct {
	AccessToken *AccessToken
	ClientID    string
	Scopes      []string
	UserInfo    map[string]interface{}
}

// Context keys for middleware data
type contextKey string

const (
	requestIDKey contextKey = "mcp_request_id"
	authCtxKey   contextKey = "mcp_auth_context"
	metricsKey   contextKey = "mcp_metrics"
)

// WithAuthContext adds authentication context to the request context
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authCtxKey, authCtx)
}

// GetAuthContext retrieves authentication context from the request context
func GetAuthContext(ctx context.Context) *AuthContext {
	if authCtx, ok := ctx.Value(authCtxKey).(*AuthContext); ok {
		return authCtx
	}
	return nil
}

// getOrGenerateRequestID gets or generates a unique request ID
func getOrGenerateRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// Error Response Helpers
// ======================

// ResponseError represents an MCP protocol error
type ResponseError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, code int) MCPResponse {
	return &ErrorResponseImpl{
		Error: &ResponseError{
			Code:    code,
			Message: message,
		},
	}
}

// NewAuthError creates a new authentication error
func NewAuthError(message, code string) error {
	return &OAuthError{
		Code:        code,
		Description: message,
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string) MCPResponse {
	return NewErrorResponse(message, -32001) // Custom rate limit error code
}

// ErrorResponseImpl implements the Response interface for errors
type ErrorResponseImpl struct {
	Error *ResponseError `json:"error"`
}

func (r *ErrorResponseImpl) GetResult() interface{} {
	return nil
}

func (r *ErrorResponseImpl) GetError() *ResponseError {
	return r.Error
}

func (r *ErrorResponseImpl) IsError() bool {
	return true
}
