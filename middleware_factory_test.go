package mcp

import "testing"

// TestMiddlewareFactoriesCreateRealMiddleware guards against the factories
// silently returning a NoOpMiddleware: a config-driven server that asked for
// compression, validation, or caching would otherwise get a do-nothing handler.
func TestMiddlewareFactoriesCreateRealMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		factory MiddlewareFactory
		config  interface{}
		want    string // expected concrete middleware Name()
	}{
		{"compression", &CompressionMiddlewareFactory{}, CompressionConfig{}, "compression"},
		{"validation", &ValidationMiddlewareFactory{}, MiddlewareValidationConfig{}, "validation"},
		{"caching", &CachingMiddlewareFactory{}, CachingConfig{}, "caching"},
		// nil config must fall back to defaults, not error.
		{"compression nil cfg", &CompressionMiddlewareFactory{}, nil, "compression"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, err := tt.factory.Create(tt.config)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if _, isNoOp := mw.(*NoOpMiddleware); isNoOp {
				t.Fatalf("%s factory returned NoOpMiddleware", tt.name)
			}
			if mw.Name() != tt.want {
				t.Fatalf("Name() = %q, want %q", mw.Name(), tt.want)
			}
		})
	}
}

func TestMiddlewareFactoryRejectsWrongConfig(t *testing.T) {
	f := &CompressionMiddlewareFactory{}
	if _, err := f.Create("not a CompressionConfig"); err == nil {
		t.Fatal("expected error for mismatched config type, got nil")
	}
}
