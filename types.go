package mcp

import (
	"context"
	"encoding/json"
	"io"
)

// Core interfaces

// Transport handles MCP protocol communication.
type Transport interface {
	io.ReadWriteCloser
}

// Handler processes MCP messages.
type Handler interface {
	Handle(ctx context.Context, msg []byte) ([]byte, error)
}

// Tool represents an executable MCP tool.
type Tool interface {
	Name() string
	Handler(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents tool output content.
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     []byte `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Implementation types

// BaseTool provides a basic Tool implementation.
type BaseTool struct {
	name    string
	handler func(context.Context, json.RawMessage) (*ToolResult, error)
}

func NewTool(name string, handler func(context.Context, json.RawMessage) (*ToolResult, error)) Tool {
	return &BaseTool{
		name:    name,
		handler: handler,
	}
}

func (t *BaseTool) Name() string { return t.name }

func (t *BaseTool) Handler(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	return t.handler(ctx, args)
}
