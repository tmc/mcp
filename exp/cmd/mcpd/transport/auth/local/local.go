// Package local provides local username/password authentication
package local

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/authtypes"
	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/session"
)

// Provider implements local authentication
type Provider struct {
	userStore    authtypes.UserStore
	sessionStore authtypes.SessionStore
	config       *authtypes.Config
}

// User represents a local user account
type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Email        string    `json:"email,omitempty"`
	FullName     string    `json:"full_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

// FileUserStore implements UserStore using file-based storage
type FileUserStore struct {
	users    map[string]*User
	filePath string
}

// MemoryUserStore implements UserStore using in-memory storage
type MemoryUserStore struct {
	users map[string]*User
}

// NewProvider creates a new local authentication provider
func NewProvider(config *authtypes.Config, userStore authtypes.UserStore) *Provider {
	sessionStore := session.NewMemoryStore(config.SessionTimeout)

	return &Provider{
		userStore:    userStore,
		sessionStore: sessionStore,
		config:       config,
	}
}

// NewFileUserStore creates a new file-based user store
func NewFileUserStore(filePath string) (*FileUserStore, error) {
	store := &FileUserStore{
		users:    make(map[string]*User),
		filePath: filePath,
	}

	if err := store.loadFromFile(); err != nil {
		return nil, err
	}

	return store, nil
}

// NewMemoryUserStore creates a new memory-based user store
func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users: make(map[string]*User),
	}
}

// Provider interface implementation

// Middleware returns HTTP middleware that enforces local authentication
func (p *Provider) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login and logout endpoints
		if r.URL.Path == p.config.LoginPath || r.URL.Path == p.config.LogoutPath {
			p.handleAuthEndpoints(w, r, next)
			return
		}

		// Check session cookie
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

		// Add user info to request context for logging
		r.Header.Set("X-Authenticated-User", userInfo.Username)
		next.ServeHTTP(w, r)
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "local"
}

// IsConfigured returns true if the provider is properly configured
func (p *Provider) IsConfigured() bool {
	return p.userStore != nil && len(p.userStore.ListUsers()) > 0
}

// Authentication endpoint handlers

func (p *Provider) handleAuthEndpoints(w http.ResponseWriter, r *http.Request, next http.Handler) {
	switch r.URL.Path {
	case p.config.LoginPath:
		p.handleLogin(w, r)
	case p.config.LogoutPath:
		p.handleLogout(w, r)
	default:
		next.ServeHTTP(w, r)
	}
}

func (p *Provider) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		p.serveLoginPage(w, r)
		return
	}

	if r.Method == "POST" {
		p.handleLoginSubmit(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (p *Provider) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		p.redirectToLoginWithError(w, r, "Username and password required")
		return
	}

	// Authenticate user
	userInfo, valid := p.userStore.Authenticate(username, password)
	if !valid {
		p.redirectToLoginWithError(w, r, "Invalid username or password")
		return
	}

	// Create session
	sessionID, err := p.sessionStore.Create(userInfo)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	session.SetSessionCookie(w, sessionID, p.config.SecureCookies, p.config.CookieDomain)

	// Redirect to original destination or home
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
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
		loginURL += "?redirect=" + r.URL.Path
	}
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

func (p *Provider) redirectToLoginWithError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	loginURL := p.config.LoginPath + "?error=" + errorMsg
	http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
}

func (p *Provider) serveLoginPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>MCP Local Authentication</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 400px; margin: 100px auto; padding: 20px; background: #f5f5f5; }
        .login-form { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="text"], input[type="password"] { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
        .btn { background: #007cba; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; width: 100%; font-size: 16px; }
        .btn:hover { background: #005a87; }
        .error { color: red; margin-bottom: 15px; padding: 10px; background: #fee; border: 1px solid #fcc; border-radius: 4px; }
        .info { color: #666; font-size: 0.9em; margin-top: 20px; text-align: center; }
        h2 { text-align: center; margin-bottom: 30px; color: #333; }
    </style>
</head>
<body>
    <div class="login-form">
        <h2>🔐 MCP Authentication</h2>
        {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
        <form method="POST" action="{{.LoginPath}}">
            <div class="form-group">
                <label for="username">Username:</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>
            <div class="form-group">
                <label for="password">Password:</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn">Login</button>
        </form>
        <div class="info">
            <p>Local authentication mode • No external dependencies</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("login").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Error     string
		LoginPath string
	}{
		Error:     r.URL.Query().Get("error"),
		LoginPath: p.config.LoginPath,
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

// FileUserStore implementation

func (s *FileUserStore) loadFromFile() error {
	if s.filePath == "" {
		return nil
	}

	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Empty store if file doesn't exist
		}
		return fmt.Errorf("failed to open users file: %w", err)
	}
	defer file.Close()

	// Try JSON format first
	var users []*User
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&users); err == nil {
		// JSON format
		for _, user := range users {
			s.users[user.Username] = user
		}
		return nil
	}

	// Fall back to simple text format (username:password per line)
	file.Seek(0, 0) // Reset file position
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])

		if username != "" && password != "" {
			if err := s.AddUser(username, password, "", ""); err != nil {
				continue // Skip users that fail to add
			}
		}
	}

	return scanner.Err()
}

func (s *FileUserStore) saveToFile() error {
	if s.filePath == "" {
		return nil
	}

	file, err := os.Create(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to create users file: %w", err)
	}
	defer file.Close()

	var users []*User
	for _, user := range s.users {
		users = append(users, user)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(users)
}

func (s *FileUserStore) AddUser(username, password, email, fullName string) error {
	if _, exists := s.users[username]; exists {
		return fmt.Errorf("user %s already exists", username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Username:     username,
		PasswordHash: string(hash),
		Email:        email,
		FullName:     fullName,
		CreatedAt:    time.Now(),
	}

	s.users[username] = user
	return s.saveToFile()
}

func (s *FileUserStore) Authenticate(username, password string) (authtypes.UserInfo, bool) {
	user, exists := s.users[username]
	if !exists {
		return authtypes.UserInfo{}, false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return authtypes.UserInfo{}, false
	}

	// Update last login time
	user.LastLogin = time.Now()
	s.saveToFile() // Ignore errors for last login update

	userInfo := authtypes.UserInfo{
		ID:       user.Username,
		Username: user.Username,
		Email:    user.Email,
		Name:     user.FullName,
		LoginAt:  time.Now(),
	}

	return userInfo, true
}

func (s *FileUserStore) GetUser(username string) (authtypes.UserInfo, bool) {
	user, exists := s.users[username]
	if !exists {
		return authtypes.UserInfo{}, false
	}

	userInfo := authtypes.UserInfo{
		ID:       user.Username,
		Username: user.Username,
		Email:    user.Email,
		Name:     user.FullName,
		LoginAt:  user.LastLogin,
	}

	return userInfo, true
}

func (s *FileUserStore) ListUsers() []string {
	var usernames []string
	for username := range s.users {
		usernames = append(usernames, username)
	}
	return usernames
}

// MemoryUserStore implementation

func (s *MemoryUserStore) AddUser(username, password, email, fullName string) error {
	if _, exists := s.users[username]; exists {
		return fmt.Errorf("user %s already exists", username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Username:     username,
		PasswordHash: string(hash),
		Email:        email,
		FullName:     fullName,
		CreatedAt:    time.Now(),
	}

	s.users[username] = user
	return nil
}

func (s *MemoryUserStore) Authenticate(username, password string) (authtypes.UserInfo, bool) {
	user, exists := s.users[username]
	if !exists {
		return authtypes.UserInfo{}, false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return authtypes.UserInfo{}, false
	}

	// Update last login time
	user.LastLogin = time.Now()

	userInfo := authtypes.UserInfo{
		ID:       user.Username,
		Username: user.Username,
		Email:    user.Email,
		Name:     user.FullName,
		LoginAt:  time.Now(),
	}

	return userInfo, true
}

func (s *MemoryUserStore) GetUser(username string) (authtypes.UserInfo, bool) {
	user, exists := s.users[username]
	if !exists {
		return authtypes.UserInfo{}, false
	}

	userInfo := authtypes.UserInfo{
		ID:       user.Username,
		Username: user.Username,
		Email:    user.Email,
		Name:     user.FullName,
		LoginAt:  user.LastLogin,
	}

	return userInfo, true
}

func (s *MemoryUserStore) ListUsers() []string {
	var usernames []string
	for username := range s.users {
		usernames = append(usernames, username)
	}
	return usernames
}

// Helper functions

// CreateUsersFromEnv creates users from environment variables
func CreateUsersFromEnv(store interface {
	AddUser(username, password, email, fullName string) error
}) error {
	usersEnv := os.Getenv("MCPD_USERS")
	if usersEnv == "" {
		return nil
	}

	pairs := strings.Split(usersEnv, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			continue
		}

		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])

		if username != "" && password != "" {
			store.AddUser(username, password, "", "")
		}
	}

	return nil
}

// CreateUsersFromCommandLine creates users from command-line format
func CreateUsersFromCommandLine(store interface {
	AddUser(username, password, email, fullName string) error
}, usersStr string) error {
	if usersStr == "" {
		return nil
	}

	pairs := strings.Split(usersStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			continue
		}

		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])

		if username != "" && password != "" {
			store.AddUser(username, password, "", "")
		}
	}

	return nil
}
