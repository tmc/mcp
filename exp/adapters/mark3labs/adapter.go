// Package mark3labs provides an adapter for mark3labs-mcp-go server implementations.
package mark3labs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tmc/mcp/exp/adapters"
	"github.com/tmc/mcp/modelcontextprotocol"
)

// Mark3LabsAdapter adapts mark3labs-mcp-go servers to work with the standard MCP SDK.
// It handles the translation between the mark3labs implementation patterns and
// the SDK server interface.
// Server is a minimal interface for the adapter
type Server interface{}

type Mark3LabsAdapter struct {
	server Server
	// Mark3labs-specific server instance
	mark3labsServer *server.MCPServer
	// Store handlers from the mark3labs server
	tools     map[string]server.ServerTool
	resources map[string]resourceEntry
	prompts   map[string]promptEntry
}

// resourceEntry holds both a resource and its handler
type resourceEntry struct {
	resource mcp.Resource
	handler  server.ResourceHandlerFunc
}

// promptEntry holds both a prompt and its handler
type promptEntry struct {
	prompt  mcp.Prompt
	handler server.PromptHandlerFunc
}

// NewAdapter creates a new Mark3Labs adapter
func NewAdapter() adapters.Adapter {
	return &Mark3LabsAdapter{
		tools:     make(map[string]server.ServerTool),
		resources: make(map[string]resourceEntry),
		prompts:   make(map[string]promptEntry),
	}
}

// Initialize sets up the adapter with the target server
func (a *Mark3LabsAdapter) Initialize(ctx context.Context, srv interface{}) error {
	a.server = srv

	// Initialize mark3labs server with default options
	// Note: We'll create a simple mock since mark3labs implementation isn't available
	// a.mark3labsServer = server.NewMCPServer(
	//     server.WithName("mark3labs-adapter"),
	//     server.WithVersion("0.1.0"),
	// )

	return nil
}

// RegisterTool adds a mark3labs tool to the adapter
func (a *Mark3LabsAdapter) RegisterTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	serverTool := server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
	a.tools[tool.Name] = serverTool
}

// RegisterResource adds a mark3labs resource to the adapter
func (a *Mark3LabsAdapter) RegisterResource(resource mcp.Resource, handler server.ResourceHandlerFunc) {
	a.resources[resource.URI] = resourceEntry{
		resource: resource,
		handler:  handler,
	}
}

// RegisterPrompt adds a mark3labs prompt to the adapter
func (a *Mark3LabsAdapter) RegisterPrompt(prompt mcp.Prompt, handler server.PromptHandlerFunc) {
	a.prompts[prompt.Name] = promptEntry{
		prompt:  prompt,
		handler: handler,
	}
}

// HandleRequest processes incoming requests for mark3labs servers
func (a *Mark3LabsAdapter) HandleRequest(ctx context.Context, method string, params any) (any, error) {
	// Translate between mark3labs patterns and SDK patterns
	switch method {
	case "initialize":
		return a.handleInitialize(ctx, params)
	case "tools/list":
		return a.handleListTools(ctx, params)
	case "tools/call":
		return a.handleCallTool(ctx, params)
	case "resources/list":
		return a.handleListResources(ctx, params)
	case "resources/read":
		return a.handleReadResource(ctx, params)
	case "prompts/list":
		return a.handleListPrompts(ctx, params)
	case "prompts/get":
		return a.handleGetPrompt(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

// GetCapabilities returns the server capabilities
func (a *Mark3LabsAdapter) GetCapabilities() modelcontextprotocol.ServerCapabilities {
	// Convert mark3labs capabilities to SDK capabilities
	capabilities := modelcontextprotocol.ServerCapabilities{}

	// Check what capabilities are available based on registered items
	if len(a.tools) > 0 {
		capabilities.Tools = &modelcontextprotocol.ToolsCapability{}
	}
	if len(a.resources) > 0 {
		capabilities.Resources = &modelcontextprotocol.ResourcesCapability{}
	}
	if len(a.prompts) > 0 {
		capabilities.Prompts = &modelcontextprotocol.PromptsCapability{}
	}

	return capabilities
}

func (a *Mark3LabsAdapter) handleInitialize(ctx context.Context, params any) (any, error) {
	// Simply return the initialization result
	return modelcontextprotocol.InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: modelcontextprotocol.Implementation{
			Name:    "mark3labs-adapter",
			Version: "0.1.0",
		},
		Capabilities: a.GetCapabilities(),
	}, nil
}

func (a *Mark3LabsAdapter) handleListTools(ctx context.Context, params any) (any, error) {
	// Convert mark3labs tools to protocol tools
	protoTools := make([]modelcontextprotocol.Tool, 0, len(a.tools))
	for _, serverTool := range a.tools {
		tool := serverTool.Tool
		protoTool := modelcontextprotocol.Tool{
			Name:        tool.Name,
			Description: tool.Description,
		}

		// Convert input schema if present
		if tool.RawInputSchema != nil {
			protoTool.InputSchema = tool.RawInputSchema
		} else if tool.InputSchema != nil {
			schemaData, err := json.Marshal(tool.InputSchema)
			if err == nil {
				protoTool.InputSchema = json.RawMessage(schemaData)
			}
		}

		protoTools = append(protoTools, protoTool)
	}

	return modelcontextprotocol.ListToolsResult{
		Tools: protoTools,
	}, nil
}

func (a *Mark3LabsAdapter) handleCallTool(ctx context.Context, params any) (any, error) {
	if callParams, ok := params.(map[string]interface{}); ok {
		name, _ := callParams["name"].(string)
		args, _ := callParams["arguments"].(map[string]interface{})

		// Find the tool handler in mark3labs server
		if serverTool, ok := a.tools[name]; ok {
			// Create mark3labs CallToolRequest
			req := mcp.CallToolRequest{
				Request: mcp.Request{
					Method: string(mcp.MethodToolsCall),
				},
			}
			req.Params.Name = name
			req.Params.Arguments = args

			// Call tool handler
			result, err := serverTool.Handler(ctx, req)
			if err != nil {
				return nil, err
			}

			// Convert mark3labs CallToolResult to protocol
			return a.convertToolResult(result), nil
		}
	}

	return nil, fmt.Errorf("tool not found")
}

// handleListResources handles resource listing requests
func (a *Mark3LabsAdapter) handleListResources(ctx context.Context, params any) (any, error) {
	// Convert mark3labs resources to protocol resources
	protoResources := make([]modelcontextprotocol.Resource, 0, len(a.resources))
	for _, entry := range a.resources {
		resource := entry.resource
		protoResource := modelcontextprotocol.Resource{
			URI:         resource.URI,
			Name:        resource.Name,
			Description: resource.Description,
			MimeType:    resource.MIMEType,
		}
		protoResources = append(protoResources, protoResource)
	}

	return modelcontextprotocol.ListResourcesResult{
		Resources: protoResources,
	}, nil
}

// handleReadResource handles resource reading requests
func (a *Mark3LabsAdapter) handleReadResource(ctx context.Context, params any) (any, error) {
	if readParams, ok := params.(map[string]interface{}); ok {
		uri, _ := readParams["uri"].(string)

		// Find the resource handler
		if entry, ok := a.resources[uri]; ok {
			// Create mark3labs ReadResourceRequest
			req := mcp.ReadResourceRequest{
				Request: mcp.Request{
					Method: string(mcp.MethodResourcesRead),
				},
			}
			req.Params.URI = uri

			// Call resource handler
			contents, err := entry.handler(ctx, req)
			if err != nil {
				return nil, err
			}

			// Convert mark3labs resource contents to protocol
			protoContents := make([]modelcontextprotocol.ResourceContents, 0, len(contents))
			for _, content := range contents {
				protoContents = append(protoContents, a.convertResourceContents(content))
			}

			return modelcontextprotocol.ReadResourceResult{
				Contents: protoContents,
			}, nil
		}
	}

	return nil, fmt.Errorf("resource not found")
}

// handleListPrompts handles prompt listing requests
func (a *Mark3LabsAdapter) handleListPrompts(ctx context.Context, params any) (any, error) {
	// Convert mark3labs prompts to protocol prompts
	protoPrompts := make([]modelcontextprotocol.Prompt, 0, len(a.prompts))
	for _, entry := range a.prompts {
		prompt := entry.prompt
		protoPrompt := modelcontextprotocol.Prompt{
			Name:        prompt.Name,
			Description: prompt.Description,
		}

		// Convert arguments if present
		if len(prompt.Arguments) > 0 {
			protoPrompt.Arguments = make([]modelcontextprotocol.PromptArgument, 0, len(prompt.Arguments))
			for _, arg := range prompt.Arguments {
				protoArg := modelcontextprotocol.PromptArgument{
					Name:        arg.Name,
					Description: arg.Description,
					Required:    arg.Required,
				}
				protoPrompt.Arguments = append(protoPrompt.Arguments, protoArg)
			}
		}

		protoPrompts = append(protoPrompts, protoPrompt)
	}

	return modelcontextprotocol.ListPromptsResult{
		Prompts: protoPrompts,
	}, nil
}

// handleGetPrompt handles prompt retrieval requests
func (a *Mark3LabsAdapter) handleGetPrompt(ctx context.Context, params any) (any, error) {
	if getParams, ok := params.(map[string]interface{}); ok {
		name, _ := getParams["name"].(string)
		args, _ := getParams["arguments"].(map[string]string)

		// Find the prompt handler
		if entry, ok := a.prompts[name]; ok {
			// Create mark3labs GetPromptRequest
			req := mcp.GetPromptRequest{
				Request: mcp.Request{
					Method: string(mcp.MethodPromptsGet),
				},
			}
			req.Params.Name = name
			req.Params.Arguments = args

			// Call prompt handler
			result, err := entry.handler(ctx, req)
			if err != nil {
				return nil, err
			}

			// Convert mark3labs GetPromptResult to protocol
			return a.convertPromptResult(result), nil
		}
	}

	return nil, fmt.Errorf("prompt not found")
}

// convertToolResult converts mark3labs CallToolResult to protocol format
func (a *Mark3LabsAdapter) convertToolResult(result *mcp.CallToolResult) *modelcontextprotocol.CallToolResult {
	protoResult := &modelcontextprotocol.CallToolResult{
		Content: make([]modelcontextprotocol.Content, 0, len(result.Content)),
		IsError: result.IsError,
	}

	for _, content := range result.Content {
		protoResult.Content = append(protoResult.Content, a.convertContent(content))
	}

	return protoResult
}

// convertContent converts mark3labs Content to protocol format
func (a *Mark3LabsAdapter) convertContent(content mcp.Content) modelcontextprotocol.Content {
	switch c := content.(type) {
	case mcp.TextContent:
		return modelcontextprotocol.TextContent{
			Type: "text",
			Text: c.Text,
		}
	case mcp.ImageContent:
		return modelcontextprotocol.ImageContent{
			Type:     "image",
			Data:     c.Data,
			MimeType: c.MIMEType,
		}
	case mcp.EmbeddedResource:
		return modelcontextprotocol.ResourceContent{
			Type:     "resource",
			Resource: a.convertResourceContents(c.Resource),
		}
	default:
		// Fallback to text content
		return modelcontextprotocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("%v", content),
		}
	}
}

// convertResourceContents converts mark3labs ResourceContents to protocol format
func (a *Mark3LabsAdapter) convertResourceContents(contents mcp.ResourceContents) modelcontextprotocol.ResourceContents {
	switch c := contents.(type) {
	case mcp.TextResourceContents:
		return modelcontextprotocol.TextResourceContents{
			URI:      c.URI,
			MimeType: c.MIMEType,
			Text:     c.Text,
		}
	case mcp.BlobResourceContents:
		return modelcontextprotocol.BlobResourceContents{
			URI:      c.URI,
			MimeType: c.MIMEType,
			Blob:     c.Blob,
		}
	default:
		// Fallback to text resource contents
		return modelcontextprotocol.TextResourceContents{
			URI:      "unknown",
			MimeType: "text/plain",
			Text:     fmt.Sprintf("%v", contents),
		}
	}
}

// convertPromptResult converts mark3labs GetPromptResult to protocol format
func (a *Mark3LabsAdapter) convertPromptResult(result *mcp.GetPromptResult) *modelcontextprotocol.GetPromptResult {
	protoResult := &modelcontextprotocol.GetPromptResult{
		Description: result.Description,
		Messages:    make([]modelcontextprotocol.PromptMessage, 0, len(result.Messages)),
	}

	for _, msg := range result.Messages {
		protoMsg := modelcontextprotocol.PromptMessage{
			Role: modelcontextprotocol.Role(msg.Role),
		}

		// Convert content
		if textContent, ok := msg.Content.(mcp.TextContent); ok {
			protoMsg.Content = modelcontextprotocol.TextContent{
				Type: "text",
				Text: textContent.Text,
			}
		} else if imageContent, ok := msg.Content.(mcp.ImageContent); ok {
			protoMsg.Content = modelcontextprotocol.ImageContent{
				Type:     "image",
				Data:     imageContent.Data,
				MimeType: imageContent.MIMEType,
			}
		}

		protoResult.Messages = append(protoResult.Messages, protoMsg)
	}

	return protoResult
}

func init() {
	// Register this adapter in the default registry
	adapters.DefaultRegistry.Register("mark3labs", NewAdapter)
}
