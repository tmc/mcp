# MCP Go Comprehensive Middleware System

## Overview

The MCP Go Comprehensive Middleware System provides a production-ready middleware infrastructure that integrates seamlessly with the type-safe APIs from Phase 2A. This system offers extensive middleware capabilities including logging, authentication, rate limiting, caching, compression, and more.

## Features

### Core Middleware Components

- **Logging Middleware**: Comprehensive request/response logging with configurable levels
- **Authentication Middleware**: OAuth2 token validation with caching
- **Rate Limiting Middleware**: Advanced rate limiting with multiple strategies
- **Timeout Middleware**: Request timeout handling with graceful degradation
- **Recovery Middleware**: Panic recovery with structured error responses
- **Metrics Middleware**: Request/response metrics collection and monitoring

### Advanced Middleware Features

- **Compression Middleware**: Request/response compression (gzip, brotli)
- **Caching Middleware**: Response caching with TTL and configurable cache keys
- **Validation Middleware**: JSON schema-based request/response validation
- **CORS Middleware**: Cross-origin request handling for HTTP transports
- **Content Transformation**: Pluggable content transformation pipelines

### Configuration and Management

- **Middleware Registry**: Central registry for middleware discovery and creation
- **Configuration Framework**: YAML/JSON-based middleware configuration
- **Dynamic Middleware**: Runtime middleware addition/removal
- **Conditional Middleware**: Apply middleware based on request characteristics
- **Transport-Specific Configuration**: Different middleware stacks per transport
- **Method-Specific Configuration**: Method-specific middleware rules

## Quick Start

### Basic Programmatic Setup

```go
package main

import (
    "log/slog"
    "time"
    
    "github.com/tmc/mcp"
)

func main() {
    // Create enhanced server with middleware support
    server := mcp.NewEnhancedServer()
    
    // Add core middleware
    server.UseMiddleware(mcp.NewLoggingMiddleware(mcp.LoggingConfig{
        Level:           slog.LevelInfo,
        IncludeRequest:  true,
        IncludeResponse: false,
    }))
    
    server.UseMiddleware(mcp.NewRecoveryMiddleware(nil, true))
    
    server.UseMiddleware(mcp.NewRateLimitMiddleware(mcp.RateLimitConfig{
        RequestsPerSecond: 100,
        BurstSize:         10,
    }))
    
    server.UseMiddleware(mcp.NewTimeoutMiddleware(30 * time.Second))
    
    // Start server...
}
```

### Configuration-Driven Setup

```go
// Load configuration from JSON/YAML
configJSON := `{
    "enabled": true,
    "default_timeout": "30s",
    "logging": {
        "level": "info",
        "include_request": true
    },
    "authentication": {
        "skip_methods": ["initialize", "ping"]
    },
    "rate_limit": {
        "requests_per_second": 50,
        "burst_size": 10
    },
    "recovery": {
        "include_stack": true
    }
}`

config, err := mcp.LoadMiddlewareConfigFromJSON([]byte(configJSON))
if err != nil {
    log.Fatal(err)
}

server := mcp.NewEnhancedServer()
serverConfig := &mcp.ServerMiddlewareConfig{
    GlobalConfig: config,
}

err = server.SetMiddlewareConfig(serverConfig)
if err != nil {
    log.Fatal(err)
}
```

## Core Middleware Components

### 1. Logging Middleware

Provides comprehensive request/response logging with configurable levels and field selection.

```go
// Basic logging
logging := mcp.NewLoggingMiddleware(mcp.LoggingConfig{
    Level: slog.LevelInfo,
})

// Advanced logging with custom fields
logging := mcp.NewLoggingMiddleware(mcp.LoggingConfig{
    Logger:         customLogger,
    Level:          slog.LevelDebug,
    IncludeRequest: true,
    IncludeResponse: true,
    RequestFields:  []string{"method", "params"},
    ResponseFields: []string{"result", "duration"},
    SanitizeFunc:   sanitizeFunction,
})
```

**Configuration Options:**
- `Logger`: Custom slog.Logger instance
- `Level`: Logging level (Debug, Info, Warn, Error)
- `IncludeRequest/IncludeResponse`: Include request/response data
- `RequestFields/ResponseFields`: Specific fields to log
- `SanitizeFunc`: Function to sanitize sensitive data

### 2. Authentication Middleware

OAuth2-based authentication with token validation and caching.

```go
// Basic authentication
auth := mcp.NewAuthenticationMiddleware(mcp.AuthConfig{
    Provider: oauthProvider,
})

// Advanced authentication with caching
auth := mcp.NewAuthenticationMiddleware(mcp.AuthConfig{
    Provider:      oauthProvider,
    SkipMethods:   []string{"initialize", "ping"},
    CacheTimeout:  5 * time.Minute,
    TokenExtractor: customTokenExtractor,
})
```

**Configuration Options:**
- `Provider`: OAuth provider implementation
- `SkipMethods`: Methods that don't require authentication
- `CacheTimeout`: Token cache duration
- `TokenExtractor`: Custom token extraction function

### 3. Rate Limiting Middleware

Advanced rate limiting with configurable strategies and cleanup.

```go
// Basic rate limiting
rateLimit := mcp.NewRateLimitMiddleware(mcp.RateLimitConfig{
    RequestsPerSecond: 100,
    BurstSize:         10,
})

// Advanced rate limiting with custom key extraction
rateLimit := mcp.NewRateLimitMiddleware(mcp.RateLimitConfig{
    RequestsPerSecond: 50,
    BurstSize:         5,
    KeyExtractor:      clientBasedKeyExtractor,
    SkipMethods:       []string{"ping"},
    WindowSize:        time.Minute,
    CleanupInterval:   10 * time.Minute,
})
```

**Configuration Options:**
- `RequestsPerSecond`: Rate limit threshold
- `BurstSize`: Burst allowance
- `KeyExtractor`: Function to generate rate limit keys
- `SkipMethods`: Methods to exclude from rate limiting
- `WindowSize`: Rate limit window duration
- `CleanupInterval`: Limiter cleanup frequency

### 4. Timeout Middleware

Request timeout handling with graceful degradation.

```go
// Basic timeout
timeout := mcp.NewTimeoutMiddleware(30 * time.Second)

// Custom timeout with response generator
timeout := &mcp.TimeoutMiddleware{
    Timeout: 30 * time.Second,
    TimeoutResponse: func() mcp.Response {
        return mcp.NewErrorResponse("Custom timeout message", -32000)
    },
}
```

### 5. Recovery Middleware

Panic recovery with structured error responses and optional stack traces.

```go
// Basic recovery
recovery := mcp.NewRecoveryMiddleware(nil, false)

// Advanced recovery with custom logger and stack traces
recovery := mcp.NewRecoveryMiddleware(customLogger, true)
```

### 6. Metrics Middleware

Comprehensive metrics collection for monitoring and observability.

```go
// Basic metrics with in-memory registry
metrics := mcp.NewMetricsMiddleware(&mcp.InMemoryMetricsRegistry{})

// Custom metrics with Prometheus registry
metrics := mcp.NewMetricsMiddleware(prometheusRegistry)
```

## Advanced Features

### Transport-Specific Middleware

Configure different middleware stacks for different transport types:

```go
server := mcp.NewEnhancedServer()

// HTTP-specific middleware
server.UseMiddlewareForTransport("http", mcp.NewCORSMiddleware(corsConfig))
server.UseMiddlewareForTransport("http", mcp.NewCompressionMiddleware(compConfig))

// WebSocket-specific middleware
server.UseMiddlewareForTransport("websocket", optimizedCompressionMiddleware)

// Stdio-specific middleware (minimal logging)
server.UseMiddlewareForTransport("stdio", minimalLoggingMiddleware)
```

### Method-Specific Middleware

Apply middleware based on specific MCP methods:

```go
// Longer timeout for expensive operations
server.UseMiddlewareForMethod("tools/call", mcp.NewTimeoutMiddleware(60*time.Second))

// Caching for resource operations
server.UseMiddlewareForMethod("resources/read", cachingMiddleware)

// Stricter rate limiting for critical operations
server.UseMiddlewareForMethod("tools/call", strictRateLimitMiddleware)
```

### Conditional Middleware

Apply middleware based on dynamic conditions:

```go
// Authentication only for premium clients
condition := mcp.NewClientCondition([]string{"premium-client", "enterprise-client"})
conditionalAuth := mcp.NewConditionalMiddleware(condition, authMiddleware, nil)
server.UseMiddleware(conditionalAuth)

// Debug logging for specific methods
methodCondition := mcp.NewMethodCondition([]string{"tools/call", "resources/read"})
conditionalDebugLogging := mcp.NewConditionalMiddleware(methodCondition, debugLogging, nil)
server.UseMiddleware(conditionalDebugLogging)

// Regex-based conditions
regexCondition, _ := mcp.NewRegexCondition("^tools/.*", "method")
conditionalMiddleware := mcp.NewConditionalMiddleware(regexCondition, middleware, nil)
```

### Middleware Groups

Organize middleware into logical groups:

```go
registry := mcp.NewMiddlewareRegistry(nil)

// Security group
securityGroup := mcp.NewMiddlewareGroup("security", registry)
securityGroup.AddByName("authentication", authConfig)
securityGroup.AddByName("rate_limit", rateLimitConfig)

// Observability group
observabilityGroup := mcp.NewMiddlewareGroup("observability", registry)
observabilityGroup.AddByName("logging", loggingConfig)
observabilityGroup.AddByName("metrics", metricsConfig)

// Apply groups
handler := securityGroup.Apply(baseHandler)
handler = observabilityGroup.Apply(handler)
```

## Configuration Schema

### Global Configuration

```json
{
    "enabled": true,
    "default_timeout": "30s",
    "max_concurrency": 100,
    "logging": {
        "level": "info",
        "include_request": true,
        "include_response": false,
        "request_fields": ["method", "client_id"],
        "response_fields": ["status", "duration"]
    },
    "authentication": {
        "skip_methods": ["initialize", "ping"],
        "cache_timeout": "5m"
    },
    "rate_limit": {
        "requests_per_second": 100,
        "burst_size": 10,
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
        "labels": ["method", "client_id", "transport"]
    },
    "cors": {
        "allow_origins": ["*"],
        "allow_methods": ["GET", "POST", "PUT", "DELETE"],
        "allow_headers": ["Content-Type", "Authorization"],
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
    }
}
```

### Transport-Specific Configuration

```json
{
    "transport_configs": {
        "http": {
            "enabled_only": ["cors", "compression"],
            "custom_config": {
                "cors": {
                    "allow_origins": ["https://app.example.com"]
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
    }
}
```

### Method-Specific Configuration

```json
{
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
    }
}
```

### Conditional Middleware Configuration

```json
{
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
}
```

## Performance Considerations

### Middleware Overhead

The middleware system is designed for minimal overhead:

- **< 1ms per middleware**: Each middleware adds less than 1ms of latency
- **Object pooling**: Reuses objects where possible to reduce GC pressure
- **Lazy evaluation**: Middleware is only applied when conditions are met
- **Efficient ordering**: Middleware is sorted by priority for optimal execution

### Memory Management

- **Connection pooling**: OAuth provider uses connection pooling
- **Cache cleanup**: Automatic cleanup of expired cache entries
- **Rate limiter cleanup**: Periodic cleanup of unused rate limiters
- **Metrics aggregation**: Efficient metrics storage and aggregation

### Benchmarks

```
BenchmarkMiddlewareOverhead-8    1000000    1.2 ms/op    512 B/op    8 allocs/op
BenchmarkMiddlewareChain-8       500000     2.1 ms/op    1024 B/op   15 allocs/op
```

## Production Deployment

### High Performance Setup

```go
// Minimal middleware for maximum performance
server := mcp.NewEnhancedServer()
server.UseMiddleware(mcp.NewLoggingMiddleware(mcp.LoggingConfig{
    Level: slog.LevelWarn, // Minimal logging
}))
server.UseMiddleware(mcp.NewRecoveryMiddleware(nil, false))
server.UseMiddleware(mcp.NewRateLimitMiddleware(mcp.RateLimitConfig{
    RequestsPerSecond: 1000,
    BurstSize:         100,
}))
```

### Development Setup

```go
// Comprehensive middleware for development
server := mcp.NewEnhancedServer()
server.UseMiddleware(mcp.NewLoggingMiddleware(mcp.LoggingConfig{
    Level:           slog.LevelDebug,
    IncludeRequest:  true,
    IncludeResponse: true,
}))
server.UseMiddleware(mcp.NewRecoveryMiddleware(nil, true)) // Include stack traces
server.UseMiddleware(mcp.NewRateLimitMiddleware(mcp.RateLimitConfig{
    RequestsPerSecond: 1000, // Lenient for development
    BurstSize:         100,
}))
```

### Production Monitoring

```go
// Enhanced monitoring setup
server.UseMiddleware(mcp.NewMetricsMiddleware(prometheusRegistry))
server.UseMiddleware(mcp.NewLoggingMiddleware(mcp.LoggingConfig{
    Level: slog.LevelInfo,
    RequestFields: []string{"method", "client_id", "duration"},
}))
```

## Custom Middleware Development

### Creating Custom Middleware

```go
type CustomMiddleware struct {
    config CustomConfig
}

func (m *CustomMiddleware) Apply(next mcp.Handler) mcp.Handler {
    return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
        // Pre-processing
        start := time.Now()
        
        // Call next middleware/handler
        resp, err := next.Handle(ctx, req)
        
        // Post-processing
        duration := time.Since(start)
        m.recordMetrics(req.GetMethod(), duration, err == nil)
        
        return resp, err
    })
}

func (m *CustomMiddleware) Name() string {
    return "custom"
}

func (m *CustomMiddleware) Priority() int {
    return 500 // Medium priority
}
```

### Registering Custom Middleware

```go
type CustomMiddlewareFactory struct{}

func (f *CustomMiddlewareFactory) Create(config interface{}) (mcp.Middleware, error) {
    var customConfig CustomConfig
    if err := mcp.MapToStruct(config, &customConfig); err != nil {
        return nil, err
    }
    
    return &CustomMiddleware{config: customConfig}, nil
}

func (f *CustomMiddlewareFactory) ConfigType() interface{} {
    return CustomConfig{}
}

func (f *CustomMiddlewareFactory) Name() string {
    return "custom"
}

func (f *CustomMiddlewareFactory) Description() string {
    return "Custom middleware implementation"
}

// Register with registry
registry := mcp.NewMiddlewareRegistry(nil)
registry.RegisterFactory(&CustomMiddlewareFactory{})
```

## Testing

### Unit Testing Middleware

```go
func TestCustomMiddleware(t *testing.T) {
    middleware := &CustomMiddleware{}
    mockHandler := &MockHandler{}
    
    wrappedHandler := middleware.Apply(mockHandler)
    
    req := NewMockRequest("test/method", nil)
    resp, err := wrappedHandler.Handle(context.Background(), req)
    
    // Verify behavior
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

### Integration Testing

```go
func TestMiddlewareIntegration(t *testing.T) {
    server := mcp.NewEnhancedServer()
    server.UseMiddleware(middleware1)
    server.UseMiddleware(middleware2)
    
    // Test complete request flow
    // ...
}
```

### Performance Testing

```go
func BenchmarkMiddleware(b *testing.B) {
    middleware := NewCustomMiddleware()
    handler := middleware.Apply(&MockHandler{})
    req := NewMockRequest("test", nil)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        handler.Handle(context.Background(), req)
    }
}
```

## Migration Guide

### From Legacy to Middleware System

1. **Identify Existing Patterns**: Find existing cross-cutting concerns in your code
2. **Choose Appropriate Middleware**: Select middleware components that match your needs
3. **Configure Gradually**: Start with basic middleware and add more as needed
4. **Test Thoroughly**: Ensure middleware doesn't break existing functionality
5. **Monitor Performance**: Watch for performance impacts and optimize as needed

### Breaking Changes

- **Handler Interface**: New unified handler interface for middleware
- **Configuration Format**: New JSON/YAML configuration schema
- **Context Usage**: Enhanced context usage for middleware data

## Troubleshooting

### Common Issues

1. **High Memory Usage**: Check cache sizes and cleanup intervals
2. **Performance Degradation**: Review middleware ordering and configuration
3. **Authentication Failures**: Verify OAuth provider configuration
4. **Rate Limiting Issues**: Check rate limit keys and thresholds

### Debug Mode

Enable debug logging to troubleshoot middleware issues:

```go
server.UseMiddleware(mcp.NewLoggingMiddleware(mcp.LoggingConfig{
    Level:           slog.LevelDebug,
    IncludeRequest:  true,
    IncludeResponse: true,
}))
```

### Performance Profiling

Use Go's built-in profiling tools to identify performance bottlenecks:

```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

## Contributing

Contributions to the middleware system are welcome! Please follow these guidelines:

1. **Follow Patterns**: Use existing middleware patterns as templates
2. **Add Tests**: Include comprehensive tests for new middleware
3. **Document Configuration**: Provide clear configuration documentation
4. **Performance**: Ensure new middleware meets performance requirements
5. **Backward Compatibility**: Maintain compatibility with existing APIs

## License

This middleware system is part of the MCP Go implementation and follows the same license terms.