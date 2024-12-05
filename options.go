package mcp

// Option configures a Service
type Option func(*Service)

// WithRateLimiting configures custom rate limiting for the service
func WithRateLimiting(cfg RateLimitConfig) Option {
	return func(s *Service) {
		s.limiter = NewRateLimiter(cfg)
	}
}

// WithDispatcher configures a custom notification dispatcher
func WithDispatcher(d *Dispatcher) Option {
	return func(s *Service) {
		s.dispatch = d
	}
}

// WithCapabilities configures initial service capabilities
func WithCapabilities(caps Capabilities) Option {
	return func(s *Service) {
		s.caps = caps
	}
}
