// Package main demonstrates a simple MCP server using the stdlib-idiomatic SDK2 API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tmc/mcp/exp/sdk2"
)

// EchoTool implements a simple echo tool
type EchoTool struct{}

func (e *EchoTool) HandleTool(ctx context.Context, call *sdk2.ToolCall) (*sdk2.ToolResult, error) {
	message, ok := call.Arguments["message"].(string)
	if !ok {
		return &sdk2.ToolResult{
			IsError: true,
			Content: []sdk2.Content{
				sdk2.TextContent{Text: "message parameter is required and must be a string"},
			},
		}, nil
	}

	return &sdk2.ToolResult{
		Content: []sdk2.Content{
			sdk2.TextContent{Text: fmt.Sprintf("Echo: %s", message)},
		},
	}, nil
}

func (e *EchoTool) Schema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The message to echo back",
			},
		},
		"required": []string{"message"},
	}
	bytes, _ := json.Marshal(schema)
	return json.RawMessage(bytes)
}

func (e *EchoTool) Description() string {
	return "Echoes back the input message with 'Echo: ' prefix"
}

// GreetingResource provides a simple greeting resource
type GreetingResource struct{}

func (g *GreetingResource) HandleResource(ctx context.Context, req *sdk2.ResourceRequest) (*sdk2.ResourceContent, error) {
	return &sdk2.ResourceContent{
		URI:      req.URI,
		MimeType: "text/plain",
		Content: []sdk2.Content{
			sdk2.TextContent{Text: "Hello from the echo server resource!"},
		},
	}, nil
}

func (g *GreetingResource) Metadata() sdk2.Resource {
	return sdk2.Resource{
		URI:         "greeting://hello",
		Name:        "Greeting Resource",
		Description: "A simple greeting resource",
		MimeType:    "text/plain",
	}
}

func main() {
	// Create a new server
	server := sdk2.NewServer()

	// Register tools/list handler
	sdk2.HandleFunc(sdk2.MethodToolsList, func(w sdk2.ResponseWriter, r *sdk2.Request) {
		tools := []sdk2.Tool{
			{
				Name:        "echo",
				Description: "Echoes back the input message with 'Echo: ' prefix",
				InputSchema: (&EchoTool{}).Schema(),
			},
		}

		result := struct {
			Tools []sdk2.Tool `json:"tools"`
		}{
			Tools: tools,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(result)
	})

	// Register tools/call handler
	sdk2.HandleFunc(sdk2.MethodToolsCall, func(w sdk2.ResponseWriter, r *sdk2.Request) {
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}

		if r.Params != nil {
			if err := json.Unmarshal(r.Params, &params); err != nil {
				sdk2.Error(w, "Invalid tool call parameters", sdk2.StatusBadRequest)
				return
			}
		}

		// Handle echo tool
		if params.Name == "echo" {
			tool := &EchoTool{}
			call := &sdk2.ToolCall{
				Name:      params.Name,
				Arguments: params.Arguments,
			}

			result, err := tool.HandleTool(r.Context, call)
			if err != nil {
				sdk2.Error(w, fmt.Sprintf("Tool execution failed: %v", err), sdk2.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(sdk2.StatusOK)
			json.NewEncoder(w).Encode(result)
			return
		}

		sdk2.Error(w, fmt.Sprintf("Unknown tool: %s", params.Name), sdk2.StatusNotFound)
	})

	// Register resources/list handler
	sdk2.HandleFunc(sdk2.MethodResourcesList, func(w sdk2.ResponseWriter, r *sdk2.Request) {
		resources := []sdk2.Resource{
			{
				URI:         "greeting://hello",
				Name:        "Greeting Resource",
				Description: "A simple greeting resource",
				MimeType:    "text/plain",
			},
		}

		result := struct {
			Resources []sdk2.Resource `json:"resources"`
		}{
			Resources: resources,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(sdk2.StatusOK)
		json.NewEncoder(w).Encode(result)
	})

	// Register resources/read handler
	sdk2.HandleFunc(sdk2.MethodResourcesRead, func(w sdk2.ResponseWriter, r *sdk2.Request) {
		var params struct {
			URI string `json:"uri"`
		}

		if r.Params != nil {
			if err := json.Unmarshal(r.Params, &params); err != nil {
				sdk2.Error(w, "Invalid resource read parameters", sdk2.StatusBadRequest)
				return
			}
		}

		// Handle greeting resource
		if params.URI == "greeting://hello" {
			resource := &GreetingResource{}
			req := &sdk2.ResourceRequest{URI: params.URI}

			content, err := resource.HandleResource(r.Context, req)
			if err != nil {
				sdk2.Error(w, fmt.Sprintf("Resource read failed: %v", err), sdk2.StatusInternalServerError)
				return
			}

			result := struct {
				Contents []sdk2.ResourceContent `json:"contents"`
			}{
				Contents: []sdk2.ResourceContent{*content},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(sdk2.StatusOK)
			json.NewEncoder(w).Encode(result)
			return
		}

		sdk2.Error(w, fmt.Sprintf("Unknown resource: %s", params.URI), sdk2.StatusNotFound)
	})

	log.Printf("Starting echo server with SDK2...")

	// Start the server using stdio transport
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
		os.Exit(1)
	}
}
