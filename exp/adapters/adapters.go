// Package adapters provides adapters for integrating various MCP server implementations.
// These adapters handle the translation between different implementation patterns and
// the standard MCP SDK server interface.
package adapters

import (
	"context"

	"github.com/tmc/mcp/protocol"
	"github.com/tmc/mcp/server"
)

// Adapter is the common interface that all server adapters must implement.
// It provides a way to wrap different MCP implementations to work with
// the standard SDK server interface.
type Adapter interface {
	// Initialize sets up the adapter with the target server implementation
	Initialize(ctx context.Context, server server.Server) error

	// HandleRequest processes incoming requests and routes them to the appropriate
	// handler in the wrapped implementation
	HandleRequest(ctx context.Context, method string, params any) (any, error)

	// GetCapabilities returns the capabilities of the wrapped server
	GetCapabilities() protocol.ServerCapabilities
}

// Registry maintains a collection of available adapters
type Registry struct {
	adapters map[string]func() Adapter
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]func() Adapter),
	}
}

// Register adds a new adapter to the registry
func (r *Registry) Register(name string, constructor func() Adapter) {
	r.adapters[name] = constructor
}

// Get retrieves an adapter by name
func (r *Registry) Get(name string) (Adapter, bool) {
	constructor, ok := r.adapters[name]
	if !ok {
		return nil, false
	}
	return constructor(), true
}

// DefaultRegistry is the global adapter registry
var DefaultRegistry = NewRegistry()
