package draft

import base "github.com/tmc/mcp/modelcontextprotocol"

const LATEST_PROTOCOL_VERSION = "DRAFT-2025-v2"
const JSONRPC_VERSION = base.JSONRPC_VERSION
const (
	ParseErrorCode     = base.ParseErrorCode
	InvalidRequestCode = base.InvalidRequestCode
	MethodNotFoundCode = base.MethodNotFoundCode
	InvalidParamsCode  = base.InvalidParamsCode
	InternalErrorCode  = base.InternalErrorCode
)

type Role = base.Role

const (
	RoleUser      = base.RoleUser
	RoleAssistant = base.RoleAssistant
)

type LoggingLevel = base.LoggingLevel

const (
	LogLevelDebug     = base.LogLevelDebug
	LogLevelInfo      = base.LogLevelInfo
	LogLevelNotice    = base.LogLevelNotice
	LogLevelWarning   = base.LogLevelWarning
	LogLevelError     = base.LogLevelError
	LogLevelCritical  = base.LogLevelCritical
	LogLevelAlert     = base.LogLevelAlert
	LogLevelEmergency = base.LogLevelEmergency
)
const (
	MethodInitialize                       = base.MethodInitialize
	MethodPing                             = base.MethodPing
	MethodCompletionComplete               = base.MethodCompletionComplete
	MethodLoggingSetLevel                  = base.MethodLoggingSetLevel
	MethodPromptsGet                       = base.MethodPromptsGet
	MethodPromptsList                      = base.MethodPromptsList
	MethodResourcesList                    = base.MethodResourcesList
	MethodResourcesTemplatesList           = base.MethodResourcesTemplatesList
	MethodResourcesRead                    = base.MethodResourcesRead
	MethodResourcesSubscribe               = base.MethodResourcesSubscribe
	MethodResourcesUnsubscribe             = base.MethodResourcesUnsubscribe
	MethodToolsCall                        = base.MethodToolsCall
	MethodToolsList                        = base.MethodToolsList
	MethodSamplingCreateMessage            = base.MethodSamplingCreateMessage
	MethodRootsList                        = base.MethodRootsList
	MethodNotificationCancelled            = base.MethodNotificationCancelled
	MethodNotificationInitialized          = base.MethodNotificationInitialized
	MethodNotificationProgress             = base.MethodNotificationProgress
	MethodNotificationMessage              = base.MethodNotificationMessage
	MethodNotificationElicitationComplete  = base.MethodNotificationElicitationComplete
	MethodNotificationResourcesUpdated     = base.MethodNotificationResourcesUpdated
	MethodNotificationResourcesListChanged = base.MethodNotificationResourcesListChanged
	MethodNotificationToolsListChanged     = base.MethodNotificationToolsListChanged
	MethodNotificationPromptsListChanged   = base.MethodNotificationPromptsListChanged
	MethodNotificationRootsListChanged     = base.MethodNotificationRootsListChanged
)
