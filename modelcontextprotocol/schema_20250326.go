// This is based on MCP schema version 2025-03-26.
package modelcontextprotocol

import (
	"encoding/json"
	"fmt"
)

/* JSON-RPC types */

// JSONRPCMessage represents any valid JSON-RPC object that can be decoded off the wire,
// or encoded to be sent.
// Implementations:
//   - [JSONRPCRequest]
//   - [JSONRPCNotification]
//   - [JSONRPCBatchRequest]
//   - [JSONRPCBatchResponse]
//   - [JSONRPCResponse]
//   - [JSONRPCError]
type JSONRPCMessage interface {
	isJSONRPCMessage()
}

// JSONRPCRequest represents a request that expects a response.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func (*JSONRPCRequest) isJSONRPCMessage() {}

// JSONRPCNotification represents a notification which does not expect a response.
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func (*JSONRPCNotification) isJSONRPCMessage() {}

// JSONRPCBatchRequest represents a JSON-RPC batch request, as described in
// https://www.jsonrpc.org/specification#batch.
type JSONRPCBatchRequest []json.RawMessage

func (JSONRPCBatchRequest) isJSONRPCMessage() {}

// JSONRPCBatchResponse represents a JSON-RPC batch response, as described in
// https://www.jsonrpc.org/specification#batch.
type JSONRPCBatchResponse []json.RawMessage

func (JSONRPCBatchResponse) isJSONRPCMessage() {}

// Constants for JSON-RPC and protocol versions
const (
	// LatestProtocolVersion is the current version of the Model Context Protocol
	LatestProtocolVersion = "2025-03-26"

	// JSONRPCVersion is the JSON-RPC version used by the protocol (2.0)
	JSONRPCVersion = "2.0"
)

// RequestID is a Request identifier.
// It can be either a string or a number value.
type RequestID struct {
	value any
}

// StringID creates a new string request identifier.
func StringID(s string) RequestID { return RequestID{value: s} }

// Int64ID creates a new integer request identifier.
func Int64ID(i int64) RequestID { return RequestID{value: i} }

// Float64ID creates a new float64 request identifier.
func Float64ID(f float64) RequestID { return RequestID{value: f} }

// IsValid returns true if the ID is a valid identifier.
// The default value for RequestID will return false.
func (id RequestID) IsValid() bool { return id.value != nil }

// String returns a string representation of the ID.
func (id RequestID) String() string {
	if !id.IsValid() {
		return "<invalid>"
	}
	return fmt.Sprintf("%v", id.value)
}

// MarshalJSON implements the json.Marshaler interface.
func (id RequestID) MarshalJSON() ([]byte, error) {
	if !id.IsValid() {
		return []byte("null"), nil
	}
	return json.Marshal(id.value)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (id *RequestID) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		id.value = s
		return nil
	}

	// Then try number
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		// Check if it's an integer
		if n == float64(int64(n)) {
			id.value = int64(n)
		} else {
			id.value = n
		}
		return nil
	}

	// Finally try null
	var null any
	if err := json.Unmarshal(data, &null); err == nil && null == nil {
		id.value = nil
		return nil
	}

	return fmt.Errorf("invalid request ID format: %s", string(data))
}

// AsInt64 attempts to convert the ID to an int64.
// Returns the value and a boolean indicating if the conversion was successful.
func (id RequestID) AsInt64() (int64, bool) {
	switch v := id.value.(type) {
	case int64:
		return v, true
	case float64:
		if v == float64(int64(v)) {
			return int64(v), true
		}
	}
	return 0, false
}

// AsFloat64 attempts to convert the ID to a float64.
// Returns the value and a boolean indicating if the conversion was successful.
func (id RequestID) AsFloat64() (float64, bool) {
	switch v := id.value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	}
	return 0, false
}

// AsString attempts to convert the ID to a string.
// Returns the value and a boolean indicating if the conversion was successful.
func (id RequestID) AsString() (string, bool) {
	if s, ok := id.value.(string); ok {
		return s, true
	}
	return "", false
}

// ProgressToken is used to associate progress notifications with the original request.
// It can be a string or a number.
type ProgressToken any

// Cursor is an opaque token used to represent a cursor for pagination.
type Cursor string

// Role represents the sender or recipient of messages and data in a conversation.
type Role string

// Constants for Role type
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

/* Content */

// Annotations represents optional annotations for the client.
// The client can use annotations to inform how objects are used or displayed.
type Annotations struct {
	// Audience describes who the intended customer of this object or data is.
	// It can include multiple entries to indicate content useful for multiple audiences
	// (e.g., ["user", "assistant"]).
	Audience []Role `json:"audience,omitempty"`

	// Priority describes how important this data is for operating the server.
	// A value of 1 means "most important" (effectively required),
	// while 0 means "least important" (entirely optional).
	Priority float64 `json:"priority,omitempty"`
}

// Content is an interface for different content types.
// Implementations:
//   - [TextContent]
//   - [ImageContent]
//   - [AudioContent]
//   - [EmbeddedResource]
type Content interface {
	contentType() string
}

// TextContent represents text provided to or from an LLM.
type TextContent struct {
	Type string `json:"type"`

	// Text is the text content of the message.
	Text string `json:"text"`

	// Optional annotations for the client.
	Annotations Annotations `json:"annotations,omitempty"`
}

func (TextContent) contentType() string { return "text" }

// ImageContent represents an image provided to or from an LLM.
type ImageContent struct {
	Type string `json:"type"`

	// Data is the base64-encoded image data.
	Data string `json:"data"`

	// MimeType is the MIME type of the image. Different providers may support different image types.
	MimeType string `json:"mimeType"`

	// Optional annotations for the client.
	Annotations Annotations `json:"annotations,omitempty"`
}

func (ImageContent) contentType() string { return "image" }

// AudioContent represents audio provided to or from an LLM.
type AudioContent struct {
	Type string `json:"type"`

	// Data is the base64-encoded audio data.
	Data string `json:"data"`

	// MimeType is the MIME type of the audio. Different providers may support different audio types.
	MimeType string `json:"mimeType"`

	// Optional annotations for the client.
	Annotations Annotations `json:"annotations,omitempty"`
}

func (AudioContent) contentType() string { return "audio" }

// EmbeddedResource represents the contents of a resource embedded into a prompt or tool call result.
// It is up to the client how best to render embedded resources for the benefit
// of the LLM and/or the user.
type EmbeddedResource struct {
	Type string `json:"type"`

	// Resource is the contents of the resource being embedded.
	// After unmarshaling, use UnmarshalResourceContents to convert to a concrete ResourceContents type.
	Resource json.RawMessage `json:"resource"`

	// Optional annotations for the client.
	Annotations Annotations `json:"annotations,omitempty"`
}

func (EmbeddedResource) contentType() string { return "resource" }

// RequestParams represents common parameters for requests.
type RequestParams struct {
	Meta struct {
		ProgressToken ProgressToken `json:"progressToken,omitempty"`
	} `json:"_meta,omitempty"`
}

// NotificationParams represents common parameters for notifications.
type NotificationParams struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// Result represents a generic result structure.
type Result struct {
	Meta map[string]any `json:"_meta,omitempty"`
}

// JSONRPCResponse represents a successful (non-error) response to a request.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Result  json.RawMessage `json:"result"`
}

func (*JSONRPCResponse) isJSONRPCMessage() {}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// JSONRPCError represents a response to a request that indicates an error occurred.
type JSONRPCError struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      RequestID `json:"id"`
	Error   struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data,omitempty"`
	} `json:"error"`
}

func (*JSONRPCError) isJSONRPCMessage() {}

/* Empty result */

// EmptyResult represents a response that indicates success but carries no data.
type EmptyResult Result

/* Cancellation */

// CancelledParams contains parameters for cancelled notification.
type CancelledParams struct {
	NotificationParams
	RequestID RequestID `json:"requestId"`
	Reason    string    `json:"reason,omitempty"`
}

/* Initialization */

// InitializeParams contains parameters for initialize request.
type InitializeParams struct {
	RequestParams
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// InitializeResult is sent from the server after receiving an initialize request.
type InitializeResult struct {
	Result
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// InitializedParams contains parameters for initialized notification.
type InitializedParams struct {
	NotificationParams
}

// Implementation describes the name and version of an MCP implementation.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities represents capabilities a client may support.
type ClientCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
	Roots        struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"roots,omitempty"`
	Sampling struct{} `json:"sampling,omitempty"`
}

// ServerCapabilities represents capabilities that a server may support.
type ServerCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
	Logging      struct{}       `json:"logging,omitempty"`
	Completions  struct{}       `json:"completions,omitempty"`
	Prompts      struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"prompts,omitempty"`
	Resources struct {
		Subscribe   bool `json:"subscribe,omitempty"`
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"resources,omitempty"`
	Tools struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"tools,omitempty"`
}

/* Ping */

// PingParams contains parameters for ping request.
type PingParams struct {
	RequestParams
}

// NewPingRequest creates a new JSON-RPC ping request.
func NewPingRequest(id RequestID) JSONRPCRequest {
	return JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodPing,
		// No params needed for ping
	}
}

/* Progress notifications */

// ProgressParams contains parameters for progress notification.
type ProgressParams struct {
	NotificationParams

	// ProgressToken is the progress token which was given in the initial request,
	// used to associate this notification with the request that is proceeding.
	ProgressToken ProgressToken `json:"progressToken"`

	// Progress is the progress thus far. This should increase every time progress is made,
	// even if the total is unknown.
	Progress float64 `json:"progress"`

	// Total number of items to process (or total progress required), if known.
	Total float64 `json:"total,omitempty"`

	// Message is an optional message describing the current progress.
	Message string `json:"message,omitempty"`
}

/* Pagination */

// PaginatedParams represents parameters for a request that supports pagination.
type PaginatedParams struct {
	RequestParams

	// Cursor is an opaque token representing the current pagination position.
	// If provided, the server should return results starting after this cursor.
	Cursor Cursor `json:"cursor,omitempty"`
}

// PaginatedResult represents a result that supports pagination.
type PaginatedResult struct {
	Result

	// NextCursor is an opaque token representing the pagination position after the last returned result.
	// If present, there may be more results available.
	NextCursor Cursor `json:"nextCursor,omitempty"`
}

/* Resources */

// ListResourcesParams contains parameters for list resources request.
type ListResourcesParams struct {
	PaginatedParams
}

// ListResourcesResult represents the response to a resources/list request.
type ListResourcesResult struct {
	PaginatedResult
	Resources []Resource `json:"resources"`
}

// ListResourceTemplatesParams contains parameters for list resource templates request.
type ListResourceTemplatesParams struct {
	PaginatedParams
}

// ListResourceTemplatesResult represents the response to a resources/templates/list request.
type ListResourceTemplatesResult struct {
	PaginatedResult
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// ReadResourceParams contains parameters for read resource request.
type ReadResourceParams struct {
	RequestParams
	URI string `json:"uri"`
}

// ReadResourceResult represents the response to a resources/read request.
type ReadResourceResult struct {
	Result

	// Contents is a list of resource contents.
	// After unmarshaling, use UnmarshalResourceContents on each element to convert to concrete ResourceContents types.
	Contents []json.RawMessage `json:"contents"`
}

// ResourceListChangedParams contains parameters for resource list changed notification.
type ResourceListChangedParams struct {
	NotificationParams
}

// SubscribeParams contains parameters for subscribe request.
type SubscribeParams struct {
	RequestParams
	URI string `json:"uri"`
}

// UnsubscribeParams contains parameters for unsubscribe request.
type UnsubscribeParams struct {
	RequestParams
	URI string `json:"uri"`
}

// ResourceUpdatedParams contains parameters for resource updated notification.
type ResourceUpdatedParams struct {
	NotificationParams
	URI string `json:"uri"`
}

// Resource represents a known resource that the server is capable of reading.
type Resource struct {
	// URI is the URI of this resource.
	URI string `json:"uri"`

	// Name is a human-readable name for this resource.
	// This can be used by clients to populate UI elements.
	Name string `json:"name"`

	// Description of what this resource represents.
	// This can be used by clients to improve the LLM's understanding of available resources.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`

	// MimeType of this resource, if known.
	MimeType string `json:"mimeType,omitempty"`

	// Optional annotations for the client.
	Annotations Annotations `json:"annotations,omitempty"`

	// Size of the raw resource content, in bytes (i.e., before base64 encoding or any tokenization), if known.
	// This can be used by hosts to display file sizes and estimate context window usage.
	Size int64 `json:"size,omitempty"`
}

// ResourceTemplate represents a template description for resources available on the server.
type ResourceTemplate struct {
	// URITemplate is a URI template (according to RFC 6570) that can be used to construct resource URIs.
	URITemplate string `json:"uriTemplate"`

	// Name is a human-readable name for the type of resource this template refers to.
	// This can be used by clients to populate UI elements.
	Name string `json:"name"`

	// Description of what this template is for.
	// This can be used by clients to improve the LLM's understanding of available resources.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`

	// MimeType for all resources that match this template. This should only be included if
	// all resources matching this template have the same type.
	MimeType string `json:"mimeType,omitempty"`

	// Optional annotations for the client.
	Annotations Annotations `json:"annotations,omitempty"`
}

// ResourceContents is an interface for the contents of a specific resource or sub-resource.
// Implementations:
//   - [TextResourceContents]
//   - [BlobResourceContents]
type ResourceContents interface {
	GetURI() string
	GetMimeType() string
}

// BaseResourceContents contains common fields for resource contents.
type BaseResourceContents struct {
	// URI of this resource.
	URI string `json:"uri"`

	// MimeType of this resource, if known.
	MimeType string `json:"mimeType,omitempty"`
}

func (r *BaseResourceContents) GetURI() string      { return r.URI }
func (r *BaseResourceContents) GetMimeType() string { return r.MimeType }

// TextResourceContents represents the text contents of a resource.
type TextResourceContents struct {
	BaseResourceContents

	// Text is the text of the item. This must only be set if the item can
	// actually be represented as text (not binary data).
	Text string `json:"text"`
}

// BlobResourceContents represents the binary contents of a resource.
type BlobResourceContents struct {
	BaseResourceContents

	// Blob is a base64-encoded string representing the binary data of the item.
	Blob string `json:"blob"`
}

/* Prompts */

// ListPromptsParams contains parameters for list prompts request.
type ListPromptsParams struct {
	PaginatedParams
}

// ListPromptsResult represents the response to a prompts/list request.
type ListPromptsResult struct {
	PaginatedResult
	Prompts []Prompt `json:"prompts"`
}

// GetPromptParams contains parameters for get prompt request.
type GetPromptParams struct {
	RequestParams
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult represents the response to a prompts/get request.
type GetPromptResult struct {
	Result
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// Prompt represents a prompt or prompt template that the server offers.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes an argument that a prompt can accept.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptMessage describes a message returned as part of a prompt.
type PromptMessage struct {
	Role Role `json:"role"`

	// Content represents the message content.
	// After unmarshaling, use UnmarshalContent to convert to a concrete Content type.
	Content json.RawMessage `json:"content"`
}

// PromptListChangedParams contains parameters for prompt list changed notification.
type PromptListChangedParams struct {
	NotificationParams
}

/* Tools */

// ListToolsParams contains parameters for list tools request.
type ListToolsParams struct {
	PaginatedParams
}

// ListToolsResult represents the response to a tools/list request.
type ListToolsResult struct {
	PaginatedResult
	Tools []Tool `json:"tools"`
}

// CallToolParams contains parameters for call tool request.
type CallToolParams struct {
	RequestParams
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// CallToolResult represents the response to a tools/call request.
//
// Any errors that originate from the tool SHOULD be reported inside the result
// object, with `isError` set to true, _not_ as an MCP protocol-level error
// response. Otherwise, the LLM would not be able to see that an error occurred
// and self-correct.
//
// However, any errors in _finding_ the tool, an error indicating that the
// server does not support tool calls, or any other exceptional conditions,
// should be reported as an MCP error response.
type CallToolResult struct {
	Result

	// Content represents the content returned by the tool.
	// After unmarshaling, use UnmarshalContent on each element to convert to concrete Content types.
	Content []json.RawMessage `json:"content"`

	// IsError indicates whether the tool call ended in an error.
	// If not set, this is assumed to be false (the call was successful).
	IsError bool `json:"isError,omitempty"`
}

// ToolListChangedParams contains parameters for tool list changed notification.
type ToolListChangedParams struct {
	NotificationParams
}

// ToolAnnotations represents additional properties describing a Tool to clients.
//
// NOTE: all properties in ToolAnnotations are *hints*.
// They are not guaranteed to provide a faithful description of
// tool behavior (including descriptive properties like `title`).
//
// Clients should never make tool use decisions based on ToolAnnotations
// received from untrusted servers.
type ToolAnnotations struct {
	// Title is a human-readable title for the tool.
	Title string `json:"title,omitempty"`

	// ReadOnlyHint: If true, the tool does not modify its environment.
	// Default: false
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`

	// DestructiveHint: If true, the tool may perform destructive updates to its environment.
	// If false, the tool performs only additive updates.
	// (This property is meaningful only when ReadOnlyHint == false)
	// Default: true
	DestructiveHint *bool `json:"destructiveHint,omitempty"`

	// IdempotentHint: If true, calling the tool repeatedly with the same arguments
	// will have no additional effect on the its environment.
	// (This property is meaningful only when ReadOnlyHint == false)
	// Default: false
	IdempotentHint bool `json:"idempotentHint,omitempty"`

	// OpenWorldHint: If true, this tool may interact with an "open world" of external
	// entities. If false, the tool's domain of interaction is closed.
	// For example, the world of a web search tool is open, whereas that
	// of a memory tool is not.
	// Default: true
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}

// Tool represents a definition for a tool the client can call.
type Tool struct {
	// Name of the tool.
	Name string `json:"name"`

	// Description is a human-readable description of the tool.
	// This can be used by clients to improve the LLM's understanding of available tools.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`

	// InputSchema is a JSON Schema object defining the expected parameters for the tool.
	InputSchema ToolInputSchema `json:"inputSchema"`

	// Optional additional tool information.
	Annotations ToolAnnotations `json:"annotations,omitempty"`
}

// ToolInputSchema represents the JSON Schema for tool input parameters.
type ToolInputSchema struct {
	Type       string                     `json:"type"`
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
	Required   []string                   `json:"required,omitempty"`
}

/* Logging */

// LoggingLevel represents the severity of a log message.
// These map to syslog message severities, as specified in RFC-5424:
// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.1
type LoggingLevel string

// Constants for LoggingLevel type
const (
	LogLevelDebug     LoggingLevel = "debug"
	LogLevelInfo      LoggingLevel = "info"
	LogLevelNotice    LoggingLevel = "notice"
	LogLevelWarning   LoggingLevel = "warning"
	LogLevelError     LoggingLevel = "error"
	LogLevelCritical  LoggingLevel = "critical"
	LogLevelAlert     LoggingLevel = "alert"
	LogLevelEmergency LoggingLevel = "emergency"
)

// SetLevelParams contains parameters for set level request.
type SetLevelParams struct {
	RequestParams

	// Level of logging that the client wants to receive from the server.
	// The server should send all logs at this level and higher (i.e., more severe)
	// to the client as notifications/message.
	Level LoggingLevel `json:"level"`
}

// LoggingMessageParams contains parameters for logging message notification.
type LoggingMessageParams struct {
	NotificationParams

	// Level represents the severity of this log message.
	Level LoggingLevel `json:"level"`

	// Logger is an optional name of the logger issuing this message.
	Logger string `json:"logger,omitempty"`

	// Data to be logged, such as a string message or an object.
	// Any JSON serializable type is allowed here.
	Data json.RawMessage `json:"data"`
}

/* Sampling */

// ModelPreferences represents the server's preferences for model selection, requested of the client during sampling.
//
// Because LLMs can vary along multiple dimensions, choosing the "best" model is
// rarely straightforward. Different models excel in different areas—some are
// faster but less capable, others are more capable but more expensive, and so
// on. This interface allows servers to express their priorities across multiple
// dimensions to help clients make an appropriate selection for their use case.
//
// These preferences are always advisory. The client MAY ignore them. It is also
// up to the client to decide how to interpret these preferences and how to
// balance them against other considerations.
type ModelPreferences struct {
	// Hints are optional hints to use for model selection.
	//
	// If multiple hints are specified, the client MUST evaluate them in order
	// (such that the first match is taken).
	//
	// The client SHOULD prioritize these hints over the numeric priorities, but
	// MAY still use the priorities to select from ambiguous matches.
	Hints []ModelHint `json:"hints,omitempty"`

	// CostPriority indicates how much to prioritize cost when selecting a model.
	// A value of 0 means cost is not important, while a value of 1 means cost is the most important factor.
	CostPriority float64 `json:"costPriority,omitempty"`

	// SpeedPriority indicates how much to prioritize sampling speed (latency) when selecting a model.
	// A value of 0 means speed is not important, while a value of 1 means speed is the most important factor.
	SpeedPriority float64 `json:"speedPriority,omitempty"`

	// IntelligencePriority indicates how much to prioritize intelligence and capabilities when selecting a model.
	// A value of 0 means intelligence is not important, while a value of 1 means intelligence is the most important factor.
	IntelligencePriority float64 `json:"intelligencePriority,omitempty"`
}

// ModelHint represents hints to use for model selection.
//
// Fields not declared here are currently left unspecified by the spec and are up
// to the client to interpret.
type ModelHint struct {
	// Name is a hint for a model name.
	//
	// The client SHOULD treat this as a substring of a model name; for example:
	//  - "claude-3-5-sonnet" should match "claude-3-5-sonnet-20241022"
	//  - "sonnet" should match "claude-3-5-sonnet-20241022", "claude-3-sonnet-20240229", etc.
	//  - "claude" should match any Claude model
	//
	// The client MAY also map the string to a different provider's model name or a
	// different model family, as long as it fills a similar niche; for example:
	//  - "gemini-1.5-flash" could match "claude-3-haiku-20240307"
	Name string `json:"name,omitempty"`
}

// CreateMessageParams contains parameters for create message request.
type CreateMessageParams struct {
	RequestParams

	// Messages to be sent to the model
	Messages []SamplingMessage `json:"messages"`

	// ModelPreferences indicates the server's preferences for which model to select.
	// The client MAY ignore these preferences.
	ModelPreferences ModelPreferences `json:"modelPreferences,omitempty"`

	// SystemPrompt is an optional system prompt the server wants to use for sampling.
	// The client MAY modify or omit this prompt.
	SystemPrompt string `json:"systemPrompt,omitempty"`

	// IncludeContext is a request to include context from one or more MCP servers
	// (including the caller), to be attached to the prompt. The client MAY ignore this request.
	IncludeContext string `json:"includeContext,omitempty"`

	// Temperature to use when sampling
	Temperature float64 `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens to sample, as requested by the server.
	// The client MAY choose to sample fewer tokens than requested.
	MaxTokens int `json:"maxTokens,omitempty"`

	// StopSequences are sequences that will cause sampling to stop when encountered
	StopSequences []string `json:"stopSequences,omitempty"`

	// Metadata is optional metadata to pass through to the LLM provider.
	// The format of this metadata is provider-specific.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CreateMessageResult represents the response to a sampling/create_message request.
// The client should inform the user before returning the sampled message, to allow
// them to inspect the response (human in the loop) and decide whether to allow the
// server to see it.
type CreateMessageResult struct {
	Result
	SamplingMessage

	// Model is the name of the model that generated the message.
	Model string `json:"model"`

	// StopReason is the reason why sampling stopped, if known.
	StopReason string `json:"stopReason,omitempty"`
}

// SamplingMessage represents a message issued to or received from an LLM API.
type SamplingMessage struct {
	Role Role `json:"role"`

	// Content represents the message content.
	// After unmarshaling, use UnmarshalContent to convert to a concrete Content type.
	Content json.RawMessage `json:"content"`
}

/* Autocomplete */

// CompleteParams contains parameters for complete request.
type CompleteParams struct {
	RequestParams

	// Ref is a reference to a prompt or resource.
	// After unmarshaling, use UnmarshalReference to convert to a concrete Reference type.
	Ref json.RawMessage `json:"ref"`

	// Argument contains the argument's information
	Argument struct {
		// Name of the argument
		Name string `json:"name"`

		// Value of the argument to use for completion matching
		Value string `json:"value"`
	} `json:"argument"`
}

// CompleteResult represents the response to a completion/complete request.
type CompleteResult struct {
	Result
	Completion struct {
		// Values is an array of completion values. Must not exceed 100 items.
		Values []string `json:"values"`

		// Total number of completion options available.
		// This can exceed the number of values actually sent in the response.
		Total int `json:"total,omitempty"`

		// HasMore indicates whether there are additional completion options
		// beyond those provided in the current response, even if the exact total is unknown.
		HasMore bool `json:"hasMore,omitempty"`
	} `json:"completion"`
}

// Reference is an interface for different reference types.
// Implementations:
//   - [ResourceReference]
//   - [PromptReference]
type Reference interface {
	referenceType() string
}

// ResourceReference represents a reference to a resource or resource template.
type ResourceReference struct {
	// Type of reference (always "ref/resource")
	Type string `json:"type"`

	// URI or URI template of the resource
	URI string `json:"uri"`
}

func (ResourceReference) referenceType() string { return "ref/resource" }

// PromptReference identifies a prompt.
type PromptReference struct {
	// Type of reference (always "ref/prompt")
	Type string `json:"type"`

	// Name of the prompt or prompt template
	Name string `json:"name"`
}

func (PromptReference) referenceType() string { return "ref/prompt" }

/* Roots */

// Root represents a root directory or file that the server can operate on.
type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

// ListRootsParams contains parameters for list roots request.
type ListRootsParams struct {
	RequestParams
}

// ListRootsResult represents the response to a roots/list request.
type ListRootsResult struct {
	Result
	Roots []Root `json:"roots"`
}

// RootsListChangedParams contains parameters for roots list changed notification.
type RootsListChangedParams struct {
	NotificationParams
}

/* Method Constants */

// Methods defines constants for all RPC methods
const (
	MethodInitialize             = "initialize"
	MethodPing                   = "ping"
	MethodResourcesList          = "resources/list"
	MethodResourcesTemplatesList = "resources/templates/list"
	MethodResourcesRead          = "resources/read"
	MethodResourcesSubscribe     = "resources/subscribe"
	MethodResourcesUnsubscribe   = "resources/unsubscribe"
	MethodPromptsList            = "prompts/list"
	MethodPromptsGet             = "prompts/get"
	MethodToolsList              = "tools/list"
	MethodToolsCall              = "tools/call"
	MethodLoggingSetLevel        = "logging/setLevel"
	MethodSamplingCreateMessage  = "sampling/createMessage"
	MethodCompletionComplete     = "completion/complete"
	MethodRootsList              = "roots/list"

	// Notification methods
	MethodNotificationInitialized          = "notifications/initialized"
	MethodNotificationCancelled            = "notifications/cancelled"
	MethodNotificationProgress             = "notifications/progress"
	MethodNotificationMessage              = "notifications/message"
	MethodNotificationResourcesUpdated     = "notifications/resources/updated"
	MethodNotificationResourcesListChanged = "notifications/resources/list_changed"
	MethodNotificationPromptsListChanged   = "notifications/prompts/list_changed"
	MethodNotificationToolsListChanged     = "notifications/tools/list_changed"
	MethodNotificationRootsListChanged     = "notifications/roots/list_changed"
)

/* Message Type Collections */

// ClientRequest represents all possible request types from the client.
// Implementations:
//   - [JSONRPCRequest] (for ping)
//   - [InitializeParams]
//   - [CompleteParams]
//   - [SetLevelParams]
//   - [GetPromptParams]
//   - [ListPromptsParams]
//   - [ListResourcesParams]
//   - [ListResourceTemplatesParams]
//   - [ReadResourceParams]
//   - [SubscribeParams]
//   - [UnsubscribeParams]
//   - [CallToolParams]
//   - [ListToolsParams]
type ClientRequest interface {
	isClientRequest()
}

// ClientNotification represents all possible notification types from the client.
// Implementations:
//   - [CancelledParams]
//   - [ProgressParams]
//   - [InitializedParams]
//   - [RootsListChangedParams]
type ClientNotification interface {
	isClientNotification()
}

// ClientResult represents all possible result types from the client.
// Implementations:
//   - [EmptyResult]
//   - [CreateMessageResult]
//   - [ListRootsResult]
type ClientResult interface {
	isClientResult()
}

// ServerRequest represents all possible request types from the server.
// Implementations:
//   - [JSONRPCRequest] (for ping)
//   - [CreateMessageParams]
//   - [ListRootsParams]
type ServerRequest interface {
	isServerRequest()
}

// ServerNotification represents all possible notification types from the server.
// Implementations:
//   - [CancelledParams]
//   - [ProgressParams]
//   - [LoggingMessageParams]
//   - [ResourceUpdatedParams]
//   - [ResourceListChangedParams]
//   - [ToolListChangedParams]
//   - [PromptListChangedParams]
type ServerNotification interface {
	isServerNotification()
}

// ServerResult represents all possible result types from the server.
// Implementations:
//   - [EmptyResult]
//   - [InitializeResult]
//   - [CompleteResult]
//   - [GetPromptResult]
//   - [ListPromptsResult]
//   - [ListResourceTemplatesResult]
//   - [ListResourcesResult]
//   - [ReadResourceResult]
//   - [CallToolResult]
//   - [ListToolsResult]
type ServerResult interface {
	isServerResult()
}

// Define which types implement the client request interface
func (JSONRPCRequest) isClientRequest()              {} // Use JSONRPCRequest with MethodPing for pings
func (InitializeParams) isClientRequest()            {}
func (CompleteParams) isClientRequest()              {}
func (SetLevelParams) isClientRequest()              {}
func (GetPromptParams) isClientRequest()             {}
func (ListPromptsParams) isClientRequest()           {}
func (ListResourcesParams) isClientRequest()         {}
func (ListResourceTemplatesParams) isClientRequest() {}
func (ReadResourceParams) isClientRequest()          {}
func (SubscribeParams) isClientRequest()             {}
func (UnsubscribeParams) isClientRequest()           {}
func (CallToolParams) isClientRequest()              {}
func (ListToolsParams) isClientRequest()             {}

// Define which types implement the client notification interface
func (CancelledParams) isClientNotification()        {}
func (ProgressParams) isClientNotification()         {}
func (InitializedParams) isClientNotification()      {}
func (RootsListChangedParams) isClientNotification() {}

// Define which types implement the client result interface
func (EmptyResult) isClientResult()         {}
func (CreateMessageResult) isClientResult() {}
func (ListRootsResult) isClientResult()     {}

// Define which types implement the server request interface
func (JSONRPCRequest) isServerRequest()      {} // Use JSONRPCRequest with MethodPing for pings
func (CreateMessageParams) isServerRequest() {}
func (ListRootsParams) isServerRequest()     {}

// Define which types implement the server notification interface
func (CancelledParams) isServerNotification()           {}
func (ProgressParams) isServerNotification()            {}
func (LoggingMessageParams) isServerNotification()      {}
func (ResourceUpdatedParams) isServerNotification()     {}
func (ResourceListChangedParams) isServerNotification() {}
func (ToolListChangedParams) isServerNotification()     {}
func (PromptListChangedParams) isServerNotification()   {}

// Define which types implement the server result interface
func (EmptyResult) isServerResult()                 {}
func (InitializeResult) isServerResult()            {}
func (CompleteResult) isServerResult()              {}
func (GetPromptResult) isServerResult()             {}
func (ListPromptsResult) isServerResult()           {}
func (ListResourceTemplatesResult) isServerResult() {}
func (ListResourcesResult) isServerResult()         {}
func (ReadResourceResult) isServerResult()          {}
func (CallToolResult) isServerResult()              {}
func (ListToolsResult) isServerResult()             {}
