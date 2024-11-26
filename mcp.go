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

// Service implements the MCP RPC service.
type Service struct {
	mu      sync.RWMutex
	tools   map[string]Tool
	caps    Capabilities
	version string
	name    string
}

// NewService creates a new MCP service.
func NewService(name, version string) *Service {
	return &Service{
		tools:   make(map[string]Tool),
		version: version,
		name:    name,
		caps: Capabilities{
			Tools: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{
				ListChanged: true,
			},
		},
	}
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
