package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/mcptel"
)

var (
	verbose = flag.Bool("v", false, "Enable verbose logging")
	quiet   = flag.Bool("q", false, "Enable quiet mode")
)

const (
	ServerName    = "mcp-everart-server"
	ServerVersion = "0.1.0"
)

func main() {
	flag.Parse()
	slog.SetDefault(slog.Default())
	// Redirect logs to stderr to keep stdout clean for the protocol
	if *verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else if *quiet {
		slog.SetLogLoggerLevel(slog.LevelError)
	}
	log.Println("starting MCP EverArt Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A MCP server for AI image generation using EverArt API"),
		// mcp.WithDefaultTracer()
	)

	// Register the everart tools
	registerTools(server)

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerTools(server *mcp.Server) {
	// Register generate_image tool
	generateImageTool := mcp.Tool{
		Name:        "generate_image",
		Description: "Generate an AI image using EverArt API",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"prompt": {
					"type": "string",
					"description": "The text prompt for image generation"
				},
				"style": {
					"type": "string",
					"description": "Art style (e.g., 'photorealistic', 'anime', 'digital_art')",
					"default": "photorealistic"
				},
				"size": {
					"type": "string",
					"description": "Image size (e.g., '512x512', '1024x1024')",
					"default": "1024x1024"
				}
			},
			"required": ["prompt"]
		}`),
	}

	server.RegisterTool(generateImageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "generate_image tool called")
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		prompt, ok := params["prompt"].(string)
		if !ok || prompt == "" {
			return nil, fmt.Errorf("prompt is required and must be a string")
		}

		style := "photorealistic"
		if s, ok := params["style"].(string); ok && s != "" {
			style = s
		}

		size := "1024x1024"
		if s, ok := params["size"].(string); ok && s != "" {
			size = s
		}

		// Mock image generation - in real implementation, would call EverArt API
		imageUrl := fmt.Sprintf("https://api.everart.ai/generated/%s_%s_%s.png",
			prompt[:min(20, len(prompt))], style, size)

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Generated image with prompt: %q\nStyle: %s\nSize: %s\nImage URL: %s",
						prompt, style, size, imageUrl),
				},
			},
		}, nil
	})

	// Register list_models tool
	listModelsTool := mcp.Tool{
		Name:        "list_models",
		Description: "List available AI models for image generation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}

	server.RegisterTool(listModelsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "list_models tool called")

		models := []string{
			"stable-diffusion-xl",
			"midjourney-v6",
			"dall-e-3",
			"leonardo-ai",
			"playground-v2",
		}

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Available models: %v", models),
				},
			},
		}, nil
	})

	log.Println("Registered EverArt tools: generate_image, list_models")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
