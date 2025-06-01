package adapters_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tmc/mcprepos/mcp/adapters"
	"github.com/tmc/mcprepos/mcp/adapters/golang_tools"
	"github.com/tmc/mcprepos/mcp/adapters/mark3labs"
	"github.com/tmc/mcprepos/mcp/protocol"
)

func TestMark3LabsAdapter(t *testing.T) {
	adapter := mark3labs.NewAdapter()

	// Test capabilities
	caps := adapter.GetCapabilities()
	if caps.Resources == nil {
		t.Error("Expected resources capability")
	}
	if caps.Tools == nil {
		t.Error("Expected tools capability")
	}
	if caps.Prompts == nil {
		t.Error("Expected prompts capability")
	}

	// Test initialize
	initReq := &protocol.InitializeRequest{
		ProtocolVersion: "1.0.0",
		ClientInfo: protocol.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	_, err := adapter.Initialize(ctx, initReq)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test list tools
	req := protocol.Request{
		Method: "tools/list",
		ID:     json.RawMessage(`"1"`),
	}

	_, err = adapter.HandleRequest(ctx, req)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
}

func TestGolangToolsAdapter(t *testing.T) {
	adapter := golang_tools.NewAdapter()

	// Test capabilities
	caps := adapter.GetCapabilities()
	if caps.Tools == nil {
		t.Error("Expected tools capability")
	}
	if caps.Prompts == nil {
		t.Error("Expected prompts capability")
	}

	// Test initialize
	initReq := &protocol.InitializeRequest{
		ProtocolVersion: "1.0.0",
		ClientInfo: protocol.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	_, err := adapter.Initialize(ctx, initReq)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test list tools
	req := protocol.Request{
		Method: "tools/list",
		ID:     json.RawMessage(`"1"`),
	}

	_, err = adapter.HandleRequest(ctx, req)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
}

func TestAdapterRegistry(t *testing.T) {
	// Test that adapters are registered
	registry := adapters.GetRegistry()
	
	if registry.GetAdapter("mark3labs") == nil {
		t.Error("mark3labs adapter not registered")
	}
	
	if registry.GetAdapter("golang-tools") == nil {
		t.Error("golang-tools adapter not registered")
	}
}

func TestCustomAdapter(t *testing.T) {
	// Test custom adapter implementation
	type testAdapter struct{}

	func (t *testAdapter) Initialize(ctx context.Context, req *protocol.InitializeRequest) (*protocol.InitializeResult, error) {
		return &protocol.InitializeResult{
			ServerInfo: protocol.ServerInfo{
				Name:    "test-server",
				Version: "1.0.0",
			},
		}, nil
	}

	func (t *testAdapter) HandleRequest(ctx context.Context, req protocol.Request) (protocol.Response, error) {
		return protocol.Response{
			ID: req.ID,
			Result: json.RawMessage(`{}`),
		}, nil
	}

	func (t *testAdapter) GetCapabilities() protocol.Capabilities {
		return protocol.Capabilities{}
	}

	// Register custom adapter
	adapter := &testAdapter{}
	adapters.RegisterAdapter("test", adapter)

	// Test that it can be retrieved
	retrieved := adapters.GetRegistry().GetAdapter("test")
	if retrieved == nil {
		t.Error("Custom adapter not registered")
	}
}