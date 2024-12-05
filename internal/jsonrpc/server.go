package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Method handles a JSON-RPC method call
type Method func(ctx context.Context, params json.RawMessage) (any, error)

// Server implements a JSON-RPC 2.0 server
type Server struct {
	methods sync.Map // map[string]Method
}

// NewServer creates a new JSON-RPC server
func NewServer() *Server {
	return &Server{}
}

// RegisterMethod registers a method handler
func (s *Server) RegisterMethod(name string, method Method) {
	s.methods.Store(name, method)
}

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Serve handles JSON-RPC requests on the given transport
func (s *Server) Serve(t io.ReadWriteCloser) error {
	dec := json.NewDecoder(t)
	enc := json.NewEncoder(t)

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decode error: %w", err)
		}

		if req.JSONRPC != "2.0" {
			s.writeError(enc, req.ID, -32600, "invalid JSON-RPC version")
			continue
		}

		method, ok := s.methods.Load(req.Method)
		if !ok {
			s.writeError(enc, req.ID, -32601, fmt.Sprintf("method %q not found", req.Method))
			continue
		}

		result, err := method.(Method)(context.Background(), req.Params)
		if err != nil {
			s.writeError(enc, req.ID, -32000, err.Error())
			continue
		}

		resultBytes, err := json.Marshal(result)
		if err != nil {
			s.writeError(enc, req.ID, -32603, fmt.Sprintf("marshal error: %v", err))
			continue
		}

		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resultBytes,
		}
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("encode error: %w", err)
		}
	}
}

func (s *Server) writeError(enc *json.Encoder, id any, code int, message string) error {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	return enc.Encode(resp)
}
