// Package mcp provides enhanced authentication security features.
// This file implements secure token storage, rotation, and transmission patterns
// for hardening the OAuth authentication system.
package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Token security error constants
var (
	ErrTokenRotationRequired = errors.New("auth: token rotation required")
	ErrTokenRevoked          = errors.New("auth: token has been revoked")
	ErrTokenTampered         = errors.New("auth: token integrity check failed")
	ErrInvalidTokenVersion   = errors.New("auth: unsupported token version")
)

// SecureToken represents an enhanced access token with security features
type SecureToken struct {
	*AccessToken
	Version       int       `json:"version"`
	Fingerprint   string    `json:"fingerprint"`
	IssuedAt      time.Time `json:"issuedAt"`
	LastUsed      time.Time `json:"lastUsed"`
	UseCount      int64     `json:"useCount"`
	RotationCount int       `json:"rotationCount"`
	Signature     string    `json:"signature"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// TokenRotationPolicy defines when and how tokens should be rotated
type TokenRotationPolicy struct {
	MaxAge           time.Duration `json:"maxAge"`
	MaxUseCount      int64         `json:"maxUseCount"`
	InactivityPeriod time.Duration `json:"inactivityPeriod"`
	ForceRotateAfter time.Duration `json:"forceRotateAfter"`
}

// DefaultTokenRotationPolicy returns secure default rotation policy
func DefaultTokenRotationPolicy() *TokenRotationPolicy {
	return &TokenRotationPolicy{
		MaxAge:           24 * time.Hour,
		MaxUseCount:      10000,
		InactivityPeriod: 2 * time.Hour,
		ForceRotateAfter: 7 * 24 * time.Hour,
	}
}

// SecureOAuthProvider wraps an OAuth provider with enhanced security
type SecureOAuthProvider struct {
	provider        OAuthProvider
	storage         *SecureTokenStorage
	rotationPolicy  *TokenRotationPolicy
	revokedTokens   sync.Map // token -> revocation time
	tokenMetadata   sync.Map // token -> metadata
	signingKey      []byte
	mu              sync.RWMutex
}

// NewSecureOAuthProvider creates a new secure OAuth provider
func NewSecureOAuthProvider(provider OAuthProvider, encryptionKey []byte, rotationPolicy *TokenRotationPolicy) (*SecureOAuthProvider, error) {
	if provider == nil {
		return nil, errors.New("provider cannot be nil")
	}

	storage, err := NewSecureTokenStorage(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure storage: %w", err)
	}

	if rotationPolicy == nil {
		rotationPolicy = DefaultTokenRotationPolicy()
	}

	// Generate signing key from encryption key
	signingKey := sha256.Sum256(append(encryptionKey, []byte("signing")...))

	return &SecureOAuthProvider{
		provider:       provider,
		storage:        storage,
		rotationPolicy: rotationPolicy,
		signingKey:     signingKey[:],
	}, nil
}

// CreateAccessToken creates a new secure access token
func (p *SecureOAuthProvider) CreateAccessToken(ctx context.Context, authCode *AuthorizationCode) (*AccessToken, error) {
	// Create base token
	baseToken, err := p.provider.CreateAccessToken(ctx, authCode)
	if err != nil {
		return nil, err
	}

	// Enhance with security features
	secureToken := &SecureToken{
		AccessToken:   baseToken,
		Version:       1,
		Fingerprint:   p.generateFingerprint(ctx),
		IssuedAt:      time.Now(),
		LastUsed:      time.Now(),
		UseCount:      0,
		RotationCount: 0,
		Metadata: map[string]interface{}{
			"clientInfo": p.extractClientInfo(ctx),
			"grantType":  "authorization_code",
		},
	}

	// Sign the token
	secureToken.Signature = p.signToken(secureToken)

	// Encrypt and store
	encrypted, err := p.storage.EncryptToken(baseToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Store metadata
	p.tokenMetadata.Store(baseToken.AccessToken, secureToken)

	// Return encrypted token as the access token
	baseToken.AccessToken = encrypted
	return baseToken, nil
}

// ValidateAccessToken validates a secure access token
func (p *SecureOAuthProvider) ValidateAccessToken(ctx context.Context, encryptedToken string) (*AccessToken, error) {
	// Check if token is revoked
	if _, revoked := p.revokedTokens.Load(encryptedToken); revoked {
		return nil, ErrTokenRevoked
	}

	// Decrypt token
	token, err := p.storage.DecryptToken(encryptedToken)
	if err != nil {
		return nil, err
	}

	// Load secure token metadata
	metadataValue, exists := p.tokenMetadata.Load(token.AccessToken)
	if !exists {
		return nil, ErrTokenTampered
	}

	secureToken := metadataValue.(*SecureToken)

	// Verify signature
	expectedSig := p.signToken(secureToken)
	if !p.verifySignature(secureToken.Signature, expectedSig) {
		return nil, ErrTokenTampered
	}

	// Check rotation requirements
	if p.needsRotation(secureToken) {
		return nil, ErrTokenRotationRequired
	}

	// Update usage stats
	secureToken.LastUsed = time.Now()
	secureToken.UseCount++
	p.tokenMetadata.Store(token.AccessToken, secureToken)

	// Validate with underlying provider
	return p.provider.ValidateAccessToken(ctx, token.AccessToken)
}

// RefreshAccessToken refreshes a token with rotation
func (p *SecureOAuthProvider) RefreshAccessToken(ctx context.Context, refreshToken string) (*AccessToken, error) {
	// Decrypt refresh token if encrypted
	decryptedRefresh := refreshToken
	if p.isEncryptedToken(refreshToken) {
		// Decrypt refresh token
		token, err := p.storage.DecryptToken(refreshToken)
		if err != nil {
			return nil, err
		}
		decryptedRefresh = token.RefreshToken
	}

	// Refresh with underlying provider
	newToken, err := p.provider.RefreshAccessToken(ctx, decryptedRefresh)
	if err != nil {
		return nil, err
	}

	// Get old token metadata for rotation count
	var rotationCount int
	if oldMetadata, exists := p.tokenMetadata.Load(decryptedRefresh); exists {
		if oldSecure, ok := oldMetadata.(*SecureToken); ok {
			rotationCount = oldSecure.RotationCount + 1
		}
	}

	// Create new secure token
	secureToken := &SecureToken{
		AccessToken:   newToken,
		Version:       1,
		Fingerprint:   p.generateFingerprint(ctx),
		IssuedAt:      time.Now(),
		LastUsed:      time.Now(),
		UseCount:      0,
		RotationCount: rotationCount,
		Metadata: map[string]interface{}{
			"clientInfo": p.extractClientInfo(ctx),
			"grantType":  "refresh_token",
			"rotatedAt":  time.Now(),
		},
	}

	// Sign the token
	secureToken.Signature = p.signToken(secureToken)

	// Encrypt and store
	encrypted, err := p.storage.EncryptToken(newToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Store metadata
	p.tokenMetadata.Store(newToken.AccessToken, secureToken)

	// Return encrypted token
	newToken.AccessToken = encrypted
	return newToken, nil
}

// RevokeToken revokes a token immediately
func (p *SecureOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	// Mark as revoked
	p.revokedTokens.Store(token, time.Now())

	// If encrypted, decrypt to get actual token
	if p.isEncryptedToken(token) {
		decrypted, err := p.storage.DecryptToken(token)
		if err == nil {
			// Revoke underlying token
			return p.provider.RevokeToken(ctx, decrypted.AccessToken)
		}
	}

	return p.provider.RevokeToken(ctx, token)
}

// needsRotation checks if a token needs rotation
func (p *SecureOAuthProvider) needsRotation(token *SecureToken) bool {
	now := time.Now()

	// Check max age
	if now.Sub(token.IssuedAt) > p.rotationPolicy.MaxAge {
		return true
	}

	// Check use count
	if token.UseCount >= p.rotationPolicy.MaxUseCount {
		return true
	}

	// Check inactivity
	if now.Sub(token.LastUsed) > p.rotationPolicy.InactivityPeriod {
		return true
	}

	// Check force rotation
	if now.Sub(token.IssuedAt) > p.rotationPolicy.ForceRotateAfter {
		return true
	}

	return false
}

// generateFingerprint generates a unique fingerprint for the token
func (p *SecureOAuthProvider) generateFingerprint(ctx context.Context) string {
	// Combine various factors for fingerprint
	factors := []string{
		time.Now().String(),
	}

	// Add client info if available
	if clientInfo := p.extractClientInfo(ctx); clientInfo != nil {
		if data, err := json.Marshal(clientInfo); err == nil {
			factors = append(factors, string(data))
		}
	}

	// Generate random component
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err == nil {
		factors = append(factors, hex.EncodeToString(randomBytes))
	}

	// Create fingerprint
	h := sha256.New()
	for _, factor := range factors {
		h.Write([]byte(factor))
	}

	return hex.EncodeToString(h.Sum(nil))[:32]
}

// extractClientInfo extracts client information from context
func (p *SecureOAuthProvider) extractClientInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})

	// Extract from context values
	if userAgent, ok := ctx.Value("User-Agent").(string); ok {
		info["userAgent"] = userAgent
	}
	if remoteAddr, ok := ctx.Value("RemoteAddr").(string); ok {
		info["remoteAddr"] = remoteAddr
	}
	if clientID, ok := ctx.Value("ClientID").(string); ok {
		info["clientId"] = clientID
	}

	return info
}

// signToken creates a signature for the token
func (p *SecureOAuthProvider) signToken(token *SecureToken) string {
	// Create signing data
	data := fmt.Sprintf("%s|%d|%s|%d|%d",
		token.AccessToken,
		token.Version,
		token.Fingerprint,
		token.IssuedAt.Unix(),
		token.RotationCount,
	)

	// Create HMAC signature
	h := hmac.New(sha256.New, p.signingKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// verifySignature verifies token signatures match
func (p *SecureOAuthProvider) verifySignature(sig1, sig2 string) bool {
	return subtle.ConstantTimeCompare([]byte(sig1), []byte(sig2)) == 1
}

// isEncryptedToken checks if a token appears to be encrypted
func (p *SecureOAuthProvider) isEncryptedToken(token string) bool {
	// Check if it's base64 encoded and has expected length
	if decoded, err := base64.URLEncoding.DecodeString(token); err == nil {
		// Encrypted tokens have nonce + ciphertext
		return len(decoded) > p.storage.cipher.NonceSize()
	}
	return false
}

// Implement remaining OAuthProvider interface methods by delegating

func (p *SecureOAuthProvider) RegisterClient(ctx context.Context, req *OAuthClientInfo) (*OAuthClientInfo, error) {
	return p.provider.RegisterClient(ctx, req)
}

func (p *SecureOAuthProvider) GetClient(ctx context.Context, clientID string) (*OAuthClientInfo, error) {
	return p.provider.GetClient(ctx, clientID)
}

func (p *SecureOAuthProvider) ValidateClient(ctx context.Context, clientID, clientSecret string) error {
	return p.provider.ValidateClient(ctx, clientID, clientSecret)
}

func (p *SecureOAuthProvider) CreateAuthorizationCode(ctx context.Context, req *AuthorizationRequest) (*AuthorizationCode, error) {
	return p.provider.CreateAuthorizationCode(ctx, req)
}

func (p *SecureOAuthProvider) GetAuthorizationCode(ctx context.Context, code string) (*AuthorizationCode, error) {
	return p.provider.GetAuthorizationCode(ctx, code)
}

func (p *SecureOAuthProvider) RevokeAuthorizationCode(ctx context.Context, code string) error {
	return p.provider.RevokeAuthorizationCode(ctx, code)
}

func (p *SecureOAuthProvider) ValidateScopes(ctx context.Context, clientID string, scopes []string) error {
	return p.provider.ValidateScopes(ctx, clientID, scopes)
}

// TokenTransmissionGuard provides secure token transmission patterns
type TokenTransmissionGuard struct {
	maxTransmissionAge time.Duration
	nonceCache         sync.Map // nonce -> timestamp
	mu                 sync.RWMutex
}

// NewTokenTransmissionGuard creates a new transmission guard
func NewTokenTransmissionGuard(maxAge time.Duration) *TokenTransmissionGuard {
	if maxAge <= 0 {
		maxAge = 5 * time.Minute
	}

	guard := &TokenTransmissionGuard{
		maxTransmissionAge: maxAge,
	}

	// Start cleanup routine
	go guard.cleanupNonces()

	return guard
}

// PrepareTokenForTransmission prepares a token for secure transmission
func (g *TokenTransmissionGuard) PrepareTokenForTransmission(token string) (string, error) {
	// Generate nonce
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create transmission package
	transmission := map[string]interface{}{
		"token":     token,
		"nonce":     hex.EncodeToString(nonce),
		"timestamp": time.Now().Unix(),
		"version":   1,
	}

	// Store nonce to prevent replay
	g.nonceCache.Store(hex.EncodeToString(nonce), time.Now())

	// Encode as JSON
	data, err := json.Marshal(transmission)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transmission: %w", err)
	}

	// Base64 encode for transmission
	return base64.URLEncoding.EncodeToString(data), nil
}

// ValidateTokenTransmission validates a received token transmission
func (g *TokenTransmissionGuard) ValidateTokenTransmission(transmitted string) (string, error) {
	// Decode from base64
	data, err := base64.URLEncoding.DecodeString(transmitted)
	if err != nil {
		return "", fmt.Errorf("invalid transmission format: %w", err)
	}

	// Parse transmission package
	var transmission map[string]interface{}
	if err := json.Unmarshal(data, &transmission); err != nil {
		return "", fmt.Errorf("invalid transmission data: %w", err)
	}

	// Validate version
	version, ok := transmission["version"].(float64)
	if !ok || int(version) != 1 {
		return "", ErrInvalidTokenVersion
	}

	// Validate timestamp
	timestamp, ok := transmission["timestamp"].(float64)
	if !ok {
		return "", errors.New("missing timestamp")
	}

	transmissionTime := time.Unix(int64(timestamp), 0)
	if time.Since(transmissionTime) > g.maxTransmissionAge {
		return "", errors.New("transmission expired")
	}

	// Validate nonce
	nonce, ok := transmission["nonce"].(string)
	if !ok {
		return "", errors.New("missing nonce")
	}

	// Check if nonce was already used
	if _, exists := g.nonceCache.Load(nonce); exists {
		return "", errors.New("nonce already used")
	}

	// Store nonce to prevent replay
	g.nonceCache.Store(nonce, time.Now())

	// Extract token
	token, ok := transmission["token"].(string)
	if !ok {
		return "", errors.New("missing token")
	}

	return token, nil
}

// cleanupNonces removes old nonces from cache
func (g *TokenTransmissionGuard) cleanupNonces() {
	ticker := time.NewTicker(g.maxTransmissionAge)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-g.maxTransmissionAge * 2)
		
		g.nonceCache.Range(func(key, value interface{}) bool {
			if timestamp, ok := value.(time.Time); ok {
				if timestamp.Before(cutoff) {
					g.nonceCache.Delete(key)
				}
			}
			return true
		})
	}
}

// SecureAuthenticationMiddleware provides enhanced authentication middleware
type SecureAuthenticationMiddleware struct {
	provider    *SecureOAuthProvider
	guard       *TokenTransmissionGuard
	skipMethods map[string]bool
	tokenCache  sync.Map
}

// NewSecureAuthenticationMiddleware creates secure authentication middleware
func NewSecureAuthenticationMiddleware(provider *SecureOAuthProvider, config AuthConfig) *SecureAuthenticationMiddleware {
	skipMethods := make(map[string]bool)
	
	// Default skip methods
	defaultSkip := []string{"initialize", "initialized", "ping"}
	for _, method := range defaultSkip {
		skipMethods[method] = true
	}
	
	// Add user-defined skip methods
	for _, method := range config.SkipMethods {
		skipMethods[method] = true
	}

	return &SecureAuthenticationMiddleware{
		provider:    provider,
		guard:       NewTokenTransmissionGuard(5 * time.Minute),
		skipMethods: skipMethods,
	}
}

// Apply implements the Middleware interface
func (m *SecureAuthenticationMiddleware) Apply(next MCPHandler) MCPHandler {
	return MCPHandlerFunc(func(ctx context.Context, req MCPRequest) (MCPResponse, error) {
		// Check if method should skip authentication
		if m.skipMethods[req.GetMethod()] {
			return next.Handle(ctx, req)
		}

		// Extract token from context or request
		token, err := m.extractSecureToken(ctx, req)
		if err != nil {
			return nil, NewAuthError("Authentication required", ErrorInvalidRequest)
		}

		// Validate token transmission if needed
		if transmitted, ok := ctx.Value("TransmittedToken").(string); ok {
			token, err = m.guard.ValidateTokenTransmission(transmitted)
			if err != nil {
				return nil, NewAuthError("Invalid token transmission", ErrorInvalidClient)
			}
		}

		// Validate token
		accessToken, err := m.provider.ValidateAccessToken(ctx, token)
		if err != nil {
			if err == ErrTokenRotationRequired {
				return nil, NewAuthError("Token rotation required", ErrorInvalidGrant)
			}
			return nil, NewAuthError("Invalid authentication", ErrorInvalidClient)
		}

		// Add authentication context
		authCtx := WithAuthContext(ctx, &AuthContext{
			AccessToken: accessToken,
			ClientID:    accessToken.ClientID,
			Scopes:      accessToken.Scopes,
		})

		return next.Handle(authCtx, req.WithContext(authCtx))
	})
}

// extractSecureToken extracts token with enhanced security checks
func (m *SecureAuthenticationMiddleware) extractSecureToken(ctx context.Context, req MCPRequest) (string, error) {
	// Try Authorization header
	if authHeader, ok := ctx.Value("Authorization").(string); ok {
		return ParseAuthorizationHeader(authHeader)
	}

	// Try request parameters
	params := req.GetParams()
	if params != nil {
		var paramsMap map[string]interface{}
		if err := json.Unmarshal(params, &paramsMap); err == nil {
			if token, ok := paramsMap["auth_token"].(string); ok {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("no authentication token found")
}

func (m *SecureAuthenticationMiddleware) Name() string {
	return "secure_authentication"
}

func (m *SecureAuthenticationMiddleware) Priority() int {
	return 900 // High priority, after logging
}