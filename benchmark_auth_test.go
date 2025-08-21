// Package mcp - Authentication Performance Benchmarks
//
// This file contains comprehensive performance benchmarks for the MCP Go authentication system.
// It measures OAuth token operations, validation performance, concurrent access patterns,
// and provides detailed profiling for optimization opportunities.

package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Benchmark configuration for auth operations
var (
	tokenSizes        = []int{32, 64, 128, 256}  // Token lengths in bytes
	clientCounts      = []int{1, 10, 100, 1000}  // Number of clients
	concurrencyLevels = []int{1, 5, 10, 50, 100} // Concurrent operations
	cacheSizes        = []int{100, 1000, 10000}  // Cache capacities
)

// =============================================================================
// Token Creation Benchmarks
// =============================================================================

func BenchmarkTokenCreation(b *testing.B) {
	for _, size := range tokenSizes {
		b.Run(fmt.Sprintf("TokenSize_%d", size), func(b *testing.B) {
			benchmarkTokenCreation(b, size)
		})
	}
}

func benchmarkTokenCreation(b *testing.B, tokenSize int) {
	provider := NewMemoryOAuthProvider()

	// Register a test client
	client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		Scopes:       []string{"read", "write"},
	})
	if err != nil {
		b.Fatalf("Failed to register client: %v", err)
	}

	// Create authorization request
	authReq := &AuthorizationRequest{
		ResponseType: ResponseTypeCode,
		ClientID:     client.ClientID,
		RedirectURI:  client.RedirectURIs[0],
		Scope:        "read write",
	}

	b.ResetTimer()
	b.SetBytes(int64(tokenSize))

	for i := 0; i < b.N; i++ {
		ctx := context.Background()

		// Create authorization code
		authCode, err := provider.CreateAuthorizationCode(ctx, authReq)
		if err != nil {
			b.Errorf("Failed to create auth code: %v", err)
			continue
		}

		// Create access token
		_, err = provider.CreateAccessToken(ctx, authCode)
		if err != nil {
			b.Errorf("Failed to create access token: %v", err)
		}
	}
}

func BenchmarkTokenCreation_WithRotation(b *testing.B) {
	provider := NewMemoryOAuthProvider()

	// Register client
	client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		Scopes:       []string{"read", "write"},
	})
	if err != nil {
		b.Fatalf("Failed to register client: %v", err)
	}

	authReq := &AuthorizationRequest{
		ResponseType: ResponseTypeCode,
		ClientID:     client.ClientID,
		RedirectURI:  client.RedirectURIs[0],
		Scope:        "read write",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()

		// Create initial token
		authCode, err := provider.CreateAuthorizationCode(ctx, authReq)
		if err != nil {
			b.Errorf("Failed to create auth code: %v", err)
			continue
		}

		accessToken, err := provider.CreateAccessToken(ctx, authCode)
		if err != nil {
			b.Errorf("Failed to create access token: %v", err)
			continue
		}

		// Rotate token using refresh
		_, err = provider.RefreshAccessToken(ctx, accessToken.RefreshToken)
		if err != nil {
			b.Errorf("Failed to refresh token: %v", err)
		}
	}
}

// =============================================================================
// Token Validation Benchmarks
// =============================================================================

func BenchmarkTokenValidation(b *testing.B) {
	provider := NewMemoryOAuthProvider()
	tokens := setupTokensForValidation(b, provider, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		token := tokens[i%len(tokens)]
		ctx := context.Background()

		_, err := provider.ValidateAccessToken(ctx, token.AccessToken)
		if err != nil {
			b.Errorf("Token validation failed: %v", err)
		}
	}
}

func BenchmarkTokenValidation_WithCache(b *testing.B) {
	for _, cacheSize := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("CacheSize_%d", cacheSize), func(b *testing.B) {
			benchmarkTokenValidationWithCache(b, cacheSize)
		})
	}
}

func benchmarkTokenValidationWithCache(b *testing.B, cacheSize int) {
	provider := NewMemoryOAuthProvider()

	// Create middleware with cache
	authMiddleware := NewAuthenticationMiddleware(AuthConfig{
		Provider:     provider,
		CacheTimeout: 5 * time.Minute,
	})

	// Pre-populate tokens
	tokens := setupTokensForValidation(b, provider, cacheSize)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		token := tokens[i%len(tokens)]

		// Test cache hit scenario (90% of the time)
		if i%10 < 9 {
			// This should hit cache after first validation
			_, err := authMiddleware.validateTokenWithCache(context.Background(), token.AccessToken)
			if err != nil {
				b.Errorf("Cached token validation failed: %v", err)
			}
		} else {
			// Generate new token for cache miss (10% of the time)
			newToken := generateRandomToken(64)
			_, _ = authMiddleware.validateTokenWithCache(context.Background(), newToken)
		}
	}
}

func BenchmarkTokenValidation_CacheHitVsMiss(b *testing.B) {
	provider := NewMemoryOAuthProvider()
	authMiddleware := NewAuthenticationMiddleware(AuthConfig{
		Provider:     provider,
		CacheTimeout: 5 * time.Minute,
	})

	// Setup valid token
	tokens := setupTokensForValidation(b, provider, 1)
	validToken := tokens[0].AccessToken

	b.Run("CacheHit", func(b *testing.B) {
		// Prime the cache
		authMiddleware.validateTokenWithCache(context.Background(), validToken)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := authMiddleware.validateTokenWithCache(context.Background(), validToken)
			if err != nil {
				b.Errorf("Cache hit validation failed: %v", err)
			}
		}
	})

	b.Run("CacheMiss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Generate unique token each time to force cache miss
			invalidToken := generateRandomToken(64)
			_, _ = authMiddleware.validateTokenWithCache(context.Background(), invalidToken)
		}
	})
}

// =============================================================================
// Concurrent Token Operations
// =============================================================================

func BenchmarkConcurrentTokenOperations(b *testing.B) {
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			benchmarkConcurrentTokenOps(b, concurrency)
		})
	}
}

func benchmarkConcurrentTokenOps(b *testing.B, concurrency int) {
	provider := NewMemoryOAuthProvider()

	// Register client
	client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		Scopes:       []string{"read", "write"},
	})
	if err != nil {
		b.Fatalf("Failed to register client: %v", err)
	}

	authReq := &AuthorizationRequest{
		ResponseType: ResponseTypeCode,
		ClientID:     client.ClientID,
		RedirectURI:  client.RedirectURIs[0],
		Scope:        "read write",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for j := 0; j < concurrency; j++ {
			go func() {
				defer wg.Done()

				ctx := context.Background()
				authCode, err := provider.CreateAuthorizationCode(ctx, authReq)
				if err != nil {
					b.Errorf("Failed to create auth code: %v", err)
					return
				}

				token, err := provider.CreateAccessToken(ctx, authCode)
				if err != nil {
					b.Errorf("Failed to create access token: %v", err)
					return
				}

				_, err = provider.ValidateAccessToken(ctx, token.AccessToken)
				if err != nil {
					b.Errorf("Failed to validate token: %v", err)
				}
			}()
		}
		wg.Wait()
	}
}

func BenchmarkConcurrentTokenValidation(b *testing.B) {
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			benchmarkConcurrentTokenValidation(b, concurrency)
		})
	}
}

func benchmarkConcurrentTokenValidation(b *testing.B, concurrency int) {
	provider := NewMemoryOAuthProvider()
	tokens := setupTokensForValidation(b, provider, 100) // Use 100 tokens for variety

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for j := 0; j < concurrency; j++ {
			go func(workerID int) {
				defer wg.Done()

				// Each worker validates different tokens
				token := tokens[(i*concurrency+workerID)%len(tokens)]
				ctx := context.Background()

				_, err := provider.ValidateAccessToken(ctx, token.AccessToken)
				if err != nil {
					b.Errorf("Worker %d token validation failed: %v", workerID, err)
				}
			}(j)
		}
		wg.Wait()
	}
}

// =============================================================================
// Token Rotation Performance
// =============================================================================

func BenchmarkTokenRotation(b *testing.B) {
	provider := NewMemoryOAuthProvider()

	// Setup initial tokens
	tokens := setupTokensForValidation(b, provider, 100)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		token := tokens[i%len(tokens)]
		ctx := context.Background()

		// Rotate token
		newToken, err := provider.RefreshAccessToken(ctx, token.RefreshToken)
		if err != nil {
			b.Errorf("Token rotation failed: %v", err)
			continue
		}

		// Update tokens array to use new token
		tokens[i%len(tokens)] = newToken
	}
}

func BenchmarkTokenRotation_HighFrequency(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping high-frequency rotation test in short mode")
	}

	provider := NewMemoryOAuthProvider()
	tokens := setupTokensForValidation(b, provider, 10)

	// Simulate high-frequency rotation (every 100ms)
	rotationInterval := 100 * time.Millisecond

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		token := tokens[i%len(tokens)]
		ctx := context.Background()

		start := time.Now()
		newToken, err := provider.RefreshAccessToken(ctx, token.RefreshToken)
		duration := time.Since(start)

		if err != nil {
			b.Errorf("High-frequency token rotation failed: %v", err)
			continue
		}

		// Update token for next iteration
		tokens[i%len(tokens)] = newToken

		// Report timing metrics
		if duration > rotationInterval {
			b.Logf("Token rotation took %v (target: %v)", duration, rotationInterval)
		}
	}
}

// =============================================================================
// Signature Verification Speed
// =============================================================================

func BenchmarkPKCEVerification(b *testing.B) {
	for _, size := range []int{32, 64, 128} {
		b.Run(fmt.Sprintf("VerifierSize_%d", size), func(b *testing.B) {
			benchmarkPKCEVerification(b, size)
		})
	}
}

func benchmarkPKCEVerification(b *testing.B, verifierSize int) {
	// Pre-generate verifiers and challenges
	verifiers := make([]string, 100)
	challenges := make([]string, 100)

	for i := 0; i < 100; i++ {
		verifier, challenge, err := GeneratePKCEChallenge()
		if err != nil {
			b.Fatalf("Failed to generate PKCE challenge: %v", err)
		}
		verifiers[i] = verifier
		challenges[i] = challenge
	}

	b.ResetTimer()
	b.SetBytes(int64(verifierSize))

	for i := 0; i < b.N; i++ {
		verifier := verifiers[i%len(verifiers)]
		challenge := challenges[i%len(challenges)]

		valid := ValidatePKCEChallenge(verifier, challenge)
		if !valid {
			b.Errorf("PKCE validation failed for iteration %d", i)
		}
	}
}

func BenchmarkAuthorizationHeaderParsing(b *testing.B) {
	headers := []string{
		"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"Bearer " + generateRandomToken(64),
		"Bearer " + generateRandomToken(128),
		"Bearer " + generateRandomToken(256),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		header := headers[i%len(headers)]

		token, err := ParseAuthorizationHeader(header)
		if err != nil {
			b.Errorf("Failed to parse authorization header: %v", err)
		}
		if token == "" {
			b.Errorf("Empty token extracted from header")
		}
	}
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

func BenchmarkAuthMemoryAllocation(b *testing.B) {
	provider := NewMemoryOAuthProvider()

	b.Run("ClientRegistration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			clientInfo := &OAuthClientInfo{
				ClientID:     fmt.Sprintf("client-%d", i),
				ClientSecret: generateRandomToken(64),
				RedirectURIs: []string{"http://localhost:8080/callback"},
				Scopes:       []string{"read", "write", "admin"},
			}

			_, err := provider.RegisterClient(context.Background(), clientInfo)
			if err != nil {
				b.Errorf("Client registration failed: %v", err)
			}
		}
	})

	b.Run("TokenCreation", func(b *testing.B) {
		// Pre-register client
		client, _ := provider.RegisterClient(context.Background(), &OAuthClientInfo{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURIs: []string{"http://localhost:8080/callback"},
		})

		authReq := &AuthorizationRequest{
			ResponseType: ResponseTypeCode,
			ClientID:     client.ClientID,
			RedirectURI:  client.RedirectURIs[0],
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			authCode, err := provider.CreateAuthorizationCode(context.Background(), authReq)
			if err != nil {
				b.Errorf("Auth code creation failed: %v", err)
				continue
			}

			_, err = provider.CreateAccessToken(context.Background(), authCode)
			if err != nil {
				b.Errorf("Access token creation failed: %v", err)
			}
		}
	})
}

func BenchmarkAuthMemoryPressure(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory pressure test in short mode")
	}

	provider := NewMemoryOAuthProvider()

	// Force GC to establish baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create many tokens to simulate memory pressure
	client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "stress-client",
		ClientSecret: "stress-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	})
	if err != nil {
		b.Fatalf("Failed to register client: %v", err)
	}

	authReq := &AuthorizationRequest{
		ResponseType: ResponseTypeCode,
		ClientID:     client.ClientID,
		RedirectURI:  client.RedirectURIs[0],
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create many tokens
		for j := 0; j < 100; j++ {
			authCode, err := provider.CreateAuthorizationCode(context.Background(), authReq)
			if err != nil {
				b.Errorf("Auth code creation failed: %v", err)
				continue
			}

			_, err = provider.CreateAccessToken(context.Background(), authCode)
			if err != nil {
				b.Errorf("Access token creation failed: %v", err)
			}
		}

		// Periodic GC to measure pressure
		if i%10 == 0 {
			runtime.GC()
		}
	}

	// Measure final memory usage
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
}

// =============================================================================
// Stress Tests
// =============================================================================

func BenchmarkAuthStress_MultiClient(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping stress test in short mode")
	}

	for _, clientCount := range clientCounts {
		b.Run(fmt.Sprintf("Clients_%d", clientCount), func(b *testing.B) {
			benchmarkAuthStressMultiClient(b, clientCount)
		})
	}
}

func benchmarkAuthStressMultiClient(b *testing.B, clientCount int) {
	provider := NewMemoryOAuthProvider()

	// Register multiple clients
	clients := make([]*OAuthClientInfo, clientCount)
	for i := 0; i < clientCount; i++ {
		client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
			ClientID:     fmt.Sprintf("client-%d", i),
			ClientSecret: generateRandomToken(64),
			RedirectURIs: []string{fmt.Sprintf("http://localhost:808%d/callback", i%10)},
			Scopes:       []string{"read", "write"},
		})
		if err != nil {
			b.Fatalf("Failed to register client %d: %v", i, err)
		}
		clients[i] = client
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup

		// Each iteration, each client creates and validates a token
		for j, client := range clients {
			wg.Add(1)
			go func(clientIdx int, clientInfo *OAuthClientInfo) {
				defer wg.Done()

				authReq := &AuthorizationRequest{
					ResponseType: ResponseTypeCode,
					ClientID:     clientInfo.ClientID,
					RedirectURI:  clientInfo.RedirectURIs[0],
					Scope:        "read write",
				}

				ctx := context.Background()
				authCode, err := provider.CreateAuthorizationCode(ctx, authReq)
				if err != nil {
					b.Errorf("Client %d auth code creation failed: %v", clientIdx, err)
					return
				}

				token, err := provider.CreateAccessToken(ctx, authCode)
				if err != nil {
					b.Errorf("Client %d token creation failed: %v", clientIdx, err)
					return
				}

				_, err = provider.ValidateAccessToken(ctx, token.AccessToken)
				if err != nil {
					b.Errorf("Client %d token validation failed: %v", clientIdx, err)
				}
			}(j, client)
		}
		wg.Wait()
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupTokensForValidation(b *testing.B, provider *MemoryOAuthProvider, count int) []*AccessToken {
	b.Helper()

	// Register client
	client, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		ClientID:     "bench-client",
		ClientSecret: "bench-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		Scopes:       []string{"read", "write"},
	})
	if err != nil {
		b.Fatalf("Failed to register client: %v", err)
	}

	tokens := make([]*AccessToken, count)

	for i := 0; i < count; i++ {
		authReq := &AuthorizationRequest{
			ResponseType: ResponseTypeCode,
			ClientID:     client.ClientID,
			RedirectURI:  client.RedirectURIs[0],
			Scope:        "read write",
		}

		authCode, err := provider.CreateAuthorizationCode(context.Background(), authReq)
		if err != nil {
			b.Fatalf("Failed to create auth code %d: %v", i, err)
		}

		token, err := provider.CreateAccessToken(context.Background(), authCode)
		if err != nil {
			b.Fatalf("Failed to create access token %d: %v", i, err)
		}

		tokens[i] = token
	}

	return tokens
}

func generateRandomToken(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based generation
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
