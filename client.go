package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Client implements an MCP client.
type Client struct {
	transport Transport
}

// NewClient creates a new MCP client.
func NewClient(t Transport) *Client {
	return &Client{transport: t}
}

// Call makes an MCP request and waits for the response.
func (c *Client) Call(ctx context.Context, method string, params interface{}) ([]byte, error) {
	req := struct {
		Method string      `json:"method"`
		Params interface{} `json:"params,omitempty"`
	}{
		Method: method,
		Params: params,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mcp: marshaling request: %w", err)
	}

	if _, err := c.transport.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("mcp: writing request: %w", err)
	}

	// Simple response reading
	buf := make([]byte, 32*1024)
	n, err := c.transport.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("mcp: reading response: %w", err)
	}

	return buf[:n], nil
}

// Close closes the client's transport.
func (c *Client) Close() error {
	return c.transport.Close()
}
