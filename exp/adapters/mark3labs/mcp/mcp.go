// Package mcp provides a compatibility layer for mark3labs-mcp-go types
// This allows existing mark3labs servers to use the adapter with just an import change
package mcp

import "github.com/mark3labs/mcp-go/mcp"

// Re-export all mark3labs mcp types
type (
	// Core types
	Tool             = mcp.Tool
	Resource         = mcp.Resource
	ResourceTemplate = mcp.ResourceTemplate
	Prompt           = mcp.Prompt
	PromptArgument   = mcp.PromptArgument
	PromptMessage    = mcp.PromptMessage

	// Request types
	CallToolRequest     = mcp.CallToolRequest
	ReadResourceRequest = mcp.ReadResourceRequest
	GetPromptRequest    = mcp.GetPromptRequest

	// Response types
	CallToolResult       = mcp.CallToolResult
	ResourceContents     = mcp.ResourceContents
	TextResourceContents = mcp.TextResourceContents
	BlobResourceContents = mcp.BlobResourceContents
	GetPromptResult      = mcp.GetPromptResult

	// Content types
	TextContent  = mcp.TextContent
	ImageContent = mcp.ImageContent

	// Other types
	Role = mcp.Role
)

// Re-export constants
const (
	RoleUser      = mcp.RoleUser
	RoleAssistant = mcp.RoleAssistant
)

// Re-export all mcp functions
var (
	// Tool functions
	NewTool         = mcp.NewTool
	WithDescription = mcp.WithDescription
	WithNumber      = mcp.WithNumber
	WithString      = mcp.WithString
	WithBoolean     = mcp.WithBoolean
	WithObject      = mcp.WithObject
	WithArray       = mcp.WithArray
	WithEnum        = mcp.WithEnum
	Required        = mcp.Required
	Description     = mcp.Description

	// Resource functions
	NewResource         = mcp.NewResource
	WithMIMEType        = mcp.WithMIMEType
	NewResourceTemplate = mcp.NewResourceTemplate

	// Prompt functions
	NewPrompt             = mcp.NewPrompt
	WithPromptDescription = mcp.WithPromptDescription
	WithArgument          = mcp.WithArgument
	ArgumentDescription   = mcp.ArgumentDescription
	RequiredArgument      = mcp.RequiredArgument

	// Result functions
	NewToolResultText  = mcp.NewToolResultText
	NewToolResultError = mcp.NewToolResultError
)
