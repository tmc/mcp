package mcp

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting functionality for MCP services
type RateLimiter struct {
	mu sync.RWMutex
	// Global limiter for all requests
	global *rate.Limiter
	// Per-method limiters
	methods map[string]*rate.Limiter
	// Per-tool limiters
	tools map[string]*rate.Limiter
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	// Global requests per second
	GlobalRPS float64
	// Burst size for global limit
	GlobalBurst int
	// Per-method RPS limits
	MethodRPS map[string]float64
	// Per-method burst limits
	MethodBurst map[string]int
	// Per-tool RPS limits
	ToolRPS map[string]float64
	// Per-tool burst limits
	ToolBurst map[string]int
}

// DefaultRateLimitConfig provides sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRPS:   100,
		GlobalBurst: 50,
		MethodRPS: map[string]float64{
			"resources/read":      20,
			"resources/list":      10,
			"tools/call":          5,
			"completion/complete": 10,
		},
		MethodBurst: map[string]int{
			"resources/read":      10,
			"resources/list":      5,
			"tools/call":          3,
			"completion/complete": 5,
		},
		ToolRPS: map[string]float64{
			// Default per-tool limit
			"*": 2,
		},
		ToolBurst: map[string]int{
			"*": 1,
		},
	}
}

// NewRateLimiter creates a new rate limiter with the given config
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		global:  rate.NewLimiter(rate.Limit(cfg.GlobalRPS), cfg.GlobalBurst),
		methods: make(map[string]*rate.Limiter),
		tools:   make(map[string]*rate.Limiter),
	}

	// Initialize method limiters
	for method, rps := range cfg.MethodRPS {
		burst := cfg.MethodBurst[method]
		rl.methods[method] = rate.NewLimiter(rate.Limit(rps), burst)
	}

	// Initialize tool limiters
	for tool, rps := range cfg.ToolRPS {
		burst := cfg.ToolBurst[tool]
		rl.tools[tool] = rate.NewLimiter(rate.Limit(rps), burst)
	}

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ctx context.Context, method string) error {
	// Check global limit first
	if err := rl.global.Wait(ctx); err != nil {
		return err
	}

	rl.mu.RLock()
	limiter, exists := rl.methods[method]
	rl.mu.RUnlock()

	if exists {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}

	return nil
}

// AllowTool checks if a tool invocation should be allowed
func (rl *RateLimiter) AllowTool(ctx context.Context, toolName string) error {
	// Check tool-specific limit
	rl.mu.RLock()
	limiter, exists := rl.tools[toolName]
	if !exists {
		// Fall back to default tool limit
		limiter = rl.tools["*"]
	}
	rl.mu.RUnlock()

	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}

	return nil
}

// UpdateMethodLimit updates the rate limit for a specific method
func (rl *RateLimiter) UpdateMethodLimit(method string, rps float64, burst int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.methods[method] = rate.NewLimiter(rate.Limit(rps), burst)
}

// UpdateToolLimit updates the rate limit for a specific tool
func (rl *RateLimiter) UpdateToolLimit(tool string, rps float64, burst int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.tools[tool] = rate.NewLimiter(rate.Limit(rps), burst)
}
