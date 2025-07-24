// Package mcptestutil provides mock implementations for testing MCP components.
package mcptestutil

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/tmc/mcp"
)

// MockServer provides a mock MCP server for testing.
type MockServer struct {
	mu              sync.RWMutex
	tools           map[string]*mcp.Tool
	resources       map[string]*mcp.Resource
	prompts         map[string]*mcp.Prompt
	initialized     bool
	capabilities    mcp.ServerCapabilities
	serverInfo      mcp.Implementation
	requestHandlers map[string]func(interface{}) (interface{}, error)
}

// NewMockServer creates a new mock MCP server.
func NewMockServer() *MockServer {
	return &MockServer{
		tools:           make(map[string]*mcp.Tool),
		resources:       make(map[string]*mcp.Resource),
		prompts:         make(map[string]*mcp.Prompt),
		requestHandlers: make(map[string]func(interface{}) (interface{}, error)),
		capabilities: mcp.ServerCapabilities{
			Tools: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{},
			Resources: &struct {
				Subscribe   bool `json:"subscribe,omitempty"`
				ListChanged bool `json:"listChanged,omitempty"`
			}{},
			Prompts: &struct {
				ListChanged bool `json:"listChanged,omitempty"`
			}{},
		},
		serverInfo: mcp.Implementation{
			Name:    "mock-server",
			Version: "1.0.0",
		},
	}
}

// SetServerInfo configures the server information.
func (m *MockServer) SetServerInfo(info mcp.Implementation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverInfo = info
}

// AddTool adds a tool to the mock server.
func (m *MockServer) AddTool(tool *mcp.Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name] = tool
}

// AddResource adds a resource to the mock server.
func (m *MockServer) AddResource(resource *mcp.Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources[resource.URI] = resource
}

// AddPrompt adds a prompt to the mock server.
func (m *MockServer) AddPrompt(prompt *mcp.Prompt) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prompts[prompt.Name] = prompt
}

// SetRequestHandler sets a custom handler for a specific request method.
func (m *MockServer) SetRequestHandler(method string, handler func(interface{}) (interface{}, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestHandlers[method] = handler
}

// Initialize handles the initialize request.
func (m *MockServer) Initialize(ctx context.Context, req *mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initialized = true

	return &mcp.InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    m.capabilities,
		ServerInfo:      m.serverInfo,
	}, nil
}

// ListTools returns the list of available tools.
func (m *MockServer) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, errors.New("server not initialized")
	}

	tools := make([]*mcp.Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools, nil
}

// CallTool executes a tool call.
func (m *MockServer) CallTool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, errors.New("server not initialized")
	}

	tool, exists := m.tools[req.Name]
	if !exists {
		return nil, errors.New("tool not found")
	}

	// Check if there's a custom handler
	if handler, exists := m.requestHandlers["tools/call"]; exists {
		result, err := handler(req)
		if err != nil {
			return nil, err
		}
		return result.(*mcp.CallToolResult), nil
	}

	// Default behavior: return success with tool name
	return &mcp.CallToolResult{
		Content: []any{
			mcp.TextContent{
				Type: "text",
				Text: "Tool " + tool.Name + " executed successfully",
			},
		},
	}, nil
}

// ListResources returns the list of available resources.
func (m *MockServer) ListResources(ctx context.Context) ([]*mcp.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, errors.New("server not initialized")
	}

	resources := make([]*mcp.Resource, 0, len(m.resources))
	for _, resource := range m.resources {
		resources = append(resources, resource)
	}
	return resources, nil
}

// MockClient provides a mock MCP client for testing.
type MockClient struct {
	mu           sync.RWMutex
	responses    map[string]interface{}
	errors       map[string]error
	callHistory  []string
	initialized  bool
	capabilities mcp.ClientCapabilities
	clientInfo   mcp.Implementation
}

// NewMockClient creates a new mock MCP client.
func NewMockClient() *MockClient {
	return &MockClient{
		responses: make(map[string]interface{}),
		errors:    make(map[string]error),
		capabilities: mcp.ClientCapabilities{
			Experimental: make(map[string]interface{}),
			Sampling:     &struct{}{},
		},
		clientInfo: mcp.Implementation{
			Name:    "mock-client",
			Version: "1.0.0",
		},
	}
}

// SetResponse sets a mock response for a specific method.
func (m *MockClient) SetResponse(method string, response interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[method] = response
}

// SetError sets a mock error for a specific method.
func (m *MockClient) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method] = err
}

// GetCallHistory returns the history of method calls.
func (m *MockClient) GetCallHistory() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.callHistory...)
}

// ClearCallHistory clears the call history.
func (m *MockClient) ClearCallHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callHistory = nil
}

// recordCall records a method call in the history.
func (m *MockClient) recordCall(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callHistory = append(m.callHistory, method)
}

// getResponse returns the mock response or error for a method.
func (m *MockClient) getResponse(method string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, exists := m.errors[method]; exists {
		return nil, err
	}

	if response, exists := m.responses[method]; exists {
		return response, nil
	}

	return nil, errors.New("no mock response configured for method: " + method)
}

// Initialize performs client initialization.
func (m *MockClient) Initialize(ctx context.Context, req *mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	m.recordCall("initialize")

	response, err := m.getResponse("initialize")
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.initialized = true
	m.mu.Unlock()

	return response.(*mcp.InitializeResult), nil
}

// ListTools lists available tools.
func (m *MockClient) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	m.recordCall("tools/list")

	response, err := m.getResponse("tools/list")
	if err != nil {
		return nil, err
	}

	return response.([]*mcp.Tool), nil
}

// CallTool executes a tool call.
func (m *MockClient) CallTool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.recordCall("tools/call")

	response, err := m.getResponse("tools/call")
	if err != nil {
		return nil, err
	}

	return response.(*mcp.CallToolResult), nil
}

// MockTransportConn provides a mock transport connection for testing.
type MockTransportConn struct {
	readBuffer  []byte
	writeBuffer []byte
	closed      bool
	readDelay   time.Duration
	writeDelay  time.Duration
	mu          sync.RWMutex
}

// NewMockTransportConn creates a new mock transport connection.
func NewMockTransportConn() *MockTransportConn {
	return &MockTransportConn{}
}

// SetReadData sets the data to be returned by Read calls.
func (m *MockTransportConn) SetReadData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuffer = append([]byte(nil), data...)
}

// GetWrittenData returns the data written to the connection.
func (m *MockTransportConn) GetWrittenData() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]byte(nil), m.writeBuffer...)
}

// SetReadDelay sets a delay for Read operations.
func (m *MockTransportConn) SetReadDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDelay = delay
}

// SetWriteDelay sets a delay for Write operations.
func (m *MockTransportConn) SetWriteDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeDelay = delay
}

// Read implements io.Reader.
func (m *MockTransportConn) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, errors.New("connection closed")
	}

	if m.readDelay > 0 {
		time.Sleep(m.readDelay)
	}

	if len(m.readBuffer) == 0 {
		return 0, io.EOF
	}

	n := copy(p, m.readBuffer)
	m.readBuffer = m.readBuffer[n:]
	return n, nil
}

// Write implements io.Writer.
func (m *MockTransportConn) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, errors.New("connection closed")
	}

	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}

	m.writeBuffer = append(m.writeBuffer, p...)
	return len(p), nil
}

// Close implements io.Closer.
func (m *MockTransportConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// IsClosed returns whether the connection is closed.
func (m *MockTransportConn) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}
