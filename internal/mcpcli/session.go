package mcpcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/tmc/mcp"
)

// Config describes how to connect to an MCP server and where to keep local state.
type Config struct {
	Cmd             string
	HTTPURL         string
	SSEURL          string
	Timeout         time.Duration
	ProtocolVersion string
	ServerStderr    bool
	StateDir        string
	ClientInfo      mcp.Implementation
}

// DefaultConfig returns the default shared CLI configuration.
func DefaultConfig() Config {
	return Config{
		Timeout:         30 * time.Second,
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		StateDir:        defaultStateDir(),
	}
}

// Event represents an asynchronous MCP notification observed by the session.
type Event struct {
	Time   time.Time       `json:"time"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Session owns a client connection, local root state, and notification fanout.
type Session struct {
	cfg      Config
	client   *mcp.Client
	init     *mcp.InitializeResult
	roots    *StateStore
	eventsMu sync.Mutex
	nextID   int
	events   map[int]chan Event
}

// Connect initializes a new session using cfg.
func Connect(ctx context.Context, cfg Config) (*Session, error) {
	cfg = withDefaults(cfg)
	if cfg.transportCount() == 0 {
		return nil, errors.New("no server transport configured")
	}
	if cfg.transportCount() > 1 {
		return nil, errors.New("choose exactly one of stdio, http, or sse transport")
	}

	store, err := OpenStateStore(cfg.StateDir)
	if err != nil {
		return nil, err
	}

	transport, err := newTransport(cfg)
	if err != nil {
		return nil, err
	}
	client, err := mcp.NewClient(transport)
	if err != nil {
		return nil, err
	}

	s := &Session{
		cfg:    cfg,
		client: client,
		roots:  store,
		events: make(map[int]chan Event),
	}
	client.OnNotification(func(notification mcp.JSONRPCNotification) {
		s.publish(Event{
			Time:   time.Now(),
			Method: notification.Method,
			Params: notification.Params,
		})
	})
	client.OnRequest(string(mcp.MethodRootsList), func(ctx context.Context, _ json.RawMessage) (any, error) {
		roots, err := s.roots.List()
		if err != nil {
			return nil, err
		}
		return mcp.ListRootsResult{Roots: roots}, nil
	})

	initCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	initResult, err := client.Initialize(initCtx, mcp.InitializeRequest{
		ProtocolVersion: cfg.ProtocolVersion,
		ClientInfo:      cfg.ClientInfo,
		Capabilities:    sessionCapabilities(),
	})
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("initialize server: %w", err)
	}
	s.init = initResult
	_ = client.Notify(initCtx, string(mcp.MethodNotificationInitialized), map[string]any{})
	return s, nil
}

func withDefaults(cfg Config) Config {
	def := DefaultConfig()
	if cfg.Timeout == 0 {
		cfg.Timeout = def.Timeout
	}
	if cfg.ProtocolVersion == "" {
		cfg.ProtocolVersion = def.ProtocolVersion
	}
	if cfg.StateDir == "" {
		cfg.StateDir = def.StateDir
	}
	if cfg.ClientInfo.Name == "" {
		cfg.ClientInfo.Name = "mcp"
	}
	if cfg.ClientInfo.Version == "" {
		cfg.ClientInfo.Version = "0.1.0"
	}
	return cfg
}

func defaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mcp"
	}
	return home + "/.mcp"
}

func sessionCapabilities() mcp.ClientCapabilities {
	return mcp.ClientCapabilities{
		Roots: &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{},
	}
}

func (cfg Config) transportCount() int {
	n := 0
	if cfg.Cmd != "" {
		n++
	}
	if cfg.HTTPURL != "" {
		n++
	}
	if cfg.SSEURL != "" {
		n++
	}
	return n
}

func newTransport(cfg Config) (mcp.Transport, error) {
	switch {
	case cfg.Cmd != "":
		return CommandTransport(cfg.Cmd, serverStderr(cfg.ServerStderr)), nil
	case cfg.SSEURL != "":
		return mcp.NewSSEClientTransport(cfg.SSEURL, nil)
	case cfg.HTTPURL != "":
		return mcp.NewStreamableClientTransport(cfg.HTTPURL, nil), nil
	default:
		return nil, errors.New("no server transport configured")
	}
}

func serverStderr(enabled bool) io.Writer {
	if enabled {
		return os.Stderr
	}
	return io.Discard
}

// Close closes the underlying client.
func (s *Session) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

// Client returns the underlying client.
func (s *Session) Client() *mcp.Client {
	return s.client
}

// InitializeResult returns the negotiated server metadata.
func (s *Session) InitializeResult() *mcp.InitializeResult {
	return s.init
}

// RootStore returns the persistent root store backing the session.
func (s *Session) RootStore() *StateStore {
	return s.roots
}

// Subscribe returns a best-effort event stream of session notifications.
func (s *Session) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 32
	}
	ch := make(chan Event, buffer)
	s.eventsMu.Lock()
	id := s.nextID
	s.nextID++
	s.events[id] = ch
	s.eventsMu.Unlock()
	return ch, func() {
		s.eventsMu.Lock()
		if ch, ok := s.events[id]; ok {
			delete(s.events, id)
			close(ch)
		}
		s.eventsMu.Unlock()
	}
}

func (s *Session) publish(event Event) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	for _, ch := range s.events {
		select {
		case ch <- event:
		default:
		}
	}
}

// ListToolsAll retrieves every page of tools and returns them sorted by name.
func (s *Session) ListToolsAll(ctx context.Context) ([]mcp.Tool, error) {
	cursor := ""
	var all []mcp.Tool
	for {
		result, err := s.client.ListTools(ctx, mcp.ListToolsRequest{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Tools...)
		if result.NextCursor == "" || result.NextCursor == cursor {
			break
		}
		cursor = result.NextCursor
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	return all, nil
}

// ListResourcesAll retrieves every page of resources and returns them sorted by URI.
func (s *Session) ListResourcesAll(ctx context.Context) ([]mcp.Resource, error) {
	cursor := ""
	var all []mcp.Resource
	for {
		result, err := s.client.ListResources(ctx, mcp.ListResourcesRequest{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Resources...)
		if result.NextCursor == "" || result.NextCursor == cursor {
			break
		}
		cursor = result.NextCursor
	}
	sort.Slice(all, func(i, j int) bool { return all[i].URI < all[j].URI })
	return all, nil
}

// ListPromptsAll retrieves every page of prompts and returns them sorted by name.
func (s *Session) ListPromptsAll(ctx context.Context) ([]mcp.Prompt, error) {
	cursor := ""
	var all []mcp.Prompt
	for {
		result, err := s.client.ListPrompts(ctx, mcp.ListPromptsRequest{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Prompts...)
		if result.NextCursor == "" || result.NextCursor == cursor {
			break
		}
		cursor = result.NextCursor
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	return all, nil
}

// CallRaw invokes an arbitrary method and unmarshals into out when non-nil.
func (s *Session) CallRaw(ctx context.Context, method string, params any, out any) error {
	if s.client == nil {
		return errors.New("session is not connected")
	}
	return s.client.Call(ctx, method, params, out)
}

// Supports reports whether the server advertised a given capability group.
func (s *Session) Supports(name string) bool {
	if s.init == nil {
		return false
	}
	switch name {
	case "logging":
		return s.init.Capabilities.Logging != nil
	case "completions":
		return s.init.Capabilities.Completions != nil
	case "tasks":
		return s.init.Capabilities.Tasks != nil
	case "resources":
		return s.init.Capabilities.Resources != nil
	case "prompts":
		return s.init.Capabilities.Prompts != nil
	case "tools":
		return s.init.Capabilities.Tools != nil
	default:
		return false
	}
}
