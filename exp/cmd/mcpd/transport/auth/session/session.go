// Package session provides session management for authentication
package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/authtypes"
)

// MemoryStore implements SessionStore using in-memory storage
type MemoryStore struct {
	sessions map[string]*Session
	timeout  time.Duration
	mu       sync.RWMutex
}

// Session represents a user session
type Session struct {
	ID        string           `json:"id"`
	UserInfo  authtypes.UserInfo   `json:"user_info"`
	CreatedAt time.Time        `json:"created_at"`
	ExpiresAt time.Time        `json:"expires_at"`
	LastSeen  time.Time        `json:"last_seen"`
}

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore(timeout time.Duration) *MemoryStore {
	store := &MemoryStore{
		sessions: make(map[string]*Session),
		timeout:  timeout,
	}
	
	// Start cleanup goroutine
	go store.startCleanup()
	
	return store
}

// Create creates a new session for the user
func (s *MemoryStore) Create(userInfo authtypes.UserInfo) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	sessionID := generateSessionID()
	now := time.Now()
	
	session := &Session{
		ID:        sessionID,
		UserInfo:  userInfo,
		CreatedAt: now,
		ExpiresAt: now.Add(s.timeout),
		LastSeen:  now,
	}
	
	s.sessions[sessionID] = session
	return sessionID, nil
}

// Validate validates a session and returns user info
func (s *MemoryStore) Validate(sessionID string) (authtypes.UserInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	session, exists := s.sessions[sessionID]
	if !exists {
		return authtypes.UserInfo{}, false
	}
	
	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, sessionID)
		return authtypes.UserInfo{}, false
	}
	
	// Update last seen time
	session.LastSeen = time.Now()
	
	return session.UserInfo, true
}

// Destroy destroys a session
func (s *MemoryStore) Destroy(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.sessions, sessionID)
	return nil
}

// Cleanup removes expired sessions
func (s *MemoryStore) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
	
	return nil
}

// GetSessionCount returns the number of active sessions
func (s *MemoryStore) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// GetSessionInfo returns session information (for debugging)
func (s *MemoryStore) GetSessionInfo(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, false
	}
	
	// Return a copy to avoid race conditions
	sessionCopy := *session
	return &sessionCopy, true
}

// ListActiveSessions returns all active session IDs
func (s *MemoryStore) ListActiveSessions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var sessions []string
	now := time.Now()
	
	for id, session := range s.sessions {
		if now.Before(session.ExpiresAt) {
			sessions = append(sessions, id)
		}
	}
	
	return sessions
}

// startCleanup starts a background goroutine to clean up expired sessions
func (s *MemoryStore) startCleanup() {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()
	
	for range ticker.C {
		s.Cleanup()
	}
}

// generateSessionID generates a cryptographically secure session ID
func generateSessionID() string {
	bytes := make([]byte, 32) // 256 bits
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateSecureID generates a cryptographically secure random ID (exported for use by other packages)
func GenerateSecureID() string {
	return generateSessionID()
}

// Helper functions for HTTP cookie management

// SetSessionCookie sets a session cookie on the HTTP response
func SetSessionCookie(w http.ResponseWriter, sessionID string, secure bool, domain string) {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 24 hours
	}
	
	if domain != "" {
		cookie.Domain = domain
	}
	
	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(w http.ResponseWriter, domain string) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	
	if domain != "" {
		cookie.Domain = domain
	}
	
	http.SetCookie(w, cookie)
}

// GetSessionFromRequest extracts session ID from HTTP request
func GetSessionFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}