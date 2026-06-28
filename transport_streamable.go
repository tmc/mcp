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
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const streamableSessionHeader = "Mcp-Session-Id"

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

	// MaxRequestBytes caps the size of an incoming POST body. Bodies larger
	// than this are rejected rather than read into memory. Zero selects the
	// default (4 MiB).
	MaxRequestBytes int64

	// AllowOrigin, if non-empty, is echoed as the Access-Control-Allow-Origin
	// header on SSE responses. It is empty by default, so no CORS header is
	// emitted and the stream is not exposed cross-origin; set it explicitly
	// (e.g. "*" or a specific origin) to opt in.
	AllowOrigin string

	// DisableLocalhostProtection disables DNS rebinding protection for local
	// streamable HTTP servers.
	DisableLocalhostProtection bool
}

// defaultMaxRequestBytes bounds an unconfigured POST body at 4 MiB.
const defaultMaxRequestBytes = 4 << 20

// StreamableHTTPHandler serves streamable MCP sessions as defined by the MCP spec
type StreamableHTTPHandler struct {
	getServer func(*http.Request) *Server
	opts      StreamableHTTPOptions

	sessionsMu sync.RWMutex
	sessions   map[string]*StreamableServerTransport

	reaperOnce sync.Once
	done       chan struct{}
	closeOnce  sync.Once
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
	if opts.MaxRequestBytes <= 0 {
		opts.MaxRequestBytes = defaultMaxRequestBytes
	}

	return &StreamableHTTPHandler{
		getServer: getServer,
		opts:      *opts,
		sessions:  make(map[string]*StreamableServerTransport),
		done:      make(chan struct{}),
	}
}

// touch records the current time as this session's last activity.
func (t *StreamableServerTransport) touch() {
	t.lastActive.Store(time.Now().UnixNano())
}

// startReaper lazily starts the idle-session reaper. It runs at most once per
// handler and stops when the handler is closed, so it does not leak: a session
// whose lastActive is older than SessionTimeout is cancelled, which makes its
// Serve goroutine return and self-delete from the map.
func (h *StreamableHTTPHandler) startReaper() {
	h.reaperOnce.Do(func() {
		safeGo(h.opts.Logger, "session reaper", func() {
			ticker := time.NewTicker(h.opts.SessionTimeout)
			defer ticker.Stop()
			for {
				select {
				case <-h.done:
					return
				case <-ticker.C:
					h.reapIdleSessions()
				}
			}
		})
	})
}

func (h *StreamableHTTPHandler) reapIdleSessions() {
	cutoff := time.Now().Add(-h.opts.SessionTimeout).UnixNano()
	var idle []*StreamableServerTransport
	h.sessionsMu.RLock()
	for _, s := range h.sessions {
		if s.lastActive.Load() < cutoff {
			idle = append(idle, s)
		}
	}
	h.sessionsMu.RUnlock()
	// Cancel outside the lock; each cancelled session's Serve goroutine deletes
	// itself from the map under the write lock.
	for _, s := range idle {
		if s.cancel != nil {
			s.cancel()
		}
	}
}

// Close shuts the handler down: it stops the reaper and cancels every live
// session so their Serve goroutines return. It is safe to call more than once.
func (h *StreamableHTTPHandler) Close() error {
	h.closeOnce.Do(func() {
		close(h.done)
		h.sessionsMu.RLock()
		sessions := make([]*StreamableServerTransport, 0, len(h.sessions))
		for _, s := range h.sessions {
			sessions = append(sessions, s)
		}
		h.sessionsMu.RUnlock()
		for _, s := range sessions {
			if s.cancel != nil {
				s.cancel()
			}
		}
	})
	return nil
}

// ServeHTTP implements the HTTP handler interface
func (h *StreamableHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.opts.DisableLocalhostProtection && streamableIsLocalhostRequest(r) && !streamableIsLoopback(r.Host) {
		http.Error(w, fmt.Sprintf("Forbidden: invalid Host header %q", r.Host), http.StatusForbidden)
		return
	}

	h.opts.Logger.DebugContext(r.Context(), "StreamableHTTPHandler: handling request",
		"method", r.Method, "path", r.URL.Path)

	jsonOK, streamOK := streamableAccepts(r.Header.Values("Accept"))

	switch r.Method {
	case http.MethodGet:
		if !streamOK {
			http.Error(w, "Accept header must include text/event-stream", http.StatusNotAcceptable)
			return
		}
		h.handleSSEStream(w, r)
	case http.MethodPost:
		if len(r.Header.Values("Accept")) > 0 && (!jsonOK || !streamOK) {
			http.Error(w, "Accept header must include application/json and text/event-stream", http.StatusNotAcceptable)
			return
		}
		h.handleMessage(w, r)
	case http.MethodDelete:
		h.handleSessionDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSSEStream handles SSE streaming with optional session resumption
func (h *StreamableHTTPHandler) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	sessionID := streamableSessionID(r)
	if sessionID == "" {
		sessionID = randText()
	}

	session, err := h.getOrCreateSession(r, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	session.touch()

	// Handle resumption via Last-Event-ID
	lastEventID := r.Header.Get("Last-Event-ID")
	resumeStreamID := streamID(0)
	resumeIndex := 0

	if lastEventID != "" {
		sid, idx, ok := parseEventID(lastEventID)
		// Guard against a forged Last-Event-ID: ignore negative indices/stream
		// ids. The replay loop is already bounded by the buffered message
		// count, so a too-large index simply resumes at the end.
		if ok && idx >= 0 && sid >= 0 {
			resumeStreamID = sid
			resumeIndex = idx + 1
		}
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// CORS is opt-in: only expose the stream cross-origin when an origin is
	// explicitly configured. The previous unconditional "*" let any origin read
	// session-scoped data.
	if h.opts.AllowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", h.opts.AllowOrigin)
	}
	w.Header().Set(streamableSessionHeader, session.id)

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
	// Cap the body so a peer cannot exhaust memory with an arbitrarily large or
	// infinite request. MaxBytesReader makes the read fail past the limit.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.opts.MaxRequestBytes))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusRequestEntityTooLarge)
		return
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, "Invalid JSON-RPC message", http.StatusBadRequest)
		return
	}

	sessionID := streamableSessionID(r)
	var session *StreamableServerTransport
	if sessionID == "" {
		sessionID = randText()
		session, err = h.getOrCreateSession(r, sessionID)
	} else {
		session, err = h.getSession(sessionID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	session.touch()
	sid := streamID(session.nextStreamID.Add(1))
	if err := session.receive(r.Context(), msg, sid); err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	w.Header().Set(streamableSessionHeader, session.id)
	if msg.ID == nil || msg.Method == "" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	next := 0
	for {
		out, idx, err := session.waitStreamMessage(r.Context(), sid, next)
		if err != nil {
			return
		}
		next = idx
		session.writeSSEMessage(w, out)
		flusher.Flush()
		if out.Message.ID == msg.ID && out.Message.Method == "" {
			return
		}
	}
}

// handleSessionDelete handles session termination
func (h *StreamableHTTPHandler) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := streamableSessionID(r)
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
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

func streamableSessionID(r *http.Request) string {
	if sessionID := r.Header.Get(streamableSessionHeader); sessionID != "" {
		return sessionID
	}
	return r.URL.Query().Get("session")
}

func streamableAccepts(values []string) (jsonOK, streamOK bool) {
	for _, value := range values {
		for _, raw := range strings.Split(value, ",") {
			token := strings.TrimSpace(raw)
			base, _, _ := strings.Cut(token, ";")
			switch strings.ToLower(strings.TrimSpace(base)) {
			case "application/json", "application/*":
				jsonOK = true
			case "text/event-stream", "text/*":
				streamOK = true
			case "*/*":
				jsonOK = true
				streamOK = true
			}
		}
	}
	return jsonOK, streamOK
}

func streamableIsLocalhostRequest(r *http.Request) bool {
	localAddr, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	return ok && localAddr != nil && streamableIsLoopback(localAddr.String())
}

func streamableIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = strings.Trim(addr, "[]")
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

func (h *StreamableHTTPHandler) getSession(sessionID string) (*StreamableServerTransport, error) {
	h.sessionsMu.RLock()
	session := h.sessions[sessionID]
	h.sessionsMu.RUnlock()
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (h *StreamableHTTPHandler) getOrCreateSession(r *http.Request, sessionID string) (*StreamableServerTransport, error) {
	h.sessionsMu.RLock()
	session := h.sessions[sessionID]
	h.sessionsMu.RUnlock()
	if session != nil {
		return session, nil
	}

	server := h.getServer(r)
	if server == nil {
		return nil, fmt.Errorf("no server available")
	}
	ctx, cancel := context.WithCancel(context.Background())
	session = newStreamableServerTransport(sessionID, h.opts.Logger)
	session.cancel = cancel

	h.sessionsMu.Lock()
	if existing := h.sessions[sessionID]; existing != nil {
		h.sessionsMu.Unlock()
		cancel()
		return existing, nil
	}
	// Enforce the session cap before inserting. Without this, a peer sending
	// unbounded unique session IDs (or omitting the header so a fresh ID is
	// minted each time) grows this map without bound and leaks a Serve
	// goroutine per session — a memory/goroutine-exhaustion DoS.
	if len(h.sessions) >= h.opts.MaxSessions {
		h.sessionsMu.Unlock()
		cancel()
		return nil, fmt.Errorf("session limit reached (%d)", h.opts.MaxSessions)
	}
	session.touch()
	h.sessions[sessionID] = session
	h.sessionsMu.Unlock()

	h.startReaper()

	safeGo(h.opts.Logger, "session serve", func() {
		_ = server.Serve(ctx, session)
		h.sessionsMu.Lock()
		if h.sessions[sessionID] == session {
			delete(h.sessions, sessionID)
		}
		h.sessionsMu.Unlock()
	})
	return session, nil
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
	cancel       context.CancelFunc
	// lastActive is the unix-nano time of the most recent activity on this
	// session, read by the handler's reaper to evict idle sessions.
	lastActive atomic.Int64

	mu               sync.RWMutex
	isDone           bool
	done             chan struct{}
	outgoingMessages map[streamID][]*streamableMsg
	signals          map[streamID]chan struct{}
	// clientRequestStreams maps an inbound client request id to the stream it
	// arrived on. Its response and any server requests it triggers route there.
	clientRequestStreams map[interface{}]streamID
	// serverRequestStreams maps an outbound server-initiated request id to the
	// stream it was emitted on. Kept separate from clientRequestStreams so client
	// and server request id spaces (both small integers) cannot collide.
	serverRequestStreams map[interface{}]streamID
	// lastRequestStream is the stream of the client request currently being
	// handled, used to route server-initiated requests and notifications.
	lastRequestStream streamID
}

// newStreamableServerTransport creates a new streamable server transport
func newStreamableServerTransport(sessionID string, logger *slog.Logger) *StreamableServerTransport {
	return &StreamableServerTransport{
		id:                   sessionID,
		incoming:             make(chan JSONRPCMessage, 100),
		logger:               logger,
		done:                 make(chan struct{}),
		outgoingMessages:     make(map[streamID][]*streamableMsg),
		signals:              make(map[streamID]chan struct{}),
		clientRequestStreams: make(map[interface{}]streamID),
		serverRequestStreams: make(map[interface{}]streamID),
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
		return JSONRPCMessage{}, transportClosedError("streamable read")
	case msg := <-t.incoming:
		return msg, nil
	}
}

// Write implements the Connection interface
func (t *StreamableServerTransport) Write(ctx context.Context, msg JSONRPCMessage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isDone {
		return transportClosedError("streamable write")
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

	// A response to a client request completes that request and frees its stream.
	if msg.Method == "" && msg.ID != nil {
		t.releaseClientRequest(msg.ID)
	}

	return nil
}

// releaseClientRequest drops the correlation for a completed client request and
// any server-initiated requests still pinned to its stream (for example server
// requests abandoned on timeout), and clears lastRequestStream when it pointed
// at that now-finished stream. It must be called with t.mu held.
func (t *StreamableServerTransport) releaseClientRequest(id interface{}) {
	sid, ok := t.clientRequestStreams[id]
	if !ok {
		return
	}
	delete(t.clientRequestStreams, id)
	for srvID, srvSid := range t.serverRequestStreams {
		if srvSid == sid {
			delete(t.serverRequestStreams, srvID)
		}
	}
	if t.lastRequestStream == sid {
		t.lastRequestStream = 0
	}
}

// Close implements the Connection interface
func (t *StreamableServerTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isDone {
		t.isDone = true
		close(t.done)
		if t.cancel != nil {
			t.cancel()
		}
	}

	return nil
}

func (t *StreamableServerTransport) receive(ctx context.Context, msg JSONRPCMessage, sid streamID) error {
	switch {
	case msg.ID != nil && msg.Method != "":
		// An inbound client request owns the stream it arrived on. Its response
		// and any server requests it triggers route back to this stream.
		t.mu.Lock()
		t.clientRequestStreams[msg.ID] = sid
		t.lastRequestStream = sid
		t.mu.Unlock()
	case msg.ID != nil && msg.Method == "":
		// An inbound response answers a server-initiated request, completing it.
		t.mu.Lock()
		delete(t.serverRequestStreams, msg.ID)
		t.mu.Unlock()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.done:
		return transportClosedError("streamable receive")
	case t.incoming <- msg:
		return nil
	}
}

// getStreamID determines the appropriate stream for an outbound message. It must
// be called with t.mu held.
func (t *StreamableServerTransport) getStreamID(msg JSONRPCMessage) streamID {
	// Server-initiated notification: follow the active client request stream when
	// one is in flight, otherwise the standalone GET stream (0).
	if msg.Method != "" && msg.ID == nil {
		return t.lastRequestStream
	}

	// Server-initiated request: route to the active client request stream (the
	// stream of the request currently being handled), or the standalone GET
	// stream (0) when emitted out of band. Tracked separately from client
	// requests so id spaces cannot collide.
	if msg.Method != "" {
		if sid, exists := t.serverRequestStreams[msg.ID]; exists {
			return sid
		}
		sid := t.lastRequestStream
		t.serverRequestStreams[msg.ID] = sid
		return sid
	}

	// Response to a client request: use the stream the request arrived on.
	if sid, exists := t.clientRequestStreams[msg.ID]; exists {
		return sid
	}

	// Default stream.
	return streamID(1)
}

// streamMessages streams messages to SSE client
func (t *StreamableServerTransport) streamMessages(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, resumeStreamID streamID, resumeIndex int) {
	t.mu.RLock()

	// Flush messages already buffered on this stream. This both replays from a
	// resumption point (Last-Event-ID) and delivers out-of-band server
	// requests/notifications that were routed to the standalone GET stream (0)
	// before this GET connected. resumeIndex is advanced so the live loop below
	// does not re-deliver them.
	if messages, exists := t.outgoingMessages[resumeStreamID]; exists {
		for i := resumeIndex; i < len(messages); i++ {
			t.writeSSEMessage(w, messages[i])
			flusher.Flush()
			resumeIndex = i + 1
		}
	}
	t.mu.RUnlock()

	// Register the signal channel under a write lock: t.signals is a shared
	// map, so the assignment must hold Lock (matching waitStreamMessage), not
	// RLock — two concurrent GET streams writing it under RLock would be a
	// fatal concurrent map write. Re-read outgoingMessages under the same lock
	// and flush anything published between the RUnlock above and here, so no
	// message is missed in the gap.
	signalCh := make(chan struct{}, 1)
	t.mu.Lock()
	if messages, exists := t.outgoingMessages[resumeStreamID]; exists {
		for i := resumeIndex; i < len(messages); i++ {
			t.writeSSEMessage(w, messages[i])
			flusher.Flush()
			resumeIndex = i + 1
		}
	}
	t.signals[resumeStreamID] = signalCh
	t.mu.Unlock()

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

func (t *StreamableServerTransport) waitStreamMessage(ctx context.Context, sid streamID, idx int) (*streamableMsg, int, error) {
	signalCh := make(chan struct{}, 1)
	t.mu.Lock()
	t.signals[sid] = signalCh
	t.mu.Unlock()
	defer func() {
		t.mu.Lock()
		if t.signals[sid] == signalCh {
			delete(t.signals, sid)
		}
		t.mu.Unlock()
	}()

	for {
		t.mu.RLock()
		if messages := t.outgoingMessages[sid]; idx < len(messages) {
			msg := messages[idx]
			t.mu.RUnlock()
			return msg, idx + 1, nil
		}
		done := t.isDone
		t.mu.RUnlock()
		if done {
			return nil, idx, transportClosedError("streamable wait")
		}

		select {
		case <-ctx.Done():
			return nil, idx, ctx.Err()
		case <-t.done:
			return nil, idx, transportClosedError("streamable wait")
		case <-signalCh:
		}
	}
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
