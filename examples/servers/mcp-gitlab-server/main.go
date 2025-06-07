package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func main() {
	server := mcp.NewServer(mcp.ServerOptions{
		Name:    "gitlab-server",
		Version: "1.0.0",
	})

	// GitLab Tools
	server.AddTool("search_repositories", mcp.Tool{
		Name:        "search_repositories",
		Description: "Search GitLab repositories",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for repositories",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 10)",
					"default":     10,
				},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		if args.Limit == 0 {
			args.Limit = 10
		}

		// Mock GitLab repository search
		results := []map[string]interface{}{
			{
				"id":          123,
				"name":        fmt.Sprintf("project-%s", args.Query),
				"path":        fmt.Sprintf("project-%s", strings.ToLower(args.Query)),
				"description": fmt.Sprintf("A GitLab project matching '%s'", args.Query),
				"visibility":  "public",
				"web_url":     fmt.Sprintf("https://gitlab.com/example/project-%s", strings.ToLower(args.Query)),
				"created_at":  "2024-01-01T00:00:00Z",
				"updated_at":  "2024-01-15T12:00:00Z",
			},
		}

		content, _ := json.MarshalIndent(map[string]interface{}{
			"query":   args.Query,
			"limit":   args.Limit,
			"results": results[:min(len(results), args.Limit)],
		}, "", "  ")

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: string(content),
				},
			},
		}, nil
	})

	server.AddTool("get_project", mcp.Tool{
		Name:        "get_project",
		Description: "Get GitLab project details",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "GitLab project ID or path",
				},
			},
			"required": []string{"project_id"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		var args struct {
			ProjectID string `json:"project_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		// Mock project details
		project := map[string]interface{}{
			"id":                args.ProjectID,
			"name":              "Example Project",
			"path":              "example-project",
			"description":       "An example GitLab project",
			"visibility":        "public",
			"web_url":           fmt.Sprintf("https://gitlab.com/example/%s", args.ProjectID),
			"default_branch":    "main",
			"created_at":        "2024-01-01T00:00:00Z",
			"last_activity_at":  "2024-01-15T12:00:00Z",
			"star_count":        42,
			"forks_count":       7,
			"open_issues_count": 3,
		}

		content, _ := json.MarshalIndent(project, "", "  ")

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: string(content),
				},
			},
		}, nil
	})

	server.AddTool("list_issues", mcp.Tool{
		Name:        "list_issues",
		Description: "List GitLab project issues",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "GitLab project ID or path",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Issue state: opened, closed, all",
					"enum":        []string{"opened", "closed", "all"},
					"default":     "opened",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of issues (default: 10)",
					"default":     10,
				},
			},
			"required": []string{"project_id"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		var args struct {
			ProjectID string `json:"project_id"`
			State     string `json:"state"`
			Limit     int    `json:"limit"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		if args.State == "" {
			args.State = "opened"
		}
		if args.Limit == 0 {
			args.Limit = 10
		}

		// Mock issues
		issues := []map[string]interface{}{
			{
				"id":          1,
				"iid":         1,
				"title":       "Fix authentication bug",
				"description": "Users cannot login with special characters in password",
				"state":       "opened",
				"created_at":  "2024-01-10T09:00:00Z",
				"updated_at":  "2024-01-15T10:30:00Z",
				"web_url":     fmt.Sprintf("https://gitlab.com/example/%s/-/issues/1", args.ProjectID),
				"labels":      []string{"bug", "authentication"},
				"assignees":   []string{"developer1"},
			},
			{
				"id":          2,
				"iid":         2,
				"title":       "Add dark mode support",
				"description": "Implement dark theme for better UX",
				"state":       "opened",
				"created_at":  "2024-01-12T14:20:00Z",
				"updated_at":  "2024-01-14T16:45:00Z",
				"web_url":     fmt.Sprintf("https://gitlab.com/example/%s/-/issues/2", args.ProjectID),
				"labels":      []string{"enhancement", "ui"},
				"assignees":   []string{},
			},
		}

		// Filter by state
		if args.State != "all" {
			filtered := []map[string]interface{}{}
			for _, issue := range issues {
				if issue["state"] == args.State {
					filtered = append(filtered, issue)
				}
			}
			issues = filtered
		}

		content, _ := json.MarshalIndent(map[string]interface{}{
			"project_id": args.ProjectID,
			"state":      args.State,
			"limit":      args.Limit,
			"issues":     issues[:min(len(issues), args.Limit)],
		}, "", "  ")

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: string(content),
				},
			},
		}, nil
	})

	server.AddTool("create_issue", mcp.Tool{
		Name:        "create_issue",
		Description: "Create a new GitLab issue",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "GitLab project ID or path",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Issue title",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Issue description",
				},
				"labels": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Issue labels",
				},
			},
			"required": []string{"project_id", "title"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		var args struct {
			ProjectID   string   `json:"project_id"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Labels      []string `json:"labels"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		// Mock issue creation
		newIssue := map[string]interface{}{
			"id":          999,
			"iid":         999,
			"title":       args.Title,
			"description": args.Description,
			"state":       "opened",
			"created_at":  "2024-01-15T12:00:00Z",
			"updated_at":  "2024-01-15T12:00:00Z",
			"web_url":     fmt.Sprintf("https://gitlab.com/example/%s/-/issues/999", args.ProjectID),
			"labels":      args.Labels,
			"assignees":   []string{},
		}

		content, _ := json.MarshalIndent(map[string]interface{}{
			"message": "Issue created successfully",
			"issue":   newIssue,
		}, "", "  ")

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: string(content),
				},
			},
		}, nil
	})

	if err := server.Serve(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
