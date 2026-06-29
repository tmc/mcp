// Package mcp - Middleware request/response adapters and metrics.
//
// This file holds the adapters that bridge the JSON-RPC request handler to the
// middleware MCPRequest/MCPResponse interfaces, along with middleware metrics
// and configuration helpers.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Request/Response Adapters
// ========================

// UnifiedRequest implements the Request interface for the middleware system
type UnifiedRequest struct {
	method string
	id     interface{}
	params json.RawMessage
	ctx    context.Context
}

func (r *UnifiedRequest) Method() string {
	return r.method
}

func (r *UnifiedRequest) ID() interface{} {
	return r.id
}

func (r *UnifiedRequest) Params() json.RawMessage {
	return r.params
}

func (r *UnifiedRequest) Context() context.Context {
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

// successResponse implements Response for successful responses
type successResponse struct {
	result interface{}
}

func (r *successResponse) Result() interface{} {
	return r.result
}

func (r *successResponse) Error() *ResponseError {
	return nil
}

func (r *successResponse) IsError() bool {
	return false
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
