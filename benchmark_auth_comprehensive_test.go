// Package mcp - Comprehensive Authentication Performance Analysis
//
// This file contains comprehensive end-to-end benchmarks that test the complete
// authentication pipeline including the recent race condition fixes and security improvements.

package mcp

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
)

// BenchmarkAuthPipeline_Complete tests the complete auth pipeline
func BenchmarkAuthPipeline_Complete(b *testing.B) {
	// Test both regular and secure providers
	testCases := []struct {
		name      string
		setupFunc func() (OAuthProvider, func())
	}{
		{
			name: "MemoryProvider",
			setupFunc: func() (OAuthProvider, func()) {
				provider := NewMemoryOAuthProvider()
				return provider, func() {}
			},
		},
		{
			name: "SecureProvider",
			setupFunc: func() (OAuthProvider, func()) {
				baseProvider := NewMemoryOAuthProvider()
				encryptionKey := []byte("test-key-32-bytes-long!!!!!!!!!!!")
				secureProvider, _ := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)
				return secureProvider, func() {}
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			provider, cleanup := tc.setupFunc()
			defer cleanup()

			// Pre-register client
			client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
				ClientID:     "benchmark-client",
				ClientSecret: "benchmark-secret",
				RedirectURIs: []string{"http://localhost:8080/callback"},
				Scopes:       []string{"read", "write"},
			})
			if err != nil {
				b.Fatalf("Failed to register client: %v", err)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Complete OAuth flow
				authReq := &AuthorizationRequest{
					ResponseType: ResponseTypeCode,
					ClientID:     client.ClientID,
					RedirectURI:  client.RedirectURIs[0],
					Scope:        "read write",
				}

				// Create authorization code
				authCode, err := provider.CreateAuthorizationCode(context.Background(), authReq)
				if err != nil {
					b.Errorf("Failed to create auth code: %v", err)
					continue
				}

				// Create access token
				token, err := provider.CreateAccessToken(context.Background(), authCode)
				if err != nil {
					b.Errorf("Failed to create access token: %v", err)
					continue
				}

				// Validate access token
				_, err = provider.ValidateAccessToken(context.Background(), token.AccessToken)
				if err != nil {
					b.Errorf("Failed to validate access token: %v", err)
					continue
				}

				// Optional: Refresh token (every 10th iteration to reduce overhead)
				if i%10 == 0 && token.RefreshToken != "" {
					_, err = provider.RefreshAccessToken(context.Background(), token.RefreshToken)
					if err != nil {
						b.Errorf("Failed to refresh token: %v", err)
					}
				}
			}
		})
	}
}

// BenchmarkAuthPipeline_ConcurrentUsers simulates multiple concurrent users
func BenchmarkAuthPipeline_ConcurrentUsers(b *testing.B) {
	userCounts := []int{1, 10, 50, 100}

	for _, userCount := range userCounts {
		b.Run(fmt.Sprintf("Users_%d", userCount), func(b *testing.B) {
			provider := NewMemoryOAuthProvider()

			// Pre-register clients for all users
			clients := make([]*OAuthClientInfo, userCount)
			for i := 0; i < userCount; i++ {
				client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
					ClientID:     fmt.Sprintf("client-%d", i),
					ClientSecret: fmt.Sprintf("secret-%d", i),
					RedirectURIs: []string{fmt.Sprintf("http://localhost:808%d/callback", i%10)},
					Scopes:       []string{"read", "write"},
				})
				if err != nil {
					b.Fatalf("Failed to register client %d: %v", i, err)
				}
				clients[i] = client
			}

			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				userID := 0
				for pb.Next() {
					client := clients[userID%len(clients)]
					userID++

					// Create auth code
					authReq := &AuthorizationRequest{
						ResponseType: ResponseTypeCode,
						ClientID:     client.ClientID,
						RedirectURI:  client.RedirectURIs[0],
						Scope:        "read write",
					}

					authCode, err := provider.CreateAuthorizationCode(context.Background(), authReq)
					if err != nil {
						b.Errorf("Failed to create auth code: %v", err)
						continue
					}

					// Create access token
					token, err := provider.CreateAccessToken(context.Background(), authCode)
					if err != nil {
						b.Errorf("Failed to create access token: %v", err)
						continue
					}

					// Validate access token
					_, err = provider.ValidateAccessToken(context.Background(), token.AccessToken)
					if err != nil {
						b.Errorf("Failed to validate access token: %v", err)
					}
				}
			})
		})
	}
}

// BenchmarkAuthPipeline_Performance measures key performance metrics
func BenchmarkAuthPipeline_Performance(b *testing.B) {
	b.Run("PerformanceAnalysis", func(b *testing.B) {
		provider := NewMemoryOAuthProvider()

		// Register client
		client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
			ClientID:     "perf-client",
			ClientSecret: "perf-secret",
			RedirectURIs: []string{"http://localhost:8080/callback"},
		})

		// Measure different operations separately
		authReq := &AuthorizationRequest{
			ResponseType: ResponseTypeCode,
			ClientID:     client.ClientID,
			RedirectURI:  client.RedirectURIs[0],
		}

		b.Run("AuthCodeCreation", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = provider.CreateAuthorizationCode(context.Background(), authReq)
			}
		})

		// Pre-create auth codes for token creation benchmark
		var authCodes []*AuthorizationCode
		for i := 0; i < 1000; i++ {
			code, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
			authCodes = append(authCodes, code)
		}

		b.Run("TokenCreation", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				authCode := authCodes[i%len(authCodes)]
				_, _ = provider.CreateAccessToken(context.Background(), authCode)
			}
		})

		// Pre-create tokens for validation benchmark
		var tokens []string
		for i := 0; i < 1000; i++ {
			authCode := authCodes[i%len(authCodes)]
			token, _ := provider.CreateAccessToken(context.Background(), authCode)
			tokens = append(tokens, token.AccessToken)
		}

		b.Run("TokenValidation", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				token := tokens[i%len(tokens)]
				_, _ = provider.ValidateAccessToken(context.Background(), token)
			}
		})
	})
}

// BenchmarkAuthSecurity_CompareProviders compares regular vs secure providers
func BenchmarkAuthSecurity_CompareProviders(b *testing.B) {
	operations := []struct {
		name string
		fn   func(provider OAuthProvider, client *OAuthClientInfo) error
	}{
		{
			name: "CreateToken",
			fn: func(provider OAuthProvider, client *OAuthClientInfo) error {
				authReq := &AuthorizationRequest{
					ResponseType: ResponseTypeCode,
					ClientID:     client.ClientID,
					RedirectURI:  client.RedirectURIs[0],
				}
				authCode, err := provider.CreateAuthorizationCode(context.Background(), authReq)
				if err != nil {
					return err
				}
				_, err = provider.CreateAccessToken(context.Background(), authCode)
				return err
			},
		},
		{
			name: "ValidateToken",
			fn: func(provider OAuthProvider, client *OAuthClientInfo) error {
				// Pre-create a token
				authReq := &AuthorizationRequest{
					ResponseType: ResponseTypeCode,
					ClientID:     client.ClientID,
					RedirectURI:  client.RedirectURIs[0],
				}
				authCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
				token, _ := provider.CreateAccessToken(context.Background(), authCode)

				_, err := provider.ValidateAccessToken(context.Background(), token.AccessToken)
				return err
			},
		},
	}

	providers := []struct {
		name      string
		setupFunc func() OAuthProvider
	}{
		{
			name: "Memory",
			setupFunc: func() OAuthProvider {
				return NewMemoryOAuthProvider()
			},
		},
		{
			name: "Secure",
			setupFunc: func() OAuthProvider {
				baseProvider := NewMemoryOAuthProvider()
				encryptionKey := []byte("benchmark-key-32-bytes-long!!!!!")
				secureProvider, _ := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)
				return secureProvider
			},
		},
	}

	for _, op := range operations {
		for _, prov := range providers {
			b.Run(fmt.Sprintf("%s_%s", op.name, prov.name), func(b *testing.B) {
				provider := prov.setupFunc()

				client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
					ClientID:     "bench-client",
					ClientSecret: "bench-secret",
					RedirectURIs: []string{"http://localhost:8080/callback"},
				})

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := op.fn(provider, client); err != nil {
						b.Errorf("Operation failed: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkAuthRegressionDetection tests for performance regressions
func BenchmarkAuthRegressionDetection(b *testing.B) {
	// This benchmark establishes performance baselines for regression detection

	provider := NewMemoryOAuthProvider()
	client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "regression-client",
		ClientSecret: "regression-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	})

	// Baseline metrics for regression detection
	baselineMetrics := map[string]struct {
		maxLatency   time.Duration
		minOpsPerSec float64
	}{
		"TokenCreation":   {maxLatency: 5 * time.Millisecond, minOpsPerSec: 500},
		"TokenValidation": {maxLatency: 1 * time.Millisecond, minOpsPerSec: 10000},
		"ConcurrentOps":   {maxLatency: 10 * time.Millisecond, minOpsPerSec: 100},
	}

	for metricName, baseline := range baselineMetrics {
		b.Run(metricName, func(b *testing.B) {
			start := time.Now()

			switch metricName {
			case "TokenCreation":
				for i := 0; i < b.N; i++ {
					authReq := &AuthorizationRequest{
						ResponseType: ResponseTypeCode,
						ClientID:     client.ClientID,
						RedirectURI:  client.RedirectURIs[0],
					}
					authCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
					_, _ = provider.CreateAccessToken(context.Background(), authCode)
				}
			case "TokenValidation":
				// Pre-create token
				authReq := &AuthorizationRequest{
					ResponseType: ResponseTypeCode,
					ClientID:     client.ClientID,
					RedirectURI:  client.RedirectURIs[0],
				}
				authCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
				token, _ := provider.CreateAccessToken(context.Background(), authCode)

				for i := 0; i < b.N; i++ {
					_, _ = provider.ValidateAccessToken(context.Background(), token.AccessToken)
				}
			case "ConcurrentOps":
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						authReq := &AuthorizationRequest{
							ResponseType: ResponseTypeCode,
							ClientID:     client.ClientID,
							RedirectURI:  client.RedirectURIs[0],
						}
						authCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
						token, _ := provider.CreateAccessToken(context.Background(), authCode)
						_, _ = provider.ValidateAccessToken(context.Background(), token.AccessToken)
					}
				})
			}

			elapsed := time.Since(start)
			avgLatency := elapsed / time.Duration(b.N)
			opsPerSec := float64(b.N) / elapsed.Seconds()

			// Report metrics
			b.ReportMetric(float64(avgLatency.Nanoseconds())/1e6, "avg-latency-ms")
			b.ReportMetric(opsPerSec, "ops/sec")

			// Check for regressions
			if avgLatency > baseline.maxLatency {
				b.Logf("REGRESSION WARNING: Average latency %.2fms exceeds baseline %.2fms",
					float64(avgLatency.Nanoseconds())/1e6, float64(baseline.maxLatency.Nanoseconds())/1e6)
			}
			if opsPerSec < baseline.minOpsPerSec {
				b.Logf("REGRESSION WARNING: Performance %.2f ops/sec below baseline %.2f ops/sec",
					opsPerSec, baseline.minOpsPerSec)
			}
		})
	}
}

// BenchmarkAuthProfile_MemoryUsage profiles memory usage patterns
func BenchmarkAuthProfile_MemoryUsage(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory profiling in short mode")
	}

	// Force GC and capture baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	provider := NewMemoryOAuthProvider()
	client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "memory-client",
		ClientSecret: "memory-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	})

	b.ResetTimer()

	// Create many tokens to analyze memory patterns
	for i := 0; i < b.N; i++ {
		authReq := &AuthorizationRequest{
			ResponseType: ResponseTypeCode,
			ClientID:     client.ClientID,
			RedirectURI:  client.RedirectURIs[0],
		}

		authCode, err := provider.CreateAuthorizationCode(context.Background(), authReq)
		if err != nil {
			b.Errorf("Failed to create auth code: %v", err)
			continue
		}

		_, err = provider.CreateAccessToken(context.Background(), authCode)
		if err != nil {
			b.Errorf("Failed to create access token: %v", err)
		}

		// Sample memory usage every 100 operations
		if i%100 == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			b.ReportMetric(float64(m.Alloc-m1.Alloc), "current-alloc-bytes")
			b.ReportMetric(float64(m.TotalAlloc-m1.TotalAlloc), "total-alloc-bytes")
		}
	}

	// Final memory measurement
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	avgAllocPerOp := float64(m2.TotalAlloc-m1.TotalAlloc) / float64(b.N)
	finalAllocated := float64(m2.Alloc - m1.Alloc)

	b.ReportMetric(avgAllocPerOp, "avg-bytes/op")
	b.ReportMetric(finalAllocated, "final-allocated-bytes")

	// Memory usage warnings
	const maxExpectedAllocPerOp = 2000 // 2KB per operation
	if avgAllocPerOp > maxExpectedAllocPerOp {
		b.Logf("MEMORY WARNING: Average %.2f bytes/op exceeds expected %d bytes/op",
			avgAllocPerOp, maxExpectedAllocPerOp)
	}
}

// BenchmarkAuthProfile_CPUUsage profiles CPU usage patterns
func BenchmarkAuthProfile_CPUUsage(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping CPU profiling in short mode")
	}

	// Test CPU-intensive operations
	operations := []struct {
		name string
		fn   func() error
	}{
		{
			name: "PKCEGeneration",
			fn: func() error {
				_, _, err := GeneratePKCEChallenge()
				return err
			},
		},
		{
			name: "TokenEncryption",
			fn: func() error {
				provider := NewMemoryOAuthProvider()
				encryptionKey := []byte("cpu-test-key-32-bytes-long!!!!!!!!")
				secureProvider, _ := NewSecureOAuthProvider(provider, encryptionKey, nil)

				client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
					ClientID:     "cpu-client",
					ClientSecret: "cpu-secret",
					RedirectURIs: []string{"http://localhost:8080/callback"},
				})

				authReq := &AuthorizationRequest{
					ResponseType: ResponseTypeCode,
					ClientID:     client.ClientID,
					RedirectURI:  client.RedirectURIs[0],
				}

				authCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
				_, err := secureProvider.CreateAccessToken(context.Background(), authCode)
				return err
			},
		},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			start := time.Now()
			for i := 0; i < b.N; i++ {
				if err := op.fn(); err != nil {
					b.Errorf("Operation failed: %v", err)
				}
			}
			elapsed := time.Since(start)

			cpuTime := elapsed.Nanoseconds()
			avgCPUTimePerOp := float64(cpuTime) / float64(b.N) / 1e6 // Convert to milliseconds

			b.ReportMetric(avgCPUTimePerOp, "avg-cpu-ms/op")

			// CPU usage warnings (operations should be fast)
			const maxExpectedCPUTimeMs = 1.0 // 1ms per operation
			if avgCPUTimePerOp > maxExpectedCPUTimeMs {
				b.Logf("CPU WARNING: Average %.2fms/op exceeds expected %.2fms/op",
					avgCPUTimePerOp, maxExpectedCPUTimeMs)
			}
		})
	}
}

// BenchmarkAuthBottleneckAnalysis identifies performance bottlenecks
func BenchmarkAuthBottleneckAnalysis(b *testing.B) {
	// Test individual components to identify bottlenecks
	components := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "RandomGeneration",
			fn: func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, _ = generateRandomString(64)
				}
			},
		},
		{
			name: "MapOperations",
			fn: func(b *testing.B) {
				provider := NewMemoryOAuthProvider()
				client := &OAuthClientInfo{
					ClientID:     "bottleneck-client",
					ClientSecret: "bottleneck-secret",
					RedirectURIs: []string{"http://localhost:8080/callback"},
				}

				for i := 0; i < b.N; i++ {
					// Test map write performance (now with mutex)
					provider.RegisterClient(context.Background(), client)
				}
			},
		},
		{
			name: "MutexContention",
			fn: func(b *testing.B) {
				provider := NewMemoryOAuthProvider()
				client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
					ClientID:     "mutex-client",
					ClientSecret: "mutex-secret",
					RedirectURIs: []string{"http://localhost:8080/callback"},
				})

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						// Test concurrent access with mutex protection
						authReq := &AuthorizationRequest{
							ResponseType: ResponseTypeCode,
							ClientID:     client.ClientID,
							RedirectURI:  client.RedirectURIs[0],
						}
						authCode, _ := provider.CreateAuthorizationCode(context.Background(), authReq)
						_, _ = provider.CreateAccessToken(context.Background(), authCode)
					}
				})
			},
		},
	}

	for _, comp := range components {
		b.Run(comp.name, comp.fn)
	}
}

// init function to enable debug mode for comprehensive testing
func init() {
	if os.Getenv("BENCHMARK_DEBUG") == "1" {
		os.Setenv("DEBUG_AUTH", "1")
	}
}
