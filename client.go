package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/exp/jsonrpc2"
)

// Client represents an MCP client capable of connecting to and interacting with an MCP server.
type Client struct {
	conn               *jsonrpc2.Connection
	notificationMu     sync.RWMutex
	notifyHandler      func(notification JSONRPCNotification)
	serverInfo         Implementation
	serverCapabilities ServerCapabilities
	initialized        bool
	initMu             sync.RWMutex
}

// ClientOption defines a function for configuring a Client instance.
type ClientOption func(*Client)

// WithNotificationHandler sets a notification handler for the client.
func WithNotificationHandler(handler func(notification JSONRPCNotification)) ClientOption {
	return func(c *Client) {
		c.notificationMu.Lock()
		c.notifyHandler = handler
		c.notificationMu.Unlock()
	}
}

// connectionBinder implements jsonrpc2.Binder
type connectionBinder struct {
	handler jsonrpc2.Handler
}

// Bind implements the jsonrpc2.Binder interface
func (b connectionBinder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	return jsonrpc2.ConnectionOptions{
		Handler: b.handler,
	}, nil
}

// NewClient creates a new MCP client instance using the provided transport.
func NewClient(transport Transport, opts ...ClientOption) (*Client, error) {
	ctx := context.Background()
	c := &Client{}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Create a handler for the connection
	handler := jsonrpc2.HandlerFunc(c.handleMessage)

	// Create a binder for the connection options
	binder := connectionBinder{
		handler: handler,
	}

	// Transport implements the jsonrpc2.Dialer interface directly
	// Create the connection
	conn, err := jsonrpc2.Dial(ctx, transport, binder)
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON-RPC connection: %w", err)
	}
	c.conn = conn

	return c, nil
}

// handleMessage processes incoming JSON-RPC messages
func (c *Client) handleMessage(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
	// For notifications, call the notification handler if registered
	if !req.ID.IsValid() {
		c.notificationMu.RLock()
		handler := c.notifyHandler
		c.notificationMu.RUnlock()

		if handler != nil {
			notif := JSONRPCNotification{
				Method: req.Method,
				Params: req.Params,
			}
			handler(notif)
		}
		return nil, nil
	}

	// For regular requests, reply with method not implemented
	// (clients typically don't handle requests, just responses)
	return nil, errors.New("method not implemented on client")
}

// OnNotification registers a handler function to be called when asynchronous
// notifications are received from the server.
func (c *Client) OnNotification(handler func(notification JSONRPCNotification)) {
	c.notificationMu.Lock()
	c.notifyHandler = handler
	c.notificationMu.Unlock()
}

// Initialize performs the initial MCP handshake with the server.
func (c *Client) Initialize(ctx context.Context, request InitializeRequest) (*InitializeResult, error) {
	if request.ProtocolVersion == "" {
		request.ProtocolVersion = LATEST_PROTOCOL_VERSION
	}

	var result InitializeResult
	if err := c.call(ctx, string(MethodInitialize), request, &result); err != nil {
		return nil, err
	}

	c.initMu.Lock()
	c.serverInfo = result.ServerInfo
	c.serverCapabilities = result.Capabilities
	c.initialized = true
	c.initMu.Unlock()

	return &result, nil
}

// Ping sends a ping request to check server liveness.
func (c *Client) Ping(ctx context.Context) error {
	var result interface{}
	return c.call(ctx, string(MethodPing), nil, &result)
}

// ListTools requests a list of available tools from the server.
func (c *Client) ListTools(ctx context.Context, request ListToolsRequest) (*ListToolsResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	var result ListToolsResult
	if err := c.call(ctx, string(MethodToolsList), request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CallTool invokes a specific tool on the server.
func (c *Client) CallTool(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	var result CallToolResult
	if err := c.call(ctx, string(MethodToolsCall), request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListPrompts requests a list of available prompts from the server.
func (c *Client) ListPrompts(ctx context.Context, request ListPromptsRequest) (*ListPromptsResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	var result ListPromptsResult
	if err := c.call(ctx, string(MethodPromptsList), request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPrompt retrieves a specific prompt from the server.
func (c *Client) GetPrompt(ctx context.Context, request GetPromptRequest) (*GetPromptResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	var result GetPromptResult
	if err := c.call(ctx, string(MethodPromptsGet), request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListResources requests a list of available resources from the server.
func (c *Client) ListResources(ctx context.Context, request ListResourcesRequest) (*ListResourcesResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	var result ListResourcesResult
	if err := c.call(ctx, string(MethodResourcesList), request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ReadResource reads the content of a specific resource from the server.
func (c *Client) ReadResource(ctx context.Context, request ReadResourceRequest) (*ReadResourceResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	var result ReadResourceResult
	if err := c.call(ctx, string(MethodResourcesRead), request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListResourceTemplates requests a list of available resource templates from the server.
func (c *Client) ListResourceTemplates(ctx context.Context, request ListResourceTemplatesRequest) (*ListResourceTemplatesResult, error) {
	if err := c.checkInitialized(); err != nil {
		return nil, err
	}

	var result ListResourceTemplatesResult
	if err := c.call(ctx, string(MethodResourcesTemplatesList), request, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Close terminates the connection to the server.
func (c *Client) Close() error {
	return c.conn.Close()
}

// call is a helper to perform a JSON-RPC call
func (c *Client) call(ctx context.Context, method string, params, result interface{}) error {
	// Call the method and get the AsyncCall object
	asyncCall := c.conn.Call(ctx, method, params)

	// Await the results and unmarshal into result
	if err := asyncCall.Await(ctx, result); err != nil {
		return err
	}

	return nil
}

// checkInitialized ensures the client has been initialized
func (c *Client) checkInitialized() error {
	c.initMu.RLock()
	defer c.initMu.RUnlock()

	if !c.initialized {
		return errors.New("client not initialized, call Initialize() first")
	}
	return nil
}
