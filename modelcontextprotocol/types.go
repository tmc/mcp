// Copyright 2024 The MCP Authors. All rights reserved.
// This file contains types for the Model Context Protocol, version 2025-03-26.
package modelcontextprotocol

import (
	"encoding/json"
)

// RequestID can be a string or a number, or null for certain error responses.
type RequestID = any // string | number | null

// ProgressToken is an opaque token used to associate progress notifications.
// It can be a string or a number.
type ProgressToken = any // string | number

// Cursor is an opaque token used to represent a cursor for pagination.
type Cursor string

// Annotations contains optional metadata for various protocol objects.
type Annotations struct {
	Audience []Role   `json:"audience,omitempty"`
	Priority *float64 `json:"priority,omitempty"`
}

// Content is an interface representing the union of TextContent, ImageContent, AudioContent, and EmbeddedResource.
// It is a sealed interface, meaning only types defined in this package can implement it.
type Content interface {
	isContent() // Unexported marker method
}

// TextContent represents textual content.
type TextContent struct {
	Type        string       `json:"type"` // Must be "text"
	Text        string       `json:"text"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

func (TextContent) isContent() {}

// ImageContent represents image data.
type ImageContent struct {
	Type        string       `json:"type"`
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

func (ImageContent) isContent() {}

// AudioContent represents audio data.
type AudioContent struct {
	Type        string       `json:"type"`
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

func (AudioContent) isContent() {}

// EmbeddedResource represents an embedded resource.
type EmbeddedResource struct {
	Type        string           `json:"type"`
	Resource    ResourceContents `json:"resource"` // Interface, requires custom unmarshal on this struct
	Annotations *Annotations     `json:"annotations,omitempty"`
}

func (EmbeddedResource) isContent() {}

// ResourceContents is an interface representing the union of TextResourceContents and BlobResourceContents.
// It is a sealed interface.
type ResourceContents interface {
	isResourceContents()
	GetURI() string
	GetMimeType() *string
}

// BaseResourceContents provides common fields for resource content types.
type BaseResourceContents struct {
	URI      string  `json:"uri"`
	MimeType *string `json:"mimeType,omitempty"`
}

func (brc BaseResourceContents) GetURI() string       { return brc.URI }
func (brc BaseResourceContents) GetMimeType() *string { return brc.MimeType }

type TextResourceContents struct {
	BaseResourceContents
	Text string `json:"text"`
}

func (TextResourceContents) isResourceContents() {}

type BlobResourceContents struct {
	BaseResourceContents
	Blob string `json:"blob"`
}

func (BlobResourceContents) isResourceContents() {}

// Reference is an interface representing the union of PromptReference and ResourceReference.
// It is a sealed interface.
type Reference interface {
	isReference()
}
type PromptReference struct {
	Type string `json:"type"` // Must be "ref/prompt"
	Name string `json:"name"`
}

func (PromptReference) isReference() {}

type ResourceReference struct {
	Type string `json:"type"` // Must be "ref/resource"
	URI  string `json:"uri"`  // URI or URI template (RFC 6570).
}

func (ResourceReference) isReference() {}

// --- JSON-RPC Message Structures ---
type JSONRPCMessage interface{ isJSONRPCMessage() }
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func (JSONRPCRequest) isJSONRPCMessage() {}

type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func (JSONRPCNotification) isJSONRPCMessage() {}

type JSONRPCBatchRequest []json.RawMessage

func (JSONRPCBatchRequest) isJSONRPCMessage() {}

type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

func (JSONRPCResponse) isJSONRPCMessage() {}

type JSONRPCBatchResponse []json.RawMessage    // Array of JSONRPCResponse (which can include ErrorObject)
func (JSONRPCBatchResponse) isJSONRPCMessage() {}

// --- Grouping Interfaces for MCP Messages ---
type ClientRequest interface{ isClientRequest() }
type ClientNotification interface{ isClientNotification() }
type ClientResult interface{ isClientResult() }
type ServerRequest interface{ isServerRequest() }
type ServerNotification interface{ isServerNotification() }
type ServerResult interface{ isServerResult() }

// --- MCP Specific Message Params and Results ---
type RequestMeta struct {
	ProgressToken *ProgressToken `json:"progressToken,omitempty"`
} // Pointer as it's optional

type EmptyResult struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

func (EmptyResult) isClientResult() {}
func (EmptyResult) isServerResult() {}

type CancelledNotificationParams struct {
	Meta      map[string]any `json:"_meta,omitempty"`
	RequestID RequestID      `json:"requestId"`
	Reason    *string        `json:"reason,omitempty"`
}

func (CancelledNotificationParams) isClientNotification() {}
func (CancelledNotificationParams) isServerNotification() {}

// Capabilities
type RootsClientCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}
type PromptsServerCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}
type ResourcesServerCapability struct {
	Subscribe   *bool `json:"subscribe,omitempty"`
	ListChanged *bool `json:"listChanged,omitempty"`
}
type ToolsServerCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}
type ClientCapabilities struct {
	Experimental map[string]any         `json:"experimental,omitempty"`
	Roots        *RootsClientCapability `json:"roots,omitempty"`
	Sampling     *struct{}              `json:"sampling,omitempty"`
}
type ServerCapabilities struct {
	Experimental map[string]any             `json:"experimental,omitempty"`
	Logging      *struct{}                  `json:"logging,omitempty"`
	Completions  *struct{}                  `json:"completions,omitempty"`
	Prompts      *PromptsServerCapability   `json:"prompts,omitempty"`
	Resources    *ResourcesServerCapability `json:"resources,omitempty"`
	Tools        *ToolsServerCapability     `json:"tools,omitempty"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type InitializeRequestParams struct {
	Meta            *RequestMeta       `json:"_meta,omitempty"`
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

func (InitializeRequestParams) isClientRequest() {}

type InitializeResult struct {
	Meta            map[string]any     `json:"_meta,omitempty"`
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    *string            `json:"instructions,omitempty"`
}

func (InitializeResult) isServerResult() {}

type InitializedNotificationParams struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

func (InitializedNotificationParams) isClientNotification() {}

type PingRequestParams struct {
	Meta *RequestMeta `json:"_meta,omitempty"`
}

func (PingRequestParams) isClientRequest() {}
func (PingRequestParams) isServerRequest() {}

type ProgressNotificationParams struct {
	Meta          map[string]any `json:"_meta,omitempty"`
	ProgressToken ProgressToken  `json:"progressToken"`
	Progress      float64        `json:"progress"`
	Total         *float64       `json:"total,omitempty"`
	Message       *string        `json:"message,omitempty"`
}

func (ProgressNotificationParams) isClientNotification() {}
func (ProgressNotificationParams) isServerNotification() {}

type ListResourcesRequestParams struct {
	Meta   *RequestMeta `json:"_meta,omitempty"`
	Cursor *Cursor      `json:"cursor,omitempty"`
}

func (ListResourcesRequestParams) isClientRequest() {}

type ListResourcesResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	Resources  []Resource     `json:"resources"`
	NextCursor *Cursor        `json:"nextCursor,omitempty"`
}

func (ListResourcesResult) isServerResult() {}

type Resource struct {
	URI         string       `json:"uri"`
	Name        string       `json:"name"`
	Description *string      `json:"description,omitempty"`
	MimeType    *string      `json:"mimeType,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Size        *int64       `json:"size,omitempty"`
}
type ListResourceTemplatesRequestParams struct {
	Meta   *RequestMeta `json:"_meta,omitempty"`
	Cursor *Cursor      `json:"cursor,omitempty"`
}

func (ListResourceTemplatesRequestParams) isClientRequest() {}

type ListResourceTemplatesResult struct {
	Meta              map[string]any     `json:"_meta,omitempty"`
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	NextCursor        *Cursor            `json:"nextCursor,omitempty"`
}

func (ListResourceTemplatesResult) isServerResult() {}

type ResourceTemplate struct {
	URITemplate string       `json:"uriTemplate"`
	Name        string       `json:"name"`
	Description *string      `json:"description,omitempty"`
	MimeType    *string      `json:"mimeType,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
}
type ReadResourceRequestParams struct {
	Meta *RequestMeta `json:"_meta,omitempty"`
	URI  string       `json:"uri"`
}

func (ReadResourceRequestParams) isClientRequest() {}

type ReadResourceResult struct {
	Meta     map[string]any     `json:"_meta,omitempty"`
	Contents []ResourceContents `json:"contents"`
}

func (ReadResourceResult) isServerResult() {}

type ResourceListChangedNotificationParams struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

func (ResourceListChangedNotificationParams) isServerNotification() {}

type SubscribeRequestParams struct {
	Meta *RequestMeta `json:"_meta,omitempty"`
	URI  string       `json:"uri"`
}

func (SubscribeRequestParams) isClientRequest() {}

type UnsubscribeRequestParams struct {
	Meta *RequestMeta `json:"_meta,omitempty"`
	URI  string       `json:"uri"`
}

func (UnsubscribeRequestParams) isClientRequest() {}

type ResourceUpdatedNotificationParams struct {
	Meta map[string]any `json:"_meta,omitempty"`
	URI  string         `json:"uri"`
}

func (ResourceUpdatedNotificationParams) isServerNotification() {}

type ListPromptsRequestParams struct {
	Meta   *RequestMeta `json:"_meta,omitempty"`
	Cursor *Cursor      `json:"cursor,omitempty"`
}

func (ListPromptsRequestParams) isClientRequest() {}

type ListPromptsResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	Prompts    []Prompt       `json:"prompts"`
	NextCursor *Cursor        `json:"nextCursor,omitempty"`
}

func (ListPromptsResult) isServerResult() {}

type Prompt struct {
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Arguments   []*PromptArgument `json:"arguments,omitempty"`
}
type PromptArgument struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Required    *bool   `json:"required,omitempty"`
}
type GetPromptRequestParams struct {
	Meta      *RequestMeta      `json:"_meta,omitempty"`
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

func (GetPromptRequestParams) isClientRequest() {}

type GetPromptResult struct {
	Meta        map[string]any  `json:"_meta,omitempty"`
	Description *string         `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

func (GetPromptResult) isServerResult() {}

type PromptMessage struct {
	Role    Role    `json:"role"`
	Content Content `json:"content"`
}
type PromptListChangedNotificationParams struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

func (PromptListChangedNotificationParams) isServerNotification() {}

type ListToolsRequestParams struct {
	Meta   *RequestMeta `json:"_meta,omitempty"`
	Cursor *Cursor      `json:"cursor,omitempty"`
}

func (ListToolsRequestParams) isClientRequest() {}

type ListToolsResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	Tools      []Tool         `json:"tools"`
	NextCursor *Cursor        `json:"nextCursor,omitempty"`
}

func (ListToolsResult) isServerResult() {}

type CallToolRequestParams struct {
	Meta      *RequestMeta   `json:"_meta,omitempty"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

func (CallToolRequestParams) isClientRequest() {}

type CallToolResult struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Content []Content      `json:"content"`
	IsError *bool          `json:"isError,omitempty"`
}

func (CallToolResult) isServerResult() {}

type ToolListChangedNotificationParams struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

func (ToolListChangedNotificationParams) isServerNotification() {}

type ToolAnnotations struct {
	Title           *string `json:"title,omitempty"`
	ReadOnlyHint    *bool   `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool   `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool   `json:"openWorldHint,omitempty"`
}
type ToolSchema struct {
	Type       string                     `json:"type"`
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
	Required   []string                   `json:"required,omitempty"`
}
type Tool struct {
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	InputSchema ToolSchema       `json:"inputSchema"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}
type SetLevelRequestParams struct {
	Meta  *RequestMeta `json:"_meta,omitempty"`
	Level LoggingLevel `json:"level"`
}

func (SetLevelRequestParams) isClientRequest() {}

type LoggingMessageNotificationParams struct {
	Meta   map[string]any  `json:"_meta,omitempty"`
	Level  LoggingLevel    `json:"level"`
	Logger *string         `json:"logger,omitempty"`
	Data   json.RawMessage `json:"data"`
}

func (LoggingMessageNotificationParams) isServerNotification() {}

// ElicitationCompleteNotificationParams informs the client that an out-of-band elicitation completed.
type ElicitationCompleteNotificationParams struct {
	Meta          map[string]any `json:"_meta,omitempty"`
	ElicitationID string         `json:"elicitationId"`
}

func (ElicitationCompleteNotificationParams) isServerNotification() {}

type ModelPreferences struct {
	Hints                []ModelHint `json:"hints,omitempty"`
	CostPriority         *float64    `json:"costPriority,omitempty"`
	SpeedPriority        *float64    `json:"speedPriority,omitempty"`
	IntelligencePriority *float64    `json:"intelligencePriority,omitempty"`
}
type ModelHint struct {
	Name *string `json:"name,omitempty"`
}
type CreateMessageRequestParams struct {
	Meta             *RequestMeta      `json:"_meta,omitempty"`
	Messages         []SamplingMessage `json:"messages"`
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
	SystemPrompt     *string           `json:"systemPrompt,omitempty"`
	IncludeContext   *string           `json:"includeContext,omitempty"`
	Temperature      *float64          `json:"temperature,omitempty"`
	MaxTokens        int               `json:"maxTokens"`
	StopSequences    []string          `json:"stopSequences,omitempty"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
}

func (CreateMessageRequestParams) isServerRequest() {}

type CreateMessageResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	Role       Role           `json:"role"`
	Content    Content        `json:"content"`
	Model      string         `json:"model"`
	StopReason *string        `json:"stopReason,omitempty"`
}

func (CreateMessageResult) isClientResult() {}

type SamplingMessage struct {
	Role    Role    `json:"role"`
	Content Content `json:"content"`
}
type CompleteRequestParams struct {
	Meta     *RequestMeta `json:"_meta,omitempty"`
	Ref      Reference    `json:"ref"`
	Argument struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
}

func (CompleteRequestParams) isClientRequest() {}

type CompleteResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	Completion struct {
		Values  []string `json:"values"`
		Total   *int     `json:"total,omitempty"`
		HasMore *bool    `json:"hasMore,omitempty"`
	}
}

func (CompleteResult) isServerResult() {}

type Root struct {
	URI  string  `json:"uri"`
	Name *string `json:"name,omitempty"`
}
type ListRootsRequestParams struct {
	Meta *RequestMeta `json:"_meta,omitempty"`
}

func (ListRootsRequestParams) isServerRequest() {}

type ListRootsResult struct {
	Meta  map[string]any `json:"_meta,omitempty"`
	Roots []Root         `json:"roots"`
}

func (ListRootsResult) isClientResult() {}

type RootsListChangedNotificationParams struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

func (RootsListChangedNotificationParams) isClientNotification() {}
