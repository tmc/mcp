// Package authtypes provides core authentication interfaces and types
package authtypes

import (
	"net/http"
	"time"
)

// Provider defines the interface for authentication providers
type Provider interface {
	// Middleware returns HTTP middleware that enforces authentication
	Middleware(next http.Handler) http.Handler
	
	// Name returns the provider name (e.g., "local", "google", "github")
	Name() string
	
	// IsConfigured returns true if the provider is properly configured
	IsConfigured() bool
}

// UserInfo represents authenticated user information
type UserInfo struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email,omitempty"`
	Name     string    `json:"name,omitempty"`
	LoginAt  time.Time `json:"login_at"`
}

// SessionStore defines the interface for session management
type SessionStore interface {
	// Create creates a new session for the user
	Create(userInfo UserInfo) (sessionID string, err error)
	
	// Validate validates a session and returns user info
	Validate(sessionID string) (userInfo UserInfo, valid bool)
	
	// Destroy destroys a session
	Destroy(sessionID string) error
	
	// Cleanup removes expired sessions
	Cleanup() error
}

// UserStore defines the interface for user management
type UserStore interface {
	// Authenticate verifies user credentials
	Authenticate(username, password string) (userInfo UserInfo, valid bool)
	
	// GetUser retrieves user information by username
	GetUser(username string) (userInfo UserInfo, exists bool)
	
	// ListUsers returns all usernames
	ListUsers() []string
}

// Config holds common authentication configuration
type Config struct {
	// Provider type (local, google, github, etc.)
	Provider string
	
	// Session configuration
	SessionTimeout time.Duration
	SecureCookies  bool
	CookieDomain   string
	
	// URLs
	LoginPath    string
	LogoutPath   string
	CallbackPath string
	
	// For external OAuth providers
	ClientID     string
	ClientSecret string
	RedirectURL  string
	
	// For local authentication
	AuthorizedUsers []string
}

// NewConfig creates a new auth config with defaults
func NewConfig() *Config {
	return &Config{
		SessionTimeout: 24 * time.Hour,
		LoginPath:      "/login",
		LogoutPath:     "/logout", 
		CallbackPath:   "/auth/callback",
		SecureCookies:  false, // Will be set to true for HTTPS
	}
}

// SetProvider sets the authentication provider
func (c *Config) SetProvider(provider string) *Config {
	c.Provider = provider
	return c
}

// SetSessionTimeout sets the session timeout duration
func (c *Config) SetSessionTimeout(timeout time.Duration) *Config {
	c.SessionTimeout = timeout
	return c
}

// SetOAuthCredentials sets OAuth client credentials
func (c *Config) SetOAuthCredentials(clientID, clientSecret, redirectURL string) *Config {
	c.ClientID = clientID
	c.ClientSecret = clientSecret
	c.RedirectURL = redirectURL
	return c
}

// SetAuthorizedUsers sets the list of authorized users
func (c *Config) SetAuthorizedUsers(users []string) *Config {
	c.AuthorizedUsers = users
	return c
}

// SetSecurityOptions sets security-related options
func (c *Config) SetSecurityOptions(secureCookies bool, domain string) *Config {
	c.SecureCookies = secureCookies
	c.CookieDomain = domain
	return c
}

// LocalConfig holds local authentication specific configuration
type LocalConfig struct {
	// File-based users
	UsersFile string
	
	// Command-line users (username:password,username2:password2)
	UsersString string
	
	// Persistence file for JSON user store
	PersistFile string
	
	// In-memory users map (for programmatic setup)
	Users map[string]string
}