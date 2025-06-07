package draft

import (
	base "github.com/tmc/mcp/modelcontextprotocol"
)

// Aliased types
type JSONRPCMessage = base.JSONRPCMessage
type JSONRPCRequest = base.JSONRPCRequest
type JSONRPCNotification = base.JSONRPCNotification
type JSONRPCBatchRequest = base.JSONRPCBatchRequest
type JSONRPCResponse = base.JSONRPCResponse
type ErrorObject = base.ErrorObject
type JSONRPCBatchResponse = base.JSONRPCBatchResponse
type RequestID = base.RequestID
type ProgressToken = base.ProgressToken
type Cursor = base.Cursor
type Annotations = base.Annotations
type Content = base.Content
type TextContent = base.TextContent
type ImageContent = base.ImageContent
type AudioContent = base.AudioContent
type EmbeddedResource = base.EmbeddedResource
type ResourceContents = base.ResourceContents
type BaseResourceContents = base.BaseResourceContents
type TextResourceContents = base.TextResourceContents
type BlobResourceContents = base.BlobResourceContents
type Reference = base.Reference
type PromptReference = base.PromptReference
type ResourceReference = base.ResourceReference
type RequestMeta = base.RequestMeta
type EmptyResult = base.EmptyResult
type Implementation = base.Implementation
type ClientCapabilities = base.ClientCapabilities
type ServerCapabilities = base.ServerCapabilities
type CancelledNotificationParams = base.CancelledNotificationParams
type InitializeRequestParams = base.InitializeRequestParams
type InitializeResult = base.InitializeResult
type InitializedNotificationParams = base.InitializedNotificationParams
type PingRequestParams = base.PingRequestParams
type ProgressNotificationParams = base.ProgressNotificationParams
type ListResourcesRequestParams = base.ListResourcesRequestParams
type ListResourcesResult = base.ListResourcesResult
type Resource = base.Resource
type ListResourceTemplatesRequestParams = base.ListResourceTemplatesRequestParams
type ListResourceTemplatesResult = base.ListResourceTemplatesResult
type ResourceTemplate = base.ResourceTemplate
type ReadResourceRequestParams = base.ReadResourceRequestParams
type ReadResourceResult = base.ReadResourceResult
type ResourceListChangedNotificationParams = base.ResourceListChangedNotificationParams
type SubscribeRequestParams = base.SubscribeRequestParams
type UnsubscribeRequestParams = base.UnsubscribeRequestParams
type ResourceUpdatedNotificationParams = base.ResourceUpdatedNotificationParams
type ListPromptsRequestParams = base.ListPromptsRequestParams
type ListPromptsResult = base.ListPromptsResult
type Prompt = base.Prompt
type PromptArgument = base.PromptArgument
type GetPromptRequestParams = base.GetPromptRequestParams
type GetPromptResult = base.GetPromptResult
type PromptMessage = base.PromptMessage
type PromptListChangedNotificationParams = base.PromptListChangedNotificationParams
type ListToolsRequestParams = base.ListToolsRequestParams
type CallToolRequestParams = base.CallToolRequestParams
type ToolListChangedNotificationParams = base.ToolListChangedNotificationParams
type ToolAnnotations = base.ToolAnnotations
type ToolSchema = base.ToolSchema
type SetLevelRequestParams = base.SetLevelRequestParams
type LoggingMessageNotificationParams = base.LoggingMessageNotificationParams
type ModelPreferences = base.ModelPreferences
type ModelHint = base.ModelHint
type CreateMessageRequestParams = base.CreateMessageRequestParams
type CreateMessageResult = base.CreateMessageResult
type SamplingMessage = base.SamplingMessage
type CompleteRequestParams = base.CompleteRequestParams
type CompleteResult = base.CompleteResult
type Root = base.Root
type ListRootsRequestParams = base.ListRootsRequestParams
type ListRootsResult = base.ListRootsResult
type RootsListChangedNotificationParams = base.RootsListChangedNotificationParams

// Grouping interfaces aliased
type ClientRequest = base.ClientRequest
type ClientNotification = base.ClientNotification
type ClientResult = base.ClientResult
type ServerRequest = base.ServerRequest
type ServerNotification = base.ServerNotification
type ServerResult = base.ServerResult

// --- Draft Specific Redefinitions/Extensions ---
type Tool struct {
	Name         string           `json:"name"`
	Description  *string          `json:"description,omitempty"`
	InputSchema  ToolSchema       `json:"inputSchema"`
	OutputSchema *ToolSchema      `json:"outputSchema,omitempty"`
	Annotations  *ToolAnnotations `json:"annotations,omitempty"`
}
type ListToolsResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`
	Tools      []Tool         `json:"tools"`
	NextCursor *base.Cursor   `json:"nextCursor,omitempty"`
}

//nolint:unused // Required by ServerResult interface
func (ListToolsResult) isServerResult() {} // Implements base.ServerResult

type ContentList []base.Content
type CallToolResult struct {
	Meta               map[string]any `json:"_meta,omitempty"`
	IsError            *bool          `json:"isError,omitempty"`
	StructuredContent  map[string]any `json:"-"`
	Content            *ContentList   `json:"-"`
	isStructuredResult bool
}

//nolint:unused // Required by ServerResult interface
func (CallToolResult) isServerResult() {} // Implements base.ServerResult
