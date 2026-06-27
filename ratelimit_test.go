package mcp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiterClose(t *testing.T) {
	t.Run("token bucket", func(t *testing.T) {
		rl := NewTokenBucketRateLimiter(10, 5)
		if !rl.cleanupTimer.Stop() {
			t.Fatal("cleanup timer should be pending before Close")
		}
		// Restart so Close has a live timer to stop, mirroring real use.
		rl.cleanupTimer.Reset(rl.cleanupInterval)

		if err := rl.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if !rl.closed {
			t.Fatal("limiter not marked closed")
		}
		// Idempotent.
		if err := rl.Close(); err != nil {
			t.Fatalf("second Close: %v", err)
		}
		// cleanup must not resurrect the timer after Close.
		rl.cleanup()
		if rl.cleanupTimer.Stop() {
			t.Fatal("cleanup rescheduled the timer after Close")
		}
		// Limits still enforced post-Close.
		if !rl.Allow(context.Background(), "k") {
			t.Fatal("Allow should still work after Close")
		}
	})

	t.Run("sliding window", func(t *testing.T) {
		rl := NewSlidingWindowRateLimiter(time.Second, 10)
		if err := rl.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if err := rl.Close(); err != nil {
			t.Fatalf("second Close: %v", err)
		}
		rl.cleanup()
		if rl.cleanupTimer.Stop() {
			t.Fatal("cleanup rescheduled the timer after Close")
		}
		if !rl.Allow(context.Background(), "k") {
			t.Fatal("Allow should still work after Close")
		}
	})
}

func TestInMemoryCacheClose(t *testing.T) {
	c := NewInMemoryCache(1024)
	if c.cleanupTimer == nil {
		t.Fatal("cleanup timer should be set after construction")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !c.closed {
		t.Fatal("cache not marked closed")
	}
	// Idempotent.
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	// startCleanup must be a no-op once closed (no new timer scheduled).
	c.startCleanup()
	if c.cleanupTimer.Stop() {
		t.Fatal("startCleanup rescheduled the timer after Close")
	}
	// Cache still usable post-Close.
	if err := c.Set(context.Background(), "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set after Close: %v", err)
	}
	if v, ok := c.Get(context.Background(), "k"); !ok || string(v) != "v" {
		t.Fatalf("Get after Close = %q, %v; want \"v\", true", v, ok)
	}
}

func TestTokenBucketRateLimiter(t *testing.T) {
	// Create limiter with 10 requests per second, burst of 5
	limiter := NewTokenBucketRateLimiter(10, 5)

	// Test 1: Allow burst
	for i := 0; i < 5; i++ {
		if !limiter.Allow(context.Background(), "test-key") {
			t.Errorf("Request %d should be allowed (burst)", i+1)
		}
	}

	// Test 2: Exceed burst - should be rejected
	if limiter.Allow(context.Background(), "test-key") {
		t.Error("Request exceeding burst should be rejected")
	}

	// Test 3: Wait for tokens to refill
	time.Sleep(200 * time.Millisecond) // Should refill ~2 tokens
	allowed := 0
	for i := 0; i < 3; i++ {
		if limiter.Allow(context.Background(), "test-key") {
			allowed++
		}
	}
	if allowed < 1 || allowed > 2 {
		t.Errorf("Expected 1-2 requests allowed after 200ms, got %d", allowed)
	}

	// Test 4: Different keys have separate buckets
	if !limiter.Allow(context.Background(), "other-key") {
		t.Error("Different key should have its own bucket")
	}

	// Test 5: AllowN
	limiter.Reset("test-key")
	if !limiter.AllowN(context.Background(), "test-key", 3) {
		t.Error("AllowN(3) should succeed with fresh bucket")
	}
	if limiter.AllowN(context.Background(), "test-key", 3) {
		t.Error("AllowN(3) should fail after using 3 tokens")
	}

	// Test 6: Stats
	stats := limiter.Stats()
	if stats.TotalRequests == 0 {
		t.Error("Stats should show total requests")
	}
	if stats.ActiveLimiters != 2 {
		t.Errorf("Expected 2 active limiters, got %d", stats.ActiveLimiters)
	}
}

func TestTokenBucketRateLimiterWait(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(100, 1) // High rate, burst of 1

	// Use up the burst
	if !limiter.Allow(context.Background(), "wait-test") {
		t.Fatal("Initial request should be allowed")
	}

	// Test wait with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx, "wait-test")
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait should succeed within timeout: %v", err)
	}
	if elapsed < 8*time.Millisecond {
		t.Errorf("Wait returned before token refill: %v", elapsed)
	}

	// Test wait with cancellation
	limiter.Allow(context.Background(), "wait-test") // Use token
	ctx, cancel = context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = limiter.Wait(ctx, "wait-test")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestTokenBucketPerKeyRules(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, 5)

	// Set custom rule for specific key
	limiter.SetRuleForKey("vip-client", RateLimitRule{
		RequestsPerSecond: 100,
		BurstSize:         20,
	})

	// VIP client should have higher burst
	vipAllowed := 0
	for i := 0; i < 25; i++ {
		if limiter.Allow(context.Background(), "vip-client") {
			vipAllowed++
		}
	}
	if vipAllowed != 20 {
		t.Errorf("VIP client should allow 20 requests (burst), got %d", vipAllowed)
	}

	// Regular client should have default burst
	regularAllowed := 0
	for i := 0; i < 10; i++ {
		if limiter.Allow(context.Background(), "regular-client") {
			regularAllowed++
		}
	}
	if regularAllowed != 5 {
		t.Errorf("Regular client should allow 5 requests (burst), got %d", regularAllowed)
	}
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	// 10 requests per second (600 per minute)
	limiter := NewSlidingWindowRateLimiter(time.Minute, 600)

	// Test 1: Allow requests within limit
	for i := 0; i < 10; i++ {
		if !limiter.Allow(context.Background(), "test-key") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Test 2: Different keys
	if !limiter.Allow(context.Background(), "other-key") {
		t.Error("Different key should have its own window")
	}

	// Test 3: AllowN
	if !limiter.AllowN(context.Background(), "batch-key", 50) {
		t.Error("AllowN(50) should succeed for new key")
	}

	// Test 4: Stats
	stats := limiter.Stats()
	if stats.TotalRequests < 61 {
		t.Errorf("Expected at least 61 total requests, got %d", stats.TotalRequests)
	}
	if stats.ActiveLimiters != 3 {
		t.Errorf("Expected 3 active windows, got %d", stats.ActiveLimiters)
	}
}

func TestSlidingWindowRateLimiterWindowSliding(t *testing.T) {
	// Very short window for testing
	limiter := NewSlidingWindowRateLimiter(100*time.Millisecond, 5)

	// Fill the window
	for i := 0; i < 5; i++ {
		if !limiter.Allow(context.Background(), "slide-test") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be rejected
	if limiter.Allow(context.Background(), "slide-test") {
		t.Error("6th request should be rejected")
	}

	// Wait for window to slide
	time.Sleep(120 * time.Millisecond)

	// Should allow new requests
	if !limiter.Allow(context.Background(), "slide-test") {
		t.Error("Request should be allowed after window slides")
	}
}

func TestSlidingWindowPerKeyRules(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(time.Second, 10)

	// Set custom rule
	limiter.SetRuleForKey("premium", RateLimitRule{
		RequestsPerSecond: 50,
		WindowSize:        time.Second,
	})

	// Premium should allow 50 requests
	premiumAllowed := 0
	for i := 0; i < 60; i++ {
		if limiter.Allow(context.Background(), "premium") {
			premiumAllowed++
		}
	}
	if premiumAllowed != 50 {
		t.Errorf("Premium should allow 50 requests, got %d", premiumAllowed)
	}

	// Regular should allow 10 requests
	regularAllowed := 0
	for i := 0; i < 15; i++ {
		if limiter.Allow(context.Background(), "regular") {
			regularAllowed++
		}
	}
	if regularAllowed != 10 {
		t.Errorf("Regular should allow 10 requests, got %d", regularAllowed)
	}
}

func TestCompositeRateLimiter(t *testing.T) {
	// Create two limiters
	tokenBucket := NewTokenBucketRateLimiter(10, 5)
	slidingWindow := NewSlidingWindowRateLimiter(time.Second, 8)

	// Test "all" strategy - both must pass
	allLimiter := NewCompositeRateLimiter("all", tokenBucket, slidingWindow)

	// Should allow up to burst (5) which is less than sliding window (8)
	allowed := 0
	for i := 0; i < 10; i++ {
		if allLimiter.Allow(context.Background(), "test") {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("'all' strategy should limit to 5 (token bucket burst), got %d", allowed)
	}

	// Test "any" strategy - at least one must pass
	tokenBucket2 := NewTokenBucketRateLimiter(1, 1)                 // Very restrictive
	slidingWindow2 := NewSlidingWindowRateLimiter(time.Second, 100) // Very permissive

	anyLimiter := NewCompositeRateLimiter("any", tokenBucket2, slidingWindow2)

	// Should allow many requests (sliding window is permissive)
	allowed = 0
	for i := 0; i < 20; i++ {
		if anyLimiter.Allow(context.Background(), "test2") {
			allowed++
		}
	}
	if allowed < 15 {
		t.Errorf("'any' strategy should allow many requests, got %d", allowed)
	}

	// Test reset
	allLimiter.Reset("test")
	if !allLimiter.Allow(context.Background(), "test") {
		t.Error("After reset, request should be allowed")
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(1000, 100) // High rate to avoid timing issues

	var wg sync.WaitGroup
	var allowed atomic.Int64
	var rejected atomic.Int64

	// Run 10 goroutines, each making 20 requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if limiter.Allow(context.Background(), "concurrent-test") {
					allowed.Add(1)
				} else {
					rejected.Add(1)
				}
				time.Sleep(time.Millisecond) // Small delay
			}
		}(i)
	}

	wg.Wait()

	total := allowed.Load() + rejected.Load()
	if total != 200 {
		t.Errorf("Expected 200 total requests, got %d", total)
	}

	// With burst of 100, at least 100 should be allowed
	if allowed.Load() < 100 {
		t.Errorf("Expected at least 100 allowed requests, got %d", allowed.Load())
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	// Create limiter using proper constructor
	limiter := NewTokenBucketRateLimiter(10, 5)
	// Override cleanup interval for testing
	limiter.cleanupInterval = 100 * time.Millisecond

	// Create buckets for multiple keys
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("cleanup-test-%d", i)
		limiter.Allow(context.Background(), key)
	}

	// Verify buckets exist
	count := 0
	limiter.limiters.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count != 5 {
		t.Errorf("Expected 5 buckets, got %d", count)
	}

	// Manually trigger cleanup with old cutoff
	limiter.cleanup()

	// All buckets should still exist (they're recent)
	count = 0
	limiter.limiters.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count != 5 {
		t.Errorf("Recent buckets should not be cleaned up, got %d", count)
	}
}

func TestEnhancedRateLimitMiddleware(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(5, 2)
	config := RateLimitConfig{
		SkipMethods: []string{"ping"},
		KeyExtractor: func(ctx context.Context, req MCPRequest) string {
			return "test-client"
		},
	}

	middleware := NewEnhancedRateLimitMiddleware(limiter, config)

	// Create test handler that returns success when rate limiting allows the request
	handler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		return &SuccessResponseImpl{Result: "success"}, nil
	})

	protected := middleware.Apply(handler)

	// Test skipped method
	skipReq := &mockMCPRequest{method: "ping"}
	resp, err := protected.Handle(context.Background(), skipReq)
	if err != nil || resp.IsError() {
		t.Error("Ping should be skipped from rate limiting")
	}

	// Test rate limited method
	for i := 0; i < 3; i++ {
		req := &mockMCPRequest{method: "tools/call"}
		resp, err = protected.Handle(context.Background(), req)

		if i < 2 {
			// First 2 should succeed (burst size)
			if err != nil || resp.IsError() {
				t.Errorf("Request %d should succeed", i+1)
			}
		} else {
			// 3rd should be rate limited
			if err != nil || !resp.IsError() {
				t.Error("3rd request should be rate limited")
			}
		}
	}
}

func BenchmarkTokenBucketRateLimiter(b *testing.B) {
	limiter := NewTokenBucketRateLimiter(10000, 1000)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-key-%d", i%100)
			limiter.Allow(ctx, key)
			i++
		}
	})
}

func BenchmarkSlidingWindowRateLimiter(b *testing.B) {
	limiter := NewSlidingWindowRateLimiter(time.Second, 10000)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-key-%d", i%100)
			limiter.Allow(ctx, key)
			i++
		}
	})
}

func BenchmarkRateLimiterWithStats(b *testing.B) {
	limiter := NewTokenBucketRateLimiter(10000, 1000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i%100)
		limiter.Allow(ctx, key)

		// Periodically check stats
		if i%1000 == 0 {
			_ = limiter.Stats()
		}
	}
}
