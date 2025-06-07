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
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/mcptel"
)

var (
	verbose = flag.Bool("v", false, "Enable verbose logging")
	quiet   = flag.Bool("q", false, "Enable quiet mode")
)

const (
	ServerName    = "mcp-sentry-server"
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
	log.Println("starting MCP Sentry Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A MCP server for error monitoring and tracking with Sentry"),
		// mcp.WithDefaultTracer()
	)

	// Register the sentry tools
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
	// Register capture_exception tool
	captureExceptionTool := mcp.Tool{
		Name:        "capture_exception",
		Description: "Capture and track an exception in Sentry",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"exception": {
					"type": "string",
					"description": "The exception message or error details"
				},
				"level": {
					"type": "string",
					"description": "Error level (error, warning, info, debug)",
					"default": "error"
				},
				"tags": {
					"type": "object",
					"description": "Additional tags for the error"
				},
				"user": {
					"type": "object",
					"description": "User information associated with the error"
				}
			},
			"required": ["exception"]
		}`),
	}

	server.RegisterTool(captureExceptionTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "capture_exception tool called")
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		exception, ok := params["exception"].(string)
		if !ok || exception == "" {
			return nil, fmt.Errorf("exception is required and must be a string")
		}

		level := "error"
		if l, ok := params["level"].(string); ok && l != "" {
			level = l
		}

		// Mock Sentry capture - in real implementation, would call Sentry API
		eventID := fmt.Sprintf("sentry-%d", time.Now().Unix())

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Exception captured in Sentry\nEvent ID: %s\nLevel: %s\nException: %s",
						eventID, level, exception),
				},
			},
		}, nil
	})

	// Register list_issues tool
	listIssuesTool := mcp.Tool{
		Name:        "list_issues",
		Description: "List recent issues from Sentry",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"project": {
					"type": "string",
					"description": "Project name to filter issues"
				},
				"limit": {
					"type": "integer",
					"description": "Number of issues to return",
					"default": 10
				},
				"status": {
					"type": "string",
					"description": "Issue status filter (unresolved, resolved, ignored)"
				}
			}
		}`),
	}

	server.RegisterTool(listIssuesTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "list_issues tool called")
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		limit := 10
		if l, ok := params["limit"].(float64); ok {
			limit = int(l)
		}

		project := "default"
		if p, ok := params["project"].(string); ok && p != "" {
			project = p
		}

		// Mock issues - in real implementation, would call Sentry API
		issues := make([]string, 0, limit)
		for i := 0; i < limit; i++ {
			issues = append(issues, fmt.Sprintf("Issue #%d: Error in %s - %d events",
				i+1, project, (i+1)*5))
		}

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Recent issues in project %s:\n%v", project, issues),
				},
			},
		}, nil
	})

	// Register create_release tool
	createReleaseTool := mcp.Tool{
		Name:        "create_release",
		Description: "Create a new release in Sentry for tracking deployments",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"version": {
					"type": "string",
					"description": "Release version identifier"
				},
				"project": {
					"type": "string",
					"description": "Project name for the release"
				},
				"ref": {
					"type": "string",
					"description": "Git commit reference"
				}
			},
			"required": ["version"]
		}`),
	}

	server.RegisterTool(createReleaseTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "create_release tool called")
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		version, ok := params["version"].(string)
		if !ok || version == "" {
			return nil, fmt.Errorf("version is required and must be a string")
		}

		project := "default"
		if p, ok := params["project"].(string); ok && p != "" {
			project = p
		}

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Release created successfully\nVersion: %s\nProject: %s\nTimestamp: %s",
						version, project, time.Now().Format(time.RFC3339)),
				},
			},
		}, nil
	})

	log.Println("Registered Sentry tools: capture_exception, list_issues, create_release")
}
