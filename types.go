package mcp

import (
	"context"
	"encoding/json"
	"errors"
)

// Protocol constants
const (
	LATEST_PROTOCOL_VERSION = "2024-11-05"
	JSONRPC_VERSION         = "2.0"
)

// Common error values
var (
	ErrInvalidParams   = errors.New("mcp: invalid parameters")
	ErrNotFound        = errors.New("mcp: not found")
	ErrUnsupported     = errors.New("mcp: operation or capability not supported")
	ErrTransportClosed = errors.New("mcp: transport closed")
)

// Role represents the sender or recipient of messages.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// MCPMethod represents the standard method names used in the protocol.
type MCPMethod string

const (
	MethodInitialize             MCPMethod = "initialize"
	MethodPing                   MCPMethod = "ping"
	MethodResourcesList          MCPMethod = "resources/list"
	MethodResourcesTemplatesList MCPMethod = "resources/templates/list"
	MethodResourcesRead          MCPMethod = "resources/read"
	MethodPromptsList            MCPMethod = "prompts/list"
	MethodPromptsGet             MCPMethod = "prompts/get"
	MethodToolsList              MCPMethod = "tools/list"
	MethodToolsCall              MCPMethod = "tools/call"
	MethodNotificationCancelled  MCPMethod = "notifications/cancelled"

	// Notification methods
	MethodProgress            MCPMethod = "notifications/progress"
	MethodLogging             MCPMethod = "notifications/message"
	MethodResourceListChanged MCPMethod = "notifications/resources/list_changed"
	MethodPromptListChanged   MCPMethod = "notifications/prompts/list_changed"
	MethodToolListChanged     MCPMethod = "notifications/tools/list_changed"
)

// JSONRPCNotification represents a notification message in the JSON-RPC protocol.
type JSONRPCNotification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// JSONRPCRequest represents a request message in the JSON-RPC protocol.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a response message in the JSON-RPC protocol.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents an error in a JSON-RPC response.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Implementation describes the name and version of an MCP client or server.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes the features supported by an MCP client.
type ClientCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
	Sampling     *struct{}      `json:"sampling,omitempty"`
}

// ServerCapabilities describes the features supported by an MCP server.
type ServerCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
	Tools        *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"tools,omitempty"`
	Resources *struct {
		Subscribe   bool `json:"subscribe,omitempty"`
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"resources,omitempty"`
	Prompts *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"prompts,omitempty"`
}

// InitializeRequest is the client's request to initialize the connection.
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      Implementation     `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// InitializeResult is the server's response to an initialize request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	Instructions    string             `json:"instructions,omitempty"`
}

// Content represents the various data types that can be included in MCP messages.
type Content interface {
	content()
}

// TextContent represents plain text data within a message.
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (TextContent) content() {}

// ImageContent represents image data within a message.
type ImageContent struct {
	Type     string `json:"type"`
	Data     []byte `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func (ImageContent) content() {}

// ListToolsRequest is the client's request to list available tools.
type ListToolsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListToolsResult is the server's response to a tools/list request.
type ListToolsResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// Tool represents the definition for a tool the client can call.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// CallToolRequest is the client's request to call a tool.
type CallToolRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallToolResult is the server's response to a tools/call request.
type CallToolResult struct {
	Content []any `json:"content"`
	IsError bool  `json:"isError,omitempty"`
	Meta    any   `json:"_meta,omitempty"`
}

// ListPromptsRequest is the client's request to list available prompts.
type ListPromptsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListPromptsResult is the server's response to a prompts/list request.
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// Prompt represents a prompt or prompt template offered by a server.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes an argument that a prompt template can accept.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Schema      any    `json:"schema,omitempty"`
}

// GetPromptRequest is the client's request to get a specific prompt.
type GetPromptRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// GetPromptResult is the server's response to a prompts/get request.
type GetPromptResult struct {
	Messages []PromptMessage `json:"messages"`
}

// PromptMessage describes a message returned as part of a prompt.
type PromptMessage struct {
	Role    Role  `json:"role"`
	Content []any `json:"content"`
}

// ListResourcesRequest is the client's request to list available resources.
type ListResourcesRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListResourcesResult is the server's response to a resources/list request.
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// Resource represents a known resource that the server is capable of reading.
type Resource struct {
	URI         string `json:"uri"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ReadResourceRequest is the client's request to read a specific resource.
type ReadResourceRequest struct {
	URI string `json:"uri"`
}

// ReadResourceResult is the server's response to a resources/read request.
type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

// ResourceContents represents the content of a specific resource.
type ResourceContents interface {
	resourceContents()
}

// TextResourceContents represents text content for a resource.
type TextResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

func (TextResourceContents) resourceContents() {}

// BlobResourceContents represents binary content for a resource.
type BlobResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Blob     string `json:"blob"` // base64 encoded
}

func (BlobResourceContents) resourceContents() {}

// ListResourceTemplatesRequest is the client's request to list available resource templates.
type ListResourceTemplatesRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListResourceTemplatesResult is the server's response to a resources/templates/list request.
type ListResourceTemplatesResult struct {
	Templates  []ResourceTemplate `json:"templates"`
	NextCursor string             `json:"nextCursor,omitempty"`
}

// ResourceTemplate represents a template description for resources available on the server.
type ResourceTemplate struct {
	Template    string `json:"template"`
	Description string `json:"description,omitempty"`
}

// TransportInterface is a deprecated alias. Use Transport from transport.go instead.
// This comment preserves the line count.

// LoggingLevel represents the severity level of a logging message.
type LoggingLevel string

// Standard logging levels
const (
	LogLevelDebug   LoggingLevel = "debug"
	LogLevelInfo    LoggingLevel = "info"
	LogLevelWarning LoggingLevel = "warning"
	LogLevelError   LoggingLevel = "error"
)

// NotificationHandler handles MCP notifications.
type NotificationHandler func(method string, params json.RawMessage) error

// ToolHandlerFunc defines the signature for functions handling tools/call requests.
type ToolHandlerFunc func(ctx context.Context, request CallToolRequest) (*CallToolResult, error)

// GetPromptHandlerFunc defines the signature for functions handling prompts/get requests.
type GetPromptHandlerFunc func(ctx context.Context, request GetPromptRequest) (*GetPromptResult, error)

// ReadResourceHandlerFunc defines the signature for functions handling resources/read requests for specific resources.
type ReadResourceHandlerFunc func(ctx context.Context, request ReadResourceRequest) ([]ResourceContents, error)

// ResourceTemplateHandlerFunc defines the signature for functions handling resources/read requests that match a resource template.
type ResourceTemplateHandlerFunc func(ctx context.Context, request ReadResourceRequest) ([]ResourceContents, error)
