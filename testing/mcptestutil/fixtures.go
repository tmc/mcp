// Package mcptestutil provides test fixture system with proper cleanup and isolation.
package mcptestutil

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/tmc/mcp"
)

// TestFixture represents a complete test environment with isolated resources.
// It provides automatic cleanup and resource management for complex test scenarios.
type TestFixture struct {
	t             *testing.T
	cleanupFuncs  []func()
	oauthProvider *IsolatedOAuthProvider
	tokenGuard    *IsolatedTokenGuard
	tempResources map[string]interface{}
	mu            sync.RWMutex
	isSetup       bool
	isCleanedUp   bool
}

// NewTestFixture creates a new test fixture with automatic cleanup registration.
// The fixture ensures proper resource isolation and cleanup for each test.
//
// Usage:
//
//	fixture := NewTestFixture(t)
//	defer fixture.Cleanup() // Optional - automatic via t.Cleanup()
//
//	// Use fixture.OAuthProvider() and fixture.TokenGuard()
func NewTestFixture(t *testing.T) *TestFixture {
	t.Helper()

	fixture := &TestFixture{
		t:             t,
		cleanupFuncs:  make([]func(), 0),
		tempResources: make(map[string]interface{}),
	}

	// Register cleanup with testing framework
	t.Cleanup(func() {
		fixture.Cleanup()
	})

	return fixture
}

// Setup initializes the fixture with isolated providers.
// This method is called automatically when accessing providers.
func (f *TestFixture) Setup() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isSetup {
		return nil
	}

	// Generate unique encryption key for this test
	encryptionKey := make([]byte, 32)
	if _, err := rand.Read(encryptionKey); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Create isolated OAuth provider
	var err error
	f.oauthProvider, err = NewIsolatedOAuthProvider(f.t, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create isolated OAuth provider: %w", err)
	}
	f.addCleanup(func() {
		f.oauthProvider.Cleanup()
	})

	// Create isolated token guard
	f.tokenGuard, err = NewIsolatedTokenGuard(f.t, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create isolated token guard: %w", err)
	}
	f.addCleanup(func() {
		f.tokenGuard.Cleanup()
	})

	f.isSetup = true
	return nil
}

// OAuthProvider returns an isolated OAuth provider for this test.
// Each test gets its own provider instance with unique state.
func (f *TestFixture) OAuthProvider() *IsolatedOAuthProvider {
	f.t.Helper()

	if err := f.Setup(); err != nil {
		f.t.Fatalf("Failed to setup test fixture: %v", err)
	}

	return f.oauthProvider
}

// TokenGuard returns an isolated token guard for this test.
// Each test gets its own guard with unique namespace.
func (f *TestFixture) TokenGuard() *IsolatedTokenGuard {
	f.t.Helper()

	if err := f.Setup(); err != nil {
		f.t.Fatalf("Failed to setup test fixture: %v", err)
	}

	return f.tokenGuard
}

// AddResource stores a named resource for cleanup tracking.
// Resources are automatically cleaned up when the fixture is destroyed.
func (f *TestFixture) AddResource(name string, resource interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.tempResources[name] = resource
}

// GetResource retrieves a named resource from the fixture.
func (f *TestFixture) GetResource(name string) (interface{}, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	resource, exists := f.tempResources[name]
	return resource, exists
}

// AddCleanup adds a cleanup function to be called when the fixture is destroyed.
// Cleanup functions are called in reverse order (LIFO).
func (f *TestFixture) AddCleanup(cleanupFunc func()) {
	f.addCleanup(cleanupFunc)
}

// addCleanup is the internal implementation of AddCleanup.
func (f *TestFixture) addCleanup(cleanupFunc func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cleanupFuncs = append(f.cleanupFuncs, cleanupFunc)
}

// Cleanup performs cleanup of all resources managed by this fixture.
// This is automatically called via t.Cleanup() but can be called manually if needed.
func (f *TestFixture) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isCleanedUp {
		return
	}

	// Call cleanup functions in reverse order (LIFO)
	for i := len(f.cleanupFuncs) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't fail the test during cleanup
					f.t.Logf("Panic during cleanup: %v", r)
				}
			}()
			f.cleanupFuncs[i]()
		}()
	}

	f.isCleanedUp = true
}

// IsolatedOAuthProvider provides an OAuth provider that doesn't share state between tests.
// Each instance has its own token storage and configuration.
type IsolatedOAuthProvider struct {
	t              *testing.T
	baseProvider   mcp.OAuthProvider
	secureProvider *mcp.SecureOAuthProvider
	namespace      string
	encryptionKey  []byte
	tokenStorage   map[string]*mcp.AccessToken
	mu             sync.RWMutex
}

// NewIsolatedOAuthProvider creates a new isolated OAuth provider for testing.
func NewIsolatedOAuthProvider(t *testing.T, encryptionKey []byte) (*IsolatedOAuthProvider, error) {
	t.Helper()

	// Generate unique namespace for this test instance
	namespace := generateTestNamespace(t)

	// Create mock base provider
	baseProvider := NewMockOAuthProvider()

	// Create secure provider wrapper
	secureProvider, err := mcp.NewSecureOAuthProvider(baseProvider, encryptionKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure provider: %w", err)
	}

	return &IsolatedOAuthProvider{
		t:              t,
		baseProvider:   baseProvider,
		secureProvider: secureProvider,
		namespace:      namespace,
		encryptionKey:  encryptionKey,
		tokenStorage:   make(map[string]*mcp.AccessToken),
	}, nil
}

// CreateAccessToken creates an access token isolated to this test instance.
func (p *IsolatedOAuthProvider) CreateAccessToken(ctx context.Context, authCode *mcp.AuthorizationCode) (*mcp.AccessToken, error) {
	p.t.Helper()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Use the secure provider to create token
	token, err := p.secureProvider.CreateAccessToken(ctx, authCode)
	if err != nil {
		return nil, err
	}

	// Store in isolated storage
	p.tokenStorage[token.AccessToken] = token

	return token, nil
}

// ValidateToken validates a token within this test instance's namespace.
func (p *IsolatedOAuthProvider) ValidateToken(ctx context.Context, tokenString string) (*mcp.AccessToken, error) {
	p.t.Helper()

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check isolated storage first
	if token, exists := p.tokenStorage[tokenString]; exists {
		return token, nil
	}

	// Fall back to secure provider validation
	return p.secureProvider.ValidateAccessToken(ctx, tokenString)
}

// RevokeToken revokes a token within this test instance.
func (p *IsolatedOAuthProvider) RevokeToken(ctx context.Context, tokenString string) error {
	p.t.Helper()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from isolated storage
	delete(p.tokenStorage, tokenString)

	// Revoke in secure provider
	return p.secureProvider.RevokeToken(ctx, tokenString)
}

// Cleanup cleans up resources used by this isolated provider.
func (p *IsolatedOAuthProvider) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear token storage
	for key := range p.tokenStorage {
		delete(p.tokenStorage, key)
	}
}

// Namespace returns the unique namespace for this provider instance.
func (p *IsolatedOAuthProvider) Namespace() string {
	return p.namespace
}

// IsolatedTokenGuard provides token validation that's isolated per test.
// Each instance operates in its own namespace to prevent test interference.
type IsolatedTokenGuard struct {
	t         *testing.T
	namespace string
	storage   map[string]*tokenInfo
	mu        sync.RWMutex
}

// tokenInfo stores metadata about tokens in the isolated guard.
type tokenInfo struct {
	Token     *mcp.AccessToken
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
	IsRevoked bool
}

// NewIsolatedTokenGuard creates a new isolated token guard for testing.
func NewIsolatedTokenGuard(t *testing.T, encryptionKey []byte) (*IsolatedTokenGuard, error) {
	t.Helper()

	namespace := generateTestNamespace(t)

	return &IsolatedTokenGuard{
		t:         t,
		namespace: namespace,
		storage:   make(map[string]*tokenInfo),
	}, nil
}

// ValidateToken validates a token within this guard's namespace.
func (g *IsolatedTokenGuard) ValidateToken(ctx context.Context, tokenString string) (*mcp.AccessToken, error) {
	g.t.Helper()

	g.mu.Lock()
	defer g.mu.Unlock()

	info, exists := g.storage[tokenString]
	if !exists {
		return nil, fmt.Errorf("token not found in namespace %s", g.namespace)
	}

	if info.IsRevoked {
		return nil, fmt.Errorf("token has been revoked")
	}

	// Update usage statistics
	info.LastUsed = time.Now()
	info.UseCount++

	return info.Token, nil
}

// StoreToken stores a token in this guard's isolated namespace.
func (g *IsolatedTokenGuard) StoreToken(tokenString string, token *mcp.AccessToken) {
	g.t.Helper()

	g.mu.Lock()
	defer g.mu.Unlock()

	g.storage[tokenString] = &tokenInfo{
		Token:     token,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  0,
		IsRevoked: false,
	}
}

// RevokeToken marks a token as revoked in this guard's namespace.
func (g *IsolatedTokenGuard) RevokeToken(tokenString string) error {
	g.t.Helper()

	g.mu.Lock()
	defer g.mu.Unlock()

	info, exists := g.storage[tokenString]
	if !exists {
		return fmt.Errorf("token not found in namespace %s", g.namespace)
	}

	info.IsRevoked = true
	return nil
}

// GetTokenInfo returns information about a token in this guard's namespace.
func (g *IsolatedTokenGuard) GetTokenInfo(tokenString string) (*tokenInfo, bool) {
	g.t.Helper()

	g.mu.RLock()
	defer g.mu.RUnlock()

	info, exists := g.storage[tokenString]
	return info, exists
}

// ListTokens returns all tokens in this guard's namespace.
func (g *IsolatedTokenGuard) ListTokens() map[string]*tokenInfo {
	g.t.Helper()

	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]*tokenInfo)
	for key, value := range g.storage {
		result[key] = value
	}
	return result
}

// Cleanup cleans up all tokens and resources in this guard's namespace.
func (g *IsolatedTokenGuard) Cleanup() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Clear all stored tokens
	for key := range g.storage {
		delete(g.storage, key)
	}
}

// Namespace returns the unique namespace for this guard instance.
func (g *IsolatedTokenGuard) Namespace() string {
	return g.namespace
}

// generateTestNamespace creates a unique namespace for test isolation.
// The namespace includes the test name, current time, and caller information.
func generateTestNamespace(t *testing.T) string {
	t.Helper()

	// Get caller information for additional uniqueness
	_, file, line, ok := runtime.Caller(2)
	callerInfo := "unknown"
	if ok {
		callerInfo = fmt.Sprintf("%s:%d", file, line)
	}

	// Create hash from test name, time, and caller
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%s",
		t.Name(),
		time.Now().UnixNano(),
		callerInfo,
	)))

	return fmt.Sprintf("test-%s-%s",
		t.Name(),
		hex.EncodeToString(hash[:8]),
	)
}

// MockOAuthProvider provides a simple mock implementation for testing.
type MockOAuthProvider struct {
	clients       map[string]*mcp.OAuthClientInfo
	authCodes     map[string]*mcp.AuthorizationCode
	tokens        map[string]*mcp.AccessToken
	refreshTokens map[string]*mcp.RefreshToken
	mu            sync.RWMutex
}

// NewMockOAuthProvider creates a new mock OAuth provider.
func NewMockOAuthProvider() *MockOAuthProvider {
	return &MockOAuthProvider{
		clients:       make(map[string]*mcp.OAuthClientInfo),
		authCodes:     make(map[string]*mcp.AuthorizationCode),
		tokens:        make(map[string]*mcp.AccessToken),
		refreshTokens: make(map[string]*mcp.RefreshToken),
	}
}

// CreateAccessToken creates a mock access token.
func (m *MockOAuthProvider) CreateAccessToken(ctx context.Context, authCode *mcp.AuthorizationCode) (*mcp.AccessToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	token := &mcp.AccessToken{
		AccessToken:  fmt.Sprintf("mock-token-%d", time.Now().UnixNano()),
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: fmt.Sprintf("mock-refresh-%d", time.Now().UnixNano()),
		Scope:        "read write",
	}

	m.tokens[token.AccessToken] = token
	return token, nil
}

// ValidateToken validates a mock token.
func (m *MockOAuthProvider) ValidateToken(ctx context.Context, tokenString string) (*mcp.AccessToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	token, exists := m.tokens[tokenString]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

// RevokeToken revokes a mock token.
func (m *MockOAuthProvider) RevokeToken(ctx context.Context, tokenString string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokens, tokenString)
	return nil
}

// RegisterClient registers a new OAuth client.
func (m *MockOAuthProvider) RegisterClient(ctx context.Context, req *mcp.OAuthClientInfo) (*mcp.OAuthClientInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.ClientID == "" {
		req.ClientID = fmt.Sprintf("mock-client-%d", time.Now().UnixNano())
	}
	if req.ClientSecret == "" {
		req.ClientSecret = fmt.Sprintf("mock-secret-%d", time.Now().UnixNano())
	}

	m.clients[req.ClientID] = req
	return req, nil
}

// GetClient retrieves a client by ID.
func (m *MockOAuthProvider) GetClient(ctx context.Context, clientID string) (*mcp.OAuthClientInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found")
	}

	return client, nil
}

// ValidateClient validates client credentials.
func (m *MockOAuthProvider) ValidateClient(ctx context.Context, clientID, clientSecret string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[clientID]
	if !exists {
		return fmt.Errorf("client not found")
	}

	if client.ClientSecret != clientSecret {
		return fmt.Errorf("invalid client secret")
	}

	return nil
}

// CreateAuthorizationCode creates a mock authorization code.
func (m *MockOAuthProvider) CreateAuthorizationCode(ctx context.Context, req *mcp.AuthorizationRequest) (*mcp.AuthorizationCode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	code := &mcp.AuthorizationCode{
		Code:        fmt.Sprintf("mock-code-%d", time.Now().UnixNano()),
		ClientID:    req.ClientID,
		RedirectURI: req.RedirectURI,
		Scopes:      []string{"read", "write"},
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	m.authCodes[code.Code] = code
	return code, nil
}

// GetAuthorizationCode retrieves an authorization code.
func (m *MockOAuthProvider) GetAuthorizationCode(ctx context.Context, code string) (*mcp.AuthorizationCode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	authCode, exists := m.authCodes[code]
	if !exists {
		return nil, fmt.Errorf("authorization code not found")
	}

	if time.Now().After(authCode.ExpiresAt) {
		return nil, fmt.Errorf("authorization code expired")
	}

	return authCode, nil
}

// RevokeAuthorizationCode revokes an authorization code.
func (m *MockOAuthProvider) RevokeAuthorizationCode(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.authCodes, code)
	return nil
}

// RefreshAccessToken refreshes an access token.
func (m *MockOAuthProvider) RefreshAccessToken(ctx context.Context, refreshToken string) (*mcp.AccessToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the refresh token
	var refreshInfo *mcp.RefreshToken
	for _, rt := range m.refreshTokens {
		if rt.Token == refreshToken {
			refreshInfo = rt
			break
		}
	}

	if refreshInfo == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Create new access token
	token := &mcp.AccessToken{
		AccessToken:  fmt.Sprintf("mock-refreshed-%d", time.Now().UnixNano()),
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: fmt.Sprintf("mock-refresh-%d", time.Now().UnixNano()),
		Scope:        "read write",
		ClientID:     refreshInfo.ClientID,
		Scopes:       refreshInfo.Scopes,
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	m.tokens[token.AccessToken] = token
	return token, nil
}

// ValidateAccessToken validates an access token (alias for ValidateToken).
func (m *MockOAuthProvider) ValidateAccessToken(ctx context.Context, token string) (*mcp.AccessToken, error) {
	return m.ValidateToken(ctx, token)
}

// ValidateScopes validates the requested scopes for a client.
func (m *MockOAuthProvider) ValidateScopes(ctx context.Context, clientID string, scopes []string) error {
	// For mock, accept any scopes
	return nil
}
