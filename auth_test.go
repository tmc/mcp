package mcp

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test comprehensive OAuth functionality to achieve near 100% coverage

func TestMemoryOAuthProvider_RegisterClient(t *testing.T) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	tests := []struct {
		name   string
		client *OAuthClientInfo
	}{
		{
			name: "client with ID and secret",
			client: &OAuthClientInfo{
				ClientID:     "test-client-1",
				ClientSecret: "test-secret-1",
				RedirectURIs: []string{"http://localhost:8080/callback"},
				Name:         "Test Client 1",
				Description:  "A test client",
				Scopes:       []string{"read", "write"},
			},
		},
		{
			name: "client without ID (auto-generated)",
			client: &OAuthClientInfo{
				RedirectURIs: []string{"http://localhost:8080/callback"},
				Name:         "Test Client 2",
			},
		},
		{
			name: "client without secret (auto-generated)",
			client: &OAuthClientInfo{
				ClientID:     "test-client-3",
				RedirectURIs: []string{"http://localhost:8080/callback"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.RegisterClient(ctx, tt.client)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Error("Expected non-nil result")
			}

			if result.ClientID == "" {
				t.Error("Client ID should not be empty")
			}

			if result.ClientSecret == "" {
				t.Error("Client secret should not be empty")
			}

			// Verify client was stored
			stored, err := provider.GetClient(ctx, result.ClientID)
			if err != nil {
				t.Errorf("Failed to retrieve stored client: %v", err)
			}

			if stored.ClientID != result.ClientID {
				t.Errorf("Stored client ID mismatch: expected %s, got %s", result.ClientID, stored.ClientID)
			}
		})
	}
}

func TestMemoryOAuthProvider_GetClient(t *testing.T) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	// Register a test client
	client := &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	}

	_, err := provider.RegisterClient(ctx, client)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	tests := []struct {
		name        string
		clientID    string
		expectError bool
	}{
		{
			name:        "existing client",
			clientID:    "test-client",
			expectError: false,
		},
		{
			name:        "non-existent client",
			clientID:    "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.GetClient(ctx, tt.clientID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Error("Expected non-nil result")
			}

			if result.ClientID != tt.clientID {
				t.Errorf("Client ID mismatch: expected %s, got %s", tt.clientID, result.ClientID)
			}
		})
	}
}

func TestMemoryOAuthProvider_ValidateClient(t *testing.T) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	// Register a test client
	client := &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	}

	_, err := provider.RegisterClient(ctx, client)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		expectError  bool
	}{
		{
			name:         "valid credentials",
			clientID:     "test-client",
			clientSecret: "test-secret",
			expectError:  false,
		},
		{
			name:         "invalid client ID",
			clientID:     "invalid-client",
			clientSecret: "test-secret",
			expectError:  true,
		},
		{
			name:         "invalid client secret",
			clientID:     "test-client",
			clientSecret: "invalid-secret",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateClient(ctx, tt.clientID, tt.clientSecret)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateRandomString_ReadError(t *testing.T) {
	oldReader := rand.Reader
	rand.Reader = failingReader{}
	defer func() {
		rand.Reader = oldReader
	}()

	if _, err := generateRandomString(64); err == nil {
		t.Fatal("expected random generation error")
	}
}

type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestMemoryOAuthProvider_AuthorizationFlow(t *testing.T) {
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

	// Test authorization request
	authReq := &AuthorizationRequest{
		ResponseType: ResponseTypeCode,
		ClientID:     "test-client",
		RedirectURI:  "http://localhost:8080/callback",
		Scope:        "read write",
		State:        "test-state",
	}

	authCode, err := provider.CreateAuthorizationCode(ctx, authReq)
	if err != nil {
		t.Fatalf("Failed to create authorization code: %v", err)
	}

	if authCode == nil {
		t.Fatal("Expected non-nil authorization code")
	}

	if authCode.Code == "" {
		t.Error("Authorization code should not be empty")
	}

	if authCode.ClientID != "test-client" {
		t.Errorf("Client ID mismatch: expected test-client, got %s", authCode.ClientID)
	}

	// Test retrieving authorization code
	retrieved, err := provider.GetAuthorizationCode(ctx, authCode.Code)
	if err != nil {
		t.Errorf("Failed to retrieve authorization code: %v", err)
	}

	if retrieved.Code != authCode.Code {
		t.Errorf("Code mismatch: expected %s, got %s", authCode.Code, retrieved.Code)
	}

	// Test creating access token
	accessToken, err := provider.CreateAccessToken(ctx, authCode)
	if err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}

	if accessToken == nil {
		t.Fatal("Expected non-nil access token")
	}

	if accessToken.AccessToken == "" {
		t.Error("Access token should not be empty")
	}

	if accessToken.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}

	if accessToken.TokenType != TokenTypeBearer {
		t.Errorf("Expected token type Bearer, got %s", accessToken.TokenType)
	}

	// Test token validation
	validated, err := provider.ValidateAccessToken(ctx, accessToken.AccessToken)
	if err != nil {
		t.Errorf("Failed to validate access token: %v", err)
	}

	if validated.AccessToken != accessToken.AccessToken {
		t.Errorf("Token mismatch: expected %s, got %s", accessToken.AccessToken, validated.AccessToken)
	}

	// Test refresh token
	refreshed, err := provider.RefreshAccessToken(ctx, accessToken.RefreshToken)
	if err != nil {
		t.Errorf("Failed to refresh access token: %v", err)
	}

	if refreshed == nil {
		t.Error("Expected non-nil refreshed token")
	}

	if refreshed.AccessToken == accessToken.AccessToken {
		t.Error("Refreshed token should be different from original")
	}

	// Test token revocation
	err = provider.RevokeToken(ctx, accessToken.AccessToken)
	if err != nil {
		t.Errorf("Failed to revoke token: %v", err)
	}

	// Test revoked token validation (should fail)
	_, err = provider.ValidateAccessToken(ctx, accessToken.AccessToken)
	if err == nil {
		t.Error("Expected error when validating revoked token")
	}
}

func TestMemoryOAuthProvider_ScopeValidation(t *testing.T) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	// Register clients with different scope configurations
	clientWithScopes := &OAuthClientInfo{
		ClientID: "client-with-scopes",
		Scopes:   []string{"read", "write"},
	}

	clientWithoutScopes := &OAuthClientInfo{
		ClientID: "client-without-scopes",
	}

	_, err := provider.RegisterClient(ctx, clientWithScopes)
	if err != nil {
		t.Fatalf("Failed to register client with scopes: %v", err)
	}

	_, err = provider.RegisterClient(ctx, clientWithoutScopes)
	if err != nil {
		t.Fatalf("Failed to register client without scopes: %v", err)
	}

	tests := []struct {
		name        string
		clientID    string
		scopes      []string
		expectError bool
	}{
		{
			name:        "valid scopes",
			clientID:    "client-with-scopes",
			scopes:      []string{"read"},
			expectError: false,
		},
		{
			name:        "invalid scope",
			clientID:    "client-with-scopes",
			scopes:      []string{"admin"},
			expectError: true,
		},
		{
			name:        "client without defined scopes (any allowed)",
			clientID:    "client-without-scopes",
			scopes:      []string{"any", "scope"},
			expectError: false,
		},
		{
			name:        "non-existent client",
			clientID:    "non-existent",
			scopes:      []string{"read"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateScopes(ctx, tt.clientID, tt.scopes)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestPKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCEChallenge()
	if err != nil {
		t.Fatalf("Failed to generate PKCE challenge: %v", err)
	}

	if verifier == "" {
		t.Error("Verifier should not be empty")
	}

	if challenge == "" {
		t.Error("Challenge should not be empty")
	}

	// Test valid verification
	if !ValidatePKCEChallenge(verifier, challenge) {
		t.Error("PKCE challenge validation should succeed")
	}

	// Test invalid verification
	if ValidatePKCEChallenge("invalid-verifier", challenge) {
		t.Error("PKCE challenge validation should fail for invalid verifier")
	}

	if ValidatePKCEChallenge(verifier, "invalid-challenge") {
		t.Error("PKCE challenge validation should fail for invalid challenge")
	}
}

func TestParseAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name          string
		header        string
		expectedToken string
		expectError   bool
	}{
		{
			name:          "valid bearer token",
			header:        "Bearer abc123",
			expectedToken: "abc123",
			expectError:   false,
		},
		{
			name:          "bearer token with case variation",
			header:        "bearer xyz789",
			expectedToken: "xyz789",
			expectError:   false,
		},
		{
			name:        "empty header",
			header:      "",
			expectError: true,
		},
		{
			name:        "invalid format",
			header:      "Invalid format",
			expectError: true,
		},
		{
			name:        "missing token",
			header:      "Bearer",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ParseAuthorizationHeader(tt.header)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if token != tt.expectedToken {
				t.Errorf("Token mismatch: expected %s, got %s", tt.expectedToken, token)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	// Register client and create access token
	client := &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	}

	_, err := provider.RegisterClient(ctx, client)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	authReq := &AuthorizationRequest{
		ResponseType: ResponseTypeCode,
		ClientID:     "test-client",
		RedirectURI:  "http://localhost:8080/callback",
	}

	authCode, err := provider.CreateAuthorizationCode(ctx, authReq)
	if err != nil {
		t.Fatalf("Failed to create authorization code: %v", err)
	}

	accessToken, err := provider.CreateAccessToken(ctx, authCode)
	if err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := GetAccessTokenFromContext(r.Context())
		if !ok {
			http.Error(w, "No token in context", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Success: " + token.ClientID))
	})

	// Wrap with auth middleware
	middleware := AuthMiddleware(provider)
	handler := middleware(testHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + accessToken.AccessToken,
			expectedStatus: http.StatusOK,
			expectedBody:   "Success: test-client",
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing authorization",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid format",
			authHeader:     "Invalid format",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("Status code mismatch: expected %d, got %d", tt.expectedStatus, recorder.Code)
			}

			if tt.expectedBody != "" {
				body := strings.TrimSpace(recorder.Body.String())
				if body != tt.expectedBody {
					t.Errorf("Body mismatch: expected %s, got %s", tt.expectedBody, body)
				}
			}
		})
	}
}

func TestOAuthError(t *testing.T) {
	tests := []struct {
		name     string
		error    *OAuthError
		expected string
	}{
		{
			name: "error with description",
			error: &OAuthError{
				Code:        ErrorInvalidClient,
				Description: "Client not found",
			},
			expected: "invalid_client: Client not found",
		},
		{
			name: "error without description",
			error: &OAuthError{
				Code: ErrorInvalidRequest,
			},
			expected: "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			if result != tt.expected {
				t.Errorf("Error string mismatch: expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	// Register client
	client := &OAuthClientInfo{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURIs: []string{"http://localhost:8080/callback"},
	}

	_, err := provider.RegisterClient(ctx, client)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	// Create expired authorization code
	authCode := &AuthorizationCode{
		Code:      "expired-code",
		ClientID:  "test-client",
		ExpiresAt: time.Now().Add(-time.Hour), // Expired
	}

	provider.authCodes["expired-code"] = authCode

	// Try to retrieve expired code
	_, err = provider.GetAuthorizationCode(ctx, "expired-code")
	if err == nil {
		t.Error("Expected error for expired authorization code")
	}

	// Code should be automatically removed
	if _, exists := provider.authCodes["expired-code"]; exists {
		t.Error("Expired authorization code should be automatically removed")
	}

	// Test expired access token
	expiredToken := &AccessToken{
		AccessToken: "expired-token",
		ClientID:    "test-client",
		ExpiresAt:   time.Now().Add(-time.Hour), // Expired
	}

	provider.accessTokens["expired-token"] = expiredToken

	// Try to validate expired token
	_, err = provider.ValidateAccessToken(ctx, "expired-token")
	if err == nil {
		t.Error("Expected error for expired access token")
	}

	// Token should be automatically removed
	if _, exists := provider.accessTokens["expired-token"]; exists {
		t.Error("Expired access token should be automatically removed")
	}
}

func BenchmarkGeneratePKCEChallenge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := GeneratePKCEChallenge()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidatePKCEChallenge(b *testing.B) {
	verifier, challenge, err := GeneratePKCEChallenge()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidatePKCEChallenge(verifier, challenge)
	}
}

func BenchmarkCreateAccessToken(b *testing.B) {
	provider := NewMemoryOAuthProvider()
	ctx := context.Background()

	// Setup client and auth code
	client := &OAuthClientInfo{
		ClientID:     "bench-client",
		ClientSecret: "bench-secret",
	}

	provider.RegisterClient(ctx, client)

	authCode := &AuthorizationCode{
		Code:      "bench-code",
		ClientID:  "bench-client",
		Scopes:    []string{"read"},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.CreateAccessToken(ctx, authCode)
		if err != nil {
			b.Fatal(err)
		}
	}
}
