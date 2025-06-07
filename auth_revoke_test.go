package mcp

import (
	"context"
	"testing"
	"time"
)

func TestRevokeAuthorizationCodeFunction(t *testing.T) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	// Register a test client
	client := &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		Scopes:       []string{"read", "write"},
	}
	_, err := provider.RegisterClient(ctx, client)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	// Create an authorization code directly in the provider's storage
	authCode := &AuthorizationCode{
		Code:          "test-auth-code",
		ClientID:      "test-client",
		Scopes:        []string{"read"},
		RedirectURI:   "http://localhost:8080/callback",
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		CodeChallenge: "test-challenge",
	}
	provider.authCodes[authCode.Code] = authCode

	// Test revoke authorization code
	err = provider.RevokeAuthorizationCode(ctx, "test-auth-code")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify code is removed
	_, exists := provider.authCodes["test-auth-code"]
	if exists {
		t.Error("Authorization code should have been removed")
	}

	// Test revoke non-existent code - this should succeed (idempotent)
	err = provider.RevokeAuthorizationCode(ctx, "non-existent")
	if err != nil {
		t.Errorf("Revoke should be idempotent, got error: %v", err)
	}
}
