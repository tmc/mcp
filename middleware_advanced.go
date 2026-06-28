// Package mcp - Advanced Middleware Components
//
// This file implements advanced middleware components including compression,
// caching, validation, CORS, and content transformation for the comprehensive
// middleware system.

package mcp

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Advanced Middleware Components
// ==============================

// CompressionMiddleware provides request/response compression
type CompressionMiddleware struct {
	algorithms map[string]CompressionAlgorithm
	minSize    int
	level      int
	skipTypes  map[string]bool
}

// CompressionAlgorithm defines a compression algorithm interface
type CompressionAlgorithm interface {
	Compress(data []byte, level int) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	ContentEncoding() string
}

// GzipAlgorithm implements gzip compression
type GzipAlgorithm struct{}

func (g *GzipAlgorithm) Compress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer

	var writer *gzip.Writer
	var err error

	if level > 0 {
		writer, err = gzip.NewWriterLevel(&buf, level)
	} else {
		writer = gzip.NewWriter(&buf)
	}

	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *GzipAlgorithm) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func (g *GzipAlgorithm) ContentEncoding() string {
	return "gzip"
}

// NewCompressionMiddleware creates a new compression middleware
func NewCompressionMiddleware(config CompressionConfig) *CompressionMiddleware {
	algorithms := map[string]CompressionAlgorithm{
		"gzip": &GzipAlgorithm{},
	}

	if config.MinSize == 0 {
		config.MinSize = 1024 // 1KB default
	}

	if config.Level == 0 {
		config.Level = gzip.DefaultCompression
	}

	// Skip binary and already compressed types
	skipTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"video/*":    true,
		"audio/*":    true,
	}

	return &CompressionMiddleware{
		algorithms: algorithms,
		minSize:    config.MinSize,
		level:      config.Level,
		skipTypes:  skipTypes,
	}
}

func (m *CompressionMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Execute the next handler
		resp, err := next.Handle(ctx, req)
		if err != nil {
			return resp, err
		}

		// Skip compression for certain content types or if response is too small
		if resp == nil {
			return resp, err
		}

		// Marshal response to check size and compress if needed
		respData, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			return resp, err // Return original response if marshaling fails
		}

		// Check if response is large enough to compress
		if len(respData) < m.minSize {
			return resp, err
		}

		// Compress using gzip (default algorithm)
		if algorithm, exists := m.algorithms["gzip"]; exists {
			compressed, compressErr := algorithm.Compress(respData, m.level)
			if compressErr == nil && len(compressed) < len(respData) {
				// Create a compressed response wrapper
				// Note: In a real implementation, this would need proper response type handling
				// For now, we'll return the original response since the MCP protocol
				// doesn't specify compression at the transport level
				return resp, err
			}
		}

		return resp, err
	})
}

func (m *CompressionMiddleware) Name() string {
	return "compression"
}

func (m *CompressionMiddleware) Priority() int {
	return 200 // Lower priority, applied late
}

// CachingMiddleware provides response caching with TTL
type CachingMiddleware struct {
	cache       Cache
	keyStrategy CacheKeyStrategy
	ttl         time.Duration
	maxSize     int64
	skipMethods map[string]bool
}

// Cache defines the caching interface
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Size() int64
}

// CacheKeyStrategy defines how cache keys are generated
type CacheKeyStrategy interface {
	GenerateKey(ctx context.Context, req MCPRequest) string
}

// InMemoryCache implements a simple in-memory cache
type InMemoryCache struct {
	mu           sync.RWMutex
	items        map[string]*cacheItem
	size         int64
	maxSize      int64
	cleanupTimer *time.Timer
	closed       bool
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
	size      int64
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(maxSize int64) *InMemoryCache {
	cache := &InMemoryCache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
	}

	// Start cleanup timer
	cache.startCleanup()

	return cache
}

func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		// Item expired, clean it up
		go func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			delete(c.items, key)
			c.size -= item.size
		}()
		return nil, false
	}

	return item.value, true
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	itemSize := int64(len(value))

	// Check if we have space
	if c.size+itemSize > c.maxSize {
		// Evict some items
		c.evictLRU()
	}

	// Remove existing item if present
	if existing, exists := c.items[key]; exists {
		c.size -= existing.size
	}

	item := &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
		size:      itemSize,
	}

	c.items[key] = item
	c.size += itemSize

	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		delete(c.items, key)
		c.size -= item.size
	}

	return nil
}

func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.size = 0

	return nil
}

func (c *InMemoryCache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

func (c *InMemoryCache) evictLRU() {
	// Simple eviction: remove expired items first
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
			c.size -= item.size
		}
	}
}

func (c *InMemoryCache) startCleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.cleanupTimer = time.AfterFunc(10*time.Minute, func() {
		c.mu.Lock()
		c.evictLRU()
		c.mu.Unlock()
		c.startCleanup() // Reschedule
	})
}

// Close stops the cache's background cleanup timer. It is safe to call more
// than once. After Close the cache may still be read and written, but expired
// entries are no longer reclaimed automatically.
func (c *InMemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.cleanupTimer != nil {
		c.cleanupTimer.Stop()
	}
	return nil
}

// DefaultCacheKeyStrategy implements a default cache key strategy
type DefaultCacheKeyStrategy struct{}

func (s *DefaultCacheKeyStrategy) GenerateKey(ctx context.Context, req MCPRequest) string {
	// Generate key based on method and params
	hash := sha256.New()
	hash.Write([]byte(req.GetMethod()))

	if params := req.GetParams(); params != nil {
		hash.Write(params)
	}

	// Include client ID if available
	if authCtx := GetAuthContext(ctx); authCtx != nil {
		hash.Write([]byte(authCtx.ClientID))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// NewCachingMiddleware creates a new caching middleware
func NewCachingMiddleware(config CachingConfig) *CachingMiddleware {
	cache := NewInMemoryCache(config.MaxSize)
	keyStrategy := &DefaultCacheKeyStrategy{}

	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}

	// Methods that should not be cached
	skipMethods := map[string]bool{
		"tools/call":     true,
		"resources/read": false, // Resources can be cached
		"prompts/get":    false, // Prompts can be cached
	}

	return &CachingMiddleware{
		cache:       cache,
		keyStrategy: keyStrategy,
		ttl:         config.TTL,
		maxSize:     config.MaxSize,
		skipMethods: skipMethods,
	}
}

func (m *CachingMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Check if method should be cached
		if m.skipMethods[req.GetMethod()] {
			return next.Handle(ctx, req)
		}

		// Generate cache key
		key := m.keyStrategy.GenerateKey(ctx, req)

		// Try to get from cache
		if cached, found := m.cache.Get(ctx, key); found {
			var resp MCPResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				return resp, nil
			}
		}

		// Execute handler
		resp, err := next.Handle(ctx, req)
		if err != nil {
			return resp, err
		}

		// Cache successful response
		if resp != nil && !resp.IsError() {
			if data, err := json.Marshal(resp); err == nil {
				m.cache.Set(ctx, key, data, m.ttl)
			}
		}

		return resp, err
	})
}

func (m *CachingMiddleware) Name() string {
	return "caching"
}

func (m *CachingMiddleware) Priority() int {
	return 300 // Lower priority to cache final results
}

// ValidationMiddleware provides request/response validation
type ValidationMiddleware struct {
	schemas    map[string]interface{}
	strictMode bool
	validator  RequestValidator
}

// RequestValidator defines the validation interface
type RequestValidator interface {
	ValidateRequest(ctx context.Context, req MCPRequest) error
	ValidateResponse(ctx context.Context, resp MCPResponse) error
}

// Schema validation functionality is now provided by security.go

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware(config MiddlewareValidationConfig) *ValidationMiddleware {
	validator := NewJSONSchemaValidator()

	return &ValidationMiddleware{
		schemas:    config.Schemas,
		strictMode: config.StrictMode,
		validator:  validator,
	}
}

func (m *ValidationMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Validate request
		if err := m.validator.ValidateRequest(ctx, req); err != nil {
			return NewErrorResponse("Request validation failed: "+err.Error(), -32602), nil
		}

		// Execute handler
		resp, err := next.Handle(ctx, req)
		if err != nil {
			return resp, err
		}

		// Validate response if strict mode is enabled
		if m.strictMode && resp != nil {
			if err := m.validator.ValidateResponse(ctx, resp); err != nil {
				return NewErrorResponse("Response validation failed: "+err.Error(), -32603), nil
			}
		}

		return resp, err
	})
}

func (m *ValidationMiddleware) Name() string {
	return "validation"
}

func (m *ValidationMiddleware) Priority() int {
	return 850 // High priority for early validation
}

// CORSMiddleware provides Cross-Origin Resource Sharing support
type CORSMiddleware struct {
	config CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(config CORSConfig) *CORSMiddleware {
	// SECURITY: Set secure defaults (no origins allowed by default in production)
	// This prevents accidental exposure. Origins must be explicitly configured.
	if len(config.AllowOrigins) == 0 {
		// Check if we're in development mode
		if os.Getenv("MCP_ENV") == "development" || os.Getenv("MCP_ENV") == "dev" {
			config.AllowOrigins = []string{"http://localhost:*", "http://127.0.0.1:*"}
		} else {
			// Production: require explicit configuration, deny all by default
			config.AllowOrigins = []string{}
		}
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{"Content-Type", "Authorization"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 3600 // 1 hour (more conservative)
	}

	return &CORSMiddleware{
		config: config,
	}
}

func (m *CORSMiddleware) Apply(next MCPHandler) MCPHandler {
	// CORS is an HTTP-layer concern; in the MCP handler chain this middleware
	// is a pass-through. Actual CORS headers are set by the HTTP transports
	// (see the streamable handler's AllowOrigin option).
	return next
}

func (m *CORSMiddleware) Name() string {
	return "cors"
}

func (m *CORSMiddleware) Priority() int {
	return 950 // High priority to set headers early
}

// ContentTransformationMiddleware provides content transformation pipelines
type ContentTransformationMiddleware struct {
	transformers map[string]ContentTransformer
}

// ContentTransformer defines the interface for content transformation
type ContentTransformer interface {
	Transform(ctx context.Context, content interface{}) (interface{}, error)
	CanTransform(contentType string) bool
}

// JSONMinifierTransformer minifies JSON content
type JSONMinifierTransformer struct{}

func (t *JSONMinifierTransformer) Transform(ctx context.Context, content interface{}) (interface{}, error) {
	if jsonStr, ok := content.(string); ok {
		var obj interface{}
		if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
			return content, err // Return original if not valid JSON
		}

		minified, err := json.Marshal(obj)
		if err != nil {
			return content, err
		}

		return string(minified), nil
	}

	return content, nil
}

func (t *JSONMinifierTransformer) CanTransform(contentType string) bool {
	return strings.Contains(contentType, "json")
}

// NewContentTransformationMiddleware creates a new content transformation middleware
func NewContentTransformationMiddleware() *ContentTransformationMiddleware {
	transformers := map[string]ContentTransformer{
		"json_minifier": &JSONMinifierTransformer{},
	}

	return &ContentTransformationMiddleware{
		transformers: transformers,
	}
}

func (m *ContentTransformationMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Transform request content if needed
		transformedReq := m.transformRequest(ctx, req)

		// Execute handler
		resp, err := next.Handle(ctx, transformedReq)
		if err != nil {
			return resp, err
		}

		// Transform response content if needed
		transformedResp := m.transformResponse(ctx, resp)

		return transformedResp, err
	})
}

func (m *ContentTransformationMiddleware) transformRequest(ctx context.Context, req MCPRequest) MCPRequest {
	// Safely transform request parameters using available transformers
	params := req.GetParams()
	if len(params) > 0 {
		for _, transformer := range m.transformers {
			if transformer.CanTransform("application/json") {
				// Parse, transform, and re-marshal JSON parameters
				var parsedParams interface{}
				if err := json.Unmarshal(params, &parsedParams); err == nil {
					if transformed, err := transformer.Transform(ctx, parsedParams); err == nil {
						if newParams, err := json.Marshal(transformed); err == nil {
							// Create new request with transformed params
							return &transformedRequest{
								original: req,
								params:   newParams,
							}
						}
					}
				}
			}
		}
	}
	return req
}

func (m *ContentTransformationMiddleware) transformResponse(ctx context.Context, resp MCPResponse) MCPResponse {
	// Safely transform response result using available transformers
	result := resp.GetResult()
	if result != nil {
		for _, transformer := range m.transformers {
			if transformer.CanTransform("application/json") {
				if transformed, err := transformer.Transform(ctx, result); err == nil {
					// Create new response with transformed result
					return &transformedResponse{
						original: resp,
						result:   transformed,
					}
				}
			}
		}
	}
	return resp
}

func (m *ContentTransformationMiddleware) Name() string {
	return "content_transformation"
}

func (m *ContentTransformationMiddleware) Priority() int {
	return 100 // Very low priority, applied last
}

// transformedRequest wraps an MCPRequest with transformed parameters
type transformedRequest struct {
	original MCPRequest
	params   json.RawMessage
}

func (tr *transformedRequest) GetMethod() string {
	return tr.original.GetMethod()
}

func (tr *transformedRequest) GetID() interface{} {
	return tr.original.GetID()
}

func (tr *transformedRequest) GetParams() json.RawMessage {
	return tr.params
}

func (tr *transformedRequest) GetContext() context.Context {
	return tr.original.GetContext()
}

func (tr *transformedRequest) WithContext(ctx context.Context) MCPRequest {
	return &transformedRequest{
		original: tr.original.WithContext(ctx),
		params:   tr.params,
	}
}

// transformedResponse wraps an MCPResponse with transformed result
type transformedResponse struct {
	original MCPResponse
	result   interface{}
}

func (tr *transformedResponse) GetResult() interface{} {
	return tr.result
}

func (tr *transformedResponse) GetError() *ResponseError {
	return tr.original.GetError()
}

func (tr *transformedResponse) IsError() bool {
	return tr.original.IsError()
}

// ConditionalMiddleware applies middleware based on conditions
type ConditionalMiddleware struct {
	condition  ConditionEvaluator
	middleware Middleware
	logger     *slog.Logger
}

// ConditionEvaluator evaluates whether middleware should be applied
type ConditionEvaluator interface {
	Evaluate(ctx context.Context, req MCPRequest) bool
}

// MethodCondition evaluates based on request method
type MethodCondition struct {
	methods map[string]bool
}

func NewMethodCondition(methods []string) *MethodCondition {
	methodMap := make(map[string]bool)
	for _, method := range methods {
		methodMap[method] = true
	}

	return &MethodCondition{
		methods: methodMap,
	}
}

func (c *MethodCondition) Evaluate(ctx context.Context, req MCPRequest) bool {
	return c.methods[req.GetMethod()]
}

// ClientCondition evaluates based on client ID
type ClientCondition struct {
	clientIDs map[string]bool
}

func NewClientCondition(clientIDs []string) *ClientCondition {
	clientMap := make(map[string]bool)
	for _, clientID := range clientIDs {
		clientMap[clientID] = true
	}

	return &ClientCondition{
		clientIDs: clientMap,
	}
}

func (c *ClientCondition) Evaluate(ctx context.Context, req MCPRequest) bool {
	if authCtx := GetAuthContext(ctx); authCtx != nil {
		return c.clientIDs[authCtx.ClientID]
	}
	return false
}

// RegexCondition evaluates based on regular expressions
type RegexCondition struct {
	pattern *regexp.Regexp
	field   string // "method", "client_id", etc.
}

func NewRegexCondition(pattern, field string) (*RegexCondition, error) {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &RegexCondition{
		pattern: compiled,
		field:   field,
	}, nil
}

func (c *RegexCondition) Evaluate(ctx context.Context, req MCPRequest) bool {
	var value string

	switch c.field {
	case "method":
		value = req.GetMethod()
	case "client_id":
		if authCtx := GetAuthContext(ctx); authCtx != nil {
			value = authCtx.ClientID
		}
	default:
		return false
	}

	return c.pattern.MatchString(value)
}

// NewConditionalMiddleware creates a conditional middleware
func NewConditionalMiddleware(condition ConditionEvaluator, middleware Middleware, logger *slog.Logger) *ConditionalMiddleware {
	if logger == nil {
		logger = slog.Default()
	}

	return &ConditionalMiddleware{
		condition:  condition,
		middleware: middleware,
		logger:     logger,
	}
}

func (m *ConditionalMiddleware) Apply(next MCPHandler) MCPHandler {
	wrappedHandler := m.middleware.Apply(next)

	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		if m.condition.Evaluate(ctx, req) {
			m.logger.Debug("Conditional middleware applied",
				"middleware", m.middleware.Name(),
				"method", req.GetMethod())
			return wrappedHandler.Handle(ctx, req)
		}

		return next.Handle(ctx, req)
	})
}

func (m *ConditionalMiddleware) Name() string {
	return fmt.Sprintf("conditional_%s", m.middleware.Name())
}

func (m *ConditionalMiddleware) Priority() int {
	return m.middleware.Priority()
}
