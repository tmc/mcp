package sdk2

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Middleware defines the middleware function signature following stdlib patterns.
type Middleware func(Handler) Handler

// Chain applies middlewares in the order they are provided.
// This follows the standard Go middleware chaining pattern.
func Chain(handler Handler, middlewares ...Middleware) Handler {
	// Apply middlewares in reverse order so the first middleware
	// is the outermost wrapper (executed first)
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// RecoveryMiddleware recovers from panics in handlers.
func RecoveryMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic
					slog.Error("Handler panic recovered",
						"error", err,
						"method", r.Method,
						"stack", string(debug.Stack()),
					)
					
					// Return error response
					Error(w, "Internal server error", StatusInternalServerError)
				}
			}()
			
			next.ServeRequest(w, r)
		})
	}
}

// LoggingMiddleware logs requests and responses.
func LoggingMiddleware() Middleware {
	return LoggingMiddlewareWithLogger(slog.Default())
}

// LoggingMiddlewareWithLogger creates logging middleware with a custom logger.
func LoggingMiddlewareWithLogger(logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			start := time.Now()
			
			// Extract request ID if available
			requestID := getRequestID(r.Context)
			
			logger.Info("Request started",
				"method", r.Method,
				"request_id", requestID,
			)
			
			// Wrap the response writer to capture status code
			rw := &responseCapture{ResponseWriter: w, statusCode: StatusOK}
			
			next.ServeRequest(rw, r)
			
			duration := time.Since(start)
			
			logger.Info("Request completed",
				"method", r.Method,
				"request_id", requestID,
				"status", rw.statusCode,
				"duration", duration,
			)
		})
	}
}

// RequestIDMiddleware adds a unique request ID to each request.
func RequestIDMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			// Generate request ID if not present
			requestID := generateRequestID()
			ctx := context.WithValue(r.Context, requestIDKey, requestID)
			
			// Add request ID to response headers
			w.Header().Set("X-Request-ID", requestID)
			
			next.ServeRequest(w, r.WithContext(ctx))
		})
	}
}

// MetricsMiddleware collects request metrics.
func MetricsMiddleware() Middleware {
	return MetricsMiddlewareWithRegistry(nil)
}

// MetricsMiddlewareWithRegistry creates metrics middleware with a custom registry.
func MetricsMiddlewareWithRegistry(registry interface{}) Middleware {
	// In a real implementation, this would use prometheus or similar
	metrics := newRequestMetrics()
	
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			start := time.Now()
			
			// Increment active requests
			metrics.activeRequests.Add(1)
			defer metrics.activeRequests.Add(-1)
			
			// Wrap response writer to capture status
			rw := &responseCapture{ResponseWriter: w, statusCode: StatusOK}
			
			next.ServeRequest(rw, r)
			
			duration := time.Since(start)
			
			// Record metrics
			metrics.recordRequest(r.Method, rw.statusCode, duration)
		})
	}
}

// AuthMiddleware provides authentication functionality.
func AuthMiddleware() Middleware {
	return AuthMiddlewareWithConfig(AuthConfig{})
}

// AuthConfig configures authentication middleware.
type AuthConfig struct {
	// TokenExtractor extracts the auth token from the request
	TokenExtractor func(*Request) (string, error)
	// TokenValidator validates the extracted token
	TokenValidator func(context.Context, string) (*UserInfo, error)
	// SkipMethods lists methods that don't require authentication
	SkipMethods []string
}

// UserInfo represents authenticated user information.
type UserInfo struct {
	ID       string            `json:"id"`
	Username string            `json:"username"`
	Roles    []string          `json:"roles"`
	Claims   map[string]string `json:"claims"`
}

// AuthMiddlewareWithConfig creates authentication middleware with configuration.
func AuthMiddlewareWithConfig(config AuthConfig) Middleware {
	// Set defaults
	if config.TokenExtractor == nil {
		config.TokenExtractor = extractTokenFromHeader
	}
	if config.TokenValidator == nil {
		config.TokenValidator = defaultTokenValidator
	}
	if config.SkipMethods == nil {
		config.SkipMethods = []string{MethodInitialize, MethodInitialized}
	}
	
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			// Check if method should skip authentication
			for _, method := range config.SkipMethods {
				if r.Method == method {
					next.ServeRequest(w, r)
					return
				}
			}
			
			// Extract token
			token, err := config.TokenExtractor(r)
			if err != nil {
				Error(w, "Authentication required", StatusUnauthorized)
				return
			}
			
			// Validate token
			userInfo, err := config.TokenValidator(r.Context, token)
			if err != nil {
				Error(w, "Invalid authentication", StatusUnauthorized)
				return
			}
			
			// Add user info to context
			ctx := context.WithValue(r.Context, userInfoKey, userInfo)
			
			next.ServeRequest(w, r.WithContext(ctx))
		})
	}
}

// RateLimitMiddleware provides rate limiting functionality.
func RateLimitMiddleware() Middleware {
	return RateLimitMiddlewareWithConfig(RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
	})
}

// RateLimitConfig configures rate limiting.
type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
	KeyExtractor      func(*Request) string
	SkipMethods       []string
}

// RateLimitMiddlewareWithConfig creates rate limiting middleware with configuration.
func RateLimitMiddlewareWithConfig(config RateLimitConfig) Middleware {
	// Set defaults
	if config.KeyExtractor == nil {
		config.KeyExtractor = func(r *Request) string {
			// Default: rate limit per client (simplified)
			return "global"
		}
	}
	
	limiters := make(map[string]*rate.Limiter)
	var mu sync.RWMutex
	
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			// Check if method should skip rate limiting
			for _, method := range config.SkipMethods {
				if r.Method == method {
					next.ServeRequest(w, r)
					return
				}
			}
			
			// Get rate limit key
			key := config.KeyExtractor(r)
			
			// Get or create limiter for this key
			mu.RLock()
			limiter, exists := limiters[key]
			mu.RUnlock()
			
			if !exists {
				mu.Lock()
				// Double-check after acquiring write lock
				if limiter, exists = limiters[key]; !exists {
					limiter = rate.NewLimiter(
						rate.Limit(config.RequestsPerSecond),
						config.BurstSize,
					)
					limiters[key] = limiter
				}
				mu.Unlock()
			}
			
			// Check rate limit
			if !limiter.Allow() {
				Error(w, "Rate limit exceeded", StatusTooManyRequests)
				return
			}
			
			next.ServeRequest(w, r)
		})
	}
}

// TimeoutMiddleware adds request timeout functionality.
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			// Create timeout context
			ctx, cancel := context.WithTimeout(r.Context, timeout)
			defer cancel()
			
			// Channel to signal completion
			done := make(chan struct{})
			
			go func() {
				defer close(done)
				next.ServeRequest(w, r.WithContext(ctx))
			}()
			
			select {
			case <-done:
				// Request completed normally
			case <-ctx.Done():
				// Request timed out
				if ctx.Err() == context.DeadlineExceeded {
					Error(w, "Request timeout", StatusRequestTimeout)
				} else {
					Error(w, "Request cancelled", StatusInternalServerError)
				}
			}
		})
	}
}

// CORSMiddleware adds CORS headers.
func CORSMiddleware() Middleware {
	return CORSMiddlewareWithConfig(CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
	})
}

// CORSConfig configures CORS middleware.
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
	MaxAge       int
}

// CORSMiddlewareWithConfig creates CORS middleware with configuration.
func CORSMiddlewareWithConfig(config CORSConfig) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			// Set CORS headers
			if len(config.AllowOrigins) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", strings.Join(config.AllowOrigins, ", "))
			}
			if len(config.AllowMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			}
			if len(config.AllowHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			}
			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}
			
			next.ServeRequest(w, r)
		})
	}
}

// ConditionalMiddleware applies middleware only when condition is met.
func ConditionalMiddleware(condition func(*Request) bool, middleware Middleware) Middleware {
	return func(next Handler) Handler {
		wrapped := middleware(next)
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			if condition(r) {
				wrapped.ServeRequest(w, r)
			} else {
				next.ServeRequest(w, r)
			}
		})
	}
}

// Helper types and functions

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userInfoKey  contextKey = "user_info"
)

// getRequestID extracts the request ID from context.
func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserInfo extracts user information from context.
func GetUserInfo(ctx context.Context) (*UserInfo, bool) {
	userInfo, ok := ctx.Value(userInfoKey).(*UserInfo)
	return userInfo, ok
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	// Simple implementation - in production, use a proper UUID library
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// responseCapture wraps ResponseWriter to capture status code.
type responseCapture struct {
	ResponseWriter
	statusCode int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

// requestMetrics holds request metrics (simplified).
type requestMetrics struct {
	activeRequests counter
	// In a real implementation, these would be prometheus metrics
}

type counter struct {
	value int64
	mu    sync.RWMutex
}

func (c *counter) Add(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
}

func (c *counter) Get() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func newRequestMetrics() *requestMetrics {
	return &requestMetrics{
		activeRequests: counter{},
	}
}

func (m *requestMetrics) recordRequest(method string, statusCode int, duration time.Duration) {
	// In a real implementation, this would record to prometheus or similar
	slog.Debug("Request metrics",
		"method", method,
		"status", statusCode,
		"duration", duration,
	)
}

// Default authentication helpers

func extractTokenFromHeader(r *Request) (string, error) {
	// Look for Authorization header
	auth := r.Context.Value("Authorization")
	if auth == nil {
		return "", fmt.Errorf("no authorization header")
	}
	
	authStr, ok := auth.(string)
	if !ok {
		return "", fmt.Errorf("invalid authorization header")
	}
	
	// Extract token from "Bearer <token>" format
	if strings.HasPrefix(authStr, "Bearer ") {
		return strings.TrimPrefix(authStr, "Bearer "), nil
	}
	
	return "", fmt.Errorf("invalid token format")
}

func defaultTokenValidator(ctx context.Context, token string) (*UserInfo, error) {
	// Simple token validation - in production, validate against your auth service
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}
	
	// Mock validation
	return &UserInfo{
		ID:       "user_123",
		Username: "demo_user",
		Roles:    []string{"user"},
		Claims:   map[string]string{"scope": "read"},
	}, nil
}

// MiddlewareFunc is a convenience type for creating simple middlewares.
type MiddlewareFunc func(ResponseWriter, *Request, func())

// ToMiddleware converts a MiddlewareFunc to a Middleware.
func (mf MiddlewareFunc) ToMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			mf(w, r, func() {
				next.ServeRequest(w, r)
			})
		})
	}
}

// Compose creates a middleware that applies multiple middlewares in sequence.
// This is an alternative to Chain that feels more functional.
func Compose(middlewares ...Middleware) Middleware {
	return func(handler Handler) Handler {
		return Chain(handler, middlewares...)
	}
}

// Group creates a middleware group with common configuration.
type MiddlewareGroup struct {
	middlewares []Middleware
}

// NewMiddlewareGroup creates a new middleware group.
func NewMiddlewareGroup() *MiddlewareGroup {
	return &MiddlewareGroup{}
}

// Use adds middleware to the group.
func (g *MiddlewareGroup) Use(middleware Middleware) *MiddlewareGroup {
	g.middlewares = append(g.middlewares, middleware)
	return g
}

// Apply applies all middlewares in the group to a handler.
func (g *MiddlewareGroup) Apply(handler Handler) Handler {
	return Chain(handler, g.middlewares...)
}

// ToMiddleware converts the group to a single middleware.
func (g *MiddlewareGroup) ToMiddleware() Middleware {
	return func(handler Handler) Handler {
		return g.Apply(handler)
	}
}