// Package mcp - Middleware Examples and Usage Patterns
//
// This file demonstrates practical usage patterns and examples for the
// comprehensive middleware system, showing how to configure and use
// different middleware components effectively.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Example Usage Patterns
// ======================

// ExampleBasicMiddlewareSetup demonstrates basic middleware configuration
func ExampleBasicMiddlewareSetup() {
	server := NewServer("example", "1.0.0")

	// Add middleware programmatically
	server.Use(NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelInfo,
		IncludeRequest:  true,
		IncludeResponse: true,
	}))

	server.Use(NewRecoveryMiddleware(nil, true))

	server.Use(NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
	}))

	fmt.Println("Basic middleware setup complete")
}

// ExampleConditionalMiddleware demonstrates conditional middleware application
func ExampleConditionalMiddleware() {
	server := NewServer("example", "1.0.0")

	// Apply authentication only for specific clients
	authMiddleware := NewAuthenticationMiddleware(AuthConfig{
		Provider: NewMemoryOAuthProvider(),
	})

	condition := NewClientCondition([]string{"premium-client", "enterprise-client"})
	conditionalAuth := NewConditionalMiddleware(condition, authMiddleware, nil)
	server.Use(conditionalAuth)

	// Apply enhanced logging for debug clients
	debugLogging := NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelDebug,
		IncludeRequest:  true,
		IncludeResponse: true,
	})

	debugCondition := NewClientCondition([]string{"debug-client", "test-client"})
	conditionalLogging := NewConditionalMiddleware(debugCondition, debugLogging, nil)
	server.Use(conditionalLogging)

	fmt.Println("Conditional middleware setup complete")
}

// ExampleCustomMiddleware demonstrates creating custom middleware
func ExampleCustomMiddleware() {
	// Create custom request ID middleware
	requestIDMiddleware := &CustomRequestIDMiddleware{}

	// Create custom audit middleware
	auditMiddleware := &CustomAuditMiddleware{
		logger: slog.Default(),
	}

	// Create custom circuit breaker middleware
	circuitBreakerMiddleware := &CustomCircuitBreakerMiddleware{
		threshold:      5,
		timeout:        30 * time.Second,
		resetAfter:     60 * time.Second,
		failureCounter: make(map[string]int),
	}

	server := NewServer("example", "1.0.0")
	server.Use(requestIDMiddleware)
	server.Use(auditMiddleware)
	server.Use(circuitBreakerMiddleware)

	fmt.Println("Custom middleware setup complete")
}

// Custom Middleware Examples
// ==========================

// CustomRequestIDMiddleware adds unique request IDs
type CustomRequestIDMiddleware struct{}

func (m *CustomRequestIDMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Generate unique request ID
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

		// Add to context
		ctx = context.WithValue(ctx, requestIDKey, requestID)

		// Continue with enriched context
		return next.Handle(ctx, req.WithContext(ctx))
	})
}

func (m *CustomRequestIDMiddleware) Name() string {
	return "request_id"
}

func (m *CustomRequestIDMiddleware) Priority() int {
	return 1050 // Very high priority
}

// CustomAuditMiddleware provides comprehensive audit logging
type CustomAuditMiddleware struct {
	logger *slog.Logger
}

func (m *CustomAuditMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		start := time.Now()

		// Extract audit information
		auditInfo := map[string]interface{}{
			"request_id": getOrGenerateRequestID(ctx),
			"method":     req.Method(),
			"timestamp":  start,
		}

		// Add authentication info if available
		if authCtx := GetAuthContext(ctx); authCtx != nil {
			auditInfo["client_id"] = authCtx.ClientID
			auditInfo["scopes"] = authCtx.Scopes
		}

		// Add request parameters (sanitized)
		if params := req.Params(); params != nil {
			auditInfo["params"] = m.sanitizeParams(params)
		}

		// Execute request
		resp, err := next.Handle(ctx, req)

		// Add response information
		auditInfo["duration"] = time.Since(start)
		auditInfo["success"] = err == nil && (resp == nil || !resp.IsError())

		if err != nil {
			auditInfo["error"] = err.Error()
		}

		// Log audit entry
		m.logger.Info("Audit log", "audit", auditInfo)

		return resp, err
	})
}

func (m *CustomAuditMiddleware) sanitizeParams(params json.RawMessage) interface{} {
	var paramsMap map[string]interface{}
	if err := json.Unmarshal(params, &paramsMap); err != nil {
		return nil
	}

	// Remove sensitive fields
	sensitiveFields := []string{"password", "token", "secret", "key"}
	for _, field := range sensitiveFields {
		if _, exists := paramsMap[field]; exists {
			paramsMap[field] = "[REDACTED]"
		}
	}

	return paramsMap
}

func (m *CustomAuditMiddleware) Name() string {
	return "audit"
}

func (m *CustomAuditMiddleware) Priority() int {
	return 950
}

// CustomCircuitBreakerMiddleware implements circuit breaker pattern
type CustomCircuitBreakerMiddleware struct {
	threshold      int
	timeout        time.Duration
	resetAfter     time.Duration
	failureCounter map[string]int
	lastFailure    map[string]time.Time
	mu             sync.RWMutex
}

func (m *CustomCircuitBreakerMiddleware) Apply(next MCPHandler) MCPHandler {
	if m.lastFailure == nil {
		m.lastFailure = make(map[string]time.Time)
	}

	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		method := req.Method()

		m.mu.RLock()
		failures := m.failureCounter[method]
		lastFail := m.lastFailure[method]
		m.mu.RUnlock()

		// Check if circuit is open
		if failures >= m.threshold {
			if time.Since(lastFail) < m.resetAfter {
				return NewErrorResponse("Circuit breaker open", -32001), nil
			} else {
				// Reset circuit
				m.mu.Lock()
				m.failureCounter[method] = 0
				m.mu.Unlock()
			}
		}

		// Execute request
		resp, err := next.Handle(ctx, req)

		// Track failures
		if err != nil || (resp != nil && resp.IsError()) {
			m.mu.Lock()
			m.failureCounter[method]++
			m.lastFailure[method] = time.Now()
			m.mu.Unlock()
		} else {
			// Reset on success
			m.mu.Lock()
			m.failureCounter[method] = 0
			m.mu.Unlock()
		}

		return resp, err
	})
}

func (m *CustomCircuitBreakerMiddleware) Name() string {
	return "circuit_breaker"
}

func (m *CustomCircuitBreakerMiddleware) Priority() int {
	return 700
}

// ExampleMiddlewareGroups demonstrates middleware grouping
func ExampleMiddlewareGroups() {
	registry := NewMiddlewareRegistry(nil)

	// Create security middleware group
	securityGroup := NewMiddlewareGroup("security", registry)
	securityGroup.AddByName("authentication", AuthConfig{
		Provider: NewMemoryOAuthProvider(),
	})
	securityGroup.AddByName("rate_limit", RateLimitConfig{
		RequestsPerSecond: 50,
		BurstSize:         10,
	})

	// Create observability middleware group
	observabilityGroup := NewMiddlewareGroup("observability", registry)
	observabilityGroup.AddByName("logging", LoggingConfig{
		Level:           slog.LevelInfo,
		IncludeRequest:  true,
		IncludeResponse: false,
	})
	observabilityGroup.AddByName("metrics", MetricsConfig{
		Registry: &InMemoryMetricsRegistry{},
	})

	// Create reliability middleware group
	reliabilityGroup := NewMiddlewareGroup("reliability", registry)
	reliabilityGroup.AddByName("recovery", RecoveryConfig{
		IncludeStack: true,
	})
	reliabilityGroup.AddByName("timeout", TimeoutConfig{
		Timeout: 30 * time.Second,
	})

	fmt.Println("Middleware groups applied successfully")
}

// Performance Optimization Examples
// =================================

// ExampleHighPerformanceSetup demonstrates optimized middleware for high performance
func ExampleHighPerformanceSetup() {
	server := NewServer("example", "1.0.0")

	// Minimal logging for production
	server.Use(NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelWarn,
		IncludeRequest:  false,
		IncludeResponse: false,
	}))

	// Essential recovery only
	server.Use(NewRecoveryMiddleware(nil, false))

	// Optimized rate limiting
	server.Use(NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
		CleanupInterval:   30 * time.Minute,
	}))

	// Efficient caching
	server.Use(NewCachingMiddleware(CachingConfig{
		TTL:         5 * time.Minute,
		MaxSize:     500 * 1024 * 1024, // 500MB
		KeyStrategy: "default",
	}))

	fmt.Println("High-performance middleware setup complete")
}

// ExampleDevelopmentSetup demonstrates middleware optimized for development
func ExampleDevelopmentSetup() {
	server := NewServer("example", "1.0.0")

	// Verbose logging for development
	server.Use(NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelDebug,
		IncludeRequest:  true,
		IncludeResponse: true,
	}))

	// Recovery with stack traces
	server.Use(NewRecoveryMiddleware(nil, true))

	// Lenient rate limiting
	server.Use(NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
	}))

	// Short cache TTL for development
	server.Use(NewCachingMiddleware(CachingConfig{
		TTL:         30 * time.Second,
		MaxSize:     10 * 1024 * 1024, // 10MB
		KeyStrategy: "default",
	}))

	fmt.Println("Development middleware setup complete")
}
