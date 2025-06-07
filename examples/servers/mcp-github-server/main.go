package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
	"golang.org/x/oauth2"
)

type GitHubServer struct {
	client *github.Client
	token  string
}

func NewGitHubServer(token string) (*GitHubServer, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &GitHubServer{
		client: client,
		token:  token,
	}, nil
}

func (gs *GitHubServer) getRepository(owner, repo string) (string, error) {
	repository, _, err := gs.client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Repository: %s/%s\n", owner, repo))
	result.WriteString(fmt.Sprintf("Description: %s\n", repository.GetDescription()))
	result.WriteString(fmt.Sprintf("Language: %s\n", repository.GetLanguage()))
	result.WriteString(fmt.Sprintf("Stars: %d\n", repository.GetStargazersCount()))
	result.WriteString(fmt.Sprintf("Forks: %d\n", repository.GetForksCount()))
	result.WriteString(fmt.Sprintf("Open Issues: %d\n", repository.GetOpenIssuesCount()))
	result.WriteString(fmt.Sprintf("Default Branch: %s\n", repository.GetDefaultBranch()))
	result.WriteString(fmt.Sprintf("Created: %s\n", repository.GetCreatedAt().Format("2006-01-02")))
	result.WriteString(fmt.Sprintf("Updated: %s\n", repository.GetUpdatedAt().Format("2006-01-02")))
	result.WriteString(fmt.Sprintf("Clone URL: %s\n", repository.GetCloneURL()))

	return result.String(), nil
}

func (gs *GitHubServer) listRepositories(owner string, limit int) (string, error) {
	if limit <= 0 {
		limit = 30
	}

	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: limit},
	}

	var repos []*github.Repository
	var err error

	if owner == "" {
		// List authenticated user's repositories
		repos, _, err = gs.client.Repositories.List(context.Background(), "", opts)
	} else {
		// List specific user's repositories
		repos, _, err = gs.client.Repositories.List(context.Background(), owner, opts)
	}

	if err != nil {
		return "", fmt.Errorf("failed to list repositories: %w", err)
	}

	var result strings.Builder
	if owner == "" {
		result.WriteString("Your repositories:\n")
	} else {
		result.WriteString(fmt.Sprintf("Repositories for %s:\n", owner))
	}
	result.WriteString("Name\t\tLanguage\tStars\tDescription\n")
	result.WriteString("----\t\t--------\t-----\t-----------\n")

	for _, repo := range repos {
		description := repo.GetDescription()
		if len(description) > 50 {
			description = description[:47] + "..."
		}
		result.WriteString(fmt.Sprintf("%s\t\t%s\t\t%d\t%s\n",
			repo.GetName(),
			repo.GetLanguage(),
			repo.GetStargazersCount(),
			description))
	}

	return result.String(), nil
}

func (gs *GitHubServer) getFileContent(owner, repo, path, ref string) (string, error) {
	opts := &github.RepositoryContentGetOptions{}
	if ref != "" {
		opts.Ref = ref
	}

	fileContent, _, _, err := gs.client.Repositories.GetContents(context.Background(), owner, repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("File: %s/%s/%s\n", owner, repo, path))
	result.WriteString(fmt.Sprintf("Size: %d bytes\n", fileContent.GetSize()))
	result.WriteString(fmt.Sprintf("SHA: %s\n", fileContent.GetSHA()))
	result.WriteString("Content:\n")
	result.WriteString("--------\n")
	result.WriteString(content)

	return result.String(), nil
}

func (gs *GitHubServer) listBranches(owner, repo string) (string, error) {
	branches, _, err := gs.client.Repositories.ListBranches(context.Background(), owner, repo, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list branches: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Branches for %s/%s:\n", owner, repo))
	result.WriteString("Name\t\tCommit SHA\n")
	result.WriteString("----\t\t----------\n")

	for _, branch := range branches {
		result.WriteString(fmt.Sprintf("%s\t\t%s\n",
			branch.GetName(),
			branch.GetCommit().GetSHA()[:8]))
	}

	return result.String(), nil
}

func (gs *GitHubServer) createIssue(owner, repo, title, body string) (string, error) {
	issueRequest := &github.IssueRequest{
		Title: &title,
		Body:  &body,
	}

	issue, _, err := gs.client.Issues.Create(context.Background(), owner, repo, issueRequest)
	if err != nil {
		return "", fmt.Errorf("failed to create issue: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Created issue #%d\n", issue.GetNumber()))
	result.WriteString(fmt.Sprintf("Title: %s\n", issue.GetTitle()))
	result.WriteString(fmt.Sprintf("URL: %s\n", issue.GetHTMLURL()))
	result.WriteString(fmt.Sprintf("State: %s\n", issue.GetState()))

	return result.String(), nil
}

func (gs *GitHubServer) listIssues(owner, repo string, state string, limit int) (string, error) {
	if limit <= 0 {
		limit = 30
	}

	opts := &github.IssueListByRepoOptions{
		State:       state,
		ListOptions: github.ListOptions{PerPage: limit},
	}

	issues, _, err := gs.client.Issues.ListByRepo(context.Background(), owner, repo, opts)
	if err != nil {
		return "", fmt.Errorf("failed to list issues: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Issues for %s/%s (state: %s):\n", owner, repo, state))
	result.WriteString("Number\tState\tTitle\t\tAuthor\n")
	result.WriteString("------\t-----\t-----\t\t------\n")

	for _, issue := range issues {
		title := issue.GetTitle()
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		result.WriteString(fmt.Sprintf("#%d\t%s\t%s\t\t%s\n",
			issue.GetNumber(),
			issue.GetState(),
			title,
			issue.GetUser().GetLogin()))
	}

	return result.String(), nil
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// Initialize GitHub server
	gs, err := NewGitHubServer(token)
	if err != nil {
		log.Fatalf("Failed to initialize GitHub server: %v", err)
	}

	// Create server with name and version
	srv := mcp.NewServer("github-server", "1.0.0")

	// Register get_repository tool
	srv.RegisterTool("get_repository", "Get information about a GitHub repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var ownerRaw, repoRaw json.RawMessage
		var exists bool

		if ownerRaw, exists = args["owner"]; !exists {
			return nil, fmt.Errorf("missing required argument: owner")
		}

		if repoRaw, exists = args["repo"]; !exists {
			return nil, fmt.Errorf("missing required argument: repo")
		}

		var owner, repo string
		if err := json.Unmarshal(ownerRaw, &owner); err != nil {
			return nil, fmt.Errorf("invalid owner argument: %w", err)
		}

		if err := json.Unmarshal(repoRaw, &repo); err != nil {
			return nil, fmt.Errorf("invalid repo argument: %w", err)
		}

		result, err := gs.getRepository(owner, repo)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting repository: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Retrieved repository info: %s/%s", owner, repo)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register list_repositories tool
	srv.RegisterTool("list_repositories", "List GitHub repositories for a user or organization", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var owner string
		if ownerRaw, exists := args["owner"]; exists {
			if err := json.Unmarshal(ownerRaw, &owner); err != nil {
				return nil, fmt.Errorf("invalid owner argument: %w", err)
			}
		}

		var limit int = 30
		if limitRaw, exists := args["limit"]; exists {
			if err := json.Unmarshal(limitRaw, &limit); err != nil {
				return nil, fmt.Errorf("invalid limit argument: %w", err)
			}
		}

		result, err := gs.listRepositories(owner, limit)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing repositories: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Listed repositories for owner: %s", owner)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register get_file_content tool
	srv.RegisterTool("get_file_content", "Get the content of a file from a GitHub repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var ownerRaw, repoRaw, pathRaw json.RawMessage
		var exists bool

		if ownerRaw, exists = args["owner"]; !exists {
			return nil, fmt.Errorf("missing required argument: owner")
		}

		if repoRaw, exists = args["repo"]; !exists {
			return nil, fmt.Errorf("missing required argument: repo")
		}

		if pathRaw, exists = args["path"]; !exists {
			return nil, fmt.Errorf("missing required argument: path")
		}

		var owner, repo, path string
		if err := json.Unmarshal(ownerRaw, &owner); err != nil {
			return nil, fmt.Errorf("invalid owner argument: %w", err)
		}

		if err := json.Unmarshal(repoRaw, &repo); err != nil {
			return nil, fmt.Errorf("invalid repo argument: %w", err)
		}

		if err := json.Unmarshal(pathRaw, &path); err != nil {
			return nil, fmt.Errorf("invalid path argument: %w", err)
		}

		var ref string
		if refRaw, exists := args["ref"]; exists {
			if err := json.Unmarshal(refRaw, &ref); err != nil {
				return nil, fmt.Errorf("invalid ref argument: %w", err)
			}
		}

		result, err := gs.getFileContent(owner, repo, path, ref)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting file content: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Retrieved file content: %s/%s/%s", owner, repo, path)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register list_branches tool
	srv.RegisterTool("list_branches", "List branches in a GitHub repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var ownerRaw, repoRaw json.RawMessage
		var exists bool

		if ownerRaw, exists = args["owner"]; !exists {
			return nil, fmt.Errorf("missing required argument: owner")
		}

		if repoRaw, exists = args["repo"]; !exists {
			return nil, fmt.Errorf("missing required argument: repo")
		}

		var owner, repo string
		if err := json.Unmarshal(ownerRaw, &owner); err != nil {
			return nil, fmt.Errorf("invalid owner argument: %w", err)
		}

		if err := json.Unmarshal(repoRaw, &repo); err != nil {
			return nil, fmt.Errorf("invalid repo argument: %w", err)
		}

		result, err := gs.listBranches(owner, repo)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing branches: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Listed branches for: %s/%s", owner, repo)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register create_issue tool
	srv.RegisterTool("create_issue", "Create a new GitHub issue", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var ownerRaw, repoRaw, titleRaw json.RawMessage
		var exists bool

		if ownerRaw, exists = args["owner"]; !exists {
			return nil, fmt.Errorf("missing required argument: owner")
		}

		if repoRaw, exists = args["repo"]; !exists {
			return nil, fmt.Errorf("missing required argument: repo")
		}

		if titleRaw, exists = args["title"]; !exists {
			return nil, fmt.Errorf("missing required argument: title")
		}

		var owner, repo, title string
		if err := json.Unmarshal(ownerRaw, &owner); err != nil {
			return nil, fmt.Errorf("invalid owner argument: %w", err)
		}

		if err := json.Unmarshal(repoRaw, &repo); err != nil {
			return nil, fmt.Errorf("invalid repo argument: %w", err)
		}

		if err := json.Unmarshal(titleRaw, &title); err != nil {
			return nil, fmt.Errorf("invalid title argument: %w", err)
		}

		var body string
		if bodyRaw, exists := args["body"]; exists {
			if err := json.Unmarshal(bodyRaw, &body); err != nil {
				return nil, fmt.Errorf("invalid body argument: %w", err)
			}
		}

		result, err := gs.createIssue(owner, repo, title, body)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error creating issue: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Created issue in: %s/%s", owner, repo)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register list_issues tool
	srv.RegisterTool("list_issues", "List issues in a GitHub repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var ownerRaw, repoRaw json.RawMessage
		var exists bool

		if ownerRaw, exists = args["owner"]; !exists {
			return nil, fmt.Errorf("missing required argument: owner")
		}

		if repoRaw, exists = args["repo"]; !exists {
			return nil, fmt.Errorf("missing required argument: repo")
		}

		var owner, repo string
		if err := json.Unmarshal(ownerRaw, &owner); err != nil {
			return nil, fmt.Errorf("invalid owner argument: %w", err)
		}

		if err := json.Unmarshal(repoRaw, &repo); err != nil {
			return nil, fmt.Errorf("invalid repo argument: %w", err)
		}

		var state string = "open"
		if stateRaw, exists := args["state"]; exists {
			if err := json.Unmarshal(stateRaw, &state); err != nil {
				return nil, fmt.Errorf("invalid state argument: %w", err)
			}
		}

		var limit int = 30
		if limitRaw, exists := args["limit"]; exists {
			if err := json.Unmarshal(limitRaw, &limit); err != nil {
				return nil, fmt.Errorf("invalid limit argument: %w", err)
			}
		}

		result, err := gs.listIssues(owner, repo, state, limit)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing issues: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Listed issues for: %s/%s", owner, repo)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Start server with stdio transport
	transport := mcp.StdioTransport{}
	log.Printf("GitHub server running on stdio")

	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
