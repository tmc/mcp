// Package transport provides communication interfaces for mcpd.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ConnectionType represents different ways clients can connect to mcpd
type ConnectionType string

const (
	ConnectionTypeUnix       ConnectionType = "unix"
	ConnectionTypeTCP        ConnectionType = "tcp"
	ConnectionTypeSSE        ConnectionType = "sse"
	ConnectionTypeWebSocket  ConnectionType = "websocket"
	ConnectionTypeHTTPStream ConnectionType = "httpstream"
)

// StreamingTransport extends the Listener with HTTP streaming capabilities
type StreamingTransport struct {
	*Listener
	httpServer     *http.Server
	httpAddr       string
	sseEnabled     bool
	wsEnabled      bool
	streamEnabled  bool
	sseClients     map[string]*SSEClient
	streamClients  map[string]*StreamClient
	wsClients      map[string]*WebSocketClient
	clientsMu      sync.RWMutex
	streamTimeout  time.Duration
	clientHandlers map[ConnectionType]func(context.Context, string) error
}

// SSEClient represents a connected SSE client
type SSEClient struct {
	ID           string
	ResponseWriter http.ResponseWriter
	Request      *http.Request
	LastEventID  string
	MessageChan  chan []byte
	Done         chan struct{}
	CloseNotify <-chan bool
}

// StreamClient represents a connected HTTP streaming client
type StreamClient struct {
	ID           string
	ResponseWriter http.ResponseWriter
	Request      *http.Request
	MessageChan  chan []byte
	Done         chan struct{}
	CloseNotify <-chan bool
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID           string
	Conn         interface{} // Will be replaced with proper WebSocket type
	MessageChan  chan []byte
	Done         chan struct{}
}

// NewStreamingTransport creates a new transport with streaming support
func NewStreamingTransport(listener *Listener, httpAddr string) *StreamingTransport {
	return &StreamingTransport{
		Listener:       listener,
		httpAddr:       httpAddr,
		sseClients:     make(map[string]*SSEClient),
		streamClients:  make(map[string]*StreamClient),
		wsClients:      make(map[string]*WebSocketClient),
		streamTimeout:  1 * time.Hour, // Default timeout
		clientHandlers: make(map[ConnectionType]func(context.Context, string) error),
	}
}

// EnableSSE enables the Server-Sent Events endpoint
func (t *StreamingTransport) EnableSSE() {
	t.sseEnabled = true
}

// EnableWebSockets enables the WebSocket endpoint
func (t *StreamingTransport) EnableWebSockets() {
	t.wsEnabled = true
}

// EnableHTTPStreaming enables the HTTP streaming endpoint
func (t *StreamingTransport) EnableHTTPStreaming() {
	t.streamEnabled = true
}

// SetStreamTimeout sets the timeout for streaming connections
func (t *StreamingTransport) SetStreamTimeout(timeout time.Duration) {
	t.streamTimeout = timeout
}

// SetClientHandler sets a handler function for a specific connection type
func (t *StreamingTransport) SetClientHandler(connType ConnectionType, handler func(context.Context, string) error) {
	t.clientHandlers[connType] = handler
}

// Start starts the streaming transport
func (t *StreamingTransport) Start(ctx context.Context) error {
	// Start the regular listener
	err := t.Listener.Listen()
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	// Start HTTP server if HTTP address is provided
	if t.httpAddr != "" {
		mux := http.NewServeMux()
		
		// Register endpoints based on enabled features
		if t.sseEnabled {
			slog.Info("Enabling SSE endpoint", "path", "/sse")
			mux.HandleFunc("/sse", t.handleSSE)
		}
		
		if t.streamEnabled {
			slog.Info("Enabling HTTP streaming endpoint", "path", "/stream")
			mux.HandleFunc("/stream", t.handleHTTPStream)
		}
		
		if t.wsEnabled {
			slog.Info("Enabling WebSocket endpoint", "path", "/ws")
			mux.HandleFunc("/ws", t.handleWebSocket)
		}
		
		// Add health check endpoint
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		
		// Create HTTP server
		t.httpServer = &http.Server{
			Addr:    t.httpAddr,
			Handler: mux,
		}
		
		// Start HTTP server in a goroutine
		go func() {
			slog.Info("Starting HTTP server for streaming", "addr", t.httpAddr)
			if err := t.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTP server error", "error", err)
			}
		}()
		
		// Wait for context cancellation to shut down HTTP server
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			slog.Info("Shutting down HTTP server")
			if err := t.httpServer.Shutdown(shutdownCtx); err != nil {
				slog.Error("Failed to shut down HTTP server gracefully", "error", err)
			}
		}()
	}
	
	return nil
}

// Close closes the streaming transport and all client connections
func (t *StreamingTransport) Close() error {
	// Close the regular listener
	err := t.Listener.Close()
	
	// Close HTTP server if it exists
	if t.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := t.httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shut down HTTP server gracefully", "error", err)
		}
	}
	
	// Close all client connections
	t.clientsMu.Lock()
	defer t.clientsMu.Unlock()
	
	// Close SSE clients
	for id, client := range t.sseClients {
		close(client.MessageChan)
		close(client.Done)
		delete(t.sseClients, id)
	}
	
	// Close HTTP streaming clients
	for id, client := range t.streamClients {
		close(client.MessageChan)
		close(client.Done)
		delete(t.streamClients, id)
	}
	
	// Close WebSocket clients
	for id, client := range t.wsClients {
		close(client.MessageChan)
		close(client.Done)
		delete(t.wsClients, id)
	}
	
	return err
}

// handleSSE handles Server-Sent Events connections
func (t *StreamingTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if SSE is enabled
	if !t.sseEnabled {
		http.Error(w, "SSE endpoint is disabled", http.StatusNotFound)
		return
	}
	
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Get or create a client ID
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("sse-%d", time.Now().UnixNano())
	}
	
	// Get last event ID if provided
	lastEventID := r.Header.Get("Last-Event-ID")
	
	// Create a communication channel
	messageChan := make(chan []byte, 10)
	doneChan := make(chan struct{})
	
	// Create a new SSE client
	client := &SSEClient{
		ID:            clientID,
		ResponseWriter: w,
		Request:       r,
		LastEventID:   lastEventID,
		MessageChan:   messageChan,
		Done:          doneChan,
		CloseNotify:   w.(http.CloseNotifier).CloseNotify(),
	}
	
	// Register the client
	t.clientsMu.Lock()
	t.sseClients[clientID] = client
	t.clientsMu.Unlock()
	
	// Log the connection
	slog.Info("SSE client connected", "client_id", clientID)
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), t.streamTimeout)
	defer cancel()
	
	// Call the client handler if registered
	if handler, ok := t.clientHandlers[ConnectionTypeSSE]; ok {
		go func() {
			if err := handler(ctx, clientID); err != nil {
				slog.Error("Error handling SSE client", "client_id", clientID, "error", err)
			}
		}()
	}
	
	// Send an initial message
	fmt.Fprintf(w, "event: connected\ndata: {\"client_id\":\"%s\"}\n\n", clientID)
	w.(http.Flusher).Flush()
	
	// Main event loop
	go func() {
		defer func() {
			// Clean up when done
			t.clientsMu.Lock()
			delete(t.sseClients, clientID)
			t.clientsMu.Unlock()
			
			slog.Info("SSE client disconnected", "client_id", clientID)
		}()
		
		for {
			select {
			case <-client.CloseNotify:
				// Client closed the connection
				return
				
			case <-ctx.Done():
				// Context cancelled or timed out
				return
				
			case <-doneChan:
				// Done channel closed
				return
				
			case msg, ok := <-messageChan:
				if !ok {
					// Channel closed
					return
				}
				
				// Send the message as an SSE event
				fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
				w.(http.Flusher).Flush()
			}
		}
	}()
	
	// Block until connection is closed
	<-ctx.Done()
}

// handleHTTPStream handles HTTP streaming connections
func (t *StreamingTransport) handleHTTPStream(w http.ResponseWriter, r *http.Request) {
	// Check if HTTP streaming is enabled
	if !t.streamEnabled {
		http.Error(w, "HTTP streaming endpoint is disabled", http.StatusNotFound)
		return
	}
	
	// Set headers for streaming
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	
	// Get or create a client ID
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("stream-%d", time.Now().UnixNano())
	}
	
	// Create a communication channel
	messageChan := make(chan []byte, 10)
	doneChan := make(chan struct{})
	
	// Create a new Stream client
	client := &StreamClient{
		ID:            clientID,
		ResponseWriter: w,
		Request:       r,
		MessageChan:   messageChan,
		Done:          doneChan,
		CloseNotify:   w.(http.CloseNotifier).CloseNotify(),
	}
	
	// Register the client
	t.clientsMu.Lock()
	t.streamClients[clientID] = client
	t.clientsMu.Unlock()
	
	// Log the connection
	slog.Info("HTTP streaming client connected", "client_id", clientID)
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), t.streamTimeout)
	defer cancel()
	
	// Call the client handler if registered
	if handler, ok := t.clientHandlers[ConnectionTypeHTTPStream]; ok {
		go func() {
			if err := handler(ctx, clientID); err != nil {
				slog.Error("Error handling HTTP streaming client", "client_id", clientID, "error", err)
			}
		}()
	}
	
	// Send an initial message
	initialMsg := map[string]string{"client_id": clientID, "status": "connected"}
	initialJSON, _ := json.Marshal(initialMsg)
	
	w.Write(initialJSON)
	w.Write([]byte("\n"))
	w.(http.Flusher).Flush()
	
	// Main event loop
	go func() {
		defer func() {
			// Clean up when done
			t.clientsMu.Lock()
			delete(t.streamClients, clientID)
			t.clientsMu.Unlock()
			
			slog.Info("HTTP streaming client disconnected", "client_id", clientID)
		}()
		
		for {
			select {
			case <-client.CloseNotify:
				// Client closed the connection
				return
				
			case <-ctx.Done():
				// Context cancelled or timed out
				return
				
			case <-doneChan:
				// Done channel closed
				return
				
			case msg, ok := <-messageChan:
				if !ok {
					// Channel closed
					return
				}
				
				// Send the message as a JSON object
				w.Write(msg)
				w.Write([]byte("\n"))
				w.(http.Flusher).Flush()
			}
		}
	}()
	
	// Block until connection is closed
	<-ctx.Done()
}

// handleWebSocket handles WebSocket connections
func (t *StreamingTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Placeholder for WebSocket implementation
	// Will be implemented in phase 3
	http.Error(w, "WebSocket support coming soon", http.StatusNotImplemented)
}

// SendMessageToClient sends a message to a specific client
func (t *StreamingTransport) SendMessageToClient(clientID string, message []byte) error {
	t.clientsMu.RLock()
	defer t.clientsMu.RUnlock()
	
	// Try to find the client in different maps
	if client, ok := t.sseClients[clientID]; ok {
		select {
		case client.MessageChan <- message:
			return nil
		default:
			return fmt.Errorf("message channel full for SSE client %s", clientID)
		}
	}
	
	if client, ok := t.streamClients[clientID]; ok {
		select {
		case client.MessageChan <- message:
			return nil
		default:
			return fmt.Errorf("message channel full for streaming client %s", clientID)
		}
	}
	
	if client, ok := t.wsClients[clientID]; ok {
		select {
		case client.MessageChan <- message:
			return nil
		default:
			return fmt.Errorf("message channel full for WebSocket client %s", clientID)
		}
	}
	
	return fmt.Errorf("client not found: %s", clientID)
}

// BroadcastMessage sends a message to all connected clients
func (t *StreamingTransport) BroadcastMessage(message []byte) {
	t.clientsMu.RLock()
	defer t.clientsMu.RUnlock()
	
	// Send to all SSE clients
	for _, client := range t.sseClients {
		select {
		case client.MessageChan <- message:
			// Message sent
		default:
			// Channel full, skip this client
		}
	}
	
	// Send to all HTTP streaming clients
	for _, client := range t.streamClients {
		select {
		case client.MessageChan <- message:
			// Message sent
		default:
			// Channel full, skip this client
		}
	}
	
	// Send to all WebSocket clients
	for _, client := range t.wsClients {
		select {
		case client.MessageChan <- message:
			// Message sent
		default:
			// Channel full, skip this client
		}
	}
}

// GetClientCount returns the number of connected clients
func (t *StreamingTransport) GetClientCount() map[string]int {
	t.clientsMu.RLock()
	defer t.clientsMu.RUnlock()
	
	return map[string]int{
		"sse":     len(t.sseClients),
		"stream":  len(t.streamClients),
		"ws":      len(t.wsClients),
	}
}