package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Protocol version constants
const (
	ProtocolVersion = "2024-11-05"
	JSONRPCVersion  = "2.0"
)

// Role represents a participant in the protocol.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// LoggingLevel represents the severity of a log message (RFC-5424)
type LoggingLevel string

const (
	LogDebug     LoggingLevel = "debug"
	LogInfo      LoggingLevel = "info"
	LogNotice    LoggingLevel = "notice"
	LogWarning   LoggingLevel = "warning"
	LogError     LoggingLevel = "error"
	LogCritical  LoggingLevel = "critical"
	LogAlert     LoggingLevel = "alert"
	LogEmergency LoggingLevel = "emergency"
)

// Service implements the MCP RPC service.
type Service struct {
	mu       sync.RWMutex
	tools    map[string]Tool
	caps     Capabilities
	version  string
	name     string
	dispatch *Dispatcher
	limiter  *RateLimiter // Add rate limiter
}

// NewService creates a new MCP service with default configuration
func NewService(name, version string, opts ...Option) *Service {
	s := &Service{
		tools:   make(map[string]Tool),
		version: version,
		name:    name,
		// Default configuration
		dispatch: NewDispatcher(),
		limiter:  NewRateLimiter(DefaultRateLimitConfig()),
		caps: Capabilities{
			Tools: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{
				ListChanged: true,
			},
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Add notification methods
func (s *Service) Handle(method string, h Handler) {
	if s.dispatch != nil {
		s.dispatch.Handle(method, h)
	}
}

func (s *Service) NotifyListChanged(method string) error {
	if s.dispatch == nil {
		return nil
	}
	switch method {
	case MethodToolListChanged:
		if s.caps.Tools == nil || !s.caps.Tools.ListChanged {
			return nil
		}
	default:
		return fmt.Errorf("unsupported list change notification: %s", method)
	}
	return s.dispatch.NotifyListChanged(method)
}

// Initialize handles client initialization.
func (s *Service) Initialize(args *InitializeArgs, reply *InitializeReply) error {
	reply.ProtocolVersion = ProtocolVersion
	reply.ServerInfo = Implementation{
		Name:    s.name,
		Version: s.version,
	}
	reply.Capabilities = s.caps
	return nil
}

// ListTools returns available tools.
func (s *Service) ListTools(args *ListToolsArgs, reply *ListToolsReply) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		// Don't expose handler in response
		t.Handler = nil
		tools = append(tools, t)
	}
	reply.Tools = tools
	return nil
}

// CallTool executes a tool.
func (s *Service) CallTool(args *CallToolArgs, reply *CallToolReply) error {
	s.mu.RLock()
	tool, ok := s.tools[args.Name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown tool: %s", args.Name)
	}

	result, err := tool.Handler(context.Background(), args.Arguments)
	if err != nil {
		return err
	}
	*reply = CallToolReply(*result)
	return nil
}

// RegisterTool adds a tool to the service.
func (s *Service) RegisterTool(t Tool) error {
	if t.Name == "" {
		return fmt.Errorf("tool name required")
	}
	if t.Handler == nil {
		return fmt.Errorf("tool handler required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tools[t.Name]; exists {
		return fmt.Errorf("tool %q already registered", t.Name)
	}

	s.tools[t.Name] = t
	return nil
}
