// This is based on MCP schema version 2025-03-26.
package modelcontextprotocol

import (
	"encoding/json"

	"github.com/tmc/mcp/modelcontextprotocol"
)

// Constants for JSON-RPC and protocol versions
const (
	// LatestProtocolVersion is the current version of the Model Context Protocol
	LatestProtocolVersion = "DRAFT-2025-v2"

	// JSONRPCVersion is the JSON-RPC version used by the protocol (2.0)
	JSONRPCVersion = "2.0"
)

type (

	// JSONRPCMessage represents any valid JSON-RPC object that can be decoded off the wire,
	// or encoded to be sent.
	// Implementations:
	//   - [JSONRPCRequest]
	//   - [JSONRPCNotification]
	//   - [JSONRPCBatchRequest]
	//   - [JSONRPCBatchResponse]
	//   - [JSONRPCResponse]
	//   - [JSONRPCError]
	JSONRPCMessage = modelcontextprotocol.JSONRPCMessage

	// JSONRPCRequest represents a request that expects a response.
	JSONRPCRequest = modelcontextprotocol.JSONRPCRequest

	// JSONRPCNotification represents a notification which does not expect a response.
	JSONRPCNotification = modelcontextprotocol.JSONRPCNotification

	// JSONRPCBatchRequest represents a JSON-RPC batch request, as described in
	// https://www.jsonrpc.org/specification#batch.
	JSONRPCBatchRequest = modelcontextprotocol.JSONRPCBatchRequest

	// JSONRPCBatchResponse represents a JSON-RPC batch response, as described in
	// https://www.jsonrpc.org/specification#batch.
	JSONRPCBatchResponse = modelcontextprotocol.JSONRPCBatchResponse

	// RequestID is a Request identifier.
	// It can be either a string or a number value.
	RequestID = modelcontextprotocol.RequestID
)

var (

	// StringID creates a new string request identifier.
	StringID = modelcontextprotocol.StringID

	// Int64ID creates a new integer request identifier.
	Int64ID = modelcontextprotocol.Int64ID

	// Float64ID creates a new float64 request identifier.
	Float64ID = modelcontextprotocol.Float64ID
)

type (

	// ProgressToken is used to associate progress notifications with the original request.
	// It can be a string or a number.
	ProgressToken = modelcontextprotocol.ProgressToken

	// Cursor is an opaque token used to represent a cursor for pagination.
	Cursor = modelcontextprotocol.Cursor

	// Role represents the sender or recipient of messages and data in a conversation.
	Role = modelcontextprotocol.Role
)

// Constants for Role type
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

/* Content */

type (

	// Annotations represents optional annotations for the client.
	// The client can use annotations to inform how objects are used or displayed.
	Annotations = modelcontextprotocol.Annotations

	// Content is an interface for different content types.
	// Implementations:
	//   - [TextContent]
	//   - [ImageContent]
	//   - [AudioContent]
	//   - [EmbeddedResource]
	Content = modelcontextprotocol.Content

	// ContentList represents a list of content objects.
	ContentList []json.RawMessage

	// TextContent represents text provided to or from an LLM.
	TextContent = modelcontextprotocol.TextContent

	// ImageContent represents an image provided to or from an LLM.
	ImageContent = modelcontextprotocol.ImageContent

	// AudioContent represents audio provided to or from an LLM.
	AudioContent = modelcontextprotocol.AudioContent

	// EmbeddedResource represents the contents of a resource embedded into a prompt or tool call result.
	// It is up to the client how best to render embedded resources for the benefit
	// of the LLM and/or the user.
	EmbeddedResource = modelcontextprotocol.EmbeddedResource

	// RequestParams represents common parameters for requests.
	RequestParams = modelcontextprotocol.RequestParams

	// NotificationParams represents common parameters for notifications.
	NotificationParams = modelcontextprotocol.NotificationParams

	// Result represents a generic result structure.
	Result = modelcontextprotocol.Result

	// JSONRPCResponse represents a successful (non-error) response to a request.
	JSONRPCResponse = modelcontextprotocol.JSONRPCResponse
)

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

type (
	// JSONRPCError represents a response to a request that indicates an error occurred.
	JSONRPCError = modelcontextprotocol.JSONRPCError

	/* Empty result */

	// EmptyResult represents a response that indicates success but carries no data.
	EmptyResult = modelcontextprotocol.EmptyResult

	/* Cancellation */

	// CancelledParams contains parameters for cancelled notification.
	CancelledParams = modelcontextprotocol.CancelledParams

	/* Initialization */

	// InitializeParams contains parameters for initialize request.
	InitializeParams = modelcontextprotocol.InitializeParams

	// InitializeResult is sent from the server after receiving an initialize request.
	InitializeResult = modelcontextprotocol.InitializeResult

	// InitializedParams contains parameters for initialized notification.
	InitializedParams = modelcontextprotocol.InitializedParams

	// Implementation describes the name and version of an MCP implementation.
	Implementation = modelcontextprotocol.Implementation

	// ClientCapabilities represents capabilities a client may support.
	ClientCapabilities = modelcontextprotocol.ClientCapabilities

	// ServerCapabilities represents capabilities that a server may support.
	ServerCapabilities = modelcontextprotocol.ServerCapabilities

	/* Ping */

	// PingParams contains parameters for ping request.
	PingParams = modelcontextprotocol.PingParams

	/* Progress notifications */

	// ProgressParams contains parameters for progress notification.
	ProgressParams = modelcontextprotocol.ProgressParams

	/* Pagination */

	// PaginatedParams represents parameters for a request that supports pagination.
	PaginatedParams = modelcontextprotocol.PaginatedParams

	// PaginatedResult represents a result that supports pagination.
	PaginatedResult = modelcontextprotocol.PaginatedResult

	/* Resources */

	// ListResourcesParams contains parameters for list resources request.
	ListResourcesParams = modelcontextprotocol.ListResourcesParams

	// ListResourcesResult represents the response to a resources/list request.
	ListResourcesResult = modelcontextprotocol.ListResourcesResult

	// ListResourceTemplatesParams contains parameters for list resource templates request.
	ListResourceTemplatesParams = modelcontextprotocol.ListResourceTemplatesParams

	// ListResourceTemplatesResult represents the response to a resources/templates/list request.
	ListResourceTemplatesResult = modelcontextprotocol.ListResourceTemplatesResult

	// ReadResourceParams contains parameters for read resource request.
	ReadResourceParams = modelcontextprotocol.ReadResourceParams

	// ReadResourceResult represents the response to a resources/read request.
	ReadResourceResult = modelcontextprotocol.ReadResourceResult

	// ResourceListChangedParams contains parameters for resource list changed notification.
	ResourceListChangedParams = modelcontextprotocol.ResourceListChangedParams

	// SubscribeParams contains parameters for subscribe request.
	SubscribeParams = modelcontextprotocol.SubscribeParams

	// UnsubscribeParams contains parameters for unsubscribe request.
	UnsubscribeParams = modelcontextprotocol.UnsubscribeParams

	// ResourceUpdatedParams contains parameters for resource updated notification.
	ResourceUpdatedParams = modelcontextprotocol.ResourceUpdatedParams

	// Resource represents a known resource that the server is capable of reading.
	Resource = modelcontextprotocol.Resource

	// ResourceTemplate represents a template description for resources available on the server.
	ResourceTemplate = modelcontextprotocol.ResourceTemplate

	// ResourceContents is an interface for the contents of a specific resource or sub-resource.
	// Implementations:
	//   - [TextResourceContents]
	//   - [BlobResourceContents]
	ResourceContents = modelcontextprotocol.ResourceContents

	// BaseResourceContents contains common fields for resource contents.
	BaseResourceContents = modelcontextprotocol.BaseResourceContents

	// TextResourceContents represents the text contents of a resource.
	TextResourceContents = modelcontextprotocol.TextResourceContents

	// BlobResourceContents represents the binary contents of a resource.
	BlobResourceContents = modelcontextprotocol.BlobResourceContents

	/* Prompts */

	// ListPromptsParams contains parameters for list prompts request.
	ListPromptsParams = modelcontextprotocol.ListPromptsParams

	// ListPromptsResult represents the response to a prompts/list request.
	ListPromptsResult = modelcontextprotocol.ListPromptsResult

	// GetPromptParams contains parameters for get prompt request.
	GetPromptParams = modelcontextprotocol.GetPromptParams

	// GetPromptResult represents the response to a prompts/get request.
	GetPromptResult = modelcontextprotocol.GetPromptResult

	// Prompt represents a prompt or prompt template that the server offers.
	Prompt = modelcontextprotocol.Prompt

	// PromptArgument describes an argument that a prompt can accept.
	PromptArgument = modelcontextprotocol.PromptArgument

	// PromptMessage describes a message returned as part of a prompt.
	PromptMessage = modelcontextprotocol.PromptMessage

	// PromptListChangedParams contains parameters for prompt list changed notification.
	PromptListChangedParams = modelcontextprotocol.PromptListChangedParams

	/* Tools */

	// ListToolsParams contains parameters for list tools request.
	ListToolsParams = modelcontextprotocol.ListToolsParams

	// ListToolsResult represents the response to a tools/list request.
	ListToolsResult = modelcontextprotocol.ListToolsResult

	// CallToolParams contains parameters for call tool request.
	CallToolParams = modelcontextprotocol.CallToolParams
)

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
type CallToolResult interface {
	isCallToolResult()
	GetIsError() bool
	isServerResult()
}

// CallToolUnstructuredResult represents a tool result for tools that do not declare an outputSchema.
type CallToolUnstructuredResult struct {
	Result

	// Content represents the content returned by the tool.
	// After unmarshaling, use UnmarshalContent on each element to convert to concrete Content types.
	Content ContentList `json:"content"`

	// structuredContent must not be provided in an unstructured tool result
	StructuredContent any `json:"-"`

	// IsError indicates whether the tool call ended in an error.
	// If not set, this is assumed to be false (the call was successful).
	IsError bool `json:"isError,omitempty"`
}

func (CallToolUnstructuredResult) isCallToolResult()  {}
func (CallToolUnstructuredResult) isServerResult()    {}
func (r CallToolUnstructuredResult) GetIsError() bool { return r.IsError }

// CallToolStructuredResult represents a tool result for tools that declare an outputSchema.
type CallToolStructuredResult struct {
	Result

	// StructuredContent contains structured tool output.
	// If the Tool defines an outputSchema, this field MUST be present in the result,
	// and contain a JSON object that matches the schema.
	StructuredContent map[string]any `json:"structuredContent"`

	// Content represents optional content returned by the tool.
	// Tools should use this field to provide compatibility with older clients that do not support structured content.
	// Clients that support structured content should ignore this field.
	Content ContentList `json:"content,omitempty"`

	// IsError indicates whether the tool call ended in an error.
	// If not set, this is assumed to be false (the call was successful).
	IsError bool `json:"isError,omitempty"`
}

func (CallToolStructuredResult) isCallToolResult()  {}
func (CallToolStructuredResult) isServerResult()    {}
func (r CallToolStructuredResult) GetIsError() bool { return r.IsError }

type (

	// ToolListChangedParams contains parameters for tool list changed notification.
	ToolListChangedParams = modelcontextprotocol.ToolListChangedParams

	// ToolAnnotations represents additional properties describing a Tool to clients.
	//
	// NOTE: all properties in ToolAnnotations are *hints*.
	// They are not guaranteed to provide a faithful description of
	// tool behavior (including descriptive properties like `title`).
	//
	// Clients should never make tool use decisions based on ToolAnnotations
	// received from untrusted servers.
	ToolAnnotations = modelcontextprotocol.ToolAnnotations

	// Tool represents a definition for a tool the client can call.
	Tool struct {
		// Name of the tool.
		Name string `json:"name"`

		// Description is a human-readable description of the tool.
		// This can be used by clients to improve the LLM's understanding of available tools.
		// It can be thought of like a "hint" to the model.
		Description string `json:"description,omitempty"`

		// InputSchema is a JSON Schema object defining the expected parameters for the tool.
		InputSchema ToolInputSchema `json:"inputSchema"`

		// OutputSchema is an optional JSON Schema object defining the structure of the tool's output.
		// If set, a CallToolResult for this Tool MUST contain a structuredContent field
		// whose contents validate against this schema.
		// If not set, a CallToolResult for this Tool MUST contain a content field.
		OutputSchema json.RawMessage `json:"outputSchema,omitempty"`

		// Optional additional tool information.
		Annotations ToolAnnotations `json:"annotations,omitempty"`
	}

	// ToolInputSchema represents the JSON Schema for tool input parameters.
	ToolInputSchema = modelcontextprotocol.ToolInputSchema
)

/* Logging */

// LoggingLevel represents the severity of a log message.
// These map to syslog message severities, as specified in RFC-5424:
// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.1
type LoggingLevel = modelcontextprotocol.LoggingLevel

// Constants for LoggingLevel type
const (
	LogLevelDebug     = modelcontextprotocol.LogLevelDebug
	LogLevelInfo      = modelcontextprotocol.LogLevelInfo
	LogLevelNotice    = modelcontextprotocol.LogLevelNotice
	LogLevelWarning   = modelcontextprotocol.LogLevelWarning
	LogLevelError     = modelcontextprotocol.LogLevelError
	LogLevelCritical  = modelcontextprotocol.LogLevelCritical
	LogLevelAlert     = modelcontextprotocol.LogLevelAlert
	LogLevelEmergency = modelcontextprotocol.LogLevelEmergency
)

type (

	// SetLevelParams contains parameters for set level request.
	SetLevelParams = modelcontextprotocol.SetLevelParams

	// LoggingMessageParams contains parameters for logging message notification.
	LoggingMessageParams = modelcontextprotocol.LoggingMessageParams

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
	ModelPreferences = modelcontextprotocol.ModelPreferences

	// ModelHint represents hints to use for model selection.
	//
	// Fields not declared here are currently left unspecified by the spec and are up
	// to the client to interpret.
	ModelHint = modelcontextprotocol.ModelHint

	// CreateMessageParams contains parameters for create message request.
	CreateMessageParams = modelcontextprotocol.CreateMessageParams

	// CreateMessageResult represents the response to a sampling/create_message request.
	// The client should inform the user before returning the sampled message, to allow
	// them to inspect the response (human in the loop) and decide whether to allow the
	// server to see it.
	CreateMessageResult = modelcontextprotocol.CreateMessageResult

	// SamplingMessage represents a message issued to or received from an LLM API.
	SamplingMessage = modelcontextprotocol.SamplingMessage

	/* Autocomplete */

	// CompleteParams contains parameters for complete request.
	CompleteParams = modelcontextprotocol.CompleteParams

	// CompleteResult represents the response to a completion/complete request.
	CompleteResult = modelcontextprotocol.CompleteResult

	// Reference is an interface for different reference .
	// Implementations:
	//   - [ResourceReference]
	//   - [PromptReference]
	Reference = modelcontextprotocol.Reference

	// ResourceReference represents a reference to a resource or resource template.
	ResourceReference = modelcontextprotocol.ResourceReference

	// PromptReference identifies a prompt.
	PromptReference = modelcontextprotocol.PromptReference

	/* Roots */

	// Root represents a root directory or file that the server can operate on.
	Root = modelcontextprotocol.Root

	// ListRootsParams contains parameters for list roots request.
	ListRootsParams = modelcontextprotocol.ListRootsParams

	// ListRootsResult represents the response to a roots/list request.
	ListRootsResult = modelcontextprotocol.ListRootsResult

	// RootsListChangedParams contains parameters for roots list changed notification.
	RootsListChangedParams = modelcontextprotocol.RootsListChangedParams
)

/* Method Constants */

// Methods defines constants for all RPC methods
const (
	MethodInitialize             = modelcontextprotocol.MethodInitialize
	MethodPing                   = modelcontextprotocol.MethodPing
	MethodResourcesList          = modelcontextprotocol.MethodResourcesList
	MethodResourcesTemplatesList = modelcontextprotocol.MethodResourcesTemplatesList
	MethodResourcesRead          = modelcontextprotocol.MethodResourcesRead
	MethodResourcesSubscribe     = modelcontextprotocol.MethodResourcesSubscribe
	MethodResourcesUnsubscribe   = modelcontextprotocol.MethodResourcesUnsubscribe
	MethodPromptsList            = modelcontextprotocol.MethodPromptsList
	MethodPromptsGet             = modelcontextprotocol.MethodPromptsGet
	MethodToolsList              = modelcontextprotocol.MethodToolsList
	MethodToolsCall              = modelcontextprotocol.MethodToolsCall
	MethodLoggingSetLevel        = modelcontextprotocol.MethodLoggingSetLevel
	MethodSamplingCreateMessage  = modelcontextprotocol.MethodSamplingCreateMessage
	MethodCompletionComplete     = modelcontextprotocol.MethodCompletionComplete
	MethodRootsList              = modelcontextprotocol.MethodRootsList

	// Notification methods
	MethodNotificationInitialized          = modelcontextprotocol.MethodNotificationInitialized
	MethodNotificationCancelled            = modelcontextprotocol.MethodNotificationCancelled
	MethodNotificationProgress             = modelcontextprotocol.MethodNotificationProgress
	MethodNotificationMessage              = modelcontextprotocol.MethodNotificationMessage
	MethodNotificationResourcesUpdated     = modelcontextprotocol.MethodNotificationResourcesUpdated
	MethodNotificationResourcesListChanged = modelcontextprotocol.MethodNotificationResourcesListChanged
	MethodNotificationPromptsListChanged   = modelcontextprotocol.MethodNotificationPromptsListChanged
	MethodNotificationToolsListChanged     = modelcontextprotocol.MethodNotificationToolsListChanged
	MethodNotificationRootsListChanged     = modelcontextprotocol.MethodNotificationRootsListChanged
)

type (

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
	ClientRequest = modelcontextprotocol.ClientRequest

	// ClientNotification represents all possible notification types from the client.
	// Implementations:
	//   - [CancelledParams]
	//   - [ProgressParams]
	//   - [InitializedParams]
	//   - [RootsListChangedParams]
	ClientNotification = modelcontextprotocol.ClientNotification

	// ClientResult represents all possible result types from the client.
	// Implementations:
	//   - [EmptyResult]
	//   - [CreateMessageResult]
	//   - [ListRootsResult]
	ClientResult = modelcontextprotocol.ClientResult

	// ServerRequest represents all possible request types from the server.
	// Implementations:
	//   - [JSONRPCRequest] (for ping)
	//   - [CreateMessageParams]
	//   - [ListRootsParams]
	ServerRequest = modelcontextprotocol.ServerRequest

	// ServerNotification represents all possible notification types from the server.
	// Implementations:
	//   - [CancelledParams]
	//   - [ProgressParams]
	//   - [LoggingMessageParams]
	//   - [ResourceUpdatedParams]
	//   - [ResourceListChangedParams]
	//   - [ToolListChangedParams]
	//   - [PromptListChangedParams]
	ServerNotification = modelcontextprotocol.ServerNotification

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
	ServerResult = modelcontextprotocol.ServerResult
)
