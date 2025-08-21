package mcp

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// StreamableTransport extends the basic Transport interface with advanced connection capabilities
type StreamableTransport interface {
	Transport
	Connect(context.Context) (Connection, error)
}

// Connection represents a logical bidirectional JSON-RPC connection with session management
type Connection interface {
	Read(context.Context) (JSONRPCMessage, error)
	Write(context.Context, JSONRPCMessage) error
	Close() error
}

// JSONRPCMessage represents a JSON-RPC message for streamable transport
type JSONRPCMessage struct {
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	JSONRPC string      `json:"jsonrpc"`
}

// StreamableHTTPOptions configures the streamable HTTP handler
type StreamableHTTPOptions struct {
	Logger         *slog.Logger
	MaxSessions    int
	SessionTimeout time.Duration
}

// StreamableHTTPHandler serves streamable MCP sessions as defined by the MCP spec
type StreamableHTTPHandler struct {
	getServer func(*http.Request) *Server
	opts      StreamableHTTPOptions

	sessionsMu sync.RWMutex
	sessions   map[string]*StreamableServerTransport
}

// NewStreamableHTTPHandler creates a new streamable HTTP handler
func NewStreamableHTTPHandler(getServer func(*http.Request) *Server, opts *StreamableHTTPOptions) *StreamableHTTPHandler {
	if opts == nil {
		opts = &StreamableHTTPOptions{}
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.MaxSessions <= 0 {
		opts.MaxSessions = 1000
	}
	if opts.SessionTimeout <= 0 {
		opts.SessionTimeout = 5 * time.Minute
	}

	return &StreamableHTTPHandler{
		getServer: getServer,
		opts:      *opts,
		sessions:  make(map[string]*StreamableServerTransport),
	}
}

// ServeHTTP implements the HTTP handler interface
func (h *StreamableHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.opts.Logger.DebugContext(r.Context(), "StreamableHTTPHandler: handling request",
		"method", r.Method, "path", r.URL.Path)

	// Check Accept header for required content types
	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "application/json") || !strings.Contains(accept, "text/event-stream") {
		http.Error(w, "Accept header must include both application/json and text/event-stream", http.StatusNotAcceptable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleSSEStream(w, r)
	case http.MethodPost:
		h.handleMessage(w, r)
	case http.MethodDelete:
		h.handleSessionDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSSEStream handles SSE streaming with optional session resumption
func (h *StreamableHTTPHandler) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		sessionID = randText()
	}

	h.sessionsMu.Lock()
	session, exists := h.sessions[sessionID]
	if !exists {
		session = newStreamableServerTransport(sessionID, h.opts.Logger)
		h.sessions[sessionID] = session
	}
	h.sessionsMu.Unlock()

	// Handle resumption via Last-Event-ID
	lastEventID := r.Header.Get("Last-Event-ID")
	resumeStreamID := streamID(0)
	resumeIndex := 0

	if lastEventID != "" {
		sid, idx, ok := parseEventID(lastEventID)
		if ok {
			resumeStreamID = sid
			resumeIndex = idx + 1
		}
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send endpoint event
	fmt.Fprintf(w, "event: endpoint\n")
	fmt.Fprintf(w, "data: /message?session=%s\n\n", sessionID)
	flusher.Flush()

	// Stream messages
	session.streamMessages(r.Context(), w, flusher, resumeStreamID, resumeIndex)
}

// handleMessage handles incoming JSON-RPC messages
func (h *StreamableHTTPHandler) handleMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	h.sessionsMu.RLock()
	session, exists := h.sessions[sessionID]
	h.sessionsMu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, "Invalid JSON-RPC message", http.StatusBadRequest)
		return
	}

	// Send message to session
	select {
	case session.incoming <- msg:
		w.WriteHeader(http.StatusNoContent)
	case <-r.Context().Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
	case <-time.After(5 * time.Second):
		http.Error(w, "Session busy", http.StatusServiceUnavailable)
	}
}

// handleSessionDelete handles session termination
func (h *StreamableHTTPHandler) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	h.sessionsMu.Lock()
	session, exists := h.sessions[sessionID]
	if exists {
		delete(h.sessions, sessionID)
		session.Close()
	}
	h.sessionsMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// streamID represents a logical stream within a session
type streamID int64

// streamableMsg represents a message with stream correlation
type streamableMsg struct {
	Message JSONRPCMessage
	EventID string
}

// StreamableServerTransport implements the streamable server transport
type StreamableServerTransport struct {
	nextStreamID atomic.Int64
	id           string
	incoming     chan JSONRPCMessage
	logger       *slog.Logger

	mu               sync.RWMutex
	isDone           bool
	done             chan struct{}
	outgoingMessages map[streamID][]*streamableMsg
	signals          map[streamID]chan struct{}
	requestStreams   map[interface{}]streamID
	streamRequests   map[streamID]map[interface{}]struct{}
}

// newStreamableServerTransport creates a new streamable server transport
func newStreamableServerTransport(sessionID string, logger *slog.Logger) *StreamableServerTransport {
	return &StreamableServerTransport{
		id:               sessionID,
		incoming:         make(chan JSONRPCMessage, 100),
		logger:           logger,
		done:             make(chan struct{}),
		outgoingMessages: make(map[streamID][]*streamableMsg),
		signals:          make(map[streamID]chan struct{}),
		requestStreams:   make(map[interface{}]streamID),
		streamRequests:   make(map[streamID]map[interface{}]struct{}),
	}
}

// Connect implements the StreamableTransport interface
func (t *StreamableServerTransport) Connect(ctx context.Context) (Connection, error) {
	return t, nil
}

// Dial implements the Transport interface for compatibility
func (t *StreamableServerTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &streamableRWCAdapter{transport: t}, nil
}

// Read implements the Connection interface
func (t *StreamableServerTransport) Read(ctx context.Context) (JSONRPCMessage, error) {
	select {
	case <-ctx.Done():
		return JSONRPCMessage{}, ctx.Err()
	case <-t.done:
		return JSONRPCMessage{}, io.EOF
	case msg := <-t.incoming:
		return msg, nil
	}
}

// Write implements the Connection interface
func (t *StreamableServerTransport) Write(ctx context.Context, msg JSONRPCMessage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isDone {
		return io.ErrClosedPipe
	}

	// Determine stream ID based on message type
	sid := t.getStreamID(msg)

	// Create streamable message
	msgIndex := len(t.outgoingMessages[sid])
	streamableMsg := &streamableMsg{
		Message: msg,
		EventID: formatEventID(sid, msgIndex),
	}

	// Store message
	t.outgoingMessages[sid] = append(t.outgoingMessages[sid], streamableMsg)

	// Signal waiting streams
	if ch, exists := t.signals[sid]; exists {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	return nil
}

// Close implements the Connection interface
func (t *StreamableServerTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isDone {
		t.isDone = true
		close(t.done)
	}

	return nil
}

// getStreamID determines the appropriate stream ID for a message
func (t *StreamableServerTransport) getStreamID(msg JSONRPCMessage) streamID {
	// For requests, create or reuse stream
	if msg.Method != "" {
		if sid, exists := t.requestStreams[msg.ID]; exists {
			return sid
		}

		// Create new stream
		sid := streamID(t.nextStreamID.Add(1))
		t.requestStreams[msg.ID] = sid
		if t.streamRequests[sid] == nil {
			t.streamRequests[sid] = make(map[interface{}]struct{})
		}
		t.streamRequests[sid][msg.ID] = struct{}{}
		return sid
	}

	// For responses, use existing stream
	if sid, exists := t.requestStreams[msg.ID]; exists {
		return sid
	}

	// Default stream
	return streamID(1)
}

// streamMessages streams messages to SSE client
func (t *StreamableServerTransport) streamMessages(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, resumeStreamID streamID, resumeIndex int) {
	t.mu.RLock()

	// Send buffered messages for resumption
	if resumeStreamID > 0 {
		if messages, exists := t.outgoingMessages[resumeStreamID]; exists {
			for i := resumeIndex; i < len(messages); i++ {
				msg := messages[i]
				t.writeSSEMessage(w, msg)
				flusher.Flush()
			}
		}
	}

	// Create signal channel for this stream
	signalCh := make(chan struct{}, 1)
	t.signals[resumeStreamID] = signalCh
	t.mu.RUnlock()

	defer func() {
		t.mu.Lock()
		delete(t.signals, resumeStreamID)
		t.mu.Unlock()
	}()

	// Stream new messages
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.done:
			return
		case <-ticker.C:
			// Send keep-alive
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		case <-signalCh:
			t.mu.RLock()
			if messages, exists := t.outgoingMessages[resumeStreamID]; exists {
				for i := resumeIndex; i < len(messages); i++ {
					msg := messages[i]
					t.writeSSEMessage(w, msg)
					flusher.Flush()
					resumeIndex = i + 1
				}
			}
			t.mu.RUnlock()
		}
	}
}

// writeSSEMessage writes a message as an SSE event
func (t *StreamableServerTransport) writeSSEMessage(w http.ResponseWriter, msg *streamableMsg) {
	data, err := json.Marshal(msg.Message)
	if err != nil {
		t.logger.Error("Failed to marshal message", "error", err)
		return
	}

	fmt.Fprintf(w, "id: %s\n", msg.EventID)
	fmt.Fprintf(w, "data: %s\n\n", string(data))
}

// Event represents an SSE event
type event struct {
	name string
	id   string
	data []byte
}

func (e event) empty() bool {
	return e.name == "" && e.id == "" && len(e.data) == 0
}

// scanEvents scans SSE events using Go 1.23+ iterators
func scanEvents(r io.Reader) iter.Seq2[event, error] {
	scanner := bufio.NewScanner(r)
	const maxTokenSize = 1 * 1024 * 1024 // 1 MiB max line size
	scanner.Buffer(nil, maxTokenSize)

	return func(yield func(event, error) bool) {
		var evt event
		var dataBuf *bytes.Buffer

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				// End of event (\n\n delimiter)
				if !evt.empty() {
					if dataBuf != nil {
						evt.data = dataBuf.Bytes()
					}
					if !yield(evt, nil) {
						return
					}
				}
				evt = event{}
				dataBuf = nil
				continue
			}

			before, after, found := bytes.Cut(line, []byte{':'})
			if !found {
				yield(event{}, fmt.Errorf("malformed line: %q", string(line)))
				return
			}

			switch {
			case bytes.Equal(before, []byte("event")):
				evt.name = strings.TrimSpace(string(after))
			case bytes.Equal(before, []byte("id")):
				evt.id = strings.TrimSpace(string(after))
			case bytes.Equal(before, []byte("data")):
				data := bytes.TrimSpace(after)
				if dataBuf != nil {
					dataBuf.WriteByte('\n')
					dataBuf.Write(data)
				} else {
					dataBuf = new(bytes.Buffer)
					dataBuf.Write(data)
				}
			}
		}

		if err := scanner.Err(); err != nil {
			yield(event{}, err)
		}
	}
}

// formatEventID formats an event ID from stream ID and message index
func formatEventID(sid streamID, idx int) string {
	return fmt.Sprintf("%d_%d", sid, idx)
}

// parseEventID parses an event ID into stream ID and message index
func parseEventID(eventID string) (sid streamID, idx int, ok bool) {
	parts := strings.Split(eventID, "_")
	if len(parts) != 2 {
		return 0, 0, false
	}

	stream, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || stream < 0 {
		return 0, 0, false
	}

	idx, err = strconv.Atoi(parts[1])
	if err != nil || idx < 0 {
		return 0, 0, false
	}

	return streamID(stream), idx, true
}

// randText generates a random session ID
func randText() string {
	// ⌈log₃₂ 2¹²⁸⌉ = 26 chars
	src := make([]byte, 26)
	rand.Read(src)
	for i := range src {
		src[i] = base32alphabet[src[i]%32]
	}
	return string(src)
}

const base32alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

// streamableRWCAdapter adapts the streamable transport to io.ReadWriteCloser
type streamableRWCAdapter struct {
	transport *StreamableServerTransport
	readBuf   bytes.Buffer
	mu        sync.Mutex
}

func (a *streamableRWCAdapter) Read(p []byte) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.readBuf.Len() > 0 {
		return a.readBuf.Read(p)
	}

	msg, err := a.transport.Read(context.Background())
	if err != nil {
		return 0, err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	data = append(data, '\n')
	a.readBuf.Write(data)

	return a.readBuf.Read(p)
}

func (a *streamableRWCAdapter) Write(p []byte) (n int, err error) {
	var msg JSONRPCMessage
	if err := json.Unmarshal(p, &msg); err != nil {
		return 0, err
	}

	if err := a.transport.Write(context.Background(), msg); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (a *streamableRWCAdapter) Close() error {
	return a.transport.Close()
}
