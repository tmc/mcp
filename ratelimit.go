// Package mcp provides comprehensive rate limiting for the Model Context Protocol.
// This file implements token bucket and sliding window rate limiting algorithms
// with per-client and global rate limiting support.
package mcp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimitRule defines rate limiting rules for specific clients or patterns
type RateLimitRule struct {
	RequestsPerSecond int           `json:"requestsPerSecond"`
	BurstSize         int           `json:"burstSize"`
	WindowSize        time.Duration `json:"windowSize"`
	Priority          int           `json:"priority"` // Higher priority rules override lower ones
	Methods           []string      `json:"methods"`  // Empty means all methods
}

// RateLimiter defines the interface for rate limiting implementations
type RateLimiter interface {
	// Allow checks if a request should be allowed
	Allow(ctx context.Context, key string) bool
	
	// AllowN checks if n requests should be allowed
	AllowN(ctx context.Context, key string, n int) bool
	
	// Wait blocks until a request can proceed or context is cancelled
	Wait(ctx context.Context, key string) error
	
	// WaitN blocks until n requests can proceed or context is cancelled
	WaitN(ctx context.Context, key string, n int) error
	
	// Reset resets the rate limiter for a specific key
	Reset(key string)
	
	// Stats returns rate limiting statistics
	Stats() RateLimitStats
}

// RateLimitStats provides rate limiting statistics
type RateLimitStats struct {
	TotalRequests    int64                      `json:"totalRequests"`
	AllowedRequests  int64                      `json:"allowedRequests"`
	RejectedRequests int64                      `json:"rejectedRequests"`
	ActiveLimiters   int                        `json:"activeLimiters"`
	PerKeyStats      map[string]*KeyStats       `json:"perKeyStats,omitempty"`
}

// KeyStats provides per-key rate limiting statistics
type KeyStats struct {
	Requests  int64     `json:"requests"`
	Allowed   int64     `json:"allowed"`
	Rejected  int64     `json:"rejected"`
	LastSeen  time.Time `json:"lastSeen"`
}

// TokenBucketRateLimiter implements token bucket algorithm
type TokenBucketRateLimiter struct {
	limiters        sync.Map
	defaultRate     float64
	defaultBurst    int
	cleanupInterval time.Duration
	cleanupTimer    *time.Timer
	stats           atomic.Value // *RateLimitStats
	perKeyRules     map[string]RateLimitRule
	mu              sync.RWMutex
}

// tokenBucket represents a single token bucket
type tokenBucket struct {
	tokens    float64
	lastFill  time.Time
	rate      float64
	burst     int
	mu        sync.Mutex
	stats     *KeyStats
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(requestsPerSecond int, burstSize int) *TokenBucketRateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 100
	}
	if burstSize <= 0 {
		burstSize = 10
	}

	rl := &TokenBucketRateLimiter{
		defaultRate:     float64(requestsPerSecond),
		defaultBurst:    burstSize,
		cleanupInterval: 10 * time.Minute,
		perKeyRules:     make(map[string]RateLimitRule),
	}

	// Initialize stats
	rl.stats.Store(&RateLimitStats{
		PerKeyStats: make(map[string]*KeyStats),
	})

	// Start cleanup timer
	rl.cleanupTimer = time.AfterFunc(rl.cleanupInterval, rl.cleanup)

	return rl
}

// SetRuleForKey sets a specific rate limit rule for a key
func (rl *TokenBucketRateLimiter) SetRuleForKey(key string, rule RateLimitRule) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.perKeyRules[key] = rule
}

// Allow implements RateLimiter
func (rl *TokenBucketRateLimiter) Allow(ctx context.Context, key string) bool {
	return rl.AllowN(ctx, key, 1)
}

// AllowN implements RateLimiter
func (rl *TokenBucketRateLimiter) AllowN(ctx context.Context, key string, n int) bool {
	bucket := rl.getBucket(key)
	allowed := bucket.allowN(n)

	// Update stats
	stats := rl.stats.Load().(*RateLimitStats)
	atomic.AddInt64(&stats.TotalRequests, int64(n))
	if allowed {
		atomic.AddInt64(&stats.AllowedRequests, int64(n))
	} else {
		atomic.AddInt64(&stats.RejectedRequests, int64(n))
	}

	return allowed
}

// Wait implements RateLimiter
func (rl *TokenBucketRateLimiter) Wait(ctx context.Context, key string) error {
	return rl.WaitN(ctx, key, 1)
}

// WaitN implements RateLimiter
func (rl *TokenBucketRateLimiter) WaitN(ctx context.Context, key string, n int) error {
	bucket := rl.getBucket(key)
	
	for {
		// Check if we can proceed
		if bucket.allowN(n) {
			// Update stats
			stats := rl.stats.Load().(*RateLimitStats)
			atomic.AddInt64(&stats.TotalRequests, int64(n))
			atomic.AddInt64(&stats.AllowedRequests, int64(n))
			return nil
		}

		// Calculate wait time
		waitTime := bucket.timeToAvailable(n)
		
		select {
		case <-ctx.Done():
			// Update rejected stats
			stats := rl.stats.Load().(*RateLimitStats)
			atomic.AddInt64(&stats.TotalRequests, int64(n))
			atomic.AddInt64(&stats.RejectedRequests, int64(n))
			return ctx.Err()
		case <-time.After(waitTime):
			// Retry after waiting
			continue
		}
	}
}

// Reset implements RateLimiter
func (rl *TokenBucketRateLimiter) Reset(key string) {
	rl.limiters.Delete(key)
}

// Stats implements RateLimiter
func (rl *TokenBucketRateLimiter) Stats() RateLimitStats {
	stats := rl.stats.Load().(*RateLimitStats)
	
	// Count active limiters
	count := 0
	rl.limiters.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	stats.ActiveLimiters = count

	// Create a copy to avoid data races
	statsCopy := RateLimitStats{
		TotalRequests:    atomic.LoadInt64(&stats.TotalRequests),
		AllowedRequests:  atomic.LoadInt64(&stats.AllowedRequests),
		RejectedRequests: atomic.LoadInt64(&stats.RejectedRequests),
		ActiveLimiters:   count,
		PerKeyStats:      make(map[string]*KeyStats),
	}

	// Copy per-key stats
	rl.limiters.Range(func(k, v interface{}) bool {
		key := k.(string)
		bucket := v.(*tokenBucket)
		bucket.mu.Lock()
		statsCopy.PerKeyStats[key] = &KeyStats{
			Requests: bucket.stats.Requests,
			Allowed:  bucket.stats.Allowed,
			Rejected: bucket.stats.Rejected,
			LastSeen: bucket.stats.LastSeen,
		}
		bucket.mu.Unlock()
		return true
	})

	return statsCopy
}

// getBucket gets or creates a token bucket for a key
func (rl *TokenBucketRateLimiter) getBucket(key string) *tokenBucket {
	// Check for existing bucket
	if v, ok := rl.limiters.Load(key); ok {
		return v.(*tokenBucket)
	}

	// Determine rate and burst for this key
	rate := rl.defaultRate
	burst := rl.defaultBurst

	rl.mu.RLock()
	if rule, ok := rl.perKeyRules[key]; ok {
		rate = float64(rule.RequestsPerSecond)
		burst = rule.BurstSize
	}
	rl.mu.RUnlock()

	// Create new bucket
	bucket := &tokenBucket{
		tokens:   float64(burst),
		lastFill: time.Now(),
		rate:     rate,
		burst:    burst,
		stats: &KeyStats{
			LastSeen: time.Now(),
		},
	}

	// Store bucket (handle race condition)
	actual, _ := rl.limiters.LoadOrStore(key, bucket)
	return actual.(*tokenBucket)
}

// allowN checks if n tokens are available
func (b *tokenBucket) allowN(n int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens = min(float64(b.burst), b.tokens+elapsed*b.rate)
	b.lastFill = now

	// Update stats
	b.stats.Requests += int64(n)
	b.stats.LastSeen = now

	// Check if we have enough tokens
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		b.stats.Allowed += int64(n)
		return true
	}

	b.stats.Rejected += int64(n)
	return false
}

// timeToAvailable calculates time until n tokens are available
func (b *tokenBucket) timeToAvailable(n int) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate tokens needed
	tokensNeeded := float64(n) - b.tokens
	if tokensNeeded <= 0 {
		return 0
	}

	// Calculate time to accumulate needed tokens
	secondsNeeded := tokensNeeded / b.rate
	return time.Duration(secondsNeeded * float64(time.Second))
}

// cleanup removes inactive token buckets
func (rl *TokenBucketRateLimiter) cleanup() {
	cutoff := time.Now().Add(-rl.cleanupInterval)

	rl.limiters.Range(func(key, value interface{}) bool {
		bucket := value.(*tokenBucket)
		bucket.mu.Lock()
		lastSeen := bucket.stats.LastSeen
		bucket.mu.Unlock()

		if lastSeen.Before(cutoff) {
			rl.limiters.Delete(key)
		}
		return true
	})

	// Schedule next cleanup
	rl.cleanupTimer.Reset(rl.cleanupInterval)
}

// SlidingWindowRateLimiter implements sliding window algorithm
type SlidingWindowRateLimiter struct {
	windows         sync.Map
	windowSize      time.Duration
	maxRequests     int
	cleanupInterval time.Duration
	cleanupTimer    *time.Timer
	stats           atomic.Value // *RateLimitStats
	perKeyRules     map[string]RateLimitRule
	mu              sync.RWMutex
}

// slidingWindow represents a single sliding window
type slidingWindow struct {
	requests []time.Time
	mu       sync.Mutex
	stats    *KeyStats
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowRateLimiter {
	if windowSize <= 0 {
		windowSize = time.Minute
	}
	if maxRequests <= 0 {
		maxRequests = 100
	}

	rl := &SlidingWindowRateLimiter{
		windowSize:      windowSize,
		maxRequests:     maxRequests,
		cleanupInterval: 10 * time.Minute,
		perKeyRules:     make(map[string]RateLimitRule),
	}

	// Initialize stats
	rl.stats.Store(&RateLimitStats{
		PerKeyStats: make(map[string]*KeyStats),
	})

	// Start cleanup timer
	rl.cleanupTimer = time.AfterFunc(rl.cleanupInterval, rl.cleanup)

	return rl
}

// SetRuleForKey sets a specific rate limit rule for a key
func (rl *SlidingWindowRateLimiter) SetRuleForKey(key string, rule RateLimitRule) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.perKeyRules[key] = rule
}

// Allow implements RateLimiter
func (rl *SlidingWindowRateLimiter) Allow(ctx context.Context, key string) bool {
	return rl.AllowN(ctx, key, 1)
}

// AllowN implements RateLimiter
func (rl *SlidingWindowRateLimiter) AllowN(ctx context.Context, key string, n int) bool {
	window := rl.getWindow(key)
	allowed := window.allowN(n, rl.getWindowSize(key), rl.getMaxRequests(key))

	// Update stats
	stats := rl.stats.Load().(*RateLimitStats)
	atomic.AddInt64(&stats.TotalRequests, int64(n))
	if allowed {
		atomic.AddInt64(&stats.AllowedRequests, int64(n))
	} else {
		atomic.AddInt64(&stats.RejectedRequests, int64(n))
	}

	return allowed
}

// Wait implements RateLimiter
func (rl *SlidingWindowRateLimiter) Wait(ctx context.Context, key string) error {
	return rl.WaitN(ctx, key, 1)
}

// WaitN implements RateLimiter
func (rl *SlidingWindowRateLimiter) WaitN(ctx context.Context, key string, n int) error {
	window := rl.getWindow(key)
	windowSize := rl.getWindowSize(key)
	maxRequests := rl.getMaxRequests(key)

	for {
		if window.allowN(n, windowSize, maxRequests) {
			// Update stats
			stats := rl.stats.Load().(*RateLimitStats)
			atomic.AddInt64(&stats.TotalRequests, int64(n))
			atomic.AddInt64(&stats.AllowedRequests, int64(n))
			return nil
		}

		// Calculate wait time
		waitTime := window.timeToAvailable(windowSize)

		select {
		case <-ctx.Done():
			// Update rejected stats
			stats := rl.stats.Load().(*RateLimitStats)
			atomic.AddInt64(&stats.TotalRequests, int64(n))
			atomic.AddInt64(&stats.RejectedRequests, int64(n))
			return ctx.Err()
		case <-time.After(waitTime):
			continue
		}
	}
}

// Reset implements RateLimiter
func (rl *SlidingWindowRateLimiter) Reset(key string) {
	rl.windows.Delete(key)
}

// Stats implements RateLimiter
func (rl *SlidingWindowRateLimiter) Stats() RateLimitStats {
	stats := rl.stats.Load().(*RateLimitStats)
	
	// Count active windows
	count := 0
	rl.windows.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	stats.ActiveLimiters = count

	// Create a copy
	statsCopy := RateLimitStats{
		TotalRequests:    atomic.LoadInt64(&stats.TotalRequests),
		AllowedRequests:  atomic.LoadInt64(&stats.AllowedRequests),
		RejectedRequests: atomic.LoadInt64(&stats.RejectedRequests),
		ActiveLimiters:   count,
		PerKeyStats:      make(map[string]*KeyStats),
	}

	// Copy per-key stats
	rl.windows.Range(func(k, v interface{}) bool {
		key := k.(string)
		window := v.(*slidingWindow)
		window.mu.Lock()
		statsCopy.PerKeyStats[key] = &KeyStats{
			Requests: window.stats.Requests,
			Allowed:  window.stats.Allowed,
			Rejected: window.stats.Rejected,
			LastSeen: window.stats.LastSeen,
		}
		window.mu.Unlock()
		return true
	})

	return statsCopy
}

// getWindow gets or creates a sliding window for a key
func (rl *SlidingWindowRateLimiter) getWindow(key string) *slidingWindow {
	if v, ok := rl.windows.Load(key); ok {
		return v.(*slidingWindow)
	}

	window := &slidingWindow{
		requests: make([]time.Time, 0),
		stats: &KeyStats{
			LastSeen: time.Now(),
		},
	}

	actual, _ := rl.windows.LoadOrStore(key, window)
	return actual.(*slidingWindow)
}

// getWindowSize returns the window size for a key
func (rl *SlidingWindowRateLimiter) getWindowSize(key string) time.Duration {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rule, ok := rl.perKeyRules[key]; ok && rule.WindowSize > 0 {
		return rule.WindowSize
	}
	return rl.windowSize
}

// getMaxRequests returns the max requests for a key
func (rl *SlidingWindowRateLimiter) getMaxRequests(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rule, ok := rl.perKeyRules[key]; ok {
		// Calculate max requests from rate and window size
		if rule.RequestsPerSecond > 0 {
			return int(float64(rule.RequestsPerSecond) * rule.WindowSize.Seconds())
		}
	}
	return rl.maxRequests
}

// allowN checks if n requests are allowed in the sliding window
func (w *slidingWindow) allowN(n int, windowSize time.Duration, maxRequests int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-windowSize)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0, len(w.requests))
	for _, reqTime := range w.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	w.requests = validRequests

	// Update stats
	w.stats.Requests += int64(n)
	w.stats.LastSeen = now

	// Check if we can add n more requests
	if len(w.requests)+n <= maxRequests {
		// Add new requests
		for i := 0; i < n; i++ {
			w.requests = append(w.requests, now)
		}
		w.stats.Allowed += int64(n)
		return true
	}

	w.stats.Rejected += int64(n)
	return false
}

// timeToAvailable calculates time until a request can be made
func (w *slidingWindow) timeToAvailable(windowSize time.Duration) time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.requests) == 0 {
		return 0
	}

	// Find the oldest request
	oldestRequest := w.requests[0]
	for _, reqTime := range w.requests {
		if reqTime.Before(oldestRequest) {
			oldestRequest = reqTime
		}
	}

	// Calculate when it will fall out of the window
	timeUntilAvailable := oldestRequest.Add(windowSize).Sub(time.Now())
	if timeUntilAvailable < 0 {
		return 0
	}

	return timeUntilAvailable
}

// cleanup removes inactive sliding windows
func (rl *SlidingWindowRateLimiter) cleanup() {
	cutoff := time.Now().Add(-rl.cleanupInterval)

	rl.windows.Range(func(key, value interface{}) bool {
		window := value.(*slidingWindow)
		window.mu.Lock()
		lastSeen := window.stats.LastSeen
		window.mu.Unlock()

		if lastSeen.Before(cutoff) {
			rl.windows.Delete(key)
		}
		return true
	})

	// Schedule next cleanup
	rl.cleanupTimer.Reset(rl.cleanupInterval)
}

// CompositeRateLimiter combines multiple rate limiters with different strategies
type CompositeRateLimiter struct {
	limiters []RateLimiter
	strategy string // "all" (all must pass) or "any" (at least one must pass)
}

// NewCompositeRateLimiter creates a new composite rate limiter
func NewCompositeRateLimiter(strategy string, limiters ...RateLimiter) *CompositeRateLimiter {
	if strategy != "all" && strategy != "any" {
		strategy = "all"
	}

	return &CompositeRateLimiter{
		limiters: limiters,
		strategy: strategy,
	}
}

// Allow implements RateLimiter
func (c *CompositeRateLimiter) Allow(ctx context.Context, key string) bool {
	if c.strategy == "all" {
		for _, limiter := range c.limiters {
			if !limiter.Allow(ctx, key) {
				return false
			}
		}
		return true
	}

	// "any" strategy
	for _, limiter := range c.limiters {
		if limiter.Allow(ctx, key) {
			return true
		}
	}
	return false
}

// AllowN implements RateLimiter
func (c *CompositeRateLimiter) AllowN(ctx context.Context, key string, n int) bool {
	if c.strategy == "all" {
		for _, limiter := range c.limiters {
			if !limiter.AllowN(ctx, key, n) {
				return false
			}
		}
		return true
	}

	// "any" strategy
	for _, limiter := range c.limiters {
		if limiter.AllowN(ctx, key, n) {
			return true
		}
	}
	return false
}

// Wait implements RateLimiter
func (c *CompositeRateLimiter) Wait(ctx context.Context, key string) error {
	if c.strategy == "all" {
		for _, limiter := range c.limiters {
			if err := limiter.Wait(ctx, key); err != nil {
				return err
			}
		}
		return nil
	}

	// "any" strategy - wait for the first one to allow
	// This is more complex and may need a different implementation
	return fmt.Errorf("wait not supported for 'any' strategy")
}

// WaitN implements RateLimiter
func (c *CompositeRateLimiter) WaitN(ctx context.Context, key string, n int) error {
	if c.strategy == "all" {
		for _, limiter := range c.limiters {
			if err := limiter.WaitN(ctx, key, n); err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("waitN not supported for 'any' strategy")
}

// Reset implements RateLimiter
func (c *CompositeRateLimiter) Reset(key string) {
	for _, limiter := range c.limiters {
		limiter.Reset(key)
	}
}

// Stats implements RateLimiter
func (c *CompositeRateLimiter) Stats() RateLimitStats {
	// Aggregate stats from all limiters
	combined := RateLimitStats{
		PerKeyStats: make(map[string]*KeyStats),
	}

	for _, limiter := range c.limiters {
		stats := limiter.Stats()
		combined.TotalRequests += stats.TotalRequests
		combined.AllowedRequests += stats.AllowedRequests
		combined.RejectedRequests += stats.RejectedRequests
		combined.ActiveLimiters += stats.ActiveLimiters
	}

	return combined
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// EnhancedRateLimitMiddleware provides advanced rate limiting middleware
type EnhancedRateLimitMiddleware struct {
	limiter      RateLimiter
	keyExtractor func(context.Context, MCPRequest) string
	skipMethods  map[string]bool
	errorHandler func(context.Context, MCPRequest) MCPResponse
}

// NewEnhancedRateLimitMiddleware creates enhanced rate limiting middleware
func NewEnhancedRateLimitMiddleware(limiter RateLimiter, config RateLimitConfig) *EnhancedRateLimitMiddleware {
	skipMethods := make(map[string]bool)
	for _, method := range config.SkipMethods {
		skipMethods[method] = true
	}

	if config.KeyExtractor == nil {
		config.KeyExtractor = func(ctx context.Context, req MCPRequest) string {
			// Default to auth-based key extraction
			if authCtx := GetAuthContext(ctx); authCtx != nil {
				return authCtx.ClientID
			}
			return "anonymous"
		}
	}

	return &EnhancedRateLimitMiddleware{
		limiter:      limiter,
		keyExtractor: config.KeyExtractor,
		skipMethods:  skipMethods,
		errorHandler: func(ctx context.Context, req MCPRequest) MCPResponse {
			return NewRateLimitError("Rate limit exceeded")
		},
	}
}

// Apply implements the Middleware interface
func (m *EnhancedRateLimitMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Check if method should skip rate limiting
		if m.skipMethods[req.GetMethod()] {
			return next.Handle(ctx, req)
		}

		// Extract rate limit key
		key := m.keyExtractor(ctx, req)

		// Check rate limit
		if !m.limiter.Allow(ctx, key) {
			return m.errorHandler(ctx, req), nil
		}

		return next.Handle(ctx, req)
	})
}

func (m *EnhancedRateLimitMiddleware) Name() string {
	return "enhanced_rate_limit"
}

func (m *EnhancedRateLimitMiddleware) Priority() int {
	return 800 // After auth, before business logic
}