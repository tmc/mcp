package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// Client implements an MCP client.
type Client struct {
	transport Transport
	name      string
	version   string
}

// NewClient creates a new MCP client.
func NewClient(name, version string, transport Transport) *Client {
	return &Client{
		transport: transport,
		name:      name,
		version:   version,
	}
}

// Initialize sends the initialize request.
func (c *Client) Initialize(ctx context.Context) (*InitializeReply, error) {
	args := InitializeArgs{
		Name:    c.name,
		Version: c.version,
	}
	data, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	resp, err := c.call(ctx, "initialize", data)
	if err != nil {
		return nil, err
	}

	var reply InitializeReply
	if err := json.Unmarshal(resp, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// ListTools sends the listTools request.
func (c *Client) ListTools(ctx context.Context) (*ListToolsReply, error) {
	resp, err := c.call(ctx, "listTools", nil)
	if err != nil {
		return nil, err
	}

	var reply ListToolsReply
	if err := json.Unmarshal(resp, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// CallTool sends a callTool request.
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (*CallToolReply, error) {
	data, err := json.Marshal(CallToolArgs{
		Name: name,
		Args: args,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.call(ctx, name, data)
	if err != nil {
		return nil, err
	}

	var reply CallToolReply
	if err := json.Unmarshal(resp, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

func (c *Client) call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	req := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.Number     `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}{
		JSONRPC: JSONRPCVersion,
		ID:      "1", // TODO: Generate unique IDs
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if _, err := c.transport.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)
	n, err := c.transport.Read(buf)
	if err != nil {
		return nil, err
	}

	var resp struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.Number     `json:"id"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("mcp: error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}
