package mcp

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	cfg := RateLimitConfig{
		GlobalRPS:   2,
		GlobalBurst: 1,
		MethodRPS: map[string]float64{
			"test/method": 1,
		},
		MethodBurst: map[string]int{
			"test/method": 1,
		},
		ToolRPS: map[string]float64{
			"test_tool": 1,
		},
		ToolBurst: map[string]int{
			"test_tool": 1,
		},
	}

	rl := NewRateLimiter(cfg)

	tests := []struct {
		name    string
		method  string
		tool    string
		wait    time.Duration
		wantErr bool
	}{
		{
			name:    "allow first request",
			method:  "test/method",
			wantErr: false,
		},
		{
			name:    "block immediate second request",
			method:  "test/method",
			wantErr: true,
		},
		{
			name:    "allow after waiting",
			method:  "test/method",
			wait:    time.Second,
			wantErr: false,
		},
		{
			name:    "allow first tool call",
			tool:    "test_tool",
			wantErr: false,
		},
		{
			name:    "block immediate second toola call",
			tool:    "test_tool",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wait > 0 {
				time.Sleep(tt.wait)
			}

			ctx := context.Background()
			var err error

			if tt.method != "" {
				err = rl.Allow(ctx, tt.method)
			} else if tt.tool != "" {
				err = rl.AllowTool(ctx, tt.tool)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("got O error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
