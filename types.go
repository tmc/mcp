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
	Context() context.Context
}

// Handler processes MCP messages.
type Handler interface {
	Handle(ctx context.Context, msg []byte) ([]byte, error)
}

// Tool represents an executable MCP tool.
type Tool interface {
	Name() string
	Description() string
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

// Capabilities represents server capabilities.
type Capabilities struct {
	Tools map[string]Tool `json:"tools"`
}

// Protocol messages

// InitializeArgs represents the arguments for the initialize method.
type InitializeArgs struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeReply represents the response to initialize.
type InitializeReply struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocolVersion"`
}

// ListToolsArgs represents the arguments for the listTools method.
type ListToolsArgs struct{}

// ListToolsReply represents the response to listTools.
type ListToolsReply struct {
	Tools []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tools"`
}

// CallToolArgs represents the arguments for the callTool method.
type CallToolArgs struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}

// CallToolReply represents the response to callTool.
type CallToolReply struct {
	Content []Content `json:"content"`
}

// Implementation types

// BaseTool provides a basic Tool implementation.
type BaseTool struct {
	name        string
	description string
	handler     func(context.Context, json.RawMessage) (*ToolResult, error)
}

func NewTool(name, description string, handler func(context.Context, json.RawMessage) (*ToolResult, error)) Tool {
	return &BaseTool{
		name:        name,
		description: description,
		handler:     handler,
	}
}

func (t *BaseTool) Name() string        { return t.name }
func (t *BaseTool) Description() string { return t.description }
func (t *BaseTool) Handler(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	return t.handler(ctx, args)
}
