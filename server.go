package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Server handles MCP protocol communication.
type Server struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{
		tools: make(map[string]Tool),
	}
}

// RegisterTool registers a tool with the server.
func (s *Server) RegisterTool(t Tool) error {
	if t == nil {
		return fmt.Errorf("mcp: nil tool")
	}

	s.mu.Lock()
	s.tools[t.Name()] = t
	s.mu.Unlock()

	return nil
}

// Handle processes an incoming message.
func (s *Server) Handle(ctx context.Context, msg []byte) ([]byte, error) {
	var req struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}

	if err := json.Unmarshal(msg, &req); err != nil {
		return nil, fmt.Errorf("mcp: invalid message: %w", err)
	}

	s.mu.RLock()
	tool, exists := s.tools[req.Method]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("mcp: unknown tool: %s", req.Method)
	}

	result, err := tool.Handler(ctx, req.Params)
	if err != nil {
		return nil, fmt.Errorf("mcp: tool execution error: %w", err)
	}

	return json.Marshal(result)
}
