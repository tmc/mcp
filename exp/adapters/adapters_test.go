package adapters_test

import (
	"context"
	"testing"

	"github.com/tmc/mcp/exp/adapters"
	"github.com/tmc/mcp/exp/adapters/golang_tools"
	"github.com/tmc/mcp/exp/adapters/mark3labs"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func TestMark3LabsAdapter(t *testing.T) {
	adapter := mark3labs.NewAdapter()

	// Test capabilities
	caps := adapter.GetCapabilities()
	_ = caps // Capabilities are optional based on registered items

	// Test initialize (requires nil server for now)
	ctx := context.Background()
	err := adapter.Initialize(ctx, nil)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test list tools
	_, err = adapter.HandleRequest(ctx, "tools/list", nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
}

func TestGolangToolsAdapter(t *testing.T) {
	adapter := golang_tools.NewAdapter()

	// Test capabilities
	caps := adapter.GetCapabilities()
	_ = caps // Capabilities are optional based on registered items

	// Test initialize (requires nil server for now)
	ctx := context.Background()
	err := adapter.Initialize(ctx, nil)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test list tools
	_, err = adapter.HandleRequest(ctx, "tools/list", nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
}

func TestAdapterRegistry(t *testing.T) {
	// Test that adapters are registered
	registry := adapters.DefaultRegistry
	
	_, ok := registry.Get("mark3labs")
	if !ok {
		t.Error("mark3labs adapter not registered")
	}
	
	_, ok = registry.Get("golang-tools")
	if !ok {
		t.Error("golang-tools adapter not registered")
	}
}

func TestCustomAdapter(t *testing.T) {
	// Test custom adapter implementation
	type testAdapter struct{}

	func (ta *testAdapter) Initialize(ctx context.Context, srv server.Server) error {
		return nil
	}

	func (ta *testAdapter) HandleRequest(ctx context.Context, method string, params any) (any, error) {
		return map[string]interface{}{}, nil
	}

	func (ta *testAdapter) GetCapabilities() modelcontextprotocol.ServerCapabilities {
		return modelcontextprotocol.ServerCapabilities{}
	}

	// Register custom adapter
	adapters.DefaultRegistry.Register("test", func() adapters.Adapter {
		return &testAdapter{}
	})

	// Test that it can be retrieved
	_, ok := adapters.DefaultRegistry.Get("test")
	if !ok {
		t.Error("Custom adapter not registered")
	}
}