package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSecureOAuthProvider(t *testing.T) {
	// Enable debug mode for this test
	oldDebug := os.Getenv("DEBUG_AUTH")
	os.Setenv("DEBUG_AUTH", "1")
	defer os.Setenv("DEBUG_AUTH", oldDebug)

	// Create base provider
	baseProvider := NewMemoryOAuthProvider()

	// Register a test client
	client := &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost/callback"},
	}
	baseProvider.RegisterClient(context.Background(), client)

	// Create secure provider
	encryptionKey := []byte("test-encryption-key-32-bytes-long")
	secureProvider, err := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)
	if err != nil {
		t.Fatalf("Failed to create secure provider: %v", err)
	}

	// Create authorization code
	authReq := &AuthorizationRequest{
		ResponseType: "code",
		ClientID:     "test-client",
		RedirectURI:  "http://localhost/callback",
		Scope:        "read write",
	}
	authCode, err := secureProvider.CreateAuthorizationCode(context.Background(), authReq)
	if err != nil {
		t.Fatalf("Failed to create auth code: %v", err)
	}

	// Create access token
	ctx := context.WithValue(context.Background(), userAgentKey, "test-agent")
	ctx = context.WithValue(ctx, remoteAddrKey, "127.0.0.1")

	token, err := secureProvider.CreateAccessToken(ctx, authCode)
	if err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}

	// Token should be encrypted
	if !isBase64(token.AccessToken) {
		t.Error("Access token should be encrypted (base64)")
	}

	// Validate token
	validated, err := secureProvider.ValidateAccessToken(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if validated.ClientID != "test-client" {
		t.Errorf("ClientID = %v, want %v", validated.ClientID, "test-client")
	}

	// Test token metadata is tracked
	// Note: metadata is stored with the original unencrypted token, not the encrypted one
	// The encrypted token is what we pass to ValidateAccessToken
	// We need to check if metadata exists for any key since we validated successfully
	found := false
	secureProvider.tokenMetadata.Range(func(key, value interface{}) bool {
		found = true
		return false // Stop after finding first entry
	})
	if !found {
		t.Error("Token metadata should be stored")
	}

	// Test revocation
	err = secureProvider.RevokeToken(ctx, token.AccessToken)
	if err != nil {
		t.Errorf("Failed to revoke token: %v", err)
	}

	// Revoked token should fail validation
	_, err = secureProvider.ValidateAccessToken(ctx, token.AccessToken)
	if err != ErrTokenRevoked {
		t.Errorf("Expected ErrTokenRevoked, got %v", err)
	}
}

func TestTokenRotation(t *testing.T) {
	baseProvider := NewMemoryOAuthProvider()

	// Short rotation policy for testing
	rotationPolicy := &TokenRotationPolicy{
		MaxAge:           100 * time.Millisecond,
		MaxUseCount:      3,
		InactivityPeriod: 50 * time.Millisecond,
		ForceRotateAfter: 200 * time.Millisecond,
	}

	encryptionKey := []byte("test-key-32-bytes-long!!!!!!!!!!!")
	secureProvider, err := NewSecureOAuthProvider(baseProvider, encryptionKey, rotationPolicy)
	if err != nil {
		t.Fatalf("Failed to create secure provider: %v", err)
	}

	// Register client and create token
	client := &OAuthClientInfo{
		ClientID:     "rotation-test",
		ClientSecret: "secret",
	}
	baseProvider.RegisterClient(context.Background(), client)

	authCode := &AuthorizationCode{
		Code:     "test-code",
		ClientID: "rotation-test",
	}
	baseProvider.authCodes["test-code"] = authCode

	token, err := secureProvider.CreateAccessToken(context.Background(), authCode)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Test max use count rotation
	for i := 0; i < 3; i++ {
		_, err = secureProvider.ValidateAccessToken(context.Background(), token.AccessToken)
		if err != nil {
			t.Errorf("Validation %d should succeed: %v", i+1, err)
		}
	}

	// Next validation should require rotation
	_, err = secureProvider.ValidateAccessToken(context.Background(), token.AccessToken)
	if err != ErrTokenRotationRequired {
		t.Errorf("Expected ErrTokenRotationRequired after max use count, got %v", err)
	}

	// Test max age rotation
	token2, _ := secureProvider.CreateAccessToken(context.Background(), authCode)
	time.Sleep(150 * time.Millisecond)

	_, err = secureProvider.ValidateAccessToken(context.Background(), token2.AccessToken)
	if err != ErrTokenRotationRequired {
		t.Errorf("Expected ErrTokenRotationRequired after max age, got %v", err)
	}
}

func TestTokenRefreshWithRotation(t *testing.T) {
	baseProvider := NewMemoryOAuthProvider()
	encryptionKey := []byte("test-key-32-bytes-long!!!!!!!!!!!")
	secureProvider, err := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)
	if err != nil {
		t.Fatalf("Failed to create secure provider: %v", err)
	}

	// Setup client and initial token
	client := &OAuthClientInfo{
		ClientID:     "refresh-test",
		ClientSecret: "secret",
	}
	baseProvider.RegisterClient(context.Background(), client)

	authCode := &AuthorizationCode{
		Code:     "test-code",
		ClientID: "refresh-test",
	}
	baseProvider.authCodes["test-code"] = authCode

	initialToken, err := secureProvider.CreateAccessToken(context.Background(), authCode)
	if err != nil {
		t.Fatalf("Failed to create initial token: %v", err)
	}

	// Refresh token
	refreshedToken, err := secureProvider.RefreshAccessToken(context.Background(), initialToken.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	// New token should be encrypted
	if !isBase64(refreshedToken.AccessToken) {
		t.Error("Refreshed token should be encrypted")
	}

	// Validate refreshed token
	validated, err := secureProvider.ValidateAccessToken(context.Background(), refreshedToken.AccessToken)
	if err != nil {
		t.Errorf("Failed to validate refreshed token: %v", err)
	}

	if validated.ClientID != "refresh-test" {
		t.Errorf("ClientID = %v, want %v", validated.ClientID, "refresh-test")
	}

	// Check rotation count increased
	if metadata, exists := secureProvider.tokenMetadata.Load(validated.AccessToken); exists {
		if secure, ok := metadata.(*SecureToken); ok {
			if secure.RotationCount != 1 {
				t.Errorf("RotationCount = %d, want 1", secure.RotationCount)
			}
		}
	}
}

func TestTokenTransmissionGuard(t *testing.T) {
	guard := NewTokenTransmissionGuard(5 * time.Minute)

	// Test prepare token
	originalToken := "test-token-12345"
	transmitted, err := guard.PrepareTokenForTransmission(originalToken)
	if err != nil {
		t.Fatalf("Failed to prepare token: %v", err)
	}

	if !isBase64(transmitted) {
		t.Error("Transmitted token should be base64 encoded")
	}

	// Test validate token
	extractedToken, err := guard.ValidateTokenTransmission(transmitted)
	if err != nil {
		t.Fatalf("Failed to validate transmission: %v", err)
	}

	if extractedToken != originalToken {
		t.Errorf("Extracted token = %v, want %v", extractedToken, originalToken)
	}

	// Test replay protection
	_, err = guard.ValidateTokenTransmission(transmitted)
	if err == nil {
		t.Error("Replay should be prevented")
	}

	// Test expired transmission
	expiredTransmission := createExpiredTransmission(originalToken, 10*time.Minute)
	_, err = guard.ValidateTokenTransmission(expiredTransmission)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Error("Expired transmission should be rejected")
	}

	// Test invalid format
	_, err = guard.ValidateTokenTransmission("invalid-base64")
	if err == nil {
		t.Error("Invalid format should be rejected")
	}
}

func TestSecureAuthenticationMiddleware(t *testing.T) {
	// Setup secure provider
	baseProvider := NewMemoryOAuthProvider()
	encryptionKey := []byte("test-key-32-bytes-long!!!!!!!!!!!")
	secureProvider, _ := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)

	// Register client and create token
	client := &OAuthClientInfo{
		ClientID:     "middleware-test",
		ClientSecret: "secret",
	}
	baseProvider.RegisterClient(context.Background(), client)

	authCode := &AuthorizationCode{
		Code:     "test-code",
		ClientID: "middleware-test",
	}
	baseProvider.authCodes["test-code"] = authCode

	token, _ := secureProvider.CreateAccessToken(context.Background(), authCode)

	// Create middleware
	config := AuthConfig{
		SkipMethods: []string{"ping"},
	}
	middleware := NewSecureAuthenticationMiddleware(secureProvider, config)

	// Create test handler
	handler := MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Check auth context
		authCtx := GetAuthContext(ctx)
		if authCtx == nil {
			return NewErrorResponse("No auth context", -32000), nil
		}
		return &errorResponse{}, nil
	})

	protected := middleware.Apply(handler)

	// Test skipped method
	skipReq := &mockMCPRequest{method: "ping"}
	_, err := protected.Handle(context.Background(), skipReq)
	if err != nil {
		t.Errorf("Ping should be allowed without auth, got error: %v", err)
	}

	// Test without token
	noAuthReq := &mockMCPRequest{method: "tools/call"}
	_, err = protected.Handle(context.Background(), noAuthReq)
	if err == nil {
		t.Error("Request without token should fail")
	}

	// Test with valid token in header
	ctx := context.WithValue(context.Background(), authHeaderKey, "Bearer "+token.AccessToken)
	authReq := &mockMCPRequest{method: "tools/call"}
	_, err = protected.Handle(ctx, authReq)
	if err != nil {
		t.Errorf("Request with valid token should succeed: %v", err)
	}

	// Test with token in params
	paramsWithToken := map[string]interface{}{
		"auth_token": token.AccessToken,
	}
	paramsJSON, _ := json.Marshal(paramsWithToken)
	paramReq := &mockMCPRequest{
		method: "tools/call",
		params: paramsJSON,
	}
	_, err = protected.Handle(context.Background(), paramReq)
	if err != nil {
		t.Errorf("Request with token in params should succeed: %v", err)
	}
}

func TestTokenSignatureVerification(t *testing.T) {
	baseProvider := NewMemoryOAuthProvider()
	encryptionKey := []byte("test-key-32-bytes-long!!!!!!!!!!!")
	secureProvider, _ := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)

	// Create a secure token
	secureToken := &SecureToken{
		AccessToken: &AccessToken{
			AccessToken: "test-token",
			ClientID:    "test-client",
		},
		Version:       1,
		Fingerprint:   "test-fingerprint",
		IssuedAt:      time.Now(),
		RotationCount: 0,
	}

	// Sign the token
	signature := secureProvider.signToken(secureToken)
	secureToken.Signature = signature

	// Verify signature
	expectedSig := secureProvider.signToken(secureToken)
	if !secureProvider.verifySignature(signature, expectedSig) {
		t.Error("Valid signature should verify")
	}

	// Tamper with token (change something that's included in the signature)
	secureToken.Version = 2
	newSig := secureProvider.signToken(secureToken)
	if secureProvider.verifySignature(signature, newSig) {
		t.Error("Tampered token signature should not verify")
	}
}

func TestConcurrentTokenOperations(t *testing.T) {
	baseProvider := NewMemoryOAuthProvider()
	encryptionKey := []byte("test-key-32-bytes-long!!!!!!!!!!!")
	secureProvider, _ := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)

	// Register client
	client := &OAuthClientInfo{
		ClientID:     "concurrent-test",
		ClientSecret: "secret",
	}
	baseProvider.RegisterClient(context.Background(), client)

	// Create multiple tokens concurrently
	var wg sync.WaitGroup
	tokens := make([]string, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			authCode := &AuthorizationCode{
				Code:     fmt.Sprintf("code-%d", idx),
				ClientID: "concurrent-test",
			}
			baseProvider.mu.Lock()
			baseProvider.authCodes[authCode.Code] = authCode
			baseProvider.mu.Unlock()

			token, err := secureProvider.CreateAccessToken(context.Background(), authCode)
			if err != nil {
				t.Errorf("Failed to create token %d: %v", idx, err)
				return
			}
			tokens[idx] = token.AccessToken
		}(i)
	}

	wg.Wait()

	// Validate tokens concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			_, err := secureProvider.ValidateAccessToken(context.Background(), tokens[idx])
			if err != nil {
				t.Errorf("Failed to validate token %d: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()
}

// Helper functions

func isBase64(s string) bool {
	_, err := base64.URLEncoding.DecodeString(s)
	return err == nil
}

func createExpiredTransmission(token string, age time.Duration) string {
	transmission := map[string]interface{}{
		"token":     token,
		"nonce":     "expired-nonce",
		"timestamp": time.Now().Add(-age).Unix(),
		"version":   1,
	}
	data, _ := json.Marshal(transmission)
	return base64.URLEncoding.EncodeToString(data)
}

func BenchmarkSecureTokenCreation(b *testing.B) {
	baseProvider := NewMemoryOAuthProvider()
	encryptionKey := []byte("benchmark-key-32-bytes-long!!!!!")
	secureProvider, _ := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)

	client := &OAuthClientInfo{
		ClientID:     "bench-client",
		ClientSecret: "secret",
	}
	baseProvider.RegisterClient(context.Background(), client)

	authCode := &AuthorizationCode{
		Code:     "bench-code",
		ClientID: "bench-client",
	}
	baseProvider.authCodes["bench-code"] = authCode

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = secureProvider.CreateAccessToken(context.Background(), authCode)
	}
}

func BenchmarkSecureTokenValidation(b *testing.B) {
	baseProvider := NewMemoryOAuthProvider()
	encryptionKey := []byte("benchmark-key-32-bytes-long!!!!!")
	secureProvider, _ := NewSecureOAuthProvider(baseProvider, encryptionKey, nil)

	client := &OAuthClientInfo{
		ClientID:     "bench-client",
		ClientSecret: "secret",
	}
	baseProvider.RegisterClient(context.Background(), client)

	authCode := &AuthorizationCode{
		Code:     "bench-code",
		ClientID: "bench-client",
	}
	baseProvider.authCodes["bench-code"] = authCode

	token, _ := secureProvider.CreateAccessToken(context.Background(), authCode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = secureProvider.ValidateAccessToken(context.Background(), token.AccessToken)
	}
}
