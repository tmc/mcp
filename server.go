package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// Standard errors
var (
	ErrInvalidMessage = errors.New("mcp: invalid message")
	ErrToolNotFound   = errors.New("mcp: tool not found")
	ErrInvalidArgs    = errors.New("mcp: invalid arguments")
)

// Server handles MCP protocol communication.
type Server struct {
	transport Transport
	handler   Handler
	tools     map[string]Tool
	mu        sync.RWMutex // protects tools
}

// NewServer creates a new MCP server.
func NewServer(t Transport) *Server {
	return &Server{
		transport: t,
		tools:     make(map[string]Tool),
	}
}

// Handle registers the main message handler.
func (s *Server) Handle(h Handler) {
	s.handler = h
}

// RegisterTool registers a tool with the server.
func (s *Server) RegisterTool(name string, t Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[name] = t
}

// Serve starts handling MCP messages.
func (s *Server) Serve(ctx context.Context) error {
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := s.transport.Read(buf)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("mcp: reading message: %w", err)
			}

			resp, err := s.handler.Handle(ctx, buf[:n])
			if err != nil {
				// Send error response
				errResp := s.errorResponse(err)
				if _, werr := s.transport.Write(errResp); werr != nil {
					return fmt.Errorf("mcp: writing error response: %w", werr)
				}
				continue
			}

			if _, err := s.transport.Write(resp); err != nil {
				return fmt.Errorf("mcp: writing response: %w", err)
			}
		}
	}
}

func (s *Server) errorResponse(err error) []byte {
	resp := struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    -32000,
			Message: err.Error(),
		},
	}
	b, _ := json.Marshal(resp)
	return b
}
