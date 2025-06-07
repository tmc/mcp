// Package transport provides integration between the auth system and HTTP transport
package transport

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth"
	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/authtypes"
)

// AuthConfig holds authentication configuration for the transport layer
type AuthConfig struct {
	Provider     string
	ClientID     string
	ClientSecret string
	RedirectURL  string

	// Local auth options
	LocalUsersFile   string
	LocalUsersString string
	LocalPersistFile string
	LocalUsers       map[string]string

	// Authorization
	AuthorizedUsers []string

	// Security
	SecureCookies bool
	CookieDomain  string
}

// SetupAuthentication creates and configures an authentication provider
func SetupAuthentication(config *AuthConfig, baseURL string) (authtypes.Provider, error) {
	if config.Provider == "" {
		return nil, fmt.Errorf("authentication provider not specified")
	}

	// Create auth config
	authConfig := authtypes.NewConfig().
		SetProvider(config.Provider).
		SetAuthorizedUsers(config.AuthorizedUsers).
		SetSecurityOptions(config.SecureCookies, config.CookieDomain)

	// Set OAuth credentials if using external provider
	if config.Provider != "local" {
		if config.ClientID == "" || config.ClientSecret == "" {
			return nil, fmt.Errorf("OAuth provider %s requires client ID and secret", config.Provider)
		}

		redirectURL := config.RedirectURL
		if redirectURL == "" {
			redirectURL = baseURL + "/auth/callback"
		}

		authConfig.SetOAuthCredentials(config.ClientID, config.ClientSecret, redirectURL)
	}

	// Create local config for local provider
	var localConfig *authtypes.LocalConfig
	if config.Provider == "local" {
		localConfig = &authtypes.LocalConfig{
			UsersFile:   config.LocalUsersFile,
			UsersString: config.LocalUsersString,
			PersistFile: config.LocalPersistFile,
			Users:       config.LocalUsers,
		}
	}

	// Create provider
	provider, err := auth.NewProvider(authConfig, localConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth provider: %w", err)
	}

	if !provider.IsConfigured() {
		return nil, fmt.Errorf("authentication provider %s is not properly configured", config.Provider)
	}

	slog.Info("Authentication configured",
		"provider", provider.Name(),
		"authorized_users", len(config.AuthorizedUsers))

	return provider, nil
}

// WrapWithAuth wraps an HTTP handler with authentication middleware
func WrapWithAuth(handler http.Handler, provider authtypes.Provider) http.Handler {
	if provider == nil {
		return handler
	}

	return provider.Middleware(handler)
}

// ParseLocalUsers parses a command-line style users string into a map
func ParseLocalUsers(usersString string) map[string]string {
	users := make(map[string]string)

	if usersString == "" {
		return users
	}

	pairs := strings.Split(usersString, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			username := strings.TrimSpace(parts[0])
			password := strings.TrimSpace(parts[1])
			if username != "" && password != "" {
				users[username] = password
			}
		}
	}

	return users
}

// Legacy compatibility functions for the old OAuth system

// OAuthConfig provides backward compatibility with the old OAuth config
type OAuthConfig struct {
	provider authtypes.Provider
}

// NewOAuthConfig creates a new OAuth config (for backward compatibility)
func NewOAuthConfig(clientID, secret, provider, callback string, authorizedUsers []string, baseURL string) *OAuthConfig {
	config := &AuthConfig{
		Provider:        provider,
		ClientID:        clientID,
		ClientSecret:    secret,
		RedirectURL:     baseURL + callback,
		AuthorizedUsers: authorizedUsers,
	}

	authProvider, err := SetupAuthentication(config, baseURL)
	if err != nil {
		slog.Error("Failed to create OAuth config", "error", err)
		return nil
	}

	return &OAuthConfig{
		provider: authProvider,
	}
}

// NewLocalAuthConfig creates a local auth config (for backward compatibility)
func NewLocalAuthConfig(userStore interface{}) *OAuthConfig {
	// This is more complex to maintain backward compatibility
	// For now, return a simple local auth setup
	config := authtypes.NewConfig().SetProvider("local")

	// Create a simple local provider - this would need more work for full compatibility
	localConfig := &authtypes.LocalConfig{
		Users: map[string]string{"admin": "admin"}, // Default
	}

	provider, err := auth.NewProvider(config, localConfig)
	if err != nil {
		slog.Error("Failed to create local auth config", "error", err)
		return nil
	}

	return &OAuthConfig{
		provider: provider,
	}
}

// OAuthMiddleware returns the authentication middleware (for backward compatibility)
func (oc *OAuthConfig) OAuthMiddleware(next http.Handler) http.Handler {
	if oc == nil || oc.provider == nil {
		return next
	}
	return oc.provider.Middleware(next)
}
