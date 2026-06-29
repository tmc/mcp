// Package mcp - Middleware Performance Benchmarks
//
// This file contains comprehensive performance benchmarks for the MCP Go middleware system.
// It measures middleware chain execution overhead, rate limiting performance, logging impact,
// and authentication middleware with various cache scenarios.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Benchmark configuration for middleware operations
var (
	middlewareChainLengths = []int{1, 3, 5, 10, 20}
	requestSizes           = []int{100, 1024, 10240}
	rateLimits             = []int{10, 100, 1000, 10000}
	cachingScenarios       = []string{"no-cache", "cache-hit", "cache-miss"}
)

// =============================================================================
// Middleware Chain Execution Overhead
// =============================================================================

func BenchmarkMiddlewareChainOverhead(b *testing.B) {
	for _, chainLength := range middlewareChainLengths {
		b.Run(fmt.Sprintf("ChainLength_%d", chainLength), func(b *testing.B) {
			benchmarkMiddlewareChainOverhead(b, chainLength)
		})
	}
}

func benchmarkMiddlewareChainOverhead(b *testing.B, chainLength int) {
	// Create base handler that does minimal work
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	// Build middleware chain
	handler := MCPHandler(baseHandler)
	for i := 0; i < chainLength; i++ {
		// Use lightweight middleware for overhead measurement
		middleware := NewLoggingMiddleware(LoggingConfig{
			Level: slog.LevelError, // Minimal logging
		})
		handler = middleware.Apply(handler)
	}

	// Create test request
	req := &MockRequestForBenchmark{
		method: "test/method",
		id:     "test-id",
		params: json.RawMessage(`{"test": "data"}`),
		ctx:    context.Background(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Errorf("Handler failed: %v", err)
		}
	}
}

func BenchmarkMiddlewareChain_DifferentTypes(b *testing.B) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	// Test different middleware combinations
	scenarios := map[string][]Middleware{
		"LoggingOnly": {
			NewLoggingMiddleware(LoggingConfig{Level: slog.LevelError}),
		},
		"RateLimitOnly": {
			NewRateLimitMiddleware(RateLimitConfig{
				RequestsPerSecond: 1000,
				BurstSize:         10,
			}),
		},
		"TimeoutOnly": {
			NewTimeoutMiddleware(30 * time.Second),
		},
		"Full": {
			NewRecoveryMiddleware(slog.Default(), false),
			NewLoggingMiddleware(LoggingConfig{Level: slog.LevelError}),
			NewRateLimitMiddleware(RateLimitConfig{
				RequestsPerSecond: 1000,
				BurstSize:         10,
			}),
			NewTimeoutMiddleware(30 * time.Second),
		},
	}

	for name, middlewares := range scenarios {
		b.Run(name, func(b *testing.B) {
			handler := MCPHandler(baseHandler)
			for _, middleware := range middlewares {
				handler = middleware.Apply(handler)
			}

			req := &MockRequestForBenchmark{
				method: "test/method",
				id:     "test-id",
				params: json.RawMessage(`{"test": "data"}`),
				ctx:    context.Background(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := handler.Handle(context.Background(), req)
				if err != nil {
					b.Errorf("Handler failed: %v", err)
				}
			}
		})
	}
}

// =============================================================================
// Rate Limiting Performance
// =============================================================================

func BenchmarkRateLimiting_UnderLoad(b *testing.B) {
	for _, limit := range rateLimits {
		b.Run(fmt.Sprintf("Limit_%d", limit), func(b *testing.B) {
			benchmarkRateLimitingUnderLoad(b, limit)
		})
	}
}

func benchmarkRateLimitingUnderLoad(b *testing.B, requestsPerSecond int) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	// Create rate limiting middleware
	rateLimitMiddleware := NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: requestsPerSecond,
		BurstSize:         requestsPerSecond / 10, // 10% burst
		KeyExtractor: func(ctx context.Context, req MCPRequest) string {
			return "global" // Global rate limiting for this benchmark
		},
	})

	handler := rateLimitMiddleware.Apply(baseHandler)

	req := &MockRequestForBenchmark{
		method: "test/method",
		id:     "test-id",
		params: json.RawMessage(`{"test": "data"}`),
		ctx:    context.Background(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Errorf("Handler failed: %v", err)
		}
	}

	// Report rate limiting effectiveness
	actualRate := float64(b.N) / b.Elapsed().Seconds()
	expectedRate := float64(requestsPerSecond)
	if actualRate > expectedRate*1.1 { // Allow 10% tolerance
		b.Logf("Rate limiting ineffective: actual=%.2f, expected=%.2f", actualRate, expectedRate)
	}
}

func BenchmarkRateLimiting_MultipleClients(b *testing.B) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	// Per-client rate limiting
	rateLimitMiddleware := NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
		KeyExtractor: func(ctx context.Context, req MCPRequest) string {
			// Extract client ID from context or generate one
			if clientID, ok := ctx.Value("client_id").(string); ok {
				return clientID
			}
			return "default"
		},
	})

	handler := rateLimitMiddleware.Apply(baseHandler)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate 10 different clients
		clientID := fmt.Sprintf("client-%d", i%10)
		ctx := context.WithValue(context.Background(), "client_id", clientID)

		req := &MockRequestForBenchmark{
			method: "test/method",
			id:     "test-id",
			params: json.RawMessage(`{"test": "data"}`),
			ctx:    ctx,
		}

		_, err := handler.Handle(ctx, req)
		if err != nil {
			b.Errorf("Handler failed: %v", err)
		}
	}
}

func BenchmarkRateLimiting_ConcurrentAccess(b *testing.B) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	rateLimitMiddleware := NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
	})

	handler := rateLimitMiddleware.Apply(baseHandler)

	req := &MockRequestForBenchmark{
		method: "test/method",
		id:     "test-id",
		params: json.RawMessage(`{"test": "data"}`),
		ctx:    context.Background(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		concurrency := 10

		wg.Add(concurrency)
		for j := 0; j < concurrency; j++ {
			go func() {
				defer wg.Done()
				_, err := handler.Handle(context.Background(), req)
				if err != nil {
					b.Errorf("Concurrent handler failed: %v", err)
				}
			}()
		}
		wg.Wait()
	}
}

// =============================================================================
// Logging Middleware Impact
// =============================================================================

func BenchmarkLoggingMiddleware_Impact(b *testing.B) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	logLevels := map[string]slog.Level{
		"Disabled": slog.Level(1000), // Very high level to disable logging
		"Error":    slog.LevelError,
		"Warn":     slog.LevelWarn,
		"Info":     slog.LevelInfo,
		"Debug":    slog.LevelDebug,
	}

	for name, level := range logLevels {
		b.Run(name, func(b *testing.B) {
			// Create logger that discards output for performance testing
			logger := slog.New(slog.NewJSONHandler(os.NewFile(0, os.DevNull), &slog.HandlerOptions{
				Level: level,
			}))

			loggingMiddleware := NewLoggingMiddleware(LoggingConfig{
				Logger:          logger,
				Level:           level,
				IncludeRequest:  true,
				IncludeResponse: true,
				IncludeParams:   true,
			})

			handler := loggingMiddleware.Apply(baseHandler)

			req := &MockRequestForBenchmark{
				method: "test/method",
				id:     "test-id",
				params: json.RawMessage(`{"test": "data"}`),
				ctx:    context.Background(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := handler.Handle(context.Background(), req)
				if err != nil {
					b.Errorf("Handler failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkLoggingMiddleware_PayloadSizes(b *testing.B) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	logger := slog.New(slog.NewJSONHandler(os.NewFile(0, os.DevNull), nil))
	loggingMiddleware := NewLoggingMiddleware(LoggingConfig{
		Logger:          logger,
		Level:           slog.LevelInfo,
		IncludeRequest:  true,
		IncludeResponse: true,
		IncludeParams:   true,
	})

	handler := loggingMiddleware.Apply(baseHandler)

	for _, size := range requestSizes {
		b.Run(fmt.Sprintf("PayloadSize_%d", size), func(b *testing.B) {
			// Generate large payload
			payload := make(map[string]interface{})
			payload["data"] = string(make([]byte, size))
			payloadJSON, _ := json.Marshal(payload)

			req := &MockRequestForBenchmark{
				method: "test/method",
				id:     "test-id",
				params: payloadJSON,
				ctx:    context.Background(),
			}

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := handler.Handle(context.Background(), req)
				if err != nil {
					b.Errorf("Handler failed: %v", err)
				}
			}
		})
	}
}

// =============================================================================
// Authentication Middleware with Cache Performance
// =============================================================================

func BenchmarkAuthMiddleware_CacheScenarios(b *testing.B) {
	provider := NewMemoryOAuthProvider()

	// Register client and create tokens
	client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "bench-client",
		ClientSecret: "bench-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	})

	authReq := &AuthorizationRequest{
		ResponseType: ResponseTypeCode,
		ClientID:     client.ClientID,
		RedirectURI:  client.RedirectURIs[0],
	}

	authCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
	validToken, _ := provider.CreateAccessToken(context.Background(), authCode)

	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	authMiddleware := NewAuthenticationMiddleware(AuthConfig{
		Provider:     provider,
		CacheTimeout: 5 * time.Minute,
	})

	handler := authMiddleware.Apply(baseHandler)

	for _, scenario := range cachingScenarios {
		b.Run(scenario, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var ctx context.Context
				var req MCPRequest

				switch scenario {
				case "no-cache":
					// Create new token each time (no cache)
					newAuthCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
					newToken, _ := provider.CreateAccessToken(context.Background(), newAuthCode)
					ctx = context.WithValue(context.Background(), authHeaderKey, "Bearer "+newToken.AccessToken)

				case "cache-hit":
					// Use same token (cache hit)
					ctx = context.WithValue(context.Background(), authHeaderKey, "Bearer "+validToken.AccessToken)

				case "cache-miss":
					// Use invalid token (cache miss)
					invalidToken := generateRandomToken(64)
					ctx = context.WithValue(context.Background(), authHeaderKey, "Bearer "+invalidToken)
				}

				req = &MockRequestForBenchmark{
					method: "tools/call",
					id:     "test-id",
					params: json.RawMessage(`{"name": "test-tool"}`),
					ctx:    ctx,
				}

				_, err := handler.Handle(ctx, req)
				if scenario == "cache-miss" && err == nil {
					// Cache miss should result in auth error, but we continue for benchmarking
				} else if scenario != "cache-miss" && err != nil {
					b.Errorf("Handler failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkAuthMiddleware_TokenExtraction(b *testing.B) {
	provider := NewMemoryOAuthProvider()
	authMiddleware := NewAuthenticationMiddleware(AuthConfig{
		Provider: provider,
	})

	extractionMethods := map[string]func(i int) (context.Context, MCPRequest){
		"HeaderExtraction": func(i int) (context.Context, MCPRequest) {
			token := generateRandomToken(64)
			ctx := context.WithValue(context.Background(), authHeaderKey, "Bearer "+token)
			req := &MockRequestForBenchmark{
				method: "tools/call",
				id:     "test-id",
				params: json.RawMessage(`{"name": "test-tool"}`),
				ctx:    ctx,
			}
			return ctx, req
		},
		"ParamsExtraction": func(i int) (context.Context, MCPRequest) {
			token := generateRandomToken(64)
			params := map[string]interface{}{
				"name":       "test-tool",
				"auth_token": token,
			}
			paramsJSON, _ := json.Marshal(params)

			req := &MockRequestForBenchmark{
				method: "tools/call",
				id:     "test-id",
				params: paramsJSON,
				ctx:    context.Background(),
			}
			return context.Background(), req
		},
	}

	for name, extractionFunc := range extractionMethods {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx, req := extractionFunc(i)

				// Just measure extraction time, not validation
				_, err := authMiddleware.extractToken(ctx, req)
				if err != nil {
					// Expected for invalid tokens, continue benchmark
				}
			}
		})
	}
}

// =============================================================================
// Metrics Middleware Performance
// =============================================================================

func BenchmarkMetricsMiddleware(b *testing.B) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	// Mock metrics registry
	metricsRegistry := &MockMetricsRegistry{
		requests: make(map[string]int),
		errors:   make(map[string]int),
	}

	metricsMiddleware := NewMetricsMiddleware(metricsRegistry)
	handler := metricsMiddleware.Apply(baseHandler)

	req := &MockRequestForBenchmark{
		method: "test/method",
		id:     "test-id",
		params: json.RawMessage(`{"test": "data"}`),
		ctx:    context.Background(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Errorf("Handler failed: %v", err)
		}
	}

	// Verify metrics were recorded
	if metricsRegistry.requests["test/method"] != b.N {
		b.Errorf("Expected %d requests recorded, got %d", b.N, metricsRegistry.requests["test/method"])
	}
}

// =============================================================================
// Recovery Middleware Performance
// =============================================================================

func BenchmarkRecoveryMiddleware(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(os.NewFile(0, os.DevNull), nil))
	recoveryMiddleware := NewRecoveryMiddleware(logger, false)

	scenarios := map[string]MCPHandlerFunc{
		"NormalOperation": func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
			return &SuccessResponseForBenchmark{result: "success"}, nil
		},
		"WithPanic": func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
			if req.Method() == "panic-method" {
				panic("test panic")
			}
			return &SuccessResponseForBenchmark{result: "success"}, nil
		},
	}

	for name, handlerFunc := range scenarios {
		b.Run(name, func(b *testing.B) {
			handler := recoveryMiddleware.Apply(handlerFunc)

			method := "test/method"
			if name == "WithPanic" {
				method = "panic-method"
			}

			req := &MockRequestForBenchmark{
				method: method,
				id:     "test-id",
				params: json.RawMessage(`{"test": "data"}`),
				ctx:    context.Background(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp, err := handler.Handle(context.Background(), req)
				if name == "WithPanic" {
					// Should recover from panic and return error response
					if resp == nil || !resp.IsError() {
						b.Errorf("Expected error response from panic recovery")
					}
				} else if err != nil {
					b.Errorf("Normal operation failed: %v", err)
				}
			}
		})
	}
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

func BenchmarkMiddlewareMemoryAllocation(b *testing.B) {
	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	b.Run("ChainCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler := MCPHandler(baseHandler)

			// Build chain
			handler = NewRecoveryMiddleware(slog.Default(), false).Apply(handler)
			handler = NewLoggingMiddleware(LoggingConfig{Level: slog.LevelError}).Apply(handler)
			handler = NewRateLimitMiddleware(RateLimitConfig{
				RequestsPerSecond: 100,
				BurstSize:         10,
			}).Apply(handler)
			handler = NewTimeoutMiddleware(30 * time.Second).Apply(handler)

			_ = handler // Prevent optimization
		}
	})

	b.Run("RequestExecution", func(b *testing.B) {
		// Pre-build middleware chain
		handler := MCPHandler(baseHandler)
		handler = NewLoggingMiddleware(LoggingConfig{Level: slog.LevelError}).Apply(handler)
		handler = NewRateLimitMiddleware(RateLimitConfig{
			RequestsPerSecond: 1000,
			BurstSize:         100,
		}).Apply(handler)

		req := &MockRequestForBenchmark{
			method: "test/method",
			id:     "test-id",
			params: json.RawMessage(`{"test": "data"}`),
			ctx:    context.Background(),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := handler.Handle(context.Background(), req)
			if err != nil {
				b.Errorf("Handler failed: %v", err)
			}
		}
	})
}

func BenchmarkMiddlewareMemoryPressure(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory pressure test in short mode")
	}

	// Force GC to establish baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	baseHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Create some allocation pressure
		data := make([]byte, 1024)
		_ = data
		return &SuccessResponseForBenchmark{result: "success"}, nil
	})

	// Build heavy middleware chain
	handler := MCPHandler(baseHandler)
	for i := 0; i < 10; i++ {
		handler = NewLoggingMiddleware(LoggingConfig{Level: slog.LevelError}).Apply(handler)
	}

	req := &MockRequestForBenchmark{
		method: "test/method",
		id:     "test-id",
		params: json.RawMessage(`{"test": "data"}`),
		ctx:    context.Background(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := handler.Handle(context.Background(), req)
		if err != nil {
			b.Errorf("Handler failed: %v", err)
		}

		// Periodic GC to measure pressure
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Measure final memory usage
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
}

// =============================================================================
// Mock Types for Benchmarking
// =============================================================================

// MockRequestForBenchmark for middleware benchmarks
type MockRequestForBenchmark struct {
	method string
	id     interface{}
	params json.RawMessage
	ctx    context.Context
}

func (r *MockRequestForBenchmark) Method() string {
	return r.method
}

func (r *MockRequestForBenchmark) ID() interface{} {
	return r.id
}

func (r *MockRequestForBenchmark) Params() json.RawMessage {
	return r.params
}

func (r *MockRequestForBenchmark) Context() context.Context {
	return r.ctx
}

func (r *MockRequestForBenchmark) WithContext(ctx context.Context) MCPRequest {
	return &MockRequestForBenchmark{
		method: r.method,
		id:     r.id,
		params: r.params,
		ctx:    ctx,
	}
}

// SuccessResponseForBenchmark for testing
type SuccessResponseForBenchmark struct {
	result interface{}
}

func (r *SuccessResponseForBenchmark) IsError() bool {
	return false
}

func (r *SuccessResponseForBenchmark) Error() *ResponseError {
	return nil
}

func (r *SuccessResponseForBenchmark) Result() interface{} {
	return r.result
}

// MockMetricsRegistry for testing metrics middleware
type MockMetricsRegistry struct {
	mu       sync.RWMutex
	requests map[string]int
	errors   map[string]int
	active   int64
}

func (m *MockMetricsRegistry) RecordRequest(method string, duration time.Duration, statusCode int, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[method]++
}

func (m *MockMetricsRegistry) RecordActiveRequests(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = count
}

func (m *MockMetricsRegistry) RecordError(method string, errorType string, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method]++
}
