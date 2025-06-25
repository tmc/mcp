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
	// Create enhanced server
	server := NewEnhancedServer()
	
	// Add middleware programmatically
	server.UseMiddleware(NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelInfo,
		IncludeRequest:  true,
		IncludeResponse: true,
	}))
	
	server.UseMiddleware(NewRecoveryMiddleware(nil, true))
	
	server.UseMiddleware(NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
	}))
	
	fmt.Println("Basic middleware setup complete")
}

// ExampleConfigurationDrivenSetup demonstrates configuration-driven middleware
func ExampleConfigurationDrivenSetup() {
	// Load configuration from JSON
	configJSON := `{
		"enabled": true,
		"default_timeout": "30s",
		"max_concurrency": 100,
		"logging": {
			"level": "info",
			"include_request": true,
			"include_response": false
		},
		"authentication": {
			"skip_methods": ["initialize", "ping"],
			"cache_timeout": "5m"
		},
		"rate_limit": {
			"requests_per_second": 50,
			"burst_size": 10
		},
		"timeout": {
			"timeout": "30s"
		},
		"recovery": {
			"include_stack": true
		},
		"metrics": {
			"labels": ["method", "client_id", "transport"]
		}
	}`
	
	// Parse configuration
	config, err := LoadMiddlewareConfigFromJSON([]byte(configJSON))
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}
	
	// Create server with configuration
	server := NewEnhancedServer()
	serverConfig := &ServerMiddlewareConfig{
		GlobalConfig: config,
	}
	
	err = server.SetMiddlewareConfig(serverConfig)
	if err != nil {
		fmt.Printf("Failed to set middleware config: %v\n", err)
		return
	}
	
	fmt.Println("Configuration-driven setup complete")
}

// ExampleTransportSpecificMiddleware demonstrates transport-specific configuration
func ExampleTransportSpecificMiddleware() {
	server := NewEnhancedServer()
	
	// HTTP-specific middleware (CORS, compression)
	server.UseMiddlewareForTransport("http", NewCORSMiddleware(CORSConfig{
		AllowOrigins: []string{"https://example.com", "https://app.example.com"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:       86400,
	}))
	
	server.UseMiddlewareForTransport("http", NewCompressionMiddleware(CompressionConfig{
		Algorithms: []string{"gzip"},
		MinSize:    1024,
		Level:      6,
	}))
	
	// WebSocket-specific middleware
	server.UseMiddlewareForTransport("websocket", NewCompressionMiddleware(CompressionConfig{
		Algorithms: []string{"gzip"},
		MinSize:    512,
		Level:      1, // Faster compression for real-time
	}))
	
	// Stdio-specific optimizations
	server.UseMiddlewareForTransport("stdio", NewLoggingMiddleware(LoggingConfig{
		Level: slog.LevelWarn, // Minimal logging for stdio
	}))
	
	fmt.Println("Transport-specific middleware setup complete")
}

// ExampleMethodSpecificMiddleware demonstrates method-specific configuration
func ExampleMethodSpecificMiddleware() {
	server := NewEnhancedServer()
	
	// Expensive operations get longer timeouts
	server.UseMiddlewareForMethod("tools/call", NewTimeoutMiddleware(60*time.Second))
	
	// Resource operations can be cached
	server.UseMiddlewareForMethod("resources/read", NewCachingMiddleware(CachingConfig{
		TTL:         10 * time.Minute,
		MaxSize:     100 * 1024 * 1024, // 100MB
		KeyStrategy: "default",
	}))
	
	// Prompts can be cached longer
	server.UseMiddlewareForMethod("prompts/get", NewCachingMiddleware(CachingConfig{
		TTL:         1 * time.Hour,
		MaxSize:     50 * 1024 * 1024, // 50MB
		KeyStrategy: "default",
	}))
	
	// Critical operations get stricter rate limiting
	server.UseMiddlewareForMethod("tools/call", NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         3,
	}))
	
	fmt.Println("Method-specific middleware setup complete")
}

// ExampleConditionalMiddleware demonstrates conditional middleware application
func ExampleConditionalMiddleware() {
	server := NewEnhancedServer()
	
	// Apply authentication only for specific clients
	authMiddleware := NewAuthenticationMiddleware(AuthConfig{
		Provider: NewMemoryOAuthProvider(),
	})
	
	condition := NewClientCondition([]string{"premium-client", "enterprise-client"})
	conditionalAuth := NewConditionalMiddleware(condition, authMiddleware, nil)
	server.UseMiddleware(conditionalAuth)
	
	// Apply enhanced logging for debug clients
	debugLogging := NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelDebug,
		IncludeRequest:  true,
		IncludeResponse: true,
	})
	
	debugCondition := NewClientCondition([]string{"debug-client", "test-client"})
	conditionalLogging := NewConditionalMiddleware(debugCondition, debugLogging, nil)
	server.UseMiddleware(conditionalLogging)
	
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
	
	server := NewEnhancedServer()
	server.UseMiddleware(requestIDMiddleware)
	server.UseMiddleware(auditMiddleware)
	server.UseMiddleware(circuitBreakerMiddleware)
	
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
			"method":     req.GetMethod(),
			"timestamp":  start,
		}
		
		// Add authentication info if available
		if authCtx := GetAuthContext(ctx); authCtx != nil {
			auditInfo["client_id"] = authCtx.ClientID
			auditInfo["scopes"] = authCtx.Scopes
		}
		
		// Add request parameters (sanitized)
		if params := req.GetParams(); params != nil {
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
		method := req.GetMethod()
		
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

// Advanced Configuration Examples
// ===============================

// ExampleComplexConfiguration demonstrates a complex real-world configuration
func ExampleComplexConfiguration() {
	configJSON := `{
		"enabled": true,
		"default_timeout": "30s",
		"max_concurrency": 1000,
		"logging": {
			"level": "info",
			"include_request": true,
			"include_response": false,
			"request_fields": ["method", "client_id"],
			"response_fields": ["status", "duration"]
		},
		"authentication": {
			"skip_methods": ["initialize", "initialized", "ping", "capabilities"],
			"cache_timeout": "10m"
		},
		"rate_limit": {
			"requests_per_second": 100,
			"burst_size": 20,
			"window_size": "1m",
			"cleanup_interval": "5m"
		},
		"timeout": {
			"timeout": "30s"
		},
		"recovery": {
			"include_stack": false
		},
		"metrics": {
			"labels": ["method", "client_id", "transport", "status"]
		},
		"cors": {
			"allow_origins": ["https://app.example.com"],
			"allow_methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
			"allow_headers": ["Content-Type", "Authorization", "X-Request-ID"],
			"max_age": 86400
		},
		"compression": {
			"algorithms": ["gzip"],
			"min_size": 1024,
			"level": 6
		},
		"validation": {
			"strict_mode": true,
			"schemas": {}
		},
		"caching": {
			"ttl": "5m",
			"max_size": 104857600,
			"key_strategy": "default"
		},
		"transport_configs": {
			"http": {
				"enabled_only": ["cors", "compression"],
				"custom_config": {
					"cors": {
						"allow_origins": ["*"]
					}
				}
			},
			"websocket": {
				"disabled_only": ["cors"],
				"custom_config": {
					"compression": {
						"level": 1
					}
				}
			},
			"stdio": {
				"disabled_only": ["cors", "compression"],
				"custom_config": {
					"logging": {
						"level": "error"
					}
				}
			}
		},
		"method_configs": {
			"tools/call": {
				"custom_timeout": "60s",
				"custom_config": {
					"rate_limit": {
						"requests_per_second": 10
					}
				}
			},
			"resources/read": {
				"enabled_only": ["caching", "compression"],
				"custom_config": {
					"caching": {
						"ttl": "30m"
					}
				}
			}
		},
		"conditional_middleware": [
			{
				"name": "authentication",
				"condition": "client:premium-client,enterprise-client",
				"priority": 900
			},
			{
				"name": "logging",
				"condition": "method:tools/call,resources/read",
				"config": {
					"level": "debug",
					"include_response": true
				}
			}
		]
	}`
	
	// Load and apply configuration
	config, err := LoadMiddlewareConfigFromJSON([]byte(configJSON))
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}
	
	server := NewEnhancedServer()
	serverConfig := &ServerMiddlewareConfig{
		GlobalConfig: config,
	}
	
	err = server.SetMiddlewareConfig(serverConfig)
	if err != nil {
		fmt.Printf("Failed to set middleware config: %v\n", err)
		return
	}
	
	fmt.Println("Complex configuration applied successfully")
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
	
	// Apply groups to handler
	// mockHandler := NewMockHandler()
	
	// Apply in order: security -> observability -> reliability
	// handler := securityGroup.Apply(mockHandler)
	// handler = observabilityGroup.Apply(handler)
	// handler = reliabilityGroup.Apply(handler)
	
	fmt.Println("Middleware groups applied successfully")
}

// Performance Optimization Examples
// =================================

// ExampleHighPerformanceSetup demonstrates optimized middleware for high performance
func ExampleHighPerformanceSetup() {
	server := NewEnhancedServer()
	
	// Minimal logging for production
	server.UseMiddleware(NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelWarn,
		IncludeRequest:  false,
		IncludeResponse: false,
	}))
	
	// Essential recovery only
	server.UseMiddleware(NewRecoveryMiddleware(nil, false))
	
	// Optimized rate limiting
	server.UseMiddleware(NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
		CleanupInterval:   30 * time.Minute,
	}))
	
	// Efficient caching
	server.UseMiddleware(NewCachingMiddleware(CachingConfig{
		TTL:         5 * time.Minute,
		MaxSize:     500 * 1024 * 1024, // 500MB
		KeyStrategy: "default",
	}))
	
	fmt.Println("High-performance middleware setup complete")
}

// ExampleDevelopmentSetup demonstrates middleware optimized for development
func ExampleDevelopmentSetup() {
	server := NewEnhancedServer()
	
	// Verbose logging for development
	server.UseMiddleware(NewLoggingMiddleware(LoggingConfig{
		Level:           slog.LevelDebug,
		IncludeRequest:  true,
		IncludeResponse: true,
	}))
	
	// Recovery with stack traces
	server.UseMiddleware(NewRecoveryMiddleware(nil, true))
	
	// Lenient rate limiting
	server.UseMiddleware(NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
	}))
	
	// Short cache TTL for development
	server.UseMiddleware(NewCachingMiddleware(CachingConfig{
		TTL:         30 * time.Second,
		MaxSize:     10 * 1024 * 1024, // 10MB
		KeyStrategy: "default",
	}))
	
	fmt.Println("Development middleware setup complete")
}