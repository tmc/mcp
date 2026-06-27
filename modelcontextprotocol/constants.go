package modelcontextprotocol

// LATEST_PROTOCOL_VERSION is the current version of the Model Context Protocol.
// It matches the canonical constant in the root mcp package so the two type
// packages negotiate the same version.
const LATEST_PROTOCOL_VERSION = "2025-11-25"

// JSONRPC_VERSION is the JSON-RPC version used ("2.0").
const JSONRPC_VERSION = "2.0"

// Standard JSON-RPC error codes
const (
	ParseErrorCode     = -32700
	InvalidRequestCode = -32600
	MethodNotFoundCode = -32601
	InvalidParamsCode  = -32602
	InternalErrorCode  = -32603
)

// Role represents the sender or recipient of messages.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// LoggingLevel defines the severity of a log message.
type LoggingLevel string

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

// ContentType constants
const (
	ContentTypeText     = "text"
	ContentTypeImage    = "image"
	ContentTypeAudio    = "audio"
	ContentTypeResource = "resource"
)

// ReferenceType constants
const (
	ReferenceTypePrompt   = "ref/prompt"
	ReferenceTypeResource = "ref/resource"
)

// Method names for MCP requests and notifications
const (
	MethodInitialize                       = "initialize"
	MethodPing                             = "ping"
	MethodCompletionComplete               = "completion/complete"
	MethodLoggingSetLevel                  = "logging/setLevel"
	MethodPromptsGet                       = "prompts/get"
	MethodPromptsList                      = "prompts/list"
	MethodResourcesList                    = "resources/list"
	MethodResourcesTemplatesList           = "resources/templates/list"
	MethodResourcesRead                    = "resources/read"
	MethodResourcesSubscribe               = "resources/subscribe"
	MethodResourcesUnsubscribe             = "resources/unsubscribe"
	MethodToolsCall                        = "tools/call"
	MethodToolsList                        = "tools/list"
	MethodSamplingCreateMessage            = "sampling/createMessage"
	MethodRootsList                        = "roots/list"
	MethodNotificationCancelled            = "notifications/cancelled"
	MethodNotificationInitialized          = "notifications/initialized"
	MethodNotificationProgress             = "notifications/progress"
	MethodNotificationMessage              = "notifications/message"
	MethodNotificationElicitationComplete  = "notifications/elicitation/complete"
	MethodNotificationResourcesUpdated     = "notifications/resources/updated"
	MethodNotificationResourcesListChanged = "notifications/resources/list_changed"
	MethodNotificationToolsListChanged     = "notifications/tools/list_changed"
	MethodNotificationPromptsListChanged   = "notifications/prompts/list_changed"
	MethodNotificationRootsListChanged     = "notifications/roots/list_changed"
)
