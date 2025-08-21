package mcp

import (
	"context"
	"os"
	"testing"
)

func TestDebugSecureOAuth(t *testing.T) {
	// Enable debug mode
	os.Setenv("DEBUG_AUTH", "1")
	defer os.Setenv("DEBUG_AUTH", "")

	// Create base provider
	baseProvider := NewMemoryOAuthProvider()
	
	// Register a test client
	client := &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost/callback"},
	}
	_, err := baseProvider.RegisterClient(context.Background(), client)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

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
	ctx := context.WithValue(context.Background(), "User-Agent", "test-agent")
	ctx = context.WithValue(ctx, "RemoteAddr", "127.0.0.1")
	
	token, err := secureProvider.CreateAccessToken(ctx, authCode)
	if err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}

	t.Logf("Created token: %+v", token)

	// Validate token - this is where it should fail
	validated, err := secureProvider.ValidateAccessToken(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if validated.ClientID != "test-client" {
		t.Errorf("ClientID = %v, want %v", validated.ClientID, "test-client")
	}
}