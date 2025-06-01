// Package server provides a compatibility layer for mark3labs-mcp-go server types
// This allows existing mark3labs servers to use the adapter with just an import change
package server

import (
	"context"
	
	"github.com/mark3labs/mcp-go/server"
	"github.com/tmc/mcprepos/mcp/adapters"
	"github.com/tmc/mcprepos/mcp/adapters/mark3labs"
)

// Re-export server types and functions
type (
	MCPServer = server.MCPServer
	Option    = server.Option
	
	// Handler function types
	ToolHandlerFunc     = server.ToolHandlerFunc
	ResourceHandlerFunc = server.ResourceHandlerFunc
	ResourceTemplateHandlerFunc = server.ResourceTemplateHandlerFunc
	PromptHandlerFunc   = server.PromptHandlerFunc
	
	// Other server types
	StdioServer = server.StdioServer
	StdioOption = server.StdioOption
)

// NewMCPServer creates a new MCPServer that includes adapter functionality
func NewMCPServer(name string, version string, options ...server.Option) *MCPServer {
	s := server.NewMCPServer(name, version, options...)
	
	// Register the adapter in the global registry
	adapter := mark3labs.NewAdapter(s, mark3labs.Mark3LabsOptions{
		ServerName:     name,
		ServerVersion:  version,
	})
	adapters.RegisterAdapter(name, adapter)
	
	// Extend the server with adapter methods
	return extendServer(s, adapter)
}

// extendServer adds adapter functionality to the mark3labs server
func extendServer(s *MCPServer, adapter adapters.Adapter) *MCPServer {
	// We can't directly modify the MCPServer struct, but we can
	// store the adapter in a global map for retrieval
	serverAdapterMap[s] = adapter
	return s
}

// serverAdapterMap stores the mapping between servers and their adapters
var serverAdapterMap = make(map[*MCPServer]adapters.Adapter)

// GetAdapter retrieves the adapter for a given server
func GetAdapter(s *MCPServer) adapters.Adapter {
	return serverAdapterMap[s]
}

// Extend the MCPServer type with adapter methods
type ExtendedMCPServer struct {
	*MCPServer
	adapter adapters.Adapter
}

// GetAdapter returns the adapter for this server
func (s *ExtendedMCPServer) GetAdapter() adapters.Adapter {
	return s.adapter
}

// Re-export server functions
var (
	WithToolCapabilities     = server.WithToolCapabilities
	WithResourceCapabilities = server.WithResourceCapabilities
	WithPromptCapabilities   = server.WithPromptCapabilities
	WithLogging              = server.WithLogging
	WithHooks                = server.WithHooks
	
	// Stdio server functions
	NewStdioServer           = server.NewStdioServer
	ServeStdio               = server.ServeStdio
	WithErrorLogger          = server.WithErrorLogger
	WithStdioContextFunc     = server.WithStdioContextFunc
)