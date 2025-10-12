package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/tmc/mcp/internal/jsonrpc2util"
	"golang.org/x/exp/jsonrpc2"
)

// Client represents an MCP client capable of connecting to and interacting with an MCP server.
//
// The client automatically handles context cancellation by sending appropriate
// notifications to the server. When using context.WithCancelCause, the cancellation
// reason is automatically propagated to the server via the notifications/cancelled message.
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

// NewClient creates a new MCP client instance using the provided transport.
func NewClient(transport Transport, opts ...ClientOption) (*Client, error) {
	ctx := context.Background()
	c := &Client{}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Create the connection
	handler := jsonrpc2.HandlerFunc(c.handleMessage)
	conn, err := jsonrpc2.Dial(ctx, transport, jsonrpc2util.ConnectionBinder{Handler: handler})
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON-RPC connection: %w", err)
	}
	c.conn = conn

	return c, nil
}

// handleMessage processes incoming JSON-RPC messages from the server.
// It distinguishes between notifications (which have no ID) and regular requests.
// For notifications, it dispatches them to the registered notification handler.
// For regular requests, it returns a "method not implemented" error since clients
// typically don't handle incoming requests, only responses.
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
	// Check if already initialized
	c.initMu.Lock()
	if c.initialized {
		c.initMu.Unlock()
		return nil, errors.New("client already initialized")
	}
	c.initMu.Unlock()

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

// call is a helper method that performs a JSON-RPC call with automatic cancellation notification.
// When the context is cancelled, it automatically sends a cancellation notification to the server
// using the notifications/cancelled method. This ensures proper cleanup of server-side operations
// when clients cancel their requests. The method supports context.WithCancelCause to propagate
// cancellation reasons to the server.
func (c *Client) call(ctx context.Context, method string, params, result interface{}) error {
	if c.conn == nil {
		return errors.New("client connection is not established")
	}

	// Call the method and get the AsyncCall object
	asyncCall := c.conn.Call(ctx, method, params)

	// Create a channel to signal when the call is done
	done := make(chan struct{})

	// Monitor context cancellation in a separate goroutine
	go func() {
		select {
		case <-ctx.Done():
			// Check if there's a cancellation cause
			cause := context.Cause(ctx)

			// Send cancellation notification if there's a specific cause
			// or if the context was cancelled (not just deadline exceeded)
			if cause != nil && (cause != context.Canceled || cause == context.Canceled) {
				// Get the raw ID value to ensure proper marshaling
				idValue := asyncCall.ID().Raw()
				if idValue == nil {
					// If ID is nil, skip cancellation
					return
				}

				cancelParams := map[string]interface{}{
					"requestId": idValue,
				}

				// Add reason from the cause
				if cause != context.Canceled {
					cancelParams["reason"] = cause.Error()
				}

				// Send the notification (best effort, ignore errors)
				_ = c.conn.Notify(context.Background(), string(MethodNotificationCancelled), cancelParams)
			}
		case <-done:
			// Call completed normally, exit goroutine
		}
	}()

	// Await the results and unmarshal into result
	err := asyncCall.Await(ctx, result)
	close(done) // Signal that the call is complete

	return err
}

// checkInitialized ensures the client has been properly initialized via the Initialize method.
// This check is performed before any MCP protocol operations to ensure the handshake has
// completed successfully. Returns an error if Initialize() has not been called.
func (c *Client) checkInitialized() error {
	c.initMu.RLock()
	defer c.initMu.RUnlock()

	if !c.initialized {
		return errors.New("client not initialized, call Initialize() first")
	}
	return nil
}
