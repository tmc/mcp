package mark3labs_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CreateMark3LabsServer creates a new mark3labs MCP server using the original API
func CreateMark3LabsServer() *server.MCPServer {
	s := server.NewMCPServer(
		"Mark3Labs Test Server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Add a test tool
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

	// Add a test resource
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

	// Add a resource template
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

	// Add a test prompt
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

func TestMark3LabsServerOriginalAPI(t *testing.T) {
	// Create a mark3labs MCP server using the original API
	s := CreateMark3LabsServer()

	// Create a stdio server to test the server
	stdioServer := server.NewStdioServer(s)

	// Create pipes for testing
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()

	// Start the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- stdioServer.Listen(ctx, inReader, outWriter)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send initialize request
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "1.0.0",
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqBytes, _ := json.Marshal(initReq)
	reqBytes = append(reqBytes, '\n') // Add newline as required by stdio protocol

	// Write request
	_, err := inWriter.Write(reqBytes)
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read response
	scanner := bufio.NewScanner(outReader)
	if !scanner.Scan() {
		t.Fatal("No response received")
	}

	response := scanner.Bytes()
	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify the response
	if result["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc 2.0, got %v", result["jsonrpc"])
	}

	if resultData, ok := result["result"].(map[string]interface{}); ok {
		if serverInfo, ok := resultData["serverInfo"].(map[string]interface{}); ok {
			if serverInfo["name"] != "Mark3Labs Test Server" {
				t.Errorf("Expected server name 'Mark3Labs Test Server', got %v", serverInfo["name"])
			}
		}

		// Verify capabilities
		if capabilities, ok := resultData["capabilities"].(map[string]interface{}); ok {
			if capabilities["tools"] == nil {
				t.Error("Expected tools capability")
			}
			if capabilities["resources"] == nil {
				t.Error("Expected resources capability")
			}
			if capabilities["prompts"] == nil {
				t.Error("Expected prompts capability")
			}
		}
	}

	// Stop the server
	cancel()
	inWriter.Close()

	// Wait for server to exit
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Server did not exit gracefully")
	}
}
