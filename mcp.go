package mcp

import (
	"context"
	"encoding/json"
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
	name     string
	version  string
	dispatch *Dispatcher
}

// NewService creates a new MCP service with default configuration
func NewService(name, version string) *Service {
	return &Service{
		tools:    make(map[string]Tool),
		name:     name,
		version:  version,
		dispatch: NewDispatcher(),
	}
}

// RegisterTool registers a tool with the service
func (s *Service) RegisterTool(t Tool) error {
	if t == nil {
		return fmt.Errorf("mcp: nil tool")
	}

	s.mu.Lock()
	s.tools[t.Name()] = t
	s.mu.Unlock()

	return nil
}

// Handle processes an incoming message
func (s *Service) Handle(ctx context.Context, msg []byte) ([]byte, error) {
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

func (s *Service) successResponse(id json.Number, result interface{}) ([]byte, error) {
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

func (s *Service) errorResponse(id json.Number, code int, message string) []byte {
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
