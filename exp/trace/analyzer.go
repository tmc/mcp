// Package trace provides MCP trace analysis capabilities
package trace

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/mcp/modelcontextprotocol"
	"github.com/tmc/mcp/modelcontextprotocol/draft"
)

// Entry represents a single trace entry
type Entry struct {
	Timestamp string
	Direction string // -> (client to server) or <- (server to client)
	Method    string
	Payload   json.RawMessage
}

// State represents the analyzed state from trace data
type State struct {
	MessageCount int
	HasServer    bool
	HasClient    bool
	IsExecutable bool

	// Discovered elements
	Tools        []*ToolInfo
	Handlers     []*HandlerInfo
	Resources    []*ResourceInfo
	ServerInfo   *modelcontextprotocol.Implementation
	ClientInfo   *modelcontextprotocol.Implementation
	Capabilities *modelcontextprotocol.ServerCapabilities

	// Interaction patterns
	ToolCalls     map[string]*ToolCallPattern
	Subscriptions map[string]*SubscriptionInfo
	Notifications []*NotificationInfo

	// Advanced features
	ResourceTemplates []*ResourceTemplateInfo
	Prompts           []*PromptInfo
	PromptCalls       map[string]*PromptCallPattern
	LoggingLevel      string
	Roots             []*RootInfo

	// Error patterns
	ErrorPatterns  map[string]int
	ProtocolErrors []ProtocolError

	// Metadata
	InitializeParams *modelcontextprotocol.InitializeRequestParams
	InitializeResult *modelcontextprotocol.InitializeResult
	Instructions     string
}

// ToolInfo represents discovered tool information
type ToolInfo struct {
	Tool      *modelcontextprotocol.Tool
	CallCount int
	Examples  []ToolCallExample
}

// ToolCallExample represents an example tool call
type ToolCallExample struct {
	Arguments map[string]any
	Result    *modelcontextprotocol.CallToolResult
	Error     *modelcontextprotocol.ErrorObject
}

// HandlerInfo represents a discovered handler pattern
type HandlerInfo struct {
	Method     string
	ParamType  string
	ResultType string
	ErrorTypes []string
}

// ResourceInfo represents discovered resource information
type ResourceInfo struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// ToolCallPattern represents usage patterns for a tool
type ToolCallPattern struct {
	Name            string
	TotalCalls      int
	SuccessfulCalls int
	ErrorPatterns   map[string]int
	TypicalDuration float64
}

// SubscriptionInfo represents a resource subscription
type SubscriptionInfo struct {
	URI         string
	Active      bool
	UpdateCount int
}

// NotificationInfo represents a notification pattern
type NotificationInfo struct {
	Method    string
	Count     int
	LastSeen  string
	Direction string // client or server
}

// ResourceTemplateInfo represents a resource template
type ResourceTemplateInfo struct {
	URITemplate string
	Name        string
	Description string
	MimeType    string
}

// PromptInfo represents a prompt definition
type PromptInfo struct {
	Name        string
	Description string
	Arguments   []*modelcontextprotocol.PromptArgument
	CallCount   int
	Examples    []PromptCallExample
}

// PromptCallExample represents an example prompt call
type PromptCallExample struct {
	Arguments map[string]string
	Messages  []modelcontextprotocol.PromptMessage
}

// PromptCallPattern represents usage patterns for a prompt
type PromptCallPattern struct {
	Name       string
	TotalCalls int
	Arguments  map[string][]string // argument name -> example values
}

// RootInfo represents a root directory
type RootInfo struct {
	URI  string
	Name string
}

// ProtocolError represents a protocol-level error
type ProtocolError struct {
	Timestamp string
	Method    string
	Code      int
	Message   string
}

// Analyzer processes trace entries and builds state
type Analyzer struct {
	state State
}

// NewAnalyzer creates a new trace analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		state: State{
			ToolCalls:      make(map[string]*ToolCallPattern),
			Subscriptions:  make(map[string]*SubscriptionInfo),
			PromptCalls:    make(map[string]*PromptCallPattern),
			ErrorPatterns:  make(map[string]int),
			Notifications:  []*NotificationInfo{},
			ProtocolErrors: []ProtocolError{},
		},
	}
}

// ProcessEntry processes a single trace entry
func (a *Analyzer) ProcessEntry(entry *Entry) error {
	a.state.MessageCount++

	// Determine source
	if entry.Direction == "->" {
		a.state.HasClient = true
	} else if entry.Direction == "<-" {
		a.state.HasServer = true
	}

	// Check for errors first
	if err := a.checkForError(entry); err != nil {
		a.recordProtocolError(entry, err)
	}

	// Route to appropriate handler
	switch entry.Method {
	case modelcontextprotocol.MethodInitialize:
		return a.processInitialize(entry)
	case modelcontextprotocol.MethodToolsList:
		return a.processToolsList(entry)
	case modelcontextprotocol.MethodToolsCall:
		return a.processToolCall(entry)
	case modelcontextprotocol.MethodResourcesList:
		return a.processResourcesList(entry)
	case modelcontextprotocol.MethodResourcesTemplatesList:
		return a.processResourceTemplatesList(entry)
	case modelcontextprotocol.MethodResourcesRead:
		return a.processResourceRead(entry)
	case modelcontextprotocol.MethodResourcesSubscribe:
		return a.processSubscribe(entry)
	case modelcontextprotocol.MethodResourcesUnsubscribe:
		return a.processUnsubscribe(entry)
	case modelcontextprotocol.MethodPromptsList:
		return a.processPromptsList(entry)
	case modelcontextprotocol.MethodPromptsGet:
		return a.processPromptGet(entry)
	case modelcontextprotocol.MethodLoggingSetLevel:
		return a.processSetLogLevel(entry)
	case modelcontextprotocol.MethodRootsList:
		return a.processRootsList(entry)
	case modelcontextprotocol.MethodPing:
		return a.processPing(entry)
	default:
		// Check for notifications
		if strings.HasPrefix(entry.Method, "notifications/") {
			return a.processNotification(entry)
		}
		// Track as potential handler
		a.trackHandler(entry)
	}

	return nil
}

// GetState returns the current analyzed state
func (a *Analyzer) GetState() State {
	return a.state
}

func (a *Analyzer) processInitialize(entry *Entry) error {
	if entry.Direction == "->" {
		// Client initialization request
		var req struct {
			Params modelcontextprotocol.InitializeRequestParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &req); err != nil {
			return fmt.Errorf("parsing initialize params: %w", err)
		}
		a.state.ClientInfo = &req.Params.ClientInfo
		a.state.InitializeParams = &req.Params
	} else {
		// Server initialization response
		var resp struct {
			Result modelcontextprotocol.InitializeResult `json:"result"`
			Error  *modelcontextprotocol.ErrorObject     `json:"error"`
		}
		if err := json.Unmarshal(entry.Payload, &resp); err != nil {
			return fmt.Errorf("parsing initialize result: %w", err)
		}

		if resp.Error != nil {
			a.state.ErrorPatterns[fmt.Sprintf("initialize-%d", resp.Error.Code)]++
			return fmt.Errorf("initialize error: %s", resp.Error.Message)
		}

		if resp.Result.ServerInfo.Name != "" {
			a.state.ServerInfo = &resp.Result.ServerInfo
		}
		a.state.Capabilities = &resp.Result.Capabilities
		if resp.Result.Instructions != nil {
			a.state.Instructions = *resp.Result.Instructions
		}
		a.state.InitializeResult = &resp.Result
	}

	return nil
}

func (a *Analyzer) processToolsList(entry *Entry) error {
	if entry.Direction == "<-" {
		// Server response with tools list
		var result struct {
			Result modelcontextprotocol.ListToolsResult `json:"result"`
		}
		if err := json.Unmarshal(entry.Payload, &result); err != nil {
			// Try draft format
			var draftResult struct {
				Result draft.ListToolsResult `json:"result"`
			}
			if err := json.Unmarshal(entry.Payload, &draftResult); err != nil {
				return fmt.Errorf("parsing tools list: %w", err)
			}
			// Convert draft tools to base tools
			for _, tool := range draftResult.Result.Tools {
				a.addTool(&modelcontextprotocol.Tool{
					Name:        tool.Name,
					Description: tool.Description,
					InputSchema: tool.InputSchema,
					Annotations: tool.Annotations,
				})
			}
		} else {
			for _, tool := range result.Result.Tools {
				a.addTool(&tool)
			}
		}
	}

	return nil
}

func (a *Analyzer) processToolCall(entry *Entry) error {
	if entry.Direction == "->" {
		// Client tool call request
		var params struct {
			Method string                                     `json:"method"`
			Params modelcontextprotocol.CallToolRequestParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &params); err != nil {
			return fmt.Errorf("parsing tool call: %w", err)
		}

		pattern, exists := a.state.ToolCalls[params.Params.Name]
		if !exists {
			pattern = &ToolCallPattern{
				Name:          params.Params.Name,
				ErrorPatterns: make(map[string]int),
			}
			a.state.ToolCalls[params.Params.Name] = pattern
		}
		pattern.TotalCalls++

		// Store example
		for _, tool := range a.state.Tools {
			if tool.Tool.Name == params.Params.Name {
				tool.CallCount++
				example := ToolCallExample{
					Arguments: params.Params.Arguments,
				}
				tool.Examples = append(tool.Examples, example)
				break
			}
		}
	} else {
		// Server tool call response
		var response struct {
			Result *modelcontextprotocol.CallToolResult `json:"result"`
			Error  *modelcontextprotocol.ErrorObject    `json:"error"`
		}
		if err := json.Unmarshal(entry.Payload, &response); err != nil {
			return fmt.Errorf("parsing tool response: %w", err)
		}

		// Update patterns and examples
		// (In real implementation, we'd correlate with the request)
	}

	return nil
}

func (a *Analyzer) processResourcesList(entry *Entry) error {
	if entry.Direction == "<-" {
		var result struct {
			Result modelcontextprotocol.ListResourcesResult `json:"result"`
		}
		if err := json.Unmarshal(entry.Payload, &result); err != nil {
			return fmt.Errorf("parsing resources list: %w", err)
		}

		for _, resource := range result.Result.Resources {
			a.state.Resources = append(a.state.Resources, &ResourceInfo{
				URI:         resource.URI,
				Name:        resource.Name,
				Description: stringValue(resource.Description),
				MimeType:    stringValue(resource.MimeType),
			})
		}
	}

	return nil
}

func (a *Analyzer) processSubscribe(entry *Entry) error {
	if entry.Direction == "->" {
		var params struct {
			Params modelcontextprotocol.SubscribeRequestParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &params); err != nil {
			return fmt.Errorf("parsing subscribe: %w", err)
		}

		sub := &SubscriptionInfo{
			URI:    params.Params.URI,
			Active: true,
		}
		a.state.Subscriptions[params.Params.URI] = sub
	}

	return nil
}

func (a *Analyzer) processUnsubscribe(entry *Entry) error {
	if entry.Direction == "->" {
		var params struct {
			Params modelcontextprotocol.UnsubscribeRequestParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &params); err != nil {
			return fmt.Errorf("parsing unsubscribe: %w", err)
		}

		if sub, exists := a.state.Subscriptions[params.Params.URI]; exists {
			sub.Active = false
		}
	}

	return nil
}

func (a *Analyzer) processResourceTemplatesList(entry *Entry) error {
	if entry.Direction == "<-" {
		var result struct {
			Result modelcontextprotocol.ListResourceTemplatesResult `json:"result"`
		}
		if err := json.Unmarshal(entry.Payload, &result); err != nil {
			return fmt.Errorf("parsing resource templates list: %w", err)
		}

		for _, template := range result.Result.ResourceTemplates {
			a.state.ResourceTemplates = append(a.state.ResourceTemplates, &ResourceTemplateInfo{
				URITemplate: template.URITemplate,
				Name:        template.Name,
				Description: stringValue(template.Description),
				MimeType:    stringValue(template.MimeType),
			})
		}
	}

	return nil
}

func (a *Analyzer) processResourceRead(entry *Entry) error {
	// Track resource read patterns
	if entry.Direction == "->" {
		var params struct {
			Params modelcontextprotocol.ReadResourceRequestParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &params); err != nil {
			return fmt.Errorf("parsing resource read: %w", err)
		}

		// Update resource info if we have it
		for _, resource := range a.state.Resources {
			if resource.URI == params.Params.URI {
				// Mark as accessed
				break
			}
		}
	}

	return nil
}

func (a *Analyzer) processPromptsList(entry *Entry) error {
	if entry.Direction == "<-" {
		var result struct {
			Result modelcontextprotocol.ListPromptsResult `json:"result"`
		}
		if err := json.Unmarshal(entry.Payload, &result); err != nil {
			return fmt.Errorf("parsing prompts list: %w", err)
		}

		for _, prompt := range result.Result.Prompts {
			a.state.Prompts = append(a.state.Prompts, &PromptInfo{
				Name:        prompt.Name,
				Description: stringValue(prompt.Description),
				Arguments:   prompt.Arguments,
				Examples:    []PromptCallExample{},
			})
		}
	}

	return nil
}

func (a *Analyzer) processPromptGet(entry *Entry) error {
	if entry.Direction == "->" {
		var params struct {
			Params modelcontextprotocol.GetPromptRequestParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &params); err != nil {
			return fmt.Errorf("parsing prompt get: %w", err)
		}

		pattern, exists := a.state.PromptCalls[params.Params.Name]
		if !exists {
			pattern = &PromptCallPattern{
				Name:      params.Params.Name,
				Arguments: make(map[string][]string),
			}
			a.state.PromptCalls[params.Params.Name] = pattern
		}
		pattern.TotalCalls++

		// Store argument examples
		for name, value := range params.Params.Arguments {
			pattern.Arguments[name] = append(pattern.Arguments[name], value)
		}

		// Find prompt and update call count
		for _, prompt := range a.state.Prompts {
			if prompt.Name == params.Params.Name {
				prompt.CallCount++
				example := PromptCallExample{
					Arguments: params.Params.Arguments,
				}
				prompt.Examples = append(prompt.Examples, example)
				break
			}
		}
	} else {
		// Process prompt response for examples
		var result struct {
			Result modelcontextprotocol.GetPromptResult `json:"result"`
		}
		if err := json.Unmarshal(entry.Payload, &result); err != nil {
			return fmt.Errorf("parsing prompt get result: %w", err)
		}

		// Could store the messages as examples
	}

	return nil
}

func (a *Analyzer) processSetLogLevel(entry *Entry) error {
	if entry.Direction == "->" {
		var params struct {
			Params modelcontextprotocol.SetLevelRequestParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &params); err != nil {
			return fmt.Errorf("parsing set log level: %w", err)
		}

		a.state.LoggingLevel = string(params.Params.Level)
	}

	return nil
}

func (a *Analyzer) processRootsList(entry *Entry) error {
	if entry.Direction == "<-" {
		var result struct {
			Result modelcontextprotocol.ListRootsResult `json:"result"`
		}
		if err := json.Unmarshal(entry.Payload, &result); err != nil {
			return fmt.Errorf("parsing roots list: %w", err)
		}

		for _, root := range result.Result.Roots {
			a.state.Roots = append(a.state.Roots, &RootInfo{
				URI:  root.URI,
				Name: stringValue(root.Name),
			})
		}
	}

	return nil
}

func (a *Analyzer) processPing(entry *Entry) error {
	// Just track that ping is being used
	return nil
}

func (a *Analyzer) processNotification(entry *Entry) error {
	notification := &NotificationInfo{
		Method:    entry.Method,
		Count:     1,
		LastSeen:  entry.Timestamp,
		Direction: entry.Direction,
	}

	// Check if we've seen this notification type before
	for _, existing := range a.state.Notifications {
		if existing.Method == entry.Method && existing.Direction == entry.Direction {
			existing.Count++
			existing.LastSeen = entry.Timestamp
			return nil
		}
	}

	a.state.Notifications = append(a.state.Notifications, notification)

	// Handle specific notifications
	switch entry.Method {
	case modelcontextprotocol.MethodNotificationResourcesUpdated:
		var params struct {
			Params modelcontextprotocol.ResourceUpdatedNotificationParams `json:"params"`
		}
		if err := json.Unmarshal(entry.Payload, &params); err == nil {
			if sub, exists := a.state.Subscriptions[params.Params.URI]; exists {
				sub.UpdateCount++
			}
		}
	}

	return nil
}

func (a *Analyzer) checkForError(entry *Entry) error {
	var msg struct {
		Error *modelcontextprotocol.ErrorObject `json:"error"`
	}

	if err := json.Unmarshal(entry.Payload, &msg); err == nil && msg.Error != nil {
		return fmt.Errorf("protocol error %d: %s", msg.Error.Code, msg.Error.Message)
	}

	return nil
}

func (a *Analyzer) recordProtocolError(entry *Entry, err error) {
	protocolErr := ProtocolError{
		Timestamp: entry.Timestamp,
		Method:    entry.Method,
	}

	// Extract error details if possible
	var msg struct {
		Error *modelcontextprotocol.ErrorObject `json:"error"`
	}
	if json.Unmarshal(entry.Payload, &msg) == nil && msg.Error != nil {
		protocolErr.Code = msg.Error.Code
		protocolErr.Message = msg.Error.Message
		a.state.ErrorPatterns[fmt.Sprintf("%s-%d", entry.Method, msg.Error.Code)]++
	}

	a.state.ProtocolErrors = append(a.state.ProtocolErrors, protocolErr)
}

func (a *Analyzer) trackHandler(entry *Entry) {
	// Extract method pattern
	method := entry.Method
	if strings.HasPrefix(method, "notifications/") {
		a.state.Notifications = append(a.state.Notifications, method)
	} else {
		// Track as potential custom handler
		handler := &HandlerInfo{
			Method: method,
		}

		// Analyze payload structure
		var payload map[string]any
		if err := json.Unmarshal(entry.Payload, &payload); err == nil {
			if params, ok := payload["params"].(map[string]any); ok {
				handler.ParamType = inferType(params)
			}
			if result, ok := payload["result"].(map[string]any); ok {
				handler.ResultType = inferType(result)
			}
			if errObj, ok := payload["error"].(map[string]any); ok {
				if code, ok := errObj["code"].(float64); ok {
					handler.ErrorTypes = append(handler.ErrorTypes, fmt.Sprintf("code-%d", int(code)))
				}
			}
		}

		a.state.Handlers = append(a.state.Handlers, handler)
	}
}

func (a *Analyzer) addTool(tool *modelcontextprotocol.Tool) {
	// Check if tool already exists
	for _, existing := range a.state.Tools {
		if existing.Tool.Name == tool.Name {
			return
		}
	}

	a.state.Tools = append(a.state.Tools, &ToolInfo{
		Tool:     tool,
		Examples: []ToolCallExample{},
	})
}

// Helper functions
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func inferType(v map[string]any) string {
	// Simple type inference from structure
	if _, hasID := v["id"]; hasID {
		if _, hasMethod := v["method"]; hasMethod {
			return "Request"
		}
		return "Response"
	}

	// Check for known patterns
	if _, hasName := v["name"]; hasName {
		if _, hasArgs := v["arguments"]; hasArgs {
			return "CallToolParams"
		}
	}

	return "Object"
}
