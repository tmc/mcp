package mcp

import (
    "io"
    "net/rpc"
    "net/rpc/jsonrpc"
)

// Server wraps an MCP service for network serving.
type Server struct {
    *rpc.Server
}

// NewServer creates a new MCP server.
func NewServer(service *Service) *Server {
    s := rpc.NewServer()
    s.RegisterName("MCP", service)
    return &Server{s}
}

// ServeConn serves a single connection.
func (s *Server) ServeConn(conn io.ReadWriteCloser) {
    s.Server.ServeCodec(jsonrpc.NewServerCodec(conn))
}

