package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Server handles MCP protocol communication.
type Server struct {
	mu       sync.RWMutex
	tools    map[string]Tool
	name     string
	version  string
	dispatch *Dispatcher
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{
		tools:    make(map[string]Tool),
		name:     name,
		version:  version,
		dispatch: NewDispatcher(),
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
		JSONRPC string          `json:"jsonrpc"`
		ID      json.Number     `json:"id,omitempty"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}

	if err := json.Unmarshal(msg, &req); err != nil {
		return nil, fmt.Errorf("mcp: invalid message: %w", err)
	}

	if req.JSONRPC != JSONRPCVersion {
		return nil, fmt.Errorf("mcp: unsupported JSON-RPC version: %s", req.JSONRPC)
	}

	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		var args InitializeArgs
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return nil, fmt.Errorf("mcp: invalid initialize params: %w", err)
		}
		result = InitializeReply{
			Name:            s.name,
			Version:         s.version,
			ProtocolVersion: ProtocolVersion,
		}

	case "listTools":
		var tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		s.mu.RLock()
		for _, tool := range s.tools {
			tools = append(tools, struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}{
				Name:        tool.Name(),
				Description: tool.Description(),
			})
		}
		s.mu.RUnlock()
		result = ListToolsReply{Tools: tools}

	default:
		s.mu.RLock()
		tool, exists := s.tools[req.Method]
		s.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("mcp: unknown method: %s", req.Method)
		}

		toolResult, err := tool.Handler(ctx, req.Params)
		if err != nil {
			return s.errorResponse(req.ID, -32000, err.Error()), nil
		}
		result = CallToolReply{
			Content: toolResult.Content,
		}
	}

	if err != nil {
		return s.errorResponse(req.ID, -32000, err.Error()), nil
	}

	return s.successResponse(req.ID, result)
}

func (s *Server) successResponse(id json.Number, result interface{}) ([]byte, error) {
	resp := struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      json.Number `json:"id,omitempty"`
		Result  interface{} `json:"result,omitempty"`
	}{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
	return json.Marshal(resp)
}

func (s *Server) errorResponse(id json.Number, code int, message string) []byte {
	resp := struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      json.Number `json:"id,omitempty"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	return data
}
