package main

import (
	"fmt"
	"log"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func main() {
	// Create a simple echo server for testing
	handler := &echoHandler{}

	server := mcp.NewServer(handler,
		mcp.WithServerInfo(modelcontextprotocol.ServerInfo{
			Name:    "test-echo-server",
			Version: "1.0.0",
		}),
	)

	if err := server.ServeStdio(); err != nil {
		log.Fatal(err)
	}
}

type echoHandler struct{}

func (h *echoHandler) Initialize(request modelcontextprotocol.InitializeRequest) (modelcontextprotocol.InitializeResult, error) {
	return modelcontextprotocol.InitializeResult{
		ServerInfo: modelcontextprotocol.ServerInfo{
			Name:    "test-echo-server",
			Version: "1.0.0",
		},
		Capabilities: modelcontextprotocol.ServerCapabilities{
			Tools: &modelcontextprotocol.ToolsServerCapabilities{
				Tools: []modelcontextprotocol.Tool{
					{
						Name:        "echo",
						Description: "Echo back the input",
						InputSchema: modelcontextprotocol.ToolInputSchema{
							"type": "object",
							"properties": map[string]any{
								"message": map[string]any{
									"type":        "string",
									"description": "Message to echo",
								},
							},
							"required": []string{"message"},
						},
					},
				},
			},
		},
	}, nil
}

func (h *echoHandler) CallTool(request modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
	if request.Name != "echo" {
		return nil, fmt.Errorf("unknown tool: %s", request.Name)
	}

	args := request.Arguments
	message, ok := args["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message must be a string")
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: modelcontextprotocol.ContentTypeText,
				Text: message,
			},
		},
	}, nil
}

func (h *echoHandler) ListTools() ([]modelcontextprotocol.Tool, error) {
	return []modelcontextprotocol.Tool{
		{
			Name:        "echo",
			Description: "Echo back the input",
			InputSchema: modelcontextprotocol.ToolInputSchema{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "Message to echo",
					},
				},
				"required": []string{"message"},
			},
		},
	}, nil
}
