package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Protocol constants
const (
	LATEST_PROTOCOL_VERSION = "2025-11-25"
	JSONRPC_VERSION         = "2.0"
)

// Common error values
var (
	ErrInvalidParams = errors.New("mcp: invalid parameters")
	ErrNotFound      = errors.New("mcp: not found")
	ErrUnsupported   = errors.New("mcp: operation or capability not supported")
	// ErrTransportClosed reports that an established transport can no longer be used.
	// Transport operations wrap this sentinel so callers can detect disconnects with
	// errors.Is without conflating them with protocol errors.
	ErrTransportClosed = errors.New("mcp: transport closed")
	ErrAlreadyExists   = errors.New("mcp: resource already exists")
	ErrMethodNotFound  = errors.New("mcp: method not found")
)

// ParameterError represents a parameter validation error with structured information
type ParameterError struct {
	Method    string `json:"method"`
	Parameter string `json:"parameter,omitempty"`
	Message   string `json:"message"`
	Cause     error  `json:"-"`
}

func (e *ParameterError) Error() string {
	if e.Parameter != "" {
		return fmt.Sprintf("mcp: invalid %s parameter for %s: %s", e.Parameter, e.Method, e.Message)
	}
	return fmt.Sprintf("mcp: invalid parameters for %s: %s", e.Method, e.Message)
}

func (e *ParameterError) Unwrap() error {
	return e.Cause
}

// NotFoundError represents a resource not found error with structured information
type NotFoundError struct {
	Type       string `json:"type"`       // "tool", "resource", "prompt", etc.
	Identifier string `json:"identifier"` // name, URI, etc.
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("mcp: %s '%s' not found", e.Type, e.Identifier)
}

// AlreadyExistsError represents a resource already exists error
type AlreadyExistsError struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("mcp: %s '%s' already exists", e.Type, e.Identifier)
}

// Role represents the sender or recipient of messages in MCP conversations.
// This is used to identify whether a message comes from a user or an assistant.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// MCPMethod represents the standard method names used in the MCP protocol.
// These constants define all the JSON-RPC method names that clients and servers
// use to communicate, including both request-response methods and notifications.
type MCPMethod string

const (
	MethodInitialize              MCPMethod = "initialize"
	MethodPing                    MCPMethod = "ping"
	MethodRootsList               MCPMethod = "roots/list"
	MethodCompletionComplete      MCPMethod = "completion/complete"
	MethodLoggingSetLevel         MCPMethod = "logging/setLevel"
	MethodSamplingCreateMessage   MCPMethod = "sampling/createMessage"
	MethodElicitationCreate       MCPMethod = "elicitation/create"
	MethodTasksList               MCPMethod = "tasks/list"
	MethodTasksGet                MCPMethod = "tasks/get"
	MethodTasksResult             MCPMethod = "tasks/result"
	MethodTasksCancel             MCPMethod = "tasks/cancel"
	MethodResourcesList           MCPMethod = "resources/list"
	MethodResourcesTemplatesList  MCPMethod = "resources/templates/list"
	MethodResourcesSubscribe      MCPMethod = "resources/subscribe"
	MethodResourcesUnsubscribe    MCPMethod = "resources/unsubscribe"
	MethodResourcesRead           MCPMethod = "resources/read"
	MethodPromptsList             MCPMethod = "prompts/list"
	MethodPromptsGet              MCPMethod = "prompts/get"
	MethodToolsList               MCPMethod = "tools/list"
	MethodToolsCall               MCPMethod = "tools/call"
	MethodNotificationCancelled   MCPMethod = "notifications/cancelled"
	MethodNotificationInitialized MCPMethod = "notifications/initialized"

	// Notification methods
	MethodProgress            MCPMethod = "notifications/progress"
	MethodLogging             MCPMethod = "notifications/message"
	MethodResourceUpdated     MCPMethod = "notifications/resources/updated"
	MethodResourceListChanged MCPMethod = "notifications/resources/list_changed"
	MethodPromptListChanged   MCPMethod = "notifications/prompts/list_changed"
	MethodToolListChanged     MCPMethod = "notifications/tools/list_changed"
	MethodRootsListChanged    MCPMethod = "notifications/roots/list_changed"
	MethodTasksStatus         MCPMethod = "notifications/tasks/status"
)

// JSONRPCNotification represents a notification message in the JSON-RPC protocol.
// Notifications are one-way messages that do not expect a response. They are used
// for events like progress updates, logging messages, and list change notifications.
type JSONRPCNotification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// JSONRPCRequest represents a request message in the JSON-RPC protocol.
// Requests expect a response and include an ID for correlation. This structure
// follows the JSON-RPC 2.0 specification.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a response message in the JSON-RPC protocol.
// Responses correlate to requests via the ID field and contain either a result
// or an error, but never both.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents an error in a JSON-RPC response.
// It follows the JSON-RPC 2.0 error object specification with code, message,
// and optional data fields for structured error information.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	if e == nil {
		return "mcp: nil JSON-RPC error"
	}
	if e.Data != nil {
		return e.Message + ": " + string(e.Data)
	}
	return e.Message
}

// Root represents a client-side filesystem root exposed to a server.
type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

// ListRootsRequest is sent by servers to retrieve the client's current roots.
type ListRootsRequest struct{}

// ListRootsResult contains the client's current roots.
type ListRootsResult struct {
	Roots []Root `json:"roots"`
}

// Implementation describes the name and version of an MCP client or server.
// This information is exchanged during the initialization handshake to identify
// the software and version being used on each side of the connection.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes the features supported by an MCP client.
// During initialization, clients advertise their capabilities to servers,
// enabling servers to optimize their behavior based on client features.
type ClientCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
	Roots        *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"roots,omitempty"`
	Sampling    *struct{} `json:"sampling,omitempty"`
	Elicitation *struct {
		Form bool `json:"form,omitempty"`
		URL  bool `json:"url,omitempty"`
	} `json:"elicitation,omitempty"`
	Tasks *struct {
		List     *struct{} `json:"list,omitempty"`
		Cancel   *struct{} `json:"cancel,omitempty"`
		Requests *struct {
			Sampling *struct {
				CreateMessage *struct{} `json:"createMessage,omitempty"`
			} `json:"sampling,omitempty"`
			Elicitation *struct {
				Create *struct{} `json:"create,omitempty"`
			} `json:"elicitation,omitempty"`
		} `json:"requests,omitempty"`
	} `json:"tasks,omitempty"`
}

// ServerCapabilities describes the features supported by an MCP server.
// Servers advertise their capabilities during initialization, informing clients
// about available features like tools, resources, prompts, and change notifications.
type ServerCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
	Logging      *struct{}      `json:"logging,omitempty"`
	Completions  *struct{}      `json:"completions,omitempty"`
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
	Tasks *struct {
		List     *struct{} `json:"list,omitempty"`
		Cancel   *struct{} `json:"cancel,omitempty"`
		Requests *struct {
			Tools *struct {
				Call *struct{} `json:"call,omitempty"`
			} `json:"tools,omitempty"`
		} `json:"requests,omitempty"`
	} `json:"tasks,omitempty"`
}

// InitializeRequest is the client's request to initialize the MCP connection.
// This is the first message sent by clients to establish protocol version,
// exchange implementation details, and negotiate capabilities.
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      Implementation     `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// InitializeResult is the server's response to an initialize request.
// It confirms the protocol version, provides server information and capabilities,
// and may include instructions for the client.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	Instructions    string             `json:"instructions,omitempty"`
}

// Content represents the various data types that can be included in MCP messages.
// This interface allows for polymorphic content handling, supporting text, images,
// and other media types within tool results, prompts, and resources.
type Content interface {
	content()
}

// TextContent represents plain text data within a message.
// This is the most common content type used for tool results,
// prompt messages, and resource contents.
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (TextContent) content() {}

// ImageContent represents image data within a message.
// Images can be embedded directly as binary data or referenced by URI,
// with optional MIME type specification for proper handling.
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

// Tool represents the definition for a tool that clients can call.
// Tools are server-provided functions that clients can invoke to perform
// operations like file system access, API calls, or data processing.
// The InputSchema provides JSON Schema validation for tool arguments.
type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"inputSchema,omitempty"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
}

// CallToolRequest is the client's request to call a specific tool.
// It specifies the tool name and provides arguments as JSON data
// that will be validated against the tool's InputSchema.
type CallToolRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallToolResult is the server's response to a tools/call request.
// It contains the tool's output as content blocks, an error flag,
// and optional metadata. Content can include text, images, or other media.
type CallToolResult struct {
	Content           []any `json:"content"`
	StructuredContent any   `json:"structuredContent,omitempty"`
	IsError           bool  `json:"isError,omitempty"`
	Meta              any   `json:"_meta,omitempty"`
}

// CompleteRequest describes a completion lookup for a prompt or resource reference.
type CompleteRequest struct {
	Ref      any `json:"ref"`
	Argument struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"argument"`
}

// CompleteResult contains server-provided completion candidates.
type CompleteResult struct {
	Completion struct {
		Values  []string `json:"values"`
		Total   *int     `json:"total,omitempty"`
		HasMore *bool    `json:"hasMore,omitempty"`
	} `json:"completion"`
}

// SetLevelRequest changes the logging level on a server that supports protocol logging.
type SetLevelRequest struct {
	Level LoggingLevel `json:"level"`
}

// LoggingMessageNotification contains structured logging notification data.
type LoggingMessageNotification struct {
	Level  LoggingLevel    `json:"level"`
	Logger string          `json:"logger,omitempty"`
	Data   json.RawMessage `json:"data"`
}

// TaskInfo describes the current state of a durable task.
type TaskInfo struct {
	TaskID        string `json:"taskId"`
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	LastUpdatedAt string `json:"lastUpdatedAt,omitempty"`
	TTL           *int64 `json:"ttl,omitempty"`
	PollInterval  *int64 `json:"pollInterval,omitempty"`
}

// ListTasksRequest requests the current set of tasks.
type ListTasksRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListTasksResult contains the current page of tasks.
type ListTasksResult struct {
	Tasks      []TaskInfo `json:"tasks"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// GetTaskRequest requests the current status of a task.
type GetTaskRequest struct {
	TaskID string `json:"taskId"`
}

// CancelTaskRequest requests cancellation of a task.
type CancelTaskRequest struct {
	TaskID string `json:"taskId"`
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
// Prompts are reusable message templates that can accept arguments
// to generate dynamic content for AI interactions.
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
	Role    Role `json:"role"`
	Content any  `json:"content"`
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

// Resource represents a known resource that the server can read.
// Resources are data sources like files, databases, or APIs that clients
// can access through the server. Each resource has a unique URI identifier.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
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

// SubscribeResourceRequest asks the server to send updates for a resource.
type SubscribeResourceRequest struct {
	URI string `json:"uri"`
}

// UnsubscribeResourceRequest cancels a resource subscription.
type UnsubscribeResourceRequest struct {
	URI string `json:"uri"`
}

// ResourceUpdatedNotificationParams identifies an updated resource.
type ResourceUpdatedNotificationParams struct {
	URI string `json:"uri"`
}

// ResourceContents represents the content of a specific resource.
// This interface allows for different content types (text, binary)
// while maintaining type safety and proper JSON serialization.
type ResourceContents interface {
	resourceContents()
}

// TextResourceContents represents text content for a resource.
// Used for resources that contain textual data like configuration files,
// source code, documentation, or any UTF-8 encoded content.
type TextResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

func (TextResourceContents) resourceContents() {}

// BlobResourceContents represents binary content for a resource.
// Used for resources containing binary data such as images, executables,
// or other non-text files. The content is base64 encoded for JSON transport.
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

// ResourceTemplate represents a template description for dynamic resources.
// Templates allow servers to advertise patterns of resources they can provide,
// such as file system paths with wildcards or parameterized API endpoints.
type ResourceTemplate struct {
	Template    string `json:"template"`
	Description string `json:"description,omitempty"`
}

// TransportInterface is a deprecated alias. Use Transport from transport.go instead.
// This comment preserves the line count.

// LoggingLevel represents the syslog severity level of a logging message.
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

// NotificationHandler handles MCP notifications.
// Notifications are one-way messages for events like progress updates,
// logging messages, and resource/tool/prompt list changes.
type NotificationHandler func(method string, params json.RawMessage) error

// ToolHandlerFunc defines the signature for functions handling tools/call requests.
type ToolHandlerFunc func(ctx context.Context, request CallToolRequest) (*CallToolResult, error)

// GetPromptHandlerFunc defines the signature for functions handling prompts/get requests.
type GetPromptHandlerFunc func(ctx context.Context, request GetPromptRequest) (*GetPromptResult, error)

// ReadResourceHandlerFunc defines the signature for functions handling resources/read requests for specific resources.
type ReadResourceHandlerFunc func(ctx context.Context, request ReadResourceRequest) ([]ResourceContents, error)

// ResourceTemplateHandlerFunc defines the signature for functions handling resources/read requests that match a resource template.
type ResourceTemplateHandlerFunc func(ctx context.Context, request ReadResourceRequest) ([]ResourceContents, error)

// CompletionHandlerFunc defines the signature for completion/complete requests.
type CompletionHandlerFunc func(ctx context.Context, request CompleteRequest) (*CompleteResult, error)

// Error helper functions for consistent error handling

// NewParameterError creates a new parameter validation error
func NewParameterError(method, parameter, message string, cause error) *ParameterError {
	return &ParameterError{
		Method:    method,
		Parameter: parameter,
		Message:   message,
		Cause:     cause,
	}
}

// NewParameterErrorFromJSON creates a parameter error from JSON unmarshaling failure
func NewParameterErrorFromJSON(method string, cause error) *ParameterError {
	return &ParameterError{
		Method:  method,
		Message: "JSON unmarshaling failed",
		Cause:   cause,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resourceType, identifier string) *NotFoundError {
	return &NotFoundError{
		Type:       resourceType,
		Identifier: identifier,
	}
}

// NewAlreadyExistsError creates a new already exists error
func NewAlreadyExistsError(resourceType, identifier string) *AlreadyExistsError {
	return &AlreadyExistsError{
		Type:       resourceType,
		Identifier: identifier,
	}
}

// ValidateRequestSize validates that a request doesn't exceed size limits
func ValidateRequestSize(data []byte, maxSize int) error {
	if len(data) > maxSize {
		return &ParameterError{
			Message: fmt.Sprintf("request size %d exceeds maximum %d bytes", len(data), maxSize),
		}
	}
	return nil
}

// ValidateRequiredFields validates that required fields are present and non-empty
func ValidateRequiredFields(method string, fields map[string]string) error {
	for field, value := range fields {
		if value == "" {
			return NewParameterError(method, field, "required field is empty", nil)
		}
	}
	return nil
}

// ValidationConfig holds configuration for request validation
type ValidationConfig struct {
	MaxRequestSize   int  `json:"maxRequestSize"`   // Maximum request size in bytes
	MaxResponseSize  int  `json:"maxResponseSize"`  // Maximum response size in bytes
	ValidateRequired bool `json:"validateRequired"` // Whether to validate required fields
	ValidateFormat   bool `json:"validateFormat"`   // Whether to validate field formats
}

// DefaultValidationConfig returns the default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxRequestSize:   1024 * 1024,      // 1MB
		MaxResponseSize:  10 * 1024 * 1024, // 10MB
		ValidateRequired: true,
		ValidateFormat:   true,
	}
}

// ParameterValidator provides comprehensive parameter validation
type ParameterValidator struct {
	config *ValidationConfig
}

// NewParameterValidator creates a new parameter validator with the given config
func NewParameterValidator(config *ValidationConfig) *ParameterValidator {
	if config == nil {
		config = DefaultValidationConfig()
	}
	return &ParameterValidator{config: config}
}

// ValidateRequest validates a complete request including size, format, and content
func (v *ParameterValidator) ValidateRequest(method string, requestData []byte) error {
	// Validate request size
	if err := ValidateRequestSize(requestData, v.config.MaxRequestSize); err != nil {
		return err
	}

	// Check if the request is valid JSON
	if !json.Valid(requestData) {
		return NewParameterError(method, "", "invalid JSON format", nil)
	}

	return nil
}

// ValidateInitializeRequest validates an initialize request
func (v *ParameterValidator) ValidateInitializeRequest(params InitializeRequest) error {
	if !v.config.ValidateRequired {
		return nil
	}

	return ValidateRequiredFields(string(MethodInitialize), map[string]string{
		"protocolVersion": params.ProtocolVersion,
		"clientInfo.name": params.ClientInfo.Name,
	})
}

// ValidateCallToolRequest validates a tools/call request
func (v *ParameterValidator) ValidateCallToolRequest(params CallToolRequest) error {
	if !v.config.ValidateRequired {
		return nil
	}

	if err := ValidateRequiredFields(string(MethodToolsCall), map[string]string{
		"name": params.Name,
	}); err != nil {
		return err
	}

	if v.config.ValidateFormat {
		// Validate tool name format (basic alphanumeric and common symbols)
		if !isValidIdentifier(params.Name) {
			return NewParameterError(string(MethodToolsCall), "name", "invalid tool name format", nil)
		}
	}

	return nil
}

// ValidateGetPromptRequest validates a prompts/get request
func (v *ParameterValidator) ValidateGetPromptRequest(params GetPromptRequest) error {
	if !v.config.ValidateRequired {
		return nil
	}

	if err := ValidateRequiredFields(string(MethodPromptsGet), map[string]string{
		"name": params.Name,
	}); err != nil {
		return err
	}

	if v.config.ValidateFormat && !isValidIdentifier(params.Name) {
		return NewParameterError(string(MethodPromptsGet), "name", "invalid prompt name format", nil)
	}

	return nil
}

// ValidateReadResourceRequest validates a resources/read request
func (v *ParameterValidator) ValidateReadResourceRequest(params ReadResourceRequest) error {
	if !v.config.ValidateRequired {
		return nil
	}

	if err := ValidateRequiredFields(string(MethodResourcesRead), map[string]string{
		"uri": params.URI,
	}); err != nil {
		return err
	}

	if v.config.ValidateFormat && !isValidURI(params.URI) {
		return NewParameterError(string(MethodResourcesRead), "uri", "invalid URI format", nil)
	}

	return nil
}

func (v *ParameterValidator) ValidateResourceSubscription(method MCPMethod, uri string) error {
	if !v.config.ValidateRequired {
		return nil
	}

	if err := ValidateRequiredFields(string(method), map[string]string{
		"uri": uri,
	}); err != nil {
		return err
	}

	if v.config.ValidateFormat && !isValidURI(uri) {
		return NewParameterError(string(method), "uri", "invalid URI format", nil)
	}

	return nil
}

// isValidIdentifier checks if a string is a valid identifier (alphanumeric, underscore, hyphen)
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

// isValidURI checks if a string looks like a valid URI (basic validation)
func isValidURI(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Basic URI validation - must contain alphanumeric chars and common URI characters
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '/' || r == ':' || r == '.' || r == '-' || r == '_' || r == '?' || r == '&' || r == '=' || r == '#') {
			return false
		}
	}
	return true
}

// ConnectionPoolConfig holds configuration for connection pooling
type ConnectionPoolConfig struct {
	MaxConnections     int           `json:"maxConnections"`     // Maximum number of concurrent connections
	MaxIdleTime        time.Duration `json:"maxIdleTime"`        // Maximum time a connection can be idle
	MaxConnectionAge   time.Duration `json:"maxConnectionAge"`   // Maximum age of a connection before forced refresh
	ConnectionTimeout  time.Duration `json:"connectionTimeout"`  // Timeout for establishing new connections
	HealthCheckTimeout time.Duration `json:"healthCheckTimeout"` // Timeout for health check operations
	CleanupInterval    time.Duration `json:"cleanupInterval"`    // Interval between idle connection cleanup
	EnableHealthChecks bool          `json:"enableHealthChecks"` // Whether to perform health checks on idle connections
}

// DefaultConnectionPoolConfig returns the default connection pool configuration
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxConnections:     10,
		MaxIdleTime:        5 * time.Minute,
		MaxConnectionAge:   1 * time.Hour,
		ConnectionTimeout:  30 * time.Second,
		HealthCheckTimeout: 5 * time.Second,
		CleanupInterval:    1 * time.Minute,
		EnableHealthChecks: true,
	}
}

// JSON marshaling optimization types and helpers

// UnmarshalHelper provides optimized parameter unmarshaling with validation
type UnmarshalHelper struct {
	validator *ParameterValidator
}

// NewUnmarshalHelper creates a new unmarshal helper with the given validator
func NewUnmarshalHelper(validator *ParameterValidator) *UnmarshalHelper {
	return &UnmarshalHelper{validator: validator}
}

// UnmarshalAndValidateInitialize unmarshals and validates an initialize request
func (u *UnmarshalHelper) UnmarshalAndValidateInitialize(data []byte) (*InitializeRequest, error) {
	if u.validator != nil {
		if err := u.validator.ValidateRequest(string(MethodInitialize), data); err != nil {
			return nil, err
		}
	}

	var params InitializeRequest
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, NewParameterErrorFromJSON(string(MethodInitialize), err)
	}

	if u.validator != nil {
		if err := u.validator.ValidateInitializeRequest(params); err != nil {
			return nil, err
		}
	}

	return &params, nil
}

// UnmarshalAndValidateCallTool unmarshals and validates a tools/call request
func (u *UnmarshalHelper) UnmarshalAndValidateCallTool(data []byte) (*CallToolRequest, error) {
	if u.validator != nil {
		if err := u.validator.ValidateRequest(string(MethodToolsCall), data); err != nil {
			return nil, err
		}
	}

	var params CallToolRequest
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, NewParameterErrorFromJSON(string(MethodToolsCall), err)
	}

	if u.validator != nil {
		if err := u.validator.ValidateCallToolRequest(params); err != nil {
			return nil, err
		}
	}

	return &params, nil
}

// UnmarshalAndValidateGetPrompt unmarshals and validates a prompts/get request
func (u *UnmarshalHelper) UnmarshalAndValidateGetPrompt(data []byte) (*GetPromptRequest, error) {
	if u.validator != nil {
		if err := u.validator.ValidateRequest(string(MethodPromptsGet), data); err != nil {
			return nil, err
		}
	}

	var params GetPromptRequest
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, NewParameterErrorFromJSON(string(MethodPromptsGet), err)
	}

	if u.validator != nil {
		if err := u.validator.ValidateGetPromptRequest(params); err != nil {
			return nil, err
		}
	}

	return &params, nil
}

// UnmarshalAndValidateReadResource unmarshals and validates a resources/read request
func (u *UnmarshalHelper) UnmarshalAndValidateReadResource(data []byte) (*ReadResourceRequest, error) {
	if u.validator != nil {
		if err := u.validator.ValidateRequest(string(MethodResourcesRead), data); err != nil {
			return nil, err
		}
	}

	var params ReadResourceRequest
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, NewParameterErrorFromJSON(string(MethodResourcesRead), err)
	}

	if u.validator != nil {
		if err := u.validator.ValidateReadResourceRequest(params); err != nil {
			return nil, err
		}
	}

	return &params, nil
}

// UnmarshalSimpleRequest unmarshals a request that only needs JSON validation (like list operations)
func (u *UnmarshalHelper) UnmarshalSimpleRequest(data []byte, method string, target interface{}) error {
	if u.validator != nil {
		if err := u.validator.ValidateRequest(method, data); err != nil {
			return err
		}
	}

	if err := json.Unmarshal(data, target); err != nil {
		return NewParameterErrorFromJSON(method, err)
	}

	return nil
}

// ObjectPool provides reusable object pooling to reduce memory allocations
// for frequently used request/response types. This improves performance
// by avoiding garbage collection pressure from repeated allocations.
type ObjectPool[T any] struct {
	pool  sync.Pool
	reset func(*T) // Optional reset function to clear object state
}

// NewObjectPool creates a new object pool for type T with optional reset function
func NewObjectPool[T any](resetFn func(*T)) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return new(T)
			},
		},
		reset: resetFn,
	}
}

// Get retrieves an object from the pool
func (p *ObjectPool[T]) Get() *T {
	obj := p.pool.Get().(*T)
	if p.reset != nil {
		p.reset(obj)
	}
	return obj
}

// Put returns an object to the pool for reuse
func (p *ObjectPool[T]) Put(obj *T) {
	if obj != nil {
		p.pool.Put(obj)
	}
}

// Pre-defined pools for commonly used types to reduce allocations
var (
	// Request type pools
	initializeRequestPool = NewObjectPool(func(r *InitializeRequest) {
		*r = InitializeRequest{}
	})

	callToolRequestPool = NewObjectPool(func(r *CallToolRequest) {
		*r = CallToolRequest{}
	})

	getPromptRequestPool = NewObjectPool(func(r *GetPromptRequest) {
		*r = GetPromptRequest{}
	})

	readResourceRequestPool = NewObjectPool(func(r *ReadResourceRequest) {
		*r = ReadResourceRequest{}
	})

	listToolsRequestPool = NewObjectPool(func(r *ListToolsRequest) {
		*r = ListToolsRequest{}
	})

	listPromptsRequestPool = NewObjectPool(func(r *ListPromptsRequest) {
		*r = ListPromptsRequest{}
	})

	listResourcesRequestPool = NewObjectPool(func(r *ListResourcesRequest) {
		*r = ListResourcesRequest{}
	})

	// Response type pools
	callToolResultPool = NewObjectPool(func(r *CallToolResult) {
		*r = CallToolResult{}
	})

	// Buffer pool for JSON unmarshaling operations
	byteBufferPool = NewObjectPool(func(b *[]byte) {
		if cap(*b) > 64*1024 { // Reset large buffers to avoid memory leaks
			*b = make([]byte, 0, 1024)
		} else {
			*b = (*b)[:0]
		}
	})
)

// PooledUnmarshalHelper extends UnmarshalHelper with object pooling for better performance
type PooledUnmarshalHelper struct {
	*UnmarshalHelper
}

// NewPooledUnmarshalHelper creates an unmarshal helper that uses object pools
func NewPooledUnmarshalHelper(validator *ParameterValidator) *PooledUnmarshalHelper {
	return &PooledUnmarshalHelper{
		UnmarshalHelper: NewUnmarshalHelper(validator),
	}
}

// UnmarshalAndValidateInitializePooled uses object pooling for better performance
func (u *PooledUnmarshalHelper) UnmarshalAndValidateInitializePooled(data []byte) (*InitializeRequest, error) {
	if u.validator != nil {
		if err := u.validator.ValidateRequest(string(MethodInitialize), data); err != nil {
			return nil, err
		}
	}

	params := initializeRequestPool.Get()
	if err := json.Unmarshal(data, params); err != nil {
		initializeRequestPool.Put(params)
		return nil, NewParameterErrorFromJSON(string(MethodInitialize), err)
	}

	if u.validator != nil {
		if err := u.validator.ValidateInitializeRequest(*params); err != nil {
			initializeRequestPool.Put(params)
			return nil, err
		}
	}

	// Note: Caller is responsible for returning object to pool when done
	return params, nil
}

// UnmarshalAndValidateCallToolPooled uses object pooling for better performance
func (u *PooledUnmarshalHelper) UnmarshalAndValidateCallToolPooled(data []byte) (*CallToolRequest, error) {
	if u.validator != nil {
		if err := u.validator.ValidateRequest(string(MethodToolsCall), data); err != nil {
			return nil, err
		}
	}

	params := callToolRequestPool.Get()
	if err := json.Unmarshal(data, params); err != nil {
		callToolRequestPool.Put(params)
		return nil, NewParameterErrorFromJSON(string(MethodToolsCall), err)
	}

	if u.validator != nil {
		if err := u.validator.ValidateCallToolRequest(*params); err != nil {
			callToolRequestPool.Put(params)
			return nil, err
		}
	}

	return params, nil
}

// ReturnInitializeRequest returns an InitializeRequest to its pool
func ReturnInitializeRequest(req *InitializeRequest) {
	if req != nil {
		initializeRequestPool.Put(req)
	}
}

// ReturnCallToolRequest returns a CallToolRequest to its pool
func ReturnCallToolRequest(req *CallToolRequest) {
	if req != nil {
		callToolRequestPool.Put(req)
	}
}

// ReturnCallToolResult returns a CallToolResult to its pool
func ReturnCallToolResult(result *CallToolResult) {
	if result != nil {
		callToolResultPool.Put(result)
	}
}
