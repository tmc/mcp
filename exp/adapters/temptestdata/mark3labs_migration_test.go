package mark3labs_test

import (
	"context"
	"fmt"
	"testing"

	// Step 1: Original import for mark3labs-mcp-go
	// "github.com/mark3labs/mcp-go/mcp"
	// "github.com/mark3labs/mcp-go/server"

	// Step 2: Replace with adapter import (single-line change!)
	"github.com/tmc/mcprepos/mcp/adapters/mark3labs/mcp"
	"github.com/tmc/mcprepos/mcp/adapters/mark3labs/server"
)

// CreateServerWithAdapter creates a server using the adapter-wrapped API
// NO CHANGES NEEDED TO THE SERVER CODE!
func CreateServerWithAdapter() *server.MCPServer {
	s := server.NewMCPServer(
		"Mark3Labs Test Server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Add a test tool - SAME CODE AS BEFORE
	testTool := mcp.NewTool("test_tool",
		mcp.WithDescription("A test tool that adds two numbers"),
		mcp.WithNumber("a",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("b",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	s.AddTool(testTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		a, ok := request.Params.Arguments["a"].(float64)
		if !ok {
			return mcp.NewToolResultError("a must be a number"), nil
		}
		b, ok := request.Params.Arguments["b"].(float64)
		if !ok {
			return mcp.NewToolResultError("b must be a number"), nil
		}

		result := a + b
		return mcp.NewToolResultText(fmt.Sprintf("Result: %f", result)), nil
	})

	// Add a test resource - SAME CODE AS BEFORE
	s.AddResource(
		mcp.NewResource("test://resource/1",
			"Test Resource",
			mcp.WithMIMEType("text/plain"),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Content for %s", request.Params.URI),
				},
			}, nil
		},
	)

	// Add a resource template - SAME CODE AS BEFORE
	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"test://resource/{id}",
			"Test resource with ID",
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Content for %s", request.Params.URI),
				},
			}, nil
		},
	)

	// Add a test prompt - SAME CODE AS BEFORE
	s.AddPrompt(
		mcp.NewPrompt("test_prompt",
			mcp.WithPromptDescription("A test prompt with placeholders"),
			mcp.WithArgument("topic",
				mcp.ArgumentDescription("The topic to ask about"),
				mcp.RequiredArgument(),
			),
		),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			topic := request.Params.Arguments["topic"]
			return &mcp.GetPromptResult{
				Description: "A test prompt about a topic",
				Messages: []mcp.PromptMessage{
					{
						Role: mcp.RoleUser,
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Tell me about %s", topic),
						},
					},
				},
			}, nil
		},
	)

	return s
}

func TestMark3LabsServerWithAdapter(t *testing.T) {
	// Create a server using the adapter
	s := CreateServerWithAdapter()

	// The underlying adapter is now compatible with the standard SDK
	adapter := server.GetAdapter(s)
	if adapter == nil {
		t.Fatal("Expected adapter to be registered")
	}

	// Verify we have the expected capabilities
	capabilities := adapter.GetCapabilities()

	if capabilities.Tools == nil {
		t.Error("Expected tools capability")
	}

	if capabilities.Resources == nil || !capabilities.Resources.Subscribe {
		t.Error("Expected resources capability with subscribe")
	}

	if capabilities.Prompts == nil || !capabilities.Prompts.ListChanged {
		t.Error("Expected prompts capability with list changed")
	}

	t.Log("Successfully created server with adapter - ready for standard SDK!")
}