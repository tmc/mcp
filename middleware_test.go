// Package mcp - Comprehensive Middleware System Tests
//
// This file contains comprehensive tests for the middleware system,
// including unit tests, integration tests, and performance benchmarks.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test Helpers and Mocks
// ======================

// MockHandler implements Handler for testing
type MockHandler struct {
	calls     []MockCall
	mu        sync.Mutex
	responses map[string]MCPResponse
	errors    map[string]error
	delay     time.Duration
}

type MockCall struct {
	Method string
	Params json.RawMessage
	Time   time.Time
}

func NewMockHandler() *MockHandler {
	return &MockHandler{
		calls:     make([]MockCall, 0),
		responses: make(map[string]MCPResponse),
		errors:    make(map[string]error),
	}
}

func (m *MockHandler) Handle(ctx context.Context, req MCPRequest) (MCPResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Record the call
	m.calls = append(m.calls, MockCall{
		Method: req.GetMethod(),
		Params: req.GetParams(),
		Time:   time.Now(),
	})
	
	// Add delay if configured
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	
	// Return configured response or error
	if err, exists := m.errors[req.GetMethod()]; exists {
		return nil, err
	}
	
	if resp, exists := m.responses[req.GetMethod()]; exists {
		return resp, nil
	}
	
	// Default response
	return &SuccessResponseImpl{
		Result: map[string]interface{}{
			"method":    req.GetMethod(),
			"processed": true,
		},
	}, nil
}

func (m *MockHandler) SetResponse(method string, response MCPResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[method] = response
}

func (m *MockHandler) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method] = err
}

func (m *MockHandler) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

func (m *MockHandler) GetCalls() []MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]MockCall, len(m.calls))
	copy(result, m.calls)
	return result
}

func (m *MockHandler) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *MockHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = m.calls[:0]
	m.responses = make(map[string]MCPResponse)
	m.errors = make(map[string]error)
	m.delay = 0
}

// MockRequest implements Request for testing
type MockRequest struct {
	method string
	id     interface{}
	params json.RawMessage
	ctx    context.Context
}

func NewMockRequest(method string, params interface{}) *MockRequest {
	var paramsJSON json.RawMessage
	if params != nil {
		data, _ := json.Marshal(params)
		paramsJSON = data
	}
	
	return &MockRequest{
		method: method,
		id:     "test-id",
		params: paramsJSON,
		ctx:    context.Background(),
	}
}

func (r *MockRequest) GetMethod() string {
	return r.method
}

func (r *MockRequest) GetID() interface{} {
	return r.id
}

func (r *MockRequest) GetParams() json.RawMessage {
	return r.params
}

func (r *MockRequest) GetContext() context.Context {
	return r.ctx
}

func (r *MockRequest) WithContext(ctx context.Context) MCPRequest {
	return &MockRequest{
		method: r.method,
		id:     r.id,
		params: r.params,
		ctx:    ctx,
	}
}

// Core Middleware Tests
// ====================

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		config LoggingConfig
		method string
		params interface{}
	}{
		{
			name: "basic logging",
			config: LoggingConfig{
				Level: slog.LevelInfo,
			},
			method: "test/method",
			params: map[string]string{"key": "value"},
		},
		{
			name: "debug logging with params",
			config: LoggingConfig{
				Level:          slog.LevelDebug,
				IncludeRequest: true,
				RequestFields:  []string{"params"},
			},
			method: "debug/method",
			params: map[string]interface{}{"debug": true, "count": 42},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := NewLoggingMiddleware(tt.config)
			
			// Create mock handler
			mockHandler := NewMockHandler()
			
			// Apply middleware
			wrappedHandler := middleware.Apply(mockHandler)
			
			// Create request
			req := NewMockRequest(tt.method, tt.params)
			
			// Execute request
			resp, err := wrappedHandler.Handle(context.Background(), req)
			
			// Verify response
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			
			if resp == nil {
				t.Error("Expected response, got nil")
			}
			
			// Verify handler was called
			if mockHandler.CallCount() != 1 {
				t.Errorf("Expected 1 call to handler, got %d", mockHandler.CallCount())
			}
		})
	}
}

func TestAuthenticationMiddleware(t *testing.T) {
	// Create test OAuth provider
	provider := NewMemoryOAuthProvider()
	
	// Register test client
	client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Name:         "Test Client",
	})
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}
	
	// Create access token
	authCode := &AuthorizationCode{
		Code:      "test-code",
		ClientID:  client.ClientID,
		Scopes:    []string{"read", "write"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	
	token, err := provider.CreateAccessToken(context.Background(), authCode)
	if err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}
	
	tests := []struct {
		name           string
		method         string
		token          string
		expectError    bool
		skipAuth       bool
	}{
		{
			name:        "valid token",
			method:      "test/protected",
			token:       token.AccessToken,
			expectError: false,
		},
		{
			name:        "invalid token",
			method:      "test/protected",
			token:       "invalid-token",
			expectError: true,
		},
		{
			name:        "no token",
			method:      "test/protected",
			token:       "",
			expectError: true,
		},
		{
			name:        "skip auth for initialize",
			method:      "initialize",
			token:       "",
			expectError: false,
			skipAuth:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			config := AuthConfig{
				Provider: provider,
			}
			middleware := NewAuthenticationMiddleware(config)
			
			// Create mock handler
			mockHandler := NewMockHandler()
			
			// Apply middleware
			wrappedHandler := middleware.Apply(mockHandler)
			
			// Create request with auth context
			req := MCPRequest(NewMockRequest(tt.method, nil))
			ctx := req.GetContext()
			if tt.token != "" {
				ctx = context.WithValue(ctx, "Authorization", "Bearer "+tt.token)
			}
			req = req.WithContext(ctx)
			
			// Execute request
			resp, err := wrappedHandler.Handle(ctx, req)
			
			// Verify results
			if tt.expectError {
				if err == nil && (resp == nil || !resp.IsError()) {
					t.Error("Expected error, but got successful response")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if resp != nil && resp.IsError() {
					t.Errorf("Expected successful response, got error: %v", resp.GetError())
				}
			}
			
			// Verify handler was called for successful auth or skip auth
			expectedCalls := 0
			if !tt.expectError || tt.skipAuth {
				expectedCalls = 1
			}
			
			if mockHandler.CallCount() != expectedCalls {
				t.Errorf("Expected %d calls to handler, got %d", expectedCalls, mockHandler.CallCount())
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		config            RateLimitConfig
		requestCount      int
		expectBlocked     int
		requestsPerSecond int
	}{
		{
			name: "basic rate limiting",
			config: RateLimitConfig{
				RequestsPerSecond: 5,
				BurstSize:         2,
			},
			requestCount:  10,
			expectBlocked: 8, // Should allow 2 burst requests, block remaining 8
		},
		{
			name: "burst allowance",
			config: RateLimitConfig{
				RequestsPerSecond: 1,
				BurstSize:         5,
			},
			requestCount:  7,
			expectBlocked: 2, // Should allow 5 burst + 0 (no time passed) = 5, block 2
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := NewRateLimitMiddleware(tt.config)
			
			// Create mock handler
			mockHandler := NewMockHandler()
			
			// Apply middleware
			wrappedHandler := middleware.Apply(mockHandler)
			
			// Make requests rapidly
			blocked := 0
			for i := 0; i < tt.requestCount; i++ {
				req := NewMockRequest("test/method", nil)
				resp, err := wrappedHandler.Handle(context.Background(), req)
				
				if err != nil || (resp != nil && resp.IsError()) {
					blocked++
				}
			}
			
			// Allow some tolerance in rate limiting due to timing
			tolerance := 2
			if blocked < tt.expectBlocked-tolerance || blocked > tt.expectBlocked+tolerance {
				t.Errorf("Expected ~%d blocked requests, got %d", tt.expectBlocked, blocked)
			}
		})
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		handlerDelay   time.Duration
		expectTimeout  bool
	}{
		{
			name:           "request completes within timeout",
			timeout:        100 * time.Millisecond,
			handlerDelay:   50 * time.Millisecond,
			expectTimeout:  false,
		},
		{
			name:           "request times out",
			timeout:        50 * time.Millisecond,
			handlerDelay:   100 * time.Millisecond,
			expectTimeout:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := NewTimeoutMiddleware(tt.timeout)
			
			// Create mock handler with delay
			mockHandler := NewMockHandler()
			mockHandler.SetDelay(tt.handlerDelay)
			
			// Apply middleware
			wrappedHandler := middleware.Apply(mockHandler)
			
			// Execute request
			req := NewMockRequest("test/method", nil)
			start := time.Now()
			resp, err := wrappedHandler.Handle(context.Background(), req)
			duration := time.Since(start)
			
			// Verify timeout behavior
			if tt.expectTimeout {
				if duration >= tt.handlerDelay {
					t.Error("Expected request to be cancelled before handler completion")
				}
				if resp == nil || !resp.IsError() {
					t.Error("Expected timeout error response")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if resp != nil && resp.IsError() {
					t.Errorf("Expected successful response, got error: %v", resp.GetError())
				}
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		shouldPanic  bool
		panicValue   interface{}
		includeStack bool
	}{
		{
			name:         "no panic",
			shouldPanic:  false,
		},
		{
			name:         "panic with string",
			shouldPanic:  true,
			panicValue:   "test panic",
			includeStack: false,
		},
		{
			name:         "panic with error and stack",
			shouldPanic:  true,
			panicValue:   fmt.Errorf("test error"),
			includeStack: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := NewRecoveryMiddleware(nil, tt.includeStack)
			
			// Create panicking handler
			panicHandler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
				if tt.shouldPanic {
					panic(tt.panicValue)
				}
				return &SuccessResponseImpl{Result: "success"}, nil
			})
			
			// Apply middleware
			wrappedHandler := middleware.Apply(panicHandler)
			
			// Execute request
			req := NewMockRequest("test/method", nil)
			
			// This should not panic, even if the handler panics
			resp, err := wrappedHandler.Handle(context.Background(), req)
			
			if tt.shouldPanic {
				// Should have recovered and returned error response
				if resp == nil || !resp.IsError() {
					t.Error("Expected error response after panic recovery")
				}
				if err != nil {
					t.Errorf("Expected no error after recovery, got %v", err)
				}
			} else {
				// Should work normally
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if resp == nil || resp.IsError() {
					t.Error("Expected successful response")
				}
			}
		})
	}
}

func TestMetricsMiddleware(t *testing.T) {
	// Create mock metrics registry
	registry := &InMemoryMetricsRegistry{}
	
	// Create middleware
	middleware := NewMetricsMiddleware(registry)
	
	// Create mock handler
	mockHandler := NewMockHandler()
	
	// Apply middleware
	wrappedHandler := middleware.Apply(mockHandler)
	
	// Execute some requests
	methods := []string{"test/method1", "test/method2", "test/method1"}
	for _, method := range methods {
		req := NewMockRequest(method, nil)
		wrappedHandler.Handle(context.Background(), req)
	}
	
	// Verify metrics were recorded
	// Note: This is a basic test - in practice you'd check specific metrics
	if registry.metrics == nil || len(registry.metrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}
}

// Middleware Chain Tests
// =====================

func TestMiddlewareChainOrdering(t *testing.T) {
	// Create middleware that records execution order
	var executionOrder []string
	var mu sync.Mutex
	
	recordingMiddleware := func(name string) Middleware {
		return &TestMiddleware{
			name:     name,
			priority: map[string]int{"first": 1000, "second": 500, "third": 100}[name],
			beforeFunc: func() {
				mu.Lock()
				executionOrder = append(executionOrder, name+":before")
				mu.Unlock()
			},
			afterFunc: func() {
				mu.Lock()
				executionOrder = append(executionOrder, name+":after")
				mu.Unlock()
			},
		}
	}
	
	// Create chain with middleware in mixed order
	chain := &MiddlewareChain{
		middlewares: []Middleware{
			recordingMiddleware("second"),
			recordingMiddleware("first"),
			recordingMiddleware("third"),
		},
	}
	
	// Create mock handler
	mockHandler := NewMockHandler()
	
	// Apply chain
	wrappedHandler := chain.Apply(mockHandler)
	
	// Execute request
	req := NewMockRequest("test/method", nil)
	wrappedHandler.Handle(context.Background(), req)
	
	// Verify execution order (should be by priority: first, second, third)
	expectedOrder := []string{
		"first:before", "second:before", "third:before",
		"third:after", "second:after", "first:after",
	}
	
	mu.Lock()
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d execution steps, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
	}
	
	// Check that higher priority middleware execute first
	if len(executionOrder) >= 3 {
		if !strings.Contains(executionOrder[0], "first") {
			t.Errorf("Expected 'first' middleware to execute first, got: %v", executionOrder)
		}
	}
	mu.Unlock()
}

// TestMiddleware for testing middleware ordering
type TestMiddleware struct {
	name       string
	priority   int
	beforeFunc func()
	afterFunc  func()
}

func (m *TestMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		if m.beforeFunc != nil {
			m.beforeFunc()
		}
		
		resp, err := next.Handle(ctx, req)
		
		if m.afterFunc != nil {
			m.afterFunc()
		}
		
		return resp, err
	})
}

func (m *TestMiddleware) Name() string {
	return m.name
}

func (m *TestMiddleware) Priority() int {
	return m.priority
}

// TestLoggingFactory for registry testing
type TestLoggingFactory struct {
	name string
}

func (f *TestLoggingFactory) Name() string {
	return f.name
}

func (f *TestLoggingFactory) Create(config interface{}) (Middleware, error) {
	return NewLoggingMiddleware(LoggingConfig{}), nil
}

func (f *TestLoggingFactory) ConfigType() interface{} {
	return LoggingConfig{}
}

func (f *TestLoggingFactory) Description() string {
	return "Test logging middleware factory"
}

// Registry Tests
// ==============

func TestMiddlewareRegistry(t *testing.T) {
	registry := NewMiddlewareRegistry(nil)
	
	// Create a test factory with unique name to avoid conflicts
	testFactoryName := "test-logging-" + t.Name()
	testFactory := &TestLoggingFactory{name: testFactoryName}
	err := registry.RegisterFactory(testFactory)
	if err != nil {
		t.Errorf("Failed to register factory: %v", err)
	}
	
	// Test duplicate registration
	err = registry.RegisterFactory(testFactory)
	if err == nil {
		t.Error("Expected error for duplicate factory registration")
	}
	
	// Test factory retrieval
	retrieved, exists := registry.GetFactory(testFactoryName)
	if !exists {
		t.Error("Factory should exist after registration")
	}
	if retrieved.Name() != testFactory.Name() {
		t.Error("Retrieved factory should be the same instance")
	}
	
	// Test middleware creation
	config := LoggingConfig{Level: slog.LevelInfo}
	middleware, err := registry.CreateMiddleware(testFactoryName, config)
	if err != nil {
		t.Errorf("Failed to create middleware: %v", err)
	}
	if middleware == nil {
		t.Error("Created middleware should not be nil")
	}
	
	// Test middleware retrieval
	retrieved2, exists := registry.GetMiddleware(testFactoryName)
	if !exists {
		t.Error("Middleware should exist after creation")
	}
	if retrieved2 != middleware {
		t.Error("Retrieved middleware should be the same instance")
	}
}

// Integration Tests
// ================

func TestEnhancedServerMiddleware(t *testing.T) {
	// Create enhanced server
	server := NewEnhancedServer()
	
	// Configure middleware
	config := &ServerMiddlewareConfig{
		GlobalConfig: &MiddlewareConfig{
			Enabled: true,
			Logging: &LoggingConfig{
				Level: slog.LevelInfo,
			},
			Recovery: &RecoveryConfig{
				IncludeStack: false,
			},
		},
	}
	
	err := server.SetMiddlewareConfig(config)
	if err != nil {
		t.Errorf("Failed to set middleware config: %v", err)
	}
	
	// Test that middleware is applied
	// Note: This is a basic integration test
	// In practice, you'd test the full request flow
	if server.middlewareManager.globalChain == nil {
		t.Error("Expected global middleware chain to be configured")
	}
}

// Performance Tests
// ================

func BenchmarkMiddlewareOverhead(b *testing.B) {
	// Test middleware overhead
	middlewares := []Middleware{
		NewLoggingMiddleware(LoggingConfig{Level: slog.LevelError}), // Minimal logging
		NewRecoveryMiddleware(nil, false),
		NewMetricsMiddleware(&InMemoryMetricsRegistry{}),
	}
	
	// Apply all middleware
	handler := MCPHandler(NewMockHandler())
	for _, middleware := range middlewares {
		handler = middleware.Apply(handler)
	}
	
	req := NewMockRequest("test/method", map[string]string{"key": "value"})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Handle(context.Background(), req)
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	// Test middleware chain performance
	chain := &MiddlewareChain{
		middlewares: []Middleware{
			NewLoggingMiddleware(LoggingConfig{Level: slog.LevelError}),
			NewRecoveryMiddleware(nil, false),
			NewMetricsMiddleware(&InMemoryMetricsRegistry{}),
		},
	}
	
	mockHandler := NewMockHandler()
	handler := chain.Apply(mockHandler)
	
	req := NewMockRequest("test/method", map[string]string{"key": "value"})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Handle(context.Background(), req)
	}
}

// Concurrency Tests
// ================

func TestMiddlewareConcurrency(t *testing.T) {
	// Test middleware safety under concurrent access
	middleware := NewRateLimitMiddleware(RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
	})
	
	mockHandler := NewMockHandler()
	wrappedHandler := middleware.Apply(mockHandler)
	
	// Run concurrent requests
	const numGoroutines = 100
	const requestsPerGoroutine = 10
	
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*requestsPerGoroutine)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerGoroutine; j++ {
				req := NewMockRequest(fmt.Sprintf("test/method/%d/%d", id, j), nil)
				_, err := wrappedHandler.Handle(context.Background(), req)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
	
	if errorCount > 0 {
		t.Errorf("Found %d errors in concurrent execution", errorCount)
	}
}