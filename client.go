package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// Client represents an MCP client.
type Client struct {
	*rpc.Client
}

// NewClient creates a new MCP client.
func NewClient(conn io.ReadWriteCloser) *Client {
	return &Client{jsonrpc.NewClient(conn)}
}

// Initialize sends the initialize request.
func (c *Client) Initialize(ctx context.Context, clientInfo Implementation) (*InitializeReply, error) {
	args := &InitializeArgs{
		ProtocolVersion: ProtocolVersion,
		ClientInfo:      clientInfo,
		Capabilities: ClientCapabilities{
			Sampling: &struct{}{},
		},
	}

	var reply InitializeReply
	err := c.Call("MCP.Initialize", args, &reply)
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

// ListTools requests available tools.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	var reply ListToolsReply
	err := c.Call("MCP.ListTools", &ListToolsArgs{}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Tools, nil
}

// CallTool executes a tool.
func (c *Client) CallTool(ctx context.Context, name string, args interface{}) (*ToolResult, error) {
	argBytes, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("marshal args: %w", err)
	}

	callArgs := &CallToolArgs{
		Name:      name,
		Arguments: argBytes,
	}

	var reply CallToolReply
	if err := c.Call("MCP.CallTool", callArgs, &reply); err != nil {
		return nil, err
	}

	return (*ToolResult)(&reply), nil
}
