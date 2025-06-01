package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type FileSystemServer struct {
	allowedDirectories []string
}

func NewFileSystemServer(allowedDirs []string) (*FileSystemServer, error) {
	var normalizedDirs []string
	
	for _, dir := range allowedDirs {
		// Expand home directory
		if strings.HasPrefix(dir, "~/") || dir == "~" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			if dir == "~" {
				dir = homeDir
			} else {
				dir = filepath.Join(homeDir, dir[2:])
			}
		}
		
		// Resolve to absolute path
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve directory %s: %w", dir, err)
		}
		
		// Validate directory exists
		if stat, err := os.Stat(absDir); err != nil {
			return nil, fmt.Errorf("directory %s does not exist: %w", absDir, err)
		} else if !stat.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", absDir)
		}
		
		normalizedDirs = append(normalizedDirs, filepath.Clean(absDir))
	}
	
	return &FileSystemServer{
		allowedDirectories: normalizedDirs,
	}, nil
}

func (fss *FileSystemServer) validatePath(requestedPath string) (string, error) {
	// Expand home directory
	if strings.HasPrefix(requestedPath, "~/") || requestedPath == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		if requestedPath == "~" {
			requestedPath = homeDir
		} else {
			requestedPath = filepath.Join(homeDir, requestedPath[2:])
		}
	}
	
	// Convert to absolute path
	var absPath string
	if filepath.IsAbs(requestedPath) {
		absPath = filepath.Clean(requestedPath)
	} else {
		wd, _ := os.Getwd()
		absPath = filepath.Clean(filepath.Join(wd, requestedPath))
	}
	
	// Check if path is within allowed directories
	allowed := false
	for _, allowedDir := range fss.allowedDirectories {
		if strings.HasPrefix(absPath, allowedDir) {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return "", fmt.Errorf("access denied - path outside allowed directories: %s", absPath)
	}
	
	// Resolve symlinks and verify they're still in allowed directories
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// For new files that don't exist, check parent directory
		parentDir := filepath.Dir(absPath)
		if realParent, err := filepath.EvalSymlinks(parentDir); err != nil {
			return "", fmt.Errorf("parent directory does not exist: %s", parentDir)
		} else {
			parentAllowed := false
			for _, allowedDir := range fss.allowedDirectories {
				if strings.HasPrefix(realParent, allowedDir) {
					parentAllowed = true
					break
				}
			}
			if !parentAllowed {
				return "", fmt.Errorf("access denied - parent directory outside allowed directories")
			}
			return absPath, nil
		}
	}
	
	// Verify real path is still allowed
	realAllowed := false
	for _, allowedDir := range fss.allowedDirectories {
		if strings.HasPrefix(realPath, allowedDir) {
			realAllowed = true
			break
		}
	}
	
	if !realAllowed {
		return "", fmt.Errorf("access denied - symlink target outside allowed directories")
	}
	
	return realPath, nil
}

func (fss *FileSystemServer) readFile(filePath string) (string, error) {
	validPath, err := fss.validatePath(filePath)
	if err != nil {
		return "", err
	}
	
	data, err := os.ReadFile(validPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	return string(data), nil
}

func (fss *FileSystemServer) writeFile(filePath, content string) error {
	validPath, err := fss.validatePath(filePath)
	if err != nil {
		return err
	}
	
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(validPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	
	if err := os.WriteFile(validPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

func (fss *FileSystemServer) listDirectory(dirPath string) ([]map[string]interface{}, error) {
	validPath, err := fss.validatePath(dirPath)
	if err != nil {
		return nil, err
	}
	
	entries, err := os.ReadDir(validPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	var result []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		entryData := map[string]interface{}{
			"name":     entry.Name(),
			"type":     getFileType(entry),
			"size":     info.Size(),
			"modified": info.ModTime().Format(time.RFC3339),
		}
		
		result = append(result, entryData)
	}
	
	return result, nil
}

func getFileType(entry fs.DirEntry) string {
	if entry.IsDir() {
		return "directory"
	}
	if entry.Type()&fs.ModeSymlink != 0 {
		return "symlink"
	}
	return "file"
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("Usage: mcp-filesystem-server <allowed-directory> [additional-directories...]")
	}
	
	// Initialize filesystem server
	fss, err := NewFileSystemServer(args)
	if err != nil {
		log.Fatalf("Failed to initialize filesystem server: %v", err)
	}
	
	// Create server with name and version
	srv := mcp.NewServer("filesystem-server", "1.0.0")
	
	// Register read_file tool
	srv.RegisterTool("read_file", "Read the contents of a file", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var pathRaw json.RawMessage
		var exists bool
		if pathRaw, exists = args["path"]; !exists {
			return nil, fmt.Errorf("missing required argument: path")
		}
		
		var filePath string
		if err := json.Unmarshal(pathRaw, &filePath); err != nil {
			return nil, fmt.Errorf("invalid path argument: %w", err)
		}
		
		content, err := fss.readFile(filePath)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error reading file: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Read file: %s (%d bytes)", filePath, len(content))
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: content,
				},
			},
		}, nil
	})
	
	// Register write_file tool
	srv.RegisterTool("write_file", "Write content to a file", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var pathRaw, contentRaw json.RawMessage
		var exists bool
		
		if pathRaw, exists = args["path"]; !exists {
			return nil, fmt.Errorf("missing required argument: path")
		}
		
		if contentRaw, exists = args["content"]; !exists {
			return nil, fmt.Errorf("missing required argument: content")
		}
		
		var filePath, content string
		if err := json.Unmarshal(pathRaw, &filePath); err != nil {
			return nil, fmt.Errorf("invalid path argument: %w", err)
		}
		
		if err := json.Unmarshal(contentRaw, &content); err != nil {
			return nil, fmt.Errorf("invalid content argument: %w", err)
		}
		
		if err := fss.writeFile(filePath, content); err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error writing file: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		log.Printf("Wrote file: %s (%d bytes)", filePath, len(content))
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), filePath),
				},
			},
		}, nil
	})
	
	// Register list_directory tool
	srv.RegisterTool("list_directory", "List the contents of a directory", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var pathRaw json.RawMessage
		var exists bool
		if pathRaw, exists = args["path"]; !exists {
			return nil, fmt.Errorf("missing required argument: path")
		}
		
		var dirPath string
		if err := json.Unmarshal(pathRaw, &dirPath); err != nil {
			return nil, fmt.Errorf("invalid path argument: %w", err)
		}
		
		entries, err := fss.listDirectory(dirPath)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing directory: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		responseJSON, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}
		
		log.Printf("Listed directory: %s (%d entries)", dirPath, len(entries))
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})
	
	// Register get_file_info tool
	srv.RegisterTool("get_file_info", "Get information about a file or directory", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var pathRaw json.RawMessage
		var exists bool
		if pathRaw, exists = args["path"]; !exists {
			return nil, fmt.Errorf("missing required argument: path")
		}
		
		var filePath string
		if err := json.Unmarshal(pathRaw, &filePath); err != nil {
			return nil, fmt.Errorf("invalid path argument: %w", err)
		}
		
		validPath, err := fss.validatePath(filePath)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error accessing path: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		info, err := os.Stat(validPath)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error getting file info: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}
		
		fileType := "file"
		if info.IsDir() {
			fileType = "directory"
		}
		
		result := map[string]interface{}{
			"name":        info.Name(),
			"path":        validPath,
			"type":        fileType,
			"size":        info.Size(),
			"modified":    info.ModTime().Format(time.RFC3339),
			"permissions": info.Mode().String(),
		}
		
		responseJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}
		
		log.Printf("Got file info: %s", filePath)
		
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})
	
	// Start server with stdio transport
	transport := mcp.StdioTransport{}
	log.Printf("Filesystem server running on stdio, allowed directories: %v", fss.allowedDirectories)
	
	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}