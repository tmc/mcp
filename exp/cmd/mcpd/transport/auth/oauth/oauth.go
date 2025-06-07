// Package oauth provides OAuth 2.0 authentication for external providers
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/authtypes"
	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/session"
)

// Provider implements OAuth 2.0 authentication
type Provider struct {
	config       *authtypes.Config
	oauthConfig  *oauth2.Config
	sessionStore authtypes.SessionStore
	stateStore   map[string]time.Time // Simple state validation
}

// UserInfo represents user information from OAuth provider
type OAuthUserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	ID    string `json:"id"`
}

// NewProvider creates a new OAuth authentication provider
func NewProvider(config *authtypes.Config) (*Provider, error) {
	oauthConfig, err := createOAuthConfig(config)
	if err != nil {
		return nil, err
	}

	sessionStore := session.NewMemoryStore(config.SessionTimeout)

	return &Provider{
		config:       config,
		oauthConfig:  oauthConfig,
		sessionStore: sessionStore,
		stateStore:   make(map[string]time.Time),
	}, nil
}

// createOAuthConfig creates OAuth2 config based on provider
func createOAuthConfig(config *authtypes.Config) (*oauth2.Config, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
	}

	switch strings.ToLower(config.Provider) {
	case "google":
		oauthConfig.Endpoint = google.Endpoint
		oauthConfig.Scopes = []string{"openid", "email", "profile"}
	case "github":
		oauthConfig.Endpoint = github.Endpoint
		oauthConfig.Scopes = []string{"user:email"}
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", config.Provider)
	}

	return oauthConfig, nil
}

// Provider interface implementation

// Middleware returns HTTP middleware that enforces OAuth authentication
func (p *Provider) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for OAuth callback and login endpoints
		if r.URL.Path == p.config.CallbackPath || r.URL.Path == p.config.LoginPath || r.URL.Path == p.config.LogoutPath {
			p.handleAuthEndpoints(w, r, next)
			return
		}

		// Check for session cookie
		sessionID, hasSession := session.GetSessionFromRequest(r)
		if !hasSession {
			p.redirectToLogin(w, r)
			return
		}

		// Validate session
		userInfo, valid := p.sessionStore.Validate(sessionID)
		if !valid {
			p.redirectToLogin(w, r)
			return
		}

		// Check if user is authorized
		if !p.isUserAuthorized(userInfo.Email) {
			http.Error(w, "Unauthorized: User not in authorized list", http.StatusForbidden)
			return
		}

		// Add user info to request context
		r.Header.Set("X-Authenticated-User", userInfo.Username)
		next.ServeHTTP(w, r)
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return p.config.Provider
}

// IsConfigured returns true if the provider is properly configured
func (p *Provider) IsConfigured() bool {
	return p.config.ClientID != "" && p.config.ClientSecret != "" && p.config.RedirectURL != ""
}

// Authentication endpoint handlers

func (p *Provider) handleAuthEndpoints(w http.ResponseWriter, r *http.Request, next http.Handler) {
	switch r.URL.Path {
	case p.config.LoginPath:
		p.handleLogin(w, r)
	case p.config.LogoutPath:
		p.handleLogout(w, r)
	case p.config.CallbackPath:
		p.handleCallback(w, r)
	default:
		next.ServeHTTP(w, r)
	}
}

func (p *Provider) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Generate state for CSRF protection
	state := generateState()
	p.stateStore[state] = time.Now().Add(10 * time.Minute) // 10 minute expiry

	// Clean up expired states
	p.cleanupExpiredStates()

	// Redirect to OAuth provider
	authURL := p.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (p *Provider) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter
	state := r.URL.Query().Get("state")
	if expiry, exists := p.stateStore[state]; !exists || time.Now().After(expiry) {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	delete(p.stateStore, state) // Use state only once

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := p.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Failed to exchange OAuth code: %v", err)
		http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Get user info from provider
	userInfo, err := p.getUserInfo(token)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Check if user is authorized
	if !p.isUserAuthorized(userInfo.Email) {
		http.Error(w, "Unauthorized: User not in authorized list", http.StatusForbidden)
		return
	}

	// Create session
	authUserInfo := authtypes.UserInfo{
		ID:       userInfo.ID,
		Username: userInfo.Email, // Use email as username for OAuth
		Email:    userInfo.Email,
		Name:     userInfo.Name,
		LoginAt:  time.Now(),
	}

	sessionID, err := p.sessionStore.Create(authUserInfo)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	session.SetSessionCookie(w, sessionID, p.config.SecureCookies, p.config.CookieDomain)

	// Redirect to home or original destination
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (p *Provider) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session
	if sessionID, hasSession := session.GetSessionFromRequest(r); hasSession {
		p.sessionStore.Destroy(sessionID)
	}

	// Clear cookie
	session.ClearSessionCookie(w, p.config.CookieDomain)

	http.Redirect(w, r, p.config.LoginPath, http.StatusTemporaryRedirect)
}

func (p *Provider) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	loginURL := p.config.LoginPath
	if r.URL.Path != "/" {
		loginURL += "?redirect=" + url.QueryEscape(r.URL.Path)
	}
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

func (p *Provider) getUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	client := p.oauthConfig.Client(context.Background(), token)

	var userInfoURL string
	switch strings.ToLower(p.config.Provider) {
	case "google":
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	case "github":
		userInfoURL = "https://api.github.com/user"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", p.config.Provider)
	}

	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	var userInfo OAuthUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// For GitHub, we need to get the email separately if it's not public
	if strings.ToLower(p.config.Provider) == "github" && userInfo.Email == "" {
		email, err := p.getGitHubEmail(client)
		if err == nil {
			userInfo.Email = email
		}
	}

	return &userInfo, nil
}

func (p *Provider) getGitHubEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email found")
}

func (p *Provider) isUserAuthorized(email string) bool {
	if len(p.config.AuthorizedUsers) == 0 {
		return true // No restrictions if no authorized users specified
	}

	for _, authorizedEmail := range p.config.AuthorizedUsers {
		if email == authorizedEmail {
			return true
		}
	}

	return false
}

func (p *Provider) cleanupExpiredStates() {
	now := time.Now()
	for state, expiry := range p.stateStore {
		if now.After(expiry) {
			delete(p.stateStore, state)
		}
	}
}

// Helper functions

func generateState() string {
	return session.GenerateSecureID() // Reuse the secure ID generation from session package
}

// ServeLoginPage serves a basic login page for OAuth providers
func (p *Provider) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>MCP OAuth Authentication</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 400px; margin: 100px auto; padding: 20px; background: #f5f5f5; }
        .login-form { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); text-align: center; }
        .btn { background: #007cba; color: white; padding: 12px 24px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; display: inline-block; font-size: 16px; margin: 10px; }
        .btn:hover { background: #005a87; }
        .btn.google { background: #4285f4; }
        .btn.github { background: #333; }
        .info { color: #666; font-size: 0.9em; margin-top: 20px; }
        h2 { margin-bottom: 30px; color: #333; }
    </style>
</head>
<body>
    <div class="login-form">
        <h2>🔐 MCP OAuth Authentication</h2>
        <p>Click below to authenticate with {{.ProviderName}}:</p>
        <a href="{{.LoginPath}}" class="btn {{.Provider}}">
            Login with {{.ProviderName}}
        </a>
        <div class="info">
            <p>You will be redirected to {{.ProviderName}} to complete authentication.</p>
        </div>
    </div>
</body>
</html>`

	// This would need template execution - simplified for now
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, tmpl)
}
