package mcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
)

func TestMemoryOAuthProvider_RegisterClient_ReadError(t *testing.T) {
	oldReader := rand.Reader
	rand.Reader = failingReader{}
	defer func() {
		rand.Reader = oldReader
	}()

	provider := NewMemoryOAuthProvider()
	_, err := provider.RegisterClient(context.Background(), &OAuthClientInfo{
		RedirectURIs: []string{"http://localhost/callback"},
	})
	if err == nil {
		t.Fatal("expected client registration to fail when random generation fails")
	}
	if !strings.Contains(err.Error(), "generate client ID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryOAuthProvider_CreateAccessToken_ReadError(t *testing.T) {
	oldReader := rand.Reader
	rand.Reader = failingReader{}
	defer func() {
		rand.Reader = oldReader
	}()

	provider := NewMemoryOAuthProvider()
	_, err := provider.CreateAccessToken(context.Background(), &AuthorizationCode{
		ClientID: "test-client",
		Scopes:   []string{"read"},
	})
	if err == nil {
		t.Fatal("expected token creation to fail when random generation fails")
	}
	if !strings.Contains(err.Error(), "generate access token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSecureOAuthProvider_ExtractClientInfoSanitizesContextValues(t *testing.T) {
	provider := &SecureOAuthProvider{}
	ctx := context.Background()
	ctx = context.WithValue(ctx, userAgentKey, "<script>alert('x')</script>\x00")
	ctx = context.WithValue(ctx, remoteAddrKey, "127.0.0.1<script>")
	ctx = context.WithValue(ctx, clientIDKey, "\"quoted\"")

	info := provider.extractClientInfo(ctx)
	if got, want := info["userAgent"], "&lt;script&gt;alert(&#39;x&#39;)&lt;/script&gt;"; got != want {
		t.Fatalf("userAgent = %v, want %v", got, want)
	}
	if got, want := info["remoteAddr"], "127.0.0.1&lt;script&gt;"; got != want {
		t.Fatalf("remoteAddr = %v, want %v", got, want)
	}
	if got, want := info["clientId"], "&#34;quoted&#34;"; got != want {
		t.Fatalf("clientId = %v, want %v", got, want)
	}
}

func TestRateLimitMiddleware_PerEndpointLimiting(t *testing.T) {
	tests := []struct {
		name                string
		perEndpointLimiting bool
		wantSecondAllowed   bool
	}{
		{
			name:                "disabled",
			perEndpointLimiting: false,
			wantSecondAllowed:   false,
		},
		{
			name:                "enabled",
			perEndpointLimiting: true,
			wantSecondAllowed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewRateLimitMiddleware(RateLimitConfig{
				RequestsPerSecond:   1,
				BurstSize:           1,
				PerEndpointLimiting: tt.perEndpointLimiting,
				KeyExtractor: func(ctx context.Context, req MCPRequest) string {
					return "client-1"
				},
			})

			handler := NewMockHandler()
			protected := middleware.Apply(handler)

			resp, err := protected.Handle(context.Background(), NewMockRequest("tools/call", nil))
			if err != nil || resp.IsError() {
				t.Fatalf("first request failed: resp=%v err=%v", resp, err)
			}

			resp, err = protected.Handle(context.Background(), NewMockRequest("resources/read", nil))
			secondAllowed := err == nil && resp != nil && !resp.IsError()
			if secondAllowed != tt.wantSecondAllowed {
				t.Fatalf("secondAllowed = %v, want %v", secondAllowed, tt.wantSecondAllowed)
			}
		})
	}
}

func TestDeriveKeyMethods(t *testing.T) {
	masterKey := []byte("01234567890123456789012345678901")
	salt := []byte("01234567890123456789012345678901")

	argon := deriveKey(masterKey, salt, "signing", KeyDerivationArgon2)
	pbkdf2Key := deriveKey(masterKey, salt, "signing", KeyDerivationPBKDF2)
	fallback := deriveKey(masterKey, salt, "signing", KeyDerivationMethod("unknown"))

	if len(argon) != 32 {
		t.Fatalf("argon2 key length = %d, want 32", len(argon))
	}
	if len(pbkdf2Key) != 32 {
		t.Fatalf("pbkdf2 key length = %d, want 32", len(pbkdf2Key))
	}
	if bytes.Equal(argon, pbkdf2Key) {
		t.Fatal("argon2 and pbkdf2 derived the same key")
	}
	if !bytes.Equal(pbkdf2Key, fallback) {
		t.Fatal("fallback key derivation should match pbkdf2")
	}
	if !bytes.Equal(argon, deriveKey(masterKey, salt, "signing", KeyDerivationArgon2)) {
		t.Fatal("argon2 key derivation is not deterministic")
	}
}

func TestSanitizeErrorModes(t *testing.T) {
	oldMode := GetErrorVerbosity()
	t.Cleanup(func() {
		SetErrorVerbosity(oldMode)
	})

	message := "database/sql: password secret at /tmp/mcp/auth.go:12"

	SetErrorVerbosity(ErrorVerbosityProduction)
	if got := SanitizeErrorMessage(message); got != "internal server error" {
		t.Fatalf("production SanitizeErrorMessage = %q, want %q", got, "internal server error")
	}
	if got := SanitizeError(fmt.Errorf("%s", message)).Error(); got != "internal server error" {
		t.Fatalf("production SanitizeError = %q, want %q", got, "internal server error")
	}

	SetErrorVerbosity(ErrorVerbosityDevelopment)
	if got := SanitizeErrorMessage(message); got != message {
		t.Fatalf("development SanitizeErrorMessage = %q, want %q", got, message)
	}
}

func TestNewCORSMiddlewareDefaults(t *testing.T) {
	t.Run("production", func(t *testing.T) {
		t.Setenv("MCP_ENV", "production")
		middleware := NewCORSMiddleware(CORSConfig{})
		if len(middleware.config.AllowOrigins) != 0 {
			t.Fatalf("production AllowOrigins = %v, want none", middleware.config.AllowOrigins)
		}
		if got := strings.Join(middleware.config.AllowMethods, ","); got != "GET,POST,OPTIONS" {
			t.Fatalf("production AllowMethods = %q", got)
		}
	})

	t.Run("development", func(t *testing.T) {
		t.Setenv("MCP_ENV", "development")
		middleware := NewCORSMiddleware(CORSConfig{})
		got := strings.Join(middleware.config.AllowOrigins, ",")
		want := "http://localhost:*,http://127.0.0.1:*"
		if got != want {
			t.Fatalf("development AllowOrigins = %q, want %q", got, want)
		}
	})
}
