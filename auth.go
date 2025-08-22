package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OAuth2 types and constants for MCP authentication

const (
	// OAuth2 grant types
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeClientCredentials = "client_credentials"

	// OAuth2 response types
	ResponseTypeCode = "code"

	// OAuth2 error codes
	ErrorInvalidRequest          = "invalid_request"
	ErrorInvalidClient           = "invalid_client"
	ErrorInvalidGrant            = "invalid_grant"
	ErrorUnauthorizedClient      = "unauthorized_client"
	ErrorUnsupportedGrantType    = "unsupported_grant_type"
	ErrorInvalidScope            = "invalid_scope"
	ErrorAccessDenied            = "access_denied"
	ErrorUnsupportedResponseType = "unsupported_response_type"
	ErrorServerError             = "server_error"
	ErrorTemporarilyUnavailable  = "temporarily_unavailable"

	// Token types
	TokenTypeBearer = "Bearer"

	// Default expiration times
	DefaultAuthCodeExpirationSeconds     = 600        // 10 minutes
	DefaultAccessTokenExpirationSeconds  = 3600       // 1 hour
	DefaultRefreshTokenExpirationSeconds = 86400 * 30 // 30 days
)

// OAuthClientInfo represents OAuth client information
type OAuthClientInfo struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
	Name         string   `json:"client_name,omitempty"`
	Description  string   `json:"client_description,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// AuthorizationRequest represents an OAuth authorization request
type AuthorizationRequest struct {
	ResponseType        string `json:"response_type"`
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri,omitempty"`
	Scope               string `json:"scope,omitempty"`
	State               string `json:"state,omitempty"`
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

// AuthorizationCode represents an OAuth authorization code
type AuthorizationCode struct {
	Code                string    `json:"code"`
	ClientID            string    `json:"client_id"`
	RedirectURI         string    `json:"redirect_uri"`
	Scopes              []string  `json:"scopes"`
	CodeChallenge       string    `json:"code_challenge,omitempty"`
	ExpiresAt           time.Time `json:"expires_at"`
	RedirectURIExplicit bool      `json:"redirect_uri_explicit"`
}

// TokenRequest represents an OAuth token request
type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// AccessToken represents an OAuth access token
type AccessToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ClientID     string    `json:"client_id"`
	Scopes       []string  `json:"scopes"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshToken represents an OAuth refresh token
type RefreshToken struct {
	Token     string    `json:"token"`
	ClientID  string    `json:"client_id"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// OAuthError represents an OAuth error response
type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description,omitempty"`
	URI         string `json:"error_uri,omitempty"`
	State       string `json:"state,omitempty"`
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

// OAuthProvider defines the interface for OAuth providers
type OAuthProvider interface {
	// Client registration
	RegisterClient(ctx context.Context, req *OAuthClientInfo) (*OAuthClientInfo, error)
	GetClient(ctx context.Context, clientID string) (*OAuthClientInfo, error)
	ValidateClient(ctx context.Context, clientID, clientSecret string) error

	// Authorization flow
	CreateAuthorizationCode(ctx context.Context, req *AuthorizationRequest) (*AuthorizationCode, error)
	GetAuthorizationCode(ctx context.Context, code string) (*AuthorizationCode, error)
	RevokeAuthorizationCode(ctx context.Context, code string) error

	// Token management
	CreateAccessToken(ctx context.Context, authCode *AuthorizationCode) (*AccessToken, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*AccessToken, error)
	ValidateAccessToken(ctx context.Context, token string) (*AccessToken, error)
	RevokeToken(ctx context.Context, token string) error

	// Scope validation
	ValidateScopes(ctx context.Context, clientID string, scopes []string) error
}

// MemoryOAuthProvider provides an in-memory OAuth provider for testing/development
type MemoryOAuthProvider struct {
	mu            sync.RWMutex
	clients       map[string]*OAuthClientInfo
	authCodes     map[string]*AuthorizationCode
	accessTokens  map[string]*AccessToken
	refreshTokens map[string]*RefreshToken
}

// NewMemoryOAuthProvider creates a new in-memory OAuth provider
func NewMemoryOAuthProvider() *MemoryOAuthProvider {
	return &MemoryOAuthProvider{
		clients:       make(map[string]*OAuthClientInfo),
		authCodes:     make(map[string]*AuthorizationCode),
		accessTokens:  make(map[string]*AccessToken),
		refreshTokens: make(map[string]*RefreshToken),
	}
}

// RegisterClient implements OAuthProvider
func (p *MemoryOAuthProvider) RegisterClient(ctx context.Context, req *OAuthClientInfo) (*OAuthClientInfo, error) {
	if req.ClientID == "" {
		req.ClientID = generateRandomString(32)
	}

	if req.ClientSecret == "" {
		req.ClientSecret = generateRandomString(64)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[req.ClientID] = req
	return req, nil
}

// GetClient implements OAuthProvider
func (p *MemoryOAuthProvider) GetClient(ctx context.Context, clientID string) (*OAuthClientInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	client, exists := p.clients[clientID]
	if !exists {
		return nil, &OAuthError{
			Code:        ErrorInvalidClient,
			Description: "Client not found",
		}
	}
	return client, nil
}

// ValidateClient implements OAuthProvider
func (p *MemoryOAuthProvider) ValidateClient(ctx context.Context, clientID, clientSecret string) error {
	client, err := p.GetClient(ctx, clientID)
	if err != nil {
		return err
	}

	if client.ClientSecret != clientSecret {
		return &OAuthError{
			Code:        ErrorInvalidClient,
			Description: "Invalid client credentials",
		}
	}

	return nil
}

// CreateAuthorizationCode implements OAuthProvider
func (p *MemoryOAuthProvider) CreateAuthorizationCode(ctx context.Context, req *AuthorizationRequest) (*AuthorizationCode, error) {
	// Validate client
	client, err := p.GetClient(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}

	// Validate redirect URI
	if req.RedirectURI != "" {
		valid := false
		for _, uri := range client.RedirectURIs {
			if uri == req.RedirectURI {
				valid = true
				break
			}
		}
		if !valid {
			return nil, &OAuthError{
				Code:        ErrorInvalidRequest,
				Description: "Invalid redirect URI",
			}
		}
	}

	// Parse scopes
	var scopes []string
	if req.Scope != "" {
		scopes = strings.Fields(req.Scope)
	}

	// Validate scopes
	if err := p.ValidateScopes(ctx, req.ClientID, scopes); err != nil {
		return nil, err
	}

	// Create authorization code
	code := &AuthorizationCode{
		Code:                generateRandomString(32),
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scopes:              scopes,
		CodeChallenge:       req.CodeChallenge,
		ExpiresAt:           time.Now().Add(time.Duration(DefaultAuthCodeExpirationSeconds) * time.Second),
		RedirectURIExplicit: req.RedirectURI != "",
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.authCodes[code.Code] = code
	return code, nil
}

// GetAuthorizationCode implements OAuthProvider
func (p *MemoryOAuthProvider) GetAuthorizationCode(ctx context.Context, code string) (*AuthorizationCode, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	authCode, exists := p.authCodes[code]
	if !exists {
		return nil, &OAuthError{
			Code:        ErrorInvalidGrant,
			Description: "Authorization code not found",
		}
	}

	if time.Now().After(authCode.ExpiresAt) {
		delete(p.authCodes, code)
		return nil, &OAuthError{
			Code:        ErrorInvalidGrant,
			Description: "Authorization code expired",
		}
	}

	return authCode, nil
}

// RevokeAuthorizationCode implements OAuthProvider
func (p *MemoryOAuthProvider) RevokeAuthorizationCode(ctx context.Context, code string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.authCodes, code)
	return nil
}

// CreateAccessToken implements OAuthProvider
func (p *MemoryOAuthProvider) CreateAccessToken(ctx context.Context, authCode *AuthorizationCode) (*AccessToken, error) {
	accessToken := generateRandomString(64)
	refreshToken := generateRandomString(64)

	expiresAt := time.Now().Add(time.Duration(DefaultAccessTokenExpirationSeconds) * time.Second)
	refreshExpiresAt := time.Now().Add(time.Duration(DefaultRefreshTokenExpirationSeconds) * time.Second)

	token := &AccessToken{
		AccessToken:  accessToken,
		TokenType:    TokenTypeBearer,
		ExpiresIn:    DefaultAccessTokenExpirationSeconds,
		RefreshToken: refreshToken,
		Scope:        strings.Join(authCode.Scopes, " "),
		ClientID:     authCode.ClientID,
		Scopes:       authCode.Scopes,
		ExpiresAt:    expiresAt,
	}

	refresh := &RefreshToken{
		Token:     refreshToken,
		ClientID:  authCode.ClientID,
		Scopes:    authCode.Scopes,
		ExpiresAt: refreshExpiresAt,
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.accessTokens[accessToken] = token
	p.refreshTokens[refreshToken] = refresh

	return token, nil
}

// RefreshAccessToken implements OAuthProvider
func (p *MemoryOAuthProvider) RefreshAccessToken(ctx context.Context, refreshTokenStr string) (*AccessToken, error) {
	p.mu.RLock()
	refresh, exists := p.refreshTokens[refreshTokenStr]
	p.mu.RUnlock()
	if !exists {
		return nil, &OAuthError{
			Code:        ErrorInvalidGrant,
			Description: "Refresh token not found",
		}
	}

	if time.Now().After(refresh.ExpiresAt) {
		p.mu.Lock()
		delete(p.refreshTokens, refreshTokenStr)
		p.mu.Unlock()
		return nil, &OAuthError{
			Code:        ErrorInvalidGrant,
			Description: "Refresh token expired",
		}
	}

	// Create new access token
	accessToken := generateRandomString(64)
	expiresAt := time.Now().Add(time.Duration(DefaultAccessTokenExpirationSeconds) * time.Second)

	token := &AccessToken{
		AccessToken:  accessToken,
		TokenType:    TokenTypeBearer,
		ExpiresIn:    DefaultAccessTokenExpirationSeconds,
		RefreshToken: refreshTokenStr, // Reuse refresh token
		Scope:        strings.Join(refresh.Scopes, " "),
		ClientID:     refresh.ClientID,
		Scopes:       refresh.Scopes,
		ExpiresAt:    expiresAt,
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.accessTokens[accessToken] = token
	return token, nil
}

// ValidateAccessToken implements OAuthProvider
func (p *MemoryOAuthProvider) ValidateAccessToken(ctx context.Context, tokenStr string) (*AccessToken, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	token, exists := p.accessTokens[tokenStr]
	if !exists {
		return nil, &OAuthError{
			Code:        ErrorInvalidClient,
			Description: "Access token not found",
		}
	}

	if time.Now().After(token.ExpiresAt) {
		delete(p.accessTokens, tokenStr)
		return nil, &OAuthError{
			Code:        ErrorInvalidClient,
			Description: "Access token expired",
		}
	}

	return token, nil
}

// RevokeToken implements OAuthProvider
func (p *MemoryOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try to revoke as access token
	if _, exists := p.accessTokens[token]; exists {
		delete(p.accessTokens, token)
		return nil
	}

	// Try to revoke as refresh token
	if _, exists := p.refreshTokens[token]; exists {
		delete(p.refreshTokens, token)
		return nil
	}

	return nil // OAuth spec says to succeed even if token doesn't exist
}

// ValidateScopes implements OAuthProvider
func (p *MemoryOAuthProvider) ValidateScopes(ctx context.Context, clientID string, scopes []string) error {
	client, err := p.GetClient(ctx, clientID)
	if err != nil {
		return err
	}

	// If client has no defined scopes, allow any
	if len(client.Scopes) == 0 {
		return nil
	}

	// Check each requested scope
	for _, requestedScope := range scopes {
		valid := false
		for _, allowedScope := range client.Scopes {
			if allowedScope == requestedScope {
				valid = true
				break
			}
		}
		if !valid {
			return &OAuthError{
				Code:        ErrorInvalidScope,
				Description: fmt.Sprintf("Scope '%s' not allowed for client", requestedScope),
			}
		}
	}

	return nil
}

// PKCE utilities

// GeneratePKCEChallenge generates a PKCE code challenge and verifier
func GeneratePKCEChallenge() (verifier, challenge string, err error) {
	// Generate code verifier (43-128 characters)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate code challenge (SHA256 of verifier)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}

// ValidatePKCEChallenge validates a PKCE code verifier against a challenge
func ValidatePKCEChallenge(verifier, challenge string) bool {
	hash := sha256.Sum256([]byte(verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	return expectedChallenge == challenge
}

// Utility functions

func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based generation
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// ParseAuthorizationHeader parses an Authorization header for Bearer tokens
func ParseAuthorizationHeader(header string) (string, error) {
	if header == "" {
		return "", &OAuthError{
			Code:        ErrorInvalidRequest,
			Description: "Missing authorization header",
		}
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", &OAuthError{
			Code:        ErrorInvalidRequest,
			Description: "Invalid authorization header format",
		}
	}

	return parts[1], nil
}

// AuthMiddleware creates HTTP middleware for OAuth token validation
func AuthMiddleware(provider OAuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			token, err := ParseAuthorizationHeader(authHeader)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// Validate token
			accessToken, err := provider.ValidateAccessToken(r.Context(), token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// Add token info to request context
			ctx := context.WithValue(r.Context(), "access_token", accessToken)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAccessTokenFromContext retrieves the access token from request context
func GetAccessTokenFromContext(ctx context.Context) (*AccessToken, bool) {
	token, ok := ctx.Value("access_token").(*AccessToken)
	return token, ok
}
