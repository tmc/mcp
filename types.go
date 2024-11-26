package mcp

import (
    "context"
    "encoding/json"
)

// Protocol types
type (
    InitializeArgs struct {
        ProtocolVersion string            `json:"protocolVersion"`
        Capabilities   ClientCapabilities `json:"capabilities"`
        ClientInfo     Implementation     `json:"clientInfo"`
    }

    InitializeReply struct {
        ProtocolVersion string         `json:"protocolVersion"`
        Capabilities   Capabilities    `json:"capabilities"`
        ServerInfo     Implementation  `json:"serverInfo"`
        Instructions   string          `json:"instructions,omitempty"`
    }

    ListToolsArgs struct {
        Cursor string `json:"cursor,omitempty"`
    }

    ListToolsReply struct {
        Tools      []Tool `json:"tools"`
        NextCursor string `json:"nextCursor,omitempty"`
    }

    CallToolArgs struct {
        Name      string          `json:"name"`
        Arguments json.RawMessage `json:"arguments,omitempty"`
    }

    CallToolReply ToolResult

    Tool struct {
        Name        string                 `json:"name"`
        Description string                 `json:"description,omitempty"`
        InputSchema map[string]any         `json:"inputSchema"`
        Handler     func(context.Context, json.RawMessage) (*ToolResult, error) `json:"-"`
    }

    ToolResult struct {
        Content []Content `json:"content"`
        IsError bool     `json:"isError,omitempty"`
        Meta    any      `json:"_meta,omitempty"`
    }

    Content struct {
        Type     string         `json:"type"`
        Text     string         `json:"text,omitempty"`
        Data     []byte         `json:"data,omitempty"`
        MimeType string         `json:"mimeType,omitempty"`
    }

    Implementation struct {
        Name    string `json:"name"`
        Version string `json:"version"`
    }

    Capabilities struct {
        Experimental map[string]any `json:"experimental,omitempty"`
        Tools       *struct {
            ListChanged bool `json:"listChanged,omitempty"`
        } `json:"tools,omitempty"`
    }

    ClientCapabilities struct {
        Experimental map[string]any `json:"experimental,omitempty"`
        Sampling     *struct{}      `json:"sampling,omitempty"`
    }
)

