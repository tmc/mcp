// mcp-tunnel server - runs on Cloud Run to provide public endpoints
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// TunnelServer manages tunnel connections
type TunnelServer struct {
	tunnels    map[string]*Tunnel
	tunnelsMux sync.RWMutex
	upgrader   websocket.Upgrader
}

// Tunnel represents a single tunnel connection
type Tunnel struct {
	ID          string
	Token       string
	ClientConn  *websocket.Conn
	Created     time.Time
	LastPing    time.Time
	mu          sync.Mutex
	Transport   string // "stdio", "sse", "http"
	sessions    map[string]*Session
	sessionsMux sync.RWMutex
}

// Session represents an SSE session
type Session struct {
	ID         string
	ClientChan chan []byte
	Created    time.Time
}

// Message types for tunnel protocol
type TunnelMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func NewTunnelServer() *TunnelServer {
	return &TunnelServer{
		tunnels: make(map[string]*Tunnel),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now (configure for production)
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket handles the WebSocket connection from local tunnel client
func (s *TunnelServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]
	token := r.Header.Get("Authorization")

	s.tunnelsMux.RLock()
	tunnel, exists := s.tunnels[tunnelID]
	s.tunnelsMux.RUnlock()

	if !exists || tunnel.Token != token {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	tunnel.mu.Lock()
	tunnel.ClientConn = conn
	tunnel.LastPing = time.Now()
	tunnel.mu.Unlock()

	log.Printf("Tunnel %s connected", tunnelID)

	// Handle tunnel messages
	for {
		var msg TunnelMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Read error for tunnel %s: %v", tunnelID, err)
			break
		}

		tunnel.LastPing = time.Now()

		switch msg.Type {
		case "pong":
			// Client is alive
		case "response":
			// Handle response from local server
			s.handleResponse(tunnel, msg)
		case "error":
			log.Printf("Tunnel %s error: %s", tunnelID, msg.Error)
		}
	}

	// Clean up on disconnect
	tunnel.mu.Lock()
	tunnel.ClientConn = nil
	tunnel.mu.Unlock()
	log.Printf("Tunnel %s disconnected", tunnelID)
}

// CreateTunnel creates a new tunnel
func (s *TunnelServer) CreateTunnel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transport string `json:"transport"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tunnelID := uuid.New().String()[:8]
	token := uuid.New().String()

	tunnel := &Tunnel{
		ID:        tunnelID,
		Token:     token,
		Created:   time.Now(),
		LastPing:  time.Now(),
		Transport: req.Transport,
		sessions:  make(map[string]*Session),
	}

	s.tunnelsMux.Lock()
	s.tunnels[tunnelID] = tunnel
	s.tunnelsMux.Unlock()

	// Schedule cleanup after 24 hours
	go func() {
		time.Sleep(24 * time.Hour)
		s.tunnelsMux.Lock()
		delete(s.tunnels, tunnelID)
		s.tunnelsMux.Unlock()
	}()

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = r.Host
	}

	resp := map[string]interface{}{
		"tunnel_id": tunnelID,
		"token":     token,
		"url":       fmt.Sprintf("https://%s/tunnels/%s", baseURL, tunnelID),
		"ws_url":    fmt.Sprintf("wss://%s/tunnels/%s/ws", baseURL, tunnelID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleMCP handles incoming MCP requests
func (s *TunnelServer) HandleMCP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	s.tunnelsMux.RLock()
	tunnel, exists := s.tunnels[tunnelID]
	s.tunnelsMux.RUnlock()

	if !exists {
		http.Error(w, "Tunnel not found", http.StatusNotFound)
		return
	}

	if tunnel.ClientConn == nil {
		http.Error(w, "Tunnel not connected", http.StatusServiceUnavailable)
		return
	}

	// Read request body
	var mcpRequest json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&mcpRequest); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Forward to tunnel client
	requestID := uuid.New().String()
	msg := TunnelMessage{
		Type:    "request",
		ID:      requestID,
		Payload: mcpRequest,
	}

	tunnel.mu.Lock()
	responseChan := make(chan TunnelMessage, 1)
	tunnel.mu.Unlock()

	// Send request to client
	if err := tunnel.ClientConn.WriteJSON(msg); err != nil {
		http.Error(w, "Tunnel error", http.StatusInternalServerError)
		return
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		if response.Error != "" {
			http.Error(w, response.Error, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(response.Payload)
	case <-time.After(30 * time.Second):
		http.Error(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// HandleSSE handles Server-Sent Events connections
func (s *TunnelServer) HandleSSE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]

	s.tunnelsMux.RLock()
	tunnel, exists := s.tunnels[tunnelID]
	s.tunnelsMux.RUnlock()

	if !exists {
		http.Error(w, "Tunnel not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create session
	sessionID := uuid.New().String()
	session := &Session{
		ID:         sessionID,
		ClientChan: make(chan []byte, 10),
		Created:    time.Now(),
	}

	tunnel.sessionsMux.Lock()
	tunnel.sessions[sessionID] = session
	tunnel.sessionsMux.Unlock()

	defer func() {
		tunnel.sessionsMux.Lock()
		delete(tunnel.sessions, sessionID)
		tunnel.sessionsMux.Unlock()
		close(session.ClientChan)
	}()

	// Send initial endpoint event
	fmt.Fprintf(w, "event: endpoint\n")
	fmt.Fprintf(w, "data: /tunnels/%s/message?sessionId=%s\n\n", tunnelID, sessionID)
	w.(http.Flusher).Flush()

	// Forward messages from tunnel to SSE client
	for {
		select {
		case msg := <-session.ClientChan:
			fmt.Fprintf(w, "event: message\n")
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// HandleMessage handles SSE message endpoint
func (s *TunnelServer) HandleMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]
	sessionID := r.URL.Query().Get("sessionId")

	s.tunnelsMux.RLock()
	tunnel, exists := s.tunnels[tunnelID]
	s.tunnelsMux.RUnlock()

	if !exists {
		http.Error(w, "Tunnel not found", http.StatusNotFound)
		return
	}

	tunnel.sessionsMux.RLock()
	session, exists := tunnel.sessions[sessionID]
	tunnel.sessionsMux.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Read request
	var mcpRequest json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&mcpRequest); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Forward to tunnel
	msg := TunnelMessage{
		Type:    "sse_request",
		ID:      sessionID,
		Payload: mcpRequest,
	}

	if err := tunnel.ClientConn.WriteJSON(msg); err != nil {
		http.Error(w, "Tunnel error", http.StatusInternalServerError)
		return
	}

	// Return 202 Accepted
	w.WriteHeader(http.StatusAccepted)
}

// handleResponse handles responses from tunnel client
func (s *TunnelServer) handleResponse(tunnel *Tunnel, msg TunnelMessage) {
	if msg.Type == "sse_response" {
		// Forward to SSE session
		tunnel.sessionsMux.RLock()
		session, exists := tunnel.sessions[msg.ID]
		tunnel.sessionsMux.RUnlock()

		if exists {
			select {
			case session.ClientChan <- msg.Payload:
			default:
				log.Printf("Session %s channel full", msg.ID)
			}
		}
	}
	// Handle regular HTTP responses via channels
}

// Ping tunnels periodically
func (s *TunnelServer) pingTunnels() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.tunnelsMux.RLock()
		tunnels := make([]*Tunnel, 0, len(s.tunnels))
		for _, t := range s.tunnels {
			tunnels = append(tunnels, t)
		}
		s.tunnelsMux.RUnlock()

		for _, tunnel := range tunnels {
			if tunnel.ClientConn != nil {
				msg := TunnelMessage{Type: "ping"}
				if err := tunnel.ClientConn.WriteJSON(msg); err != nil {
					log.Printf("Failed to ping tunnel %s: %v", tunnel.ID, err)
				}
			}
		}
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := NewTunnelServer()
	go server.pingTunnels()

	router := mux.NewRouter()
	
	// Tunnel management
	router.HandleFunc("/tunnels", server.CreateTunnel).Methods("POST")
	router.HandleFunc("/tunnels/{id}/ws", server.HandleWebSocket)
	
	// MCP endpoints
	router.HandleFunc("/tunnels/{id}/mcp", server.HandleMCP).Methods("POST")
	router.HandleFunc("/tunnels/{id}/sse", server.HandleSSE).Methods("GET")
	router.HandleFunc("/tunnels/{id}/message", server.HandleMessage).Methods("POST")
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("MCP Tunnel Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}