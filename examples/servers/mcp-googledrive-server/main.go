package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GoogleDriveServer struct {
	service *drive.Service
	ctx     context.Context
}

func NewGoogleDriveServer(credentialsJSON, token string) (*GoogleDriveServer, error) {
	ctx := context.Background()

	// Parse credentials
	config, err := google.ConfigFromJSON([]byte(credentialsJSON), drive.DriveReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Create token
	tok := &oauth2.Token{}
	if err := json.Unmarshal([]byte(token), tok); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	client := config.Client(ctx, tok)

	// Create Drive service
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	return &GoogleDriveServer{
		service: service,
		ctx:     ctx,
	}, nil
}

func (gds *GoogleDriveServer) listFiles(query string, limit int) (string, error) {
	if limit <= 0 {
		limit = 20
	}

	call := gds.service.Files.List().PageSize(int64(limit)).Fields("files(id,name,mimeType,size,modifiedTime,webViewLink)")
	if query != "" {
		call = call.Q(query)
	}

	files, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("failed to list files: %w", err)
	}

	var result strings.Builder
	if query != "" {
		result.WriteString(fmt.Sprintf("Files matching query '%s':\n", query))
	} else {
		result.WriteString("Recent files:\n")
	}
	result.WriteString("Name\t\tType\t\tSize\t\tModified\n")
	result.WriteString("----\t\t----\t\t----\t\t--------\n")

	for _, file := range files.Files {
		name := file.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		mimeType := file.MimeType
		if strings.HasPrefix(mimeType, "application/vnd.google-apps.") {
			mimeType = strings.TrimPrefix(mimeType, "application/vnd.google-apps.")
		}
		if len(mimeType) > 15 {
			mimeType = mimeType[:12] + "..."
		}

		size := "N/A"
		if file.Size > 0 {
			size = formatBytes(file.Size)
		}

		modTime := "N/A"
		if file.ModifiedTime != "" {
			if t, err := time.Parse(time.RFC3339, file.ModifiedTime); err == nil {
				modTime = t.Format("2006-01-02")
			}
		}

		result.WriteString(fmt.Sprintf("%s\t\t%s\t\t%s\t\t%s\n", name, mimeType, size, modTime))
	}

	return result.String(), nil
}

func (gds *GoogleDriveServer) getFileInfo(fileID string) (string, error) {
	file, err := gds.service.Files.Get(fileID).Fields("id,name,mimeType,size,createdTime,modifiedTime,owners,parents,webViewLink,webContentLink").Do()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("File: %s\n", file.Name))
	result.WriteString(fmt.Sprintf("ID: %s\n", file.Id))
	result.WriteString(fmt.Sprintf("MIME Type: %s\n", file.MimeType))

	if file.Size > 0 {
		result.WriteString(fmt.Sprintf("Size: %s\n", formatBytes(file.Size)))
	}

	if file.CreatedTime != "" {
		if t, err := time.Parse(time.RFC3339, file.CreatedTime); err == nil {
			result.WriteString(fmt.Sprintf("Created: %s\n", t.Format("2006-01-02 15:04:05")))
		}
	}

	if file.ModifiedTime != "" {
		if t, err := time.Parse(time.RFC3339, file.ModifiedTime); err == nil {
			result.WriteString(fmt.Sprintf("Modified: %s\n", t.Format("2006-01-02 15:04:05")))
		}
	}

	if len(file.Owners) > 0 {
		result.WriteString(fmt.Sprintf("Owner: %s\n", file.Owners[0].DisplayName))
	}

	if file.WebViewLink != "" {
		result.WriteString(fmt.Sprintf("View Link: %s\n", file.WebViewLink))
	}

	if file.WebContentLink != "" {
		result.WriteString(fmt.Sprintf("Download Link: %s\n", file.WebContentLink))
	}

	return result.String(), nil
}

func (gds *GoogleDriveServer) searchFiles(query string, limit int) (string, error) {
	if limit <= 0 {
		limit = 20
	}

	// Build search query
	searchQuery := fmt.Sprintf("name contains '%s' or fullText contains '%s'", query, query)

	call := gds.service.Files.List().
		Q(searchQuery).
		PageSize(int64(limit)).
		Fields("files(id,name,mimeType,size,modifiedTime,webViewLink)")

	files, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("failed to search files: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Search results for '%s':\n", query))
	result.WriteString("Name\t\tType\t\tSize\t\tModified\n")
	result.WriteString("----\t\t----\t\t----\t\t--------\n")

	for _, file := range files.Files {
		name := file.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		mimeType := file.MimeType
		if strings.HasPrefix(mimeType, "application/vnd.google-apps.") {
			mimeType = strings.TrimPrefix(mimeType, "application/vnd.google-apps.")
		}
		if len(mimeType) > 15 {
			mimeType = mimeType[:12] + "..."
		}

		size := "N/A"
		if file.Size > 0 {
			size = formatBytes(file.Size)
		}

		modTime := "N/A"
		if file.ModifiedTime != "" {
			if t, err := time.Parse(time.RFC3339, file.ModifiedTime); err == nil {
				modTime = t.Format("2006-01-02")
			}
		}

		result.WriteString(fmt.Sprintf("%s\t\t%s\t\t%s\t\t%s\n", name, mimeType, size, modTime))
	}

	result.WriteString(fmt.Sprintf("\nFound %d files\n", len(files.Files)))
	return result.String(), nil
}

func (gds *GoogleDriveServer) listFolders(parentID string) (string, error) {
	query := "mimeType='application/vnd.google-apps.folder'"
	if parentID != "" {
		query += fmt.Sprintf(" and '%s' in parents", parentID)
	}

	call := gds.service.Files.List().
		Q(query).
		Fields("files(id,name,modifiedTime,webViewLink)")

	files, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("failed to list folders: %w", err)
	}

	var result strings.Builder
	if parentID != "" {
		result.WriteString(fmt.Sprintf("Folders in parent %s:\n", parentID))
	} else {
		result.WriteString("Root folders:\n")
	}
	result.WriteString("Name\t\tID\t\tModified\n")
	result.WriteString("----\t\t--\t\t--------\n")

	for _, folder := range files.Files {
		name := folder.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		id := folder.Id
		if len(id) > 15 {
			id = id[:12] + "..."
		}

		modTime := "N/A"
		if folder.ModifiedTime != "" {
			if t, err := time.Parse(time.RFC3339, folder.ModifiedTime); err == nil {
				modTime = t.Format("2006-01-02")
			}
		}

		result.WriteString(fmt.Sprintf("%s\t\t%s\t\t%s\n", name, id, modTime))
	}

	return result.String(), nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func main() {
	credentialsFile := os.Getenv("GOOGLE_CREDENTIALS")
	tokenFile := os.Getenv("GOOGLE_TOKEN")

	if credentialsFile == "" || tokenFile == "" {
		log.Fatal("GOOGLE_CREDENTIALS and GOOGLE_TOKEN environment variables are required")
	}

	credentials, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatalf("Failed to read credentials file: %v", err)
	}

	token, err := os.ReadFile(tokenFile)
	if err != nil {
		log.Fatalf("Failed to read token file: %v", err)
	}

	// Initialize Google Drive server
	gds, err := NewGoogleDriveServer(string(credentials), string(token))
	if err != nil {
		log.Fatalf("Failed to initialize Google Drive server: %v", err)
	}

	// Create server with name and version
	srv := mcp.NewServer("googledrive-server", "1.0.0")

	// Register list_files tool
	srv.RegisterTool("list_files", "List files in Google Drive", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var query string
		if queryRaw, exists := args["query"]; exists {
			if err := json.Unmarshal(queryRaw, &query); err != nil {
				return nil, fmt.Errorf("invalid query argument: %w", err)
			}
		}

		var limit int = 20
		if limitRaw, exists := args["limit"]; exists {
			if err := json.Unmarshal(limitRaw, &limit); err != nil {
				return nil, fmt.Errorf("invalid limit argument: %w", err)
			}
		}

		result, err := gds.listFiles(query, limit)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing files: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Listed files with query: %s", query)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register get_file_info tool
	srv.RegisterTool("get_file_info", "Get detailed information about a Google Drive file", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var fileIDRaw json.RawMessage
		var exists bool
		if fileIDRaw, exists = args["file_id"]; !exists {
			return nil, fmt.Errorf("missing required argument: file_id")
		}

		var fileID string
		if err := json.Unmarshal(fileIDRaw, &fileID); err != nil {
			return nil, fmt.Errorf("invalid file_id argument: %w", err)
		}

		result, err := gds.getFileInfo(fileID)
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

		log.Printf("Retrieved file info for: %s", fileID)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register search_files tool
	srv.RegisterTool("search_files", "Search for files in Google Drive", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var queryRaw json.RawMessage
		var exists bool
		if queryRaw, exists = args["query"]; !exists {
			return nil, fmt.Errorf("missing required argument: query")
		}

		var query string
		if err := json.Unmarshal(queryRaw, &query); err != nil {
			return nil, fmt.Errorf("invalid query argument: %w", err)
		}

		var limit int = 20
		if limitRaw, exists := args["limit"]; exists {
			if err := json.Unmarshal(limitRaw, &limit); err != nil {
				return nil, fmt.Errorf("invalid limit argument: %w", err)
			}
		}

		result, err := gds.searchFiles(query, limit)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error searching files: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Searched files for: %s", query)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register list_folders tool
	srv.RegisterTool("list_folders", "List folders in Google Drive", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var parentID string
		if parentIDRaw, exists := args["parent_id"]; exists {
			if err := json.Unmarshal(parentIDRaw, &parentID); err != nil {
				return nil, fmt.Errorf("invalid parent_id argument: %w", err)
			}
		}

		result, err := gds.listFolders(parentID)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing folders: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Listed folders with parent: %s", parentID)

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
	log.Printf("Google Drive server running on stdio")

	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
