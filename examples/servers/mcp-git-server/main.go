package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type GitServer struct {
	allowedRepositories []string
}

func NewGitServer(allowedRepos []string) (*GitServer, error) {
	var normalizedRepos []string
	
	for _, repo := range allowedRepos {
		// Expand home directory
		if strings.HasPrefix(repo, "~/") || repo == "~" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			if repo == "~" {
				repo = homeDir
			} else {
				repo = filepath.Join(homeDir, repo[2:])
			}
		}
		
		// Resolve to absolute path
		absRepo, err := filepath.Abs(repo)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve repository %s: %w", repo, err)
		}
		
		// Validate repository exists and is a git repository
		if stat, err := os.Stat(absRepo); err != nil {
			return nil, fmt.Errorf("repository %s does not exist: %w", absRepo, err)
		} else if !stat.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", absRepo)
		}
		
		// Check if it's a git repository
		gitDir := filepath.Join(absRepo, ".git")
		if _, err := os.Stat(gitDir); err != nil {
			return nil, fmt.Errorf("%s is not a git repository", absRepo)
		}
		
		normalizedRepos = append(normalizedRepos, filepath.Clean(absRepo))
	}
	
	return &GitServer{
		allowedRepositories: normalizedRepos,
	}, nil
}

func (gs *GitServer) validateRepository(repoPath string) (string, error) {
	// Expand home directory
	if strings.HasPrefix(repoPath, "~/") || repoPath == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		if repoPath == "~" {
			repoPath = homeDir
		} else {
			repoPath = filepath.Join(homeDir, repoPath[2:])
		}
	}
	
	// Convert to absolute path
	var absPath string
	if filepath.IsAbs(repoPath) {
		absPath = filepath.Clean(repoPath)
	} else {
		wd, _ := os.Getwd()
		absPath = filepath.Clean(filepath.Join(wd, repoPath))
	}
	
	// Check if path is within allowed repositories
	allowed := false
	for _, allowedRepo := range gs.allowedRepositories {
		if strings.HasPrefix(absPath, allowedRepo) {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return "", fmt.Errorf("access denied - repository outside allowed repositories: %s", absPath)
	}
	
	return absPath, nil
}

func (gs *GitServer) runGitCommand(repoPath string, args ...string) (string, error) {
	validRepo, err := gs.validateRepository(repoPath)
	if err != nil {
		return "", err
	}
	
	cmd := exec.Command("git", args...)
	cmd.Dir = validRepo
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %s", string(output))
	}
	
	return string(output), nil
}

func (gs *GitServer) getStatus(repoPath string) (string, error) {
	return gs.runGitCommand(repoPath, "status", "--porcelain")
}

func (gs *GitServer) getLog(repoPath string, maxCount int) (string, error) {
	args := []string{"log", "--oneline"}
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", maxCount))
	}
	return gs.runGitCommand(repoPath, args...)
}

func (gs *GitServer) getBranches(repoPath string) (string, error) {
	return gs.runGitCommand(repoPath, "branch", "-a")
}

func (gs *GitServer) getDiff(repoPath string, ref1, ref2 string) (string, error) {
	args := []string{"diff"}
	if ref1 != "" {
		args = append(args, ref1)
		if ref2 != "" {
			args = append(args, ref2)
		}
	}
	return gs.runGitCommand(repoPath, args...)
}

func (gs *GitServer) createBranch(repoPath, branchName string) (string, error) {
	return gs.runGitCommand(repoPath, "checkout", "-b", branchName)
}

func (gs *GitServer) switchBranch(repoPath, branchName string) (string, error) {
	return gs.runGitCommand(repoPath, "checkout", branchName)
}

func (gs *GitServer) addFiles(repoPath string, files []string) (string, error) {
	args := append([]string{"add"}, files...)
	return gs.runGitCommand(repoPath, args...)
}

func (gs *GitServer) commit(repoPath, message string) (string, error) {
	return gs.runGitCommand(repoPath, "commit", "-m", message)
}

func (gs *GitServer) push(repoPath, remote, branch string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	if branch == "" {
		branch = "HEAD"
	}
	return gs.runGitCommand(repoPath, "push", remote, branch)
}

func (gs *GitServer) pull(repoPath, remote, branch string) (string, error) {
	args := []string{"pull"}
	if remote != "" {
		args = append(args, remote)
		if branch != "" {
			args = append(args, branch)
		}
	}
	return gs.runGitCommand(repoPath, args...)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("Usage: mcp-git-server <allowed-repository> [additional-repositories...]")
	}
	
	// Initialize git server
	gs, err := NewGitServer(args)
	if err != nil {
		log.Fatalf("Failed to initialize git server: %v", err)
	}
	
	// Create server with name and version
	srv := mcp.NewServer("git-server", "1.0.0")
	
	// Register git_status tool
	srv.RegisterTool("git_status", "Get the status of a git repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw json.RawMessage
		var exists bool
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		var repoPath string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		status, err := gs.getStatus(repoPath)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting git status: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Git status for repository: %s", repoPath)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: status,
				},
			},
		}, nil
	})
	
	// Register git_log tool
	srv.RegisterTool("git_log", "Get the commit log of a git repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw json.RawMessage
		var exists bool
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		var repoPath string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		var maxCount int = 10 // default
		if maxCountRaw, exists := args["max_count"]; exists {
			if err := json.Unmarshal(maxCountRaw, &maxCount); err != nil {
				return nil, fmt.Errorf("invalid max_count argument: %w", err)
			}
		}
		
		log, err := gs.getLog(repoPath, maxCount)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting git log: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Git log for repository: %s (max %d commits)", repoPath, maxCount)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: log,
				},
			},
		}, nil
	})
	
	// Register git_branches tool
	srv.RegisterTool("git_branches", "List branches in a git repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw json.RawMessage
		var exists bool
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		var repoPath string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		branches, err := gs.getBranches(repoPath)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting git branches: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Git branches for repository: %s", repoPath)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: branches,
				},
			},
		}, nil
	})
	
	// Register git_diff tool
	srv.RegisterTool("git_diff", "Show differences in a git repository", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw json.RawMessage
		var exists bool
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		var repoPath string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		var ref1, ref2 string
		if ref1Raw, exists := args["ref1"]; exists {
			if err := json.Unmarshal(ref1Raw, &ref1); err != nil {
				return nil, fmt.Errorf("invalid ref1 argument: %w", err)
			}
		}
		if ref2Raw, exists := args["ref2"]; exists {
			if err := json.Unmarshal(ref2Raw, &ref2); err != nil {
				return nil, fmt.Errorf("invalid ref2 argument: %w", err)
			}
		}
		
		diff, err := gs.getDiff(repoPath, ref1, ref2)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting git diff: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Git diff for repository: %s", repoPath)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: diff,
				},
			},
		}, nil
	})
	
	// Register git_add tool
	srv.RegisterTool("git_add", "Add files to git staging area", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw, filesRaw json.RawMessage
		var exists bool
		
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		if filesRaw, exists = args["files"]; !exists {
			return nil, fmt.Errorf("missing required argument: files")
		}
		
		var repoPath string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		var files []string
		if err := json.Unmarshal(filesRaw, &files); err != nil {
			return nil, fmt.Errorf("invalid files argument: %w", err)
		}
		
		result, err := gs.addFiles(repoPath, files)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error adding files: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Added files to git staging area: %v", files)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully added files: %s\n%s", strings.Join(files, ", "), result),
				},
			},
		}, nil
	})
	
	// Register git_commit tool
	srv.RegisterTool("git_commit", "Create a git commit", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw, messageRaw json.RawMessage
		var exists bool
		
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		if messageRaw, exists = args["message"]; !exists {
			return nil, fmt.Errorf("missing required argument: message")
		}
		
		var repoPath, message string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		if err := json.Unmarshal(messageRaw, &message); err != nil {
			return nil, fmt.Errorf("invalid message argument: %w", err)
		}
		
		result, err := gs.commit(repoPath, message)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error committing: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Created git commit with message: %s", message)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})
	
	// Register git_create_branch tool
	srv.RegisterTool("git_create_branch", "Create a new git branch", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw, branchRaw json.RawMessage
		var exists bool
		
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		if branchRaw, exists = args["branch"]; !exists {
			return nil, fmt.Errorf("missing required argument: branch")
		}
		
		var repoPath, branch string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		if err := json.Unmarshal(branchRaw, &branch); err != nil {
			return nil, fmt.Errorf("invalid branch argument: %w", err)
		}
		
		result, err := gs.createBranch(repoPath, branch)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error creating branch: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Created git branch: %s", branch)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})
	
	// Register git_switch_branch tool
	srv.RegisterTool("git_switch_branch", "Switch to a different git branch", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var repoPathRaw, branchRaw json.RawMessage
		var exists bool
		
		if repoPathRaw, exists = args["repository"]; !exists {
			return nil, fmt.Errorf("missing required argument: repository")
		}
		
		if branchRaw, exists = args["branch"]; !exists {
			return nil, fmt.Errorf("missing required argument: branch")
		}
		
		var repoPath, branch string
		if err := json.Unmarshal(repoPathRaw, &repoPath); err != nil {
			return nil, fmt.Errorf("invalid repository argument: %w", err)
		}
		
		if err := json.Unmarshal(branchRaw, &branch); err != nil {
			return nil, fmt.Errorf("invalid branch argument: %w", err)
		}
		
		result, err := gs.switchBranch(repoPath, branch)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error switching branch: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Switched to git branch: %s", branch)
		
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
	log.Printf("Git server running on stdio, allowed repositories: %v", gs.allowedRepositories)
	
	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}