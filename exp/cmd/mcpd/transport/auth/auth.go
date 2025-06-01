// Package auth provides authentication factories and utilities for mcpd
package auth

import (
	"fmt"
	"strings"

	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/authtypes"
	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/local"
	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/oauth"
)

// NewProvider creates a new authentication provider based on the configuration
func NewProvider(config *authtypes.Config, localConfig *authtypes.LocalConfig) (authtypes.Provider, error) {
	switch strings.ToLower(config.Provider) {
	case "local":
		return createLocalProvider(config, localConfig)
	case "google", "github":
		return createOAuthProvider(config)
	default:
		return nil, fmt.Errorf("unsupported authentication provider: %s", config.Provider)
	}
}

func createLocalProvider(config *authtypes.Config, localConfig *authtypes.LocalConfig) (authtypes.Provider, error) {
	var userStore authtypes.UserStore
	var err error
	
	// Determine which user store to use
	if localConfig.UsersFile != "" {
		// File-based user store
		userStore, err = local.NewFileUserStore(localConfig.UsersFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create file user store: %w", err)
		}
	} else {
		// Memory-based user store
		memoryStore := local.NewMemoryUserStore()
		userStore = memoryStore
		
		// Add users from various sources
		if err := addUsersToStore(memoryStore, localConfig); err != nil {
			return nil, fmt.Errorf("failed to add users: %w", err)
		}
		
		// If no users exist, create a default admin user
		if len(userStore.ListUsers()) == 0 {
			if err := memoryStore.AddUser("admin", "admin", "", "Administrator"); err != nil {
				return nil, fmt.Errorf("failed to create default admin user: %w", err)
			}
		}
	}
	
	return local.NewProvider(config, userStore), nil
}

func createOAuthProvider(config *authtypes.Config) (authtypes.Provider, error) {
	if config.ClientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("OAuth provider %s requires client ID and secret", config.Provider)
	}
	
	if config.RedirectURL == "" {
		return nil, fmt.Errorf("OAuth provider %s requires redirect URL", config.Provider)
	}
	
	return oauth.NewProvider(config)
}

func addUsersToStore(store interface {
	AddUser(username, password, email, fullName string) error
}, localConfig *authtypes.LocalConfig) error {
	
	// Add from environment variables
	if err := local.CreateUsersFromEnv(store); err != nil {
		return fmt.Errorf("failed to create users from environment: %w", err)
	}
	
	// Add from command line string
	if localConfig.UsersString != "" {
		if err := local.CreateUsersFromCommandLine(store, localConfig.UsersString); err != nil {
			return fmt.Errorf("failed to create users from command line: %w", err)
		}
	}
	
	// Add from in-memory map
	if localConfig.Users != nil {
		for username, password := range localConfig.Users {
			if err := store.AddUser(username, password, "", ""); err != nil {
				return fmt.Errorf("failed to add user %s: %w", username, err)
			}
		}
	}
	
	return nil
}

// SetupLocalAuth is a helper function to set up local authentication with various options
func SetupLocalAuth(config *authtypes.Config, options ...LocalOption) (authtypes.Provider, error) {
	localConfig := &authtypes.LocalConfig{}
	
	// Apply options
	for _, opt := range options {
		opt(localConfig)
	}
	
	return createLocalProvider(config, localConfig)
}

// LocalOption defines options for local authentication setup
type LocalOption func(*authtypes.LocalConfig)

// WithUsersFile configures local auth to use a users file
func WithUsersFile(filePath string) LocalOption {
	return func(c *authtypes.LocalConfig) {
		c.UsersFile = filePath
	}
}

// WithUsersString configures local auth with command-line style users
func WithUsersString(users string) LocalOption {
	return func(c *authtypes.LocalConfig) {
		c.UsersString = users
	}
}

// WithPersistFile configures local auth to persist users to a JSON file
func WithPersistFile(filePath string) LocalOption {
	return func(c *authtypes.LocalConfig) {
		c.PersistFile = filePath
	}
}

// WithUsers configures local auth with a map of users
func WithUsers(users map[string]string) LocalOption {
	return func(c *authtypes.LocalConfig) {
		c.Users = users
	}
}

// WithEnvironmentUsers configures local auth to read users from MCPD_USERS environment variable
func WithEnvironmentUsers() LocalOption {
	return func(c *authtypes.LocalConfig) {
		// This is handled automatically in addUsersToStore
	}
}

// QuickLocalAuth is a convenience function for simple local auth setup
func QuickLocalAuth(config *authtypes.Config, users map[string]string) (authtypes.Provider, error) {
	return SetupLocalAuth(config, WithUsers(users))
}

// FileBasedLocalAuth is a convenience function for file-based local auth
func FileBasedLocalAuth(config *authtypes.Config, usersFile string) (authtypes.Provider, error) {
	return SetupLocalAuth(config, WithUsersFile(usersFile))
}

// EnvironmentLocalAuth is a convenience function for environment-based local auth
func EnvironmentLocalAuth(config *authtypes.Config) (authtypes.Provider, error) {
	return SetupLocalAuth(config, WithEnvironmentUsers())
}