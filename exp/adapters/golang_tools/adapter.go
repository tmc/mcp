// Package golang_tools provides an adapter for golang-tools-internal-mcp server implementations.
package golang_tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/mcp/exp/adapters"
	"github.com/tmc/mcp/modelcontextprotocol"
	// The golang-tools types would normally be imported here
	// mcp "golang.org/x/tools/internal/mcp"
	// mcpProtocol "golang.org/x/tools/internal/mcp/protocol"
)

// Server is a minimal interface for the golang-tools server
type Server interface {
	GetServerInfo() ServerInfo
	GetTools() []Tool
	GetPrompts() []Prompt
	CallTool(ctx context.Context, name string, args map[string]json.RawMessage) (any, error)
	GetPrompt(ctx context.Context, name string, args map[string]string) (any, error)
}

type ServerInfo struct {
	Name            string
	Version         string
	Instructions    string
	ProtocolVersion string
}

type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type Prompt struct {
	Name        string
	Description string
	Arguments   []PromptArgument
}

// GolangToolsAdapter adapts golang-tools-internal-mcp servers to work with the standard MCP SDK.
// It handles the translation between the golang-tools implementation patterns and
// the SDK server interface.
type GolangToolsAdapter struct {
	server       Server
	golangServer interface{} // This would be *mcp.Server from golang-tools

	// Store tools and prompts locally
	tools   []ToolDefinition
	prompts []PromptDefinition

	// Map to store handlers
	toolHandlers   map[string]ToolHandler
	promptHandlers map[string]PromptHandler
}

// Types that represent golang-tools definitions
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema interface{} // Would be *jsonschema.Schema in golang-tools
}

type PromptDefinition struct {
	Name        string
	Description string
	Arguments   []PromptArgument
}

type PromptArgument struct {
	Name        string
	Description string
	Required    bool
}

// Handler types - these would match the golang-tools handler signatures
type ToolHandler func(context.Context, interface{}, map[string]json.RawMessage) (*ToolResult, error)
type PromptHandler func(context.Context, interface{}, map[string]string) (*PromptResult, error)

// Result types - these match the golang-tools result types
type ToolResult struct {
	Content []Content
	IsError bool
}

type PromptResult struct {
	Description string
	Messages    []PromptMessage
}

type PromptMessage struct {
	Role    string
	Content Content
}

type Content struct {
	Type     string
	Text     string
	Data     string
	MIMEType string
	Resource *Resource
}

type Resource struct {
	URI      string
	MIMEType string
	Text     string
	Blob     *string
}

// NewAdapter creates a new GolangTools adapter
func NewAdapter() adapters.Adapter {
	return &GolangToolsAdapter{
		tools:          []ToolDefinition{},
		prompts:        []PromptDefinition{},
		toolHandlers:   make(map[string]ToolHandler),
		promptHandlers: make(map[string]PromptHandler),
	}
}

// Initialize sets up the adapter with the target server
func (a *GolangToolsAdapter) Initialize(ctx context.Context, srv interface{}) error {
	if srv != nil {
		if s, ok := srv.(Server); ok {
			a.server = s
		}
	}

	// Extract server info from the SDK server if available
	var info ServerInfo
	if a.server != nil {
		info = a.server.GetServerInfo()
	} else {
		info = ServerInfo{
			Name:            "golang-tools-adapter",
			Version:         "0.1.0",
			ProtocolVersion: "2024-11-05",
		}
	}

	// Create the golang-tools server (simulated since we don't have the actual implementation)
	// In a real implementation, this would call the golang-tools NewServer function
	a.golangServer = struct {
		Name         string
		Version      string
		Instructions string
	}{Name: info.Name, Version: info.Version, Instructions: info.Instructions}

	// Initialize tools and prompts from the SDK server
	if err := a.initializeFeatures(ctx); err != nil {
		return fmt.Errorf("failed to initialize features: %w", err)
	}

	return nil
}

// HandleRequest processes incoming requests for golang-tools servers
func (a *GolangToolsAdapter) HandleRequest(ctx context.Context, method string, params any) (any, error) {
	switch method {
	case "initialize":
		return a.handleInitialize(ctx, params)
	case "tools/list":
		return a.handleListTools(ctx, params)
	case "tools/call":
		return a.handleCallTool(ctx, params)
	case "prompts/list":
		return a.handleListPrompts(ctx, params)
	case "prompts/get":
		return a.handleGetPrompt(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

// GetCapabilities returns the server capabilities
func (a *GolangToolsAdapter) GetCapabilities() modelcontextprotocol.ServerCapabilities {
	caps := modelcontextprotocol.ServerCapabilities{}

	// Check if we have tools registered
	if len(a.tools) > 0 {
		toolsListChanged := false
		caps.Tools = &modelcontextprotocol.ToolsServerCapability{
			ListChanged: &toolsListChanged,
		}
	}

	// Check if we have prompts registered
	if len(a.prompts) > 0 {
		promptsListChanged := false
		caps.Prompts = &modelcontextprotocol.PromptsServerCapability{
			ListChanged: &promptsListChanged,
		}
	}

	return caps
}

func (a *GolangToolsAdapter) handleInitialize(ctx context.Context, params any) (any, error) {
	// Return initialization result using the SDK server's info
	info := a.server.GetServerInfo()

	return modelcontextprotocol.InitializeResult{
		ProtocolVersion: info.ProtocolVersion,
		ServerInfo: modelcontextprotocol.Implementation{
			Name:    info.Name,
			Version: info.Version,
		},
		Capabilities: a.GetCapabilities(),
	}, nil
}

func (a *GolangToolsAdapter) handleListTools(ctx context.Context, params any) (any, error) {
	// Convert tools to SDK protocol
	tools := make([]modelcontextprotocol.Tool, len(a.tools))
	for i, tool := range a.tools {
		// Create a simple schema structure
		schema := modelcontextprotocol.ToolSchema{
			Type:       "object",
			Properties: make(map[string]json.RawMessage),
		}

		tools[i] = modelcontextprotocol.Tool{
			Name:        tool.Name,
			Description: &tool.Description,
			InputSchema: schema,
		}
	}

	return modelcontextprotocol.ListToolsResult{
		Tools: tools,
	}, nil
}

func (a *GolangToolsAdapter) handleCallTool(ctx context.Context, params any) (any, error) {
	// Parse parameters
	paramMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters type: %T", params)
	}

	name, ok := paramMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid tool name")
	}

	arguments, _ := paramMap["arguments"].(map[string]interface{})

	// Find the tool handler
	handler, ok := a.toolHandlers[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Convert arguments to json.RawMessage
	argMap := make(map[string]json.RawMessage)
	for k, v := range arguments {
		bytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal argument %s: %w", k, err)
		}
		argMap[k] = json.RawMessage(bytes)
	}

	// Create a mock server connection
	mockConn := &mockServerConnection{server: a.golangServer}

	// Call the handler
	result, err := handler(ctx, mockConn, argMap)
	if err != nil {
		return nil, err
	}

	// Convert result to SDK protocol
	content := make([]modelcontextprotocol.Content, len(result.Content))
	for i, c := range result.Content {
		content[i] = convertContent(c)
	}

	return modelcontextprotocol.CallToolResult{
		Content: content,
		IsError: &result.IsError,
	}, nil
}

func (a *GolangToolsAdapter) handleListPrompts(ctx context.Context, params any) (any, error) {
	// Convert prompts to SDK protocol
	prompts := make([]modelcontextprotocol.Prompt, len(a.prompts))
	for i, prompt := range a.prompts {
		args := make([]*modelcontextprotocol.PromptArgument, len(prompt.Arguments))
		for j, arg := range prompt.Arguments {
			args[j] = &modelcontextprotocol.PromptArgument{
				Name:        arg.Name,
				Description: &arg.Description,
				Required:    &arg.Required,
			}
		}
		prompts[i] = modelcontextprotocol.Prompt{
			Name:        prompt.Name,
			Description: &prompt.Description,
			Arguments:   args,
		}
	}

	return modelcontextprotocol.ListPromptsResult{
		Prompts: prompts,
	}, nil
}

func (a *GolangToolsAdapter) handleGetPrompt(ctx context.Context, params any) (any, error) {
	// Parse parameters
	paramMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters type: %T", params)
	}

	name, ok := paramMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid prompt name")
	}

	arguments, _ := paramMap["arguments"].(map[string]interface{})

	// Find the prompt handler
	handler, ok := a.promptHandlers[name]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}

	// Convert arguments to map[string]string
	argMap := make(map[string]string)
	for k, v := range arguments {
		if str, ok := v.(string); ok {
			argMap[k] = str
		}
	}

	// Create a mock server connection
	mockConn := &mockServerConnection{server: a.golangServer}

	// Call the handler
	result, err := handler(ctx, mockConn, argMap)
	if err != nil {
		return nil, err
	}

	// Convert messages to SDK protocol
	messages := make([]modelcontextprotocol.PromptMessage, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = modelcontextprotocol.PromptMessage{
			Role:    modelcontextprotocol.Role(msg.Role),
			Content: convertContent(msg.Content),
		}
	}

	return modelcontextprotocol.GetPromptResult{
		Description: result.Description,
		Messages:    messages,
	}, nil
}

// initializeFeatures initializes tools and prompts from the SDK server
func (a *GolangToolsAdapter) initializeFeatures(ctx context.Context) error {
	// Get tools from SDK server and register them
	if tools := a.server.GetTools(); len(tools) > 0 {
		for _, tool := range tools {
			// Create golang-tools tool definition
			golangTool := ToolDefinition{
				Name:        tool.Name,
				Description: tool.Description,
			}

			// Convert input schema if available
			if tool.InputSchema != nil {
				var schema interface{}
				if err := json.Unmarshal(tool.InputSchema, &schema); err == nil {
					golangTool.InputSchema = schema
				}
			}

			// Store the tool
			a.tools = append(a.tools, golangTool)

			// Create and store the handler
			toolName := tool.Name // Capture for closure
			a.toolHandlers[toolName] = func(ctx context.Context, conn interface{}, args map[string]json.RawMessage) (*ToolResult, error) {
				// Call the SDK server's tool handler
				result, err := a.server.CallTool(ctx, toolName, args)
				if err != nil {
					return nil, err
				}

				// Convert result to golang-tools format
				callResult, ok := result.(modelcontextprotocol.CallToolResult)
				if !ok {
					return nil, fmt.Errorf("unexpected tool result type: %T", result)
				}

				// Convert content
				golangContent := make([]Content, len(callResult.Content))
				for i, c := range callResult.Content {
					golangContent[i] = convertSDKContentToGolang(c)
				}

				return &ToolResult{
					Content: golangContent,
					IsError: callResult.IsError,
				}, nil
			}
		}
	}

	// Get prompts from SDK server and register them
	if prompts := a.server.GetPrompts(); len(prompts) > 0 {
		for _, prompt := range prompts {
			// Create golang-tools prompt definition
			golangPrompt := PromptDefinition{
				Name:        prompt.Name,
				Description: prompt.Description,
			}

			// Convert prompt arguments
			args := make([]PromptArgument, len(prompt.Arguments))
			for i, arg := range prompt.Arguments {
				args[i] = PromptArgument{
					Name:        arg.Name,
					Description: arg.Description,
					Required:    arg.Required,
				}
			}
			golangPrompt.Arguments = args

			// Store the prompt
			a.prompts = append(a.prompts, golangPrompt)

			// Create and store the handler
			promptName := prompt.Name // Capture for closure
			a.promptHandlers[promptName] = func(ctx context.Context, conn interface{}, args map[string]string) (*PromptResult, error) {
				// Call the SDK server's prompt handler
				result, err := a.server.GetPrompt(ctx, promptName, args)
				if err != nil {
					return nil, err
				}

				// Convert result to golang-tools format
				getResult, ok := result.(modelcontextprotocol.GetPromptResult)
				if !ok {
					return nil, fmt.Errorf("unexpected prompt result type: %T", result)
				}

				// Convert messages
				messages := make([]PromptMessage, len(getResult.Messages))
				for i, msg := range getResult.Messages {
					messages[i] = PromptMessage{
						Role:    string(msg.Role),
						Content: convertSDKContentToGolang(msg.Content),
					}
				}

				return &PromptResult{
					Description: getResult.Description,
					Messages:    messages,
				}, nil
			}
		}
	}

	return nil
}

// convertContent converts golang-tools content to SDK protocol content
func convertContent(content Content) modelcontextprotocol.Content {
	switch content.Type {
	case "text":
		return modelcontextprotocol.TextContent{
			Type: "text",
			Text: content.Text,
		}
	case "image":
		return modelcontextprotocol.ImageContent{
			Type:     "image",
			Data:     content.Data,
			MimeType: content.MIMEType,
		}
	case "resource":
		if content.Resource != nil {
			if content.Resource.Blob != nil {
				return modelcontextprotocol.ResourceContent{
					Type: "resource",
					Resource: modelcontextprotocol.BlobResourceContents{
						URI:      content.Resource.URI,
						MimeType: content.Resource.MIMEType,
						Blob:     *content.Resource.Blob,
					},
				}
			}
			return modelcontextprotocol.ResourceContent{
				Type: "resource",
				Resource: modelcontextprotocol.TextResourceContents{
					URI:      content.Resource.URI,
					MimeType: content.Resource.MIMEType,
					Text:     content.Resource.Text,
				},
			}
		}
	}

	// Default to text content
	return modelcontextprotocol.TextContent{
		Type: "text",
		Text: content.Text,
	}
}

// convertSDKContentToGolang converts SDK protocol content to golang-tools content
func convertSDKContentToGolang(content modelcontextprotocol.Content) Content {
	switch c := content.(type) {
	case modelcontextprotocol.TextContent:
		return Content{
			Type: "text",
			Text: c.Text,
		}
	case modelcontextprotocol.ImageContent:
		return Content{
			Type:     "image",
			Data:     c.Data,
			MIMEType: c.MimeType,
		}
	case modelcontextprotocol.ResourceContent:
		golangContent := Content{
			Type: "resource",
		}

		switch r := c.Resource.(type) {
		case modelcontextprotocol.TextResourceContents:
			golangContent.Resource = &Resource{
				URI:      r.URI,
				MIMEType: r.MimeType,
				Text:     r.Text,
			}
		case modelcontextprotocol.BlobResourceContents:
			blob := r.Blob
			golangContent.Resource = &Resource{
				URI:      r.URI,
				MIMEType: r.MimeType,
				Blob:     &blob,
			}
		}

		return golangContent
	default:
		// Default to text content
		return Content{
			Type: "text",
			Text: fmt.Sprintf("%v", content),
		}
	}
}

// mockServerConnection is a mock connection for calling golang-tools server methods
type mockServerConnection struct {
	server interface{}
}

// Implement methods required by golang-tools ServerConnection
// These methods would normally be part of the golang-tools ServerConnection interface
func (m *mockServerConnection) Ping(ctx context.Context) error {
	return nil
}

func (m *mockServerConnection) Close() error {
	return nil
}

func (m *mockServerConnection) Wait() error {
	return nil
}

func (m *mockServerConnection) Notify(ctx context.Context, method string, params interface{}) error {
	return nil
}

func init() {
	// Register this adapter in the default registry
	adapters.DefaultRegistry.Register("golang-tools", NewAdapter)
}
