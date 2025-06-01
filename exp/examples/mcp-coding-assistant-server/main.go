package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/tmc/mcp"
)

// Tool Handler Implementations

// TaskHandler handles task delegation requests.
func TaskHandler(ctx context.Context, input TaskInput) (interface{}, error) {
	log.Printf("Task requested: %s - %s", input.Description, input.Prompt)
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Task '%s' with prompt '%s' has been processed.", input.Description, input.Prompt),
			},
		},
	}, nil
}

// BashHandler executes bash commands in a controlled environment.
func BashHandler(ctx context.Context, input BashInput) (interface{}, error) {
	log.Printf("Executing bash command: %s", input.Command)

	cmd := exec.CommandContext(ctx, "bash", "-c", input.Command)

	// Capture both stdout and stderr
	stdoutStderr, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"stdout":      string(stdoutStderr),
		"stderr":      "",
		"interrupted": false,
		"isImage":     false,
		"sandbox":     false,
	}

	if err != nil {
		// If there was an error, include it in the stderr field
		result["stderr"] = err.Error()
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": mustMarshal(result),
			},
		},
	}, nil
}

// BatchHandler handles batch operations by delegating to appropriate handlers.
func BatchHandler(ctx context.Context, input BatchInput) (interface{}, error) {
	log.Printf("Batch operation requested: %s", input.Description)

	// Results from each invocation
	results := make([]interface{}, len(input.Invocations))

	for i, invocation := range input.Invocations {
		var result interface{}
		var err error

		// Process each invocation based on tool name
		switch invocation.ToolName {
		case "Bash":
			var bashInput BashInput
			if err := json.Unmarshal(invocation.Input, &bashInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling Bash input: %w", err)
			}
			result, err = BashHandler(ctx, bashInput)
		case "Task":
			var taskInput TaskInput
			if err := json.Unmarshal(invocation.Input, &taskInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling Task input: %w", err)
			}
			result, err = TaskHandler(ctx, taskInput)
		case "Glob":
			var globInput GlobInput
			if err := json.Unmarshal(invocation.Input, &globInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling Glob input: %w", err)
			}
			result, err = GlobHandler(ctx, globInput)
		case "Grep":
			var grepInput GrepInput
			if err := json.Unmarshal(invocation.Input, &grepInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling Grep input: %w", err)
			}
			result, err = GrepHandler(ctx, grepInput)
		case "LS":
			var lsInput LSInput
			if err := json.Unmarshal(invocation.Input, &lsInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling LS input: %w", err)
			}
			result, err = LSHandler(ctx, lsInput)
		case "Read":
			var readInput ReadInput
			if err := json.Unmarshal(invocation.Input, &readInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling Read input: %w", err)
			}
			result, err = ReadHandler(ctx, readInput)
		case "Edit":
			var editInput EditInput
			if err := json.Unmarshal(invocation.Input, &editInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling Edit input: %w", err)
			}
			result, err = EditHandler(ctx, editInput)
		case "MultiEdit":
			var multiEditInput MultiEditInput
			if err := json.Unmarshal(invocation.Input, &multiEditInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling MultiEdit input: %w", err)
			}
			result, err = MultiEditHandler(ctx, multiEditInput)
		case "Write":
			var writeInput WriteInput
			if err := json.Unmarshal(invocation.Input, &writeInput); err != nil {
				return nil, fmt.Errorf("error unmarshaling Write input: %w", err)
			}
			result, err = WriteHandler(ctx, writeInput)
		default:
			return nil, fmt.Errorf("unsupported tool: %s", invocation.ToolName)
		}

		if err != nil {
			return nil, fmt.Errorf("error processing %s: %w", invocation.ToolName, err)
		}

		results[i] = result
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": mustMarshal(results),
			},
		},
	}, nil
}

// GlobHandler implements file pattern matching.
func GlobHandler(ctx context.Context, input GlobInput) (interface{}, error) {
	log.Printf("Glob pattern matching: %s", input.Pattern)

	path := "."
	if input.Path != "" {
		path = input.Path
	}

	// Simplified implementation - in a real implementation, you'd use filepath.Glob
	// or a more robust glob library
	cmd := exec.CommandContext(ctx, "find", path, "-path", input.Pattern, "-type", "f")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("error executing glob: %w", err)
	}

	// Split output into lines for file paths
	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	result := map[string]interface{}{
		"files": files,
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": mustMarshal(result),
			},
		},
	}, nil
}

// GrepHandler implements content search functionality.
func GrepHandler(ctx context.Context, input GrepInput) (interface{}, error) {
	log.Printf("Grep search for pattern: %s", input.Pattern)

	path := "."
	if input.Path != "" {
		path = input.Path
	}

	args := []string{"-r", input.Pattern, path}
	if input.Include != "" {
		args = []string{"-r", "--include", input.Include, input.Pattern, path}
	}

	cmd := exec.CommandContext(ctx, "grep", args...)
	output, err := cmd.CombinedOutput()

	// Grep returns exit code 1 if no matches are found, which is not an error for us
	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		return nil, fmt.Errorf("error executing grep: %w", err)
	}

	matches := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(matches) == 1 && matches[0] == "" {
		matches = []string{}
	}

	result := map[string]interface{}{
		"matches": matches,
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": mustMarshal(result),
			},
		},
	}, nil
}

// LSHandler lists files and directories.
func LSHandler(ctx context.Context, input LSInput) (interface{}, error) {
	log.Printf("Listing directory: %s", input.Path)

	cmd := exec.CommandContext(ctx, "ls", "-la", input.Path)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("error listing directory: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter lines based on ignore patterns if provided
	if len(input.Ignore) > 0 {
		// In a real implementation, you'd use a proper glob matching library
		filteredLines := []string{}
		for _, line := range lines {
			ignored := false
			for _, pattern := range input.Ignore {
				if strings.Contains(line, pattern) {
					ignored = true
					break
				}
			}
			if !ignored {
				filteredLines = append(filteredLines, line)
			}
		}
		lines = filteredLines
	}

	result := map[string]interface{}{
		"entries": lines,
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": mustMarshal(result),
			},
		},
	}, nil
}

// ReadHandler implements file reading functionality.
func ReadHandler(ctx context.Context, input ReadInput) (interface{}, error) {
	log.Printf("Reading file: %s", input.FilePath)

	data, err := os.ReadFile(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	// Apply offset and limit if provided
	start := 0
	end := len(lines)

	if input.Offset != nil {
		start = int(*input.Offset)
		if start < 0 {
			start = 0
		}
		if start > len(lines) {
			start = len(lines)
		}
	}

	if input.Limit != nil {
		limit := int(*input.Limit)
		if limit > 0 {
			end = start + limit
			if end > len(lines) {
				end = len(lines)
			}
		}
	}

	// Apply the offset and limit
	if start < end {
		lines = lines[start:end]
	} else {
		lines = []string{}
	}

	// Format output with line numbers
	numberedLines := []string{}
	for i, line := range lines {
		numberedLines = append(numberedLines, fmt.Sprintf("%5d\t%s", start+i+1, line))
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": strings.Join(numberedLines, "\n"),
			},
		},
	}, nil
}

// EditHandler implements file editing functionality.
func EditHandler(ctx context.Context, input EditInput) (interface{}, error) {
	log.Printf("Editing file: %s", input.FilePath)

	// Check if file exists
	fileInfo, err := os.Stat(input.FilePath)

	// If creating a new file (empty old_string)
	if input.OldString == "" {
		// Create parent directories if necessary
		dir := strings.TrimSuffix(input.FilePath, "/"+baseNameFromPath(input.FilePath))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("error creating directories: %w", err)
		}

		// Write the new content
		if err := os.WriteFile(input.FilePath, []byte(input.NewString), 0644); err != nil {
			return nil, fmt.Errorf("error creating file: %w", err)
		}

		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Created new file: %s", input.FilePath),
				},
			},
		}, nil
	}

	// For existing file edits
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", input.FilePath)
	} else if err != nil {
		return nil, fmt.Errorf("error accessing file: %w", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not a file", input.FilePath)
	}

	// Read the file content
	content, err := os.ReadFile(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	fileContent := string(content)
	count := strings.Count(fileContent, input.OldString)

	// Handle expected replacements
	expectedReplacements := 1
	if input.ExpectedReplacements != nil {
		expectedReplacements = int(*input.ExpectedReplacements)
	}

	if count != expectedReplacements {
		return nil, fmt.Errorf("expected %d occurrences of the string to replace, but found %d", expectedReplacements, count)
	}

	// Perform the replacement
	newContent := strings.Replace(fileContent, input.OldString, input.NewString, expectedReplacements)

	// Write the file back
	if err := os.WriteFile(input.FilePath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("error writing file: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully edited %s, replaced %d occurrence(s)", input.FilePath, expectedReplacements),
			},
		},
	}, nil
}

// MultiEditHandler implements multiple file edits in a single operation.
func MultiEditHandler(ctx context.Context, input MultiEditInput) (interface{}, error) {
	log.Printf("Multiple edits for file: %s", input.FilePath)

	// Read the file content
	content, err := os.ReadFile(input.FilePath)
	if err != nil {
		// If file doesn't exist and first edit has empty old_string, create it
		if os.IsNotExist(err) && len(input.Edits) > 0 && input.Edits[0].OldString == "" {
			// Create parent directories if necessary
			dir := strings.TrimSuffix(input.FilePath, "/"+baseNameFromPath(input.FilePath))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("error creating directories: %w", err)
			}

			// Start with empty content
			content = []byte{}
		} else {
			return nil, fmt.Errorf("error reading file: %w", err)
		}
	}

	fileContent := string(content)

	// Apply each edit in sequence
	for i, edit := range input.Edits {
		expectedReplacements := 1
		if edit.ExpectedReplacements != nil {
			expectedReplacements = int(*edit.ExpectedReplacements)
		}

		// Special case for first edit with empty old_string on new file
		if i == 0 && edit.OldString == "" && len(content) == 0 {
			fileContent = edit.NewString
			continue
		}

		count := strings.Count(fileContent, edit.OldString)

		if count != expectedReplacements {
			return nil, fmt.Errorf("for edit #%d: expected %d occurrences, but found %d", i+1, expectedReplacements, count)
		}

		fileContent = strings.Replace(fileContent, edit.OldString, edit.NewString, expectedReplacements)
	}

	// Write the final content back to the file
	if err := os.WriteFile(input.FilePath, []byte(fileContent), 0644); err != nil {
		return nil, fmt.Errorf("error writing file: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully applied %d edits to %s", len(input.Edits), input.FilePath),
			},
		},
	}, nil
}

// WriteHandler implements file writing functionality.
func WriteHandler(ctx context.Context, input WriteInput) (interface{}, error) {
	log.Printf("Writing to file: %s", input.FilePath)

	// Create parent directories if necessary
	dir := strings.TrimSuffix(input.FilePath, "/"+baseNameFromPath(input.FilePath))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating directories: %w", err)
	}

	// Write the content to the file
	if err := os.WriteFile(input.FilePath, []byte(input.Content), 0644); err != nil {
		return nil, fmt.Errorf("error writing file: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully wrote %d bytes to %s", len(input.Content), input.FilePath),
			},
		},
	}, nil
}

// Unimplemented handlers for completeness
func NotebookReadHandler(ctx context.Context, input NotebookReadInput) (interface{}, error) {
	return nil, fmt.Errorf("NotebookRead functionality not implemented")
}

func NotebookEditHandler(ctx context.Context, input NotebookEditInput) (interface{}, error) {
	return nil, fmt.Errorf("NotebookEdit functionality not implemented")
}

func WebFetchHandler(ctx context.Context, input WebFetchInput) (interface{}, error) {
	return nil, fmt.Errorf("WebFetch functionality not implemented")
}

func TodoReadHandler(ctx context.Context, input TodoReadInput) (interface{}, error) {
	return nil, fmt.Errorf("TodoRead functionality not implemented")
}

func TodoWriteHandler(ctx context.Context, input TodoWriteInput) (interface{}, error) {
	return nil, fmt.Errorf("TodoWrite functionality not implemented")
}

func WebSearchHandler(ctx context.Context, input WebSearchInput) (interface{}, error) {
	return nil, fmt.Errorf("WebSearch functionality not implemented")
}

// Utility Functions
func baseNameFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// --- Main Function ---

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to listen on")
	enableStdio := flag.Bool("stdio", false, "Use stdio instead of HTTP")
	flag.Parse()

	// Create the server
	srv := mcp.NewServer()

	// Register tool handlers
	srv.RegisterTool("Task", TaskHandler)
	srv.RegisterTool("Bash", BashHandler)
	srv.RegisterTool("Batch", BatchHandler)
	srv.RegisterTool("Glob", GlobHandler)
	srv.RegisterTool("Grep", GrepHandler)
	srv.RegisterTool("LS", LSHandler)
	srv.RegisterTool("Read", ReadHandler)
	srv.RegisterTool("Edit", EditHandler)
	srv.RegisterTool("MultiEdit", MultiEditHandler)
	srv.RegisterTool("Write", WriteHandler)
	srv.RegisterTool("NotebookRead", NotebookReadHandler)
	srv.RegisterTool("NotebookEdit", NotebookEditHandler)
	srv.RegisterTool("WebFetch", WebFetchHandler)
	srv.RegisterTool("TodoRead", TodoReadHandler)
	srv.RegisterTool("TodoWrite", TodoWriteHandler)
	srv.RegisterTool("WebSearch", WebSearchHandler)

	// Start the server
	if *enableStdio {
		fmt.Println("Starting MCP server on stdio")
		if err := srv.ServeStdio(context.Background()); err != nil {
			log.Fatalf("Error serving on stdio: %v", err)
		}
	} else {
		addr := fmt.Sprintf(":%d", *port)
		fmt.Printf("Starting MCP server on http://localhost%s\n", addr)
		if err := srv.ServeHTTP(context.Background(), addr); err != nil {
			log.Fatalf("Error serving HTTP: %v", err)
		}
	}
}

// --- Data Structures (as per the existing code) ---

// Root is the top-level structure of the JSON response describing the tools.
type Root struct {
	Result  Result `json:"result"`
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"` // Assuming the id is an integer based on the example '1'
}

// Result represents the 'result' object containing the list of tools.
type Result struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolDefinition represents a single tool definition within the 'tools' array.
// It contains the tool's metadata and its input schema (as raw JSON).
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"` // Store schema as raw JSON
}

// --- Structs for specific Tool Input Payloads ---
// These map directly to the "properties" defined in each tool's "inputSchema".
// Optional fields use pointers or omitempty where appropriate.

// TaskInput maps to the inputSchema for the "Task" tool.
type TaskInput struct {
	Description string `json:"description"` // Required
	Prompt      string `json:"prompt"`      // Required
}

// BashInput maps to the inputSchema for the "Bash" tool.
type BashInput struct {
	Command     string   `json:"command"`               // Required
	Timeout     *float64 `json:"timeout,omitempty"`     // Optional number, using pointer to differentiate 0 from unset
	Description string   `json:"description,omitempty"` // Optional string
}

// BatchInput maps to the inputSchema for the "Batch" tool.
type BatchInput struct {
	Description string            `json:"description"` // Required
	Invocations []BatchInvocation `json:"invocations"` // Required array (minItems: 1)
}

// BatchInvocation maps to the "items" schema for the "invocations" array in BatchInput.
type BatchInvocation struct {
	ToolName string          `json:"tool_name"` // Required
	Input    json.RawMessage `json:"input"`     // Required, but content is arbitrary object
}

// GlobInput maps to the inputSchema for the "Glob" tool.
type GlobInput struct {
	Pattern string `json:"pattern"`        // Required
	Path    string `json:"path,omitempty"` // Optional string
}

// GrepInput maps to the inputSchema for the "Grep" tool.
type GrepInput struct {
	Pattern string `json:"pattern"`           // Required
	Path    string `json:"path,omitempty"`    // Optional string
	Include string `json:"include,omitempty"` // Optional string
}

// LSInput maps to the inputSchema for the "LS" tool.
type LSInput struct {
	Path   string   `json:"path"`             // Required
	Ignore []string `json:"ignore,omitempty"` // Optional array of strings
}

// ReadInput maps to the inputSchema for the "Read" tool.
type ReadInput struct {
	FilePath string   `json:"file_path"`        // Required
	Offset   *float64 `json:"offset,omitempty"` // Optional number, using pointer
	Limit    *float64 `json:"limit,omitempty"`  // Optional number, using pointer
}

// EditInput maps to the inputSchema for the "Edit" tool.
type EditInput struct {
	FilePath             string   `json:"file_path"`                       // Required
	OldString            string   `json:"old_string"`                      // Required
	NewString            string   `json:"new_string"`                      // Required
	ExpectedReplacements *float64 `json:"expected_replacements,omitempty"` // Optional number
}

// MultiEditInput maps to the inputSchema for the "MultiEdit" tool.
type MultiEditInput struct {
	FilePath string          `json:"file_path"` // Required
	Edits    []EditOperation `json:"edits"`     // Required array (minItems: 1)
}

// EditOperation maps to the "items" schema for the "edits" array in MultiEditInput.
type EditOperation struct {
	OldString            string   `json:"old_string"`                      // Required
	NewString            string   `json:"new_string"`                      // Required
	ExpectedReplacements *float64 `json:"expected_replacements,omitempty"` // Optional number
}

// WriteInput maps to the inputSchema for the "Write" tool.
type WriteInput struct {
	FilePath string `json:"file_path"` // Required
	Content  string `json:"content"`   // Required
}

// NotebookReadInput maps to the inputSchema for the "NotebookRead" tool.
type NotebookReadInput struct {
	NotebookPath string `json:"notebook_path"` // Required
}

// NotebookEditInput maps to the inputSchema for the "NotebookEdit" tool.
type NotebookEditInput struct {
	NotebookPath string  `json:"notebook_path"`       // Required
	CellNumber   float64 `json:"cell_number"`         // Required
	NewSource    string  `json:"new_source"`          // Required
	CellType     string  `json:"cell_type,omitempty"` // Optional
	EditMode     string  `json:"edit_mode,omitempty"` // Optional
}

// WebFetchInput maps to the inputSchema for the "WebFetch" tool.
type WebFetchInput struct {
	URL    string `json:"url"`    // Required
	Prompt string `json:"prompt"` // Required
}

// TodoReadInput maps to the inputSchema for the "TodoRead" tool.
type TodoReadInput struct {
	// No fields required based on schema properties: {}
}

// TodoWriteInput maps to the inputSchema for the "TodoWrite" tool.
type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"` // Required array
}

// TodoItem maps to the "items" schema for the "todos" array in TodoWriteInput.
type TodoItem struct {
	Content  string `json:"content"`  // Required
	Status   string `json:"status"`   // Required
	Priority string `json:"priority"` // Required
	ID       string `json:"id"`       // Required
}

// WebSearchInput maps to the inputSchema for the "WebSearch" tool.
type WebSearchInput struct {
	Query          string   `json:"query"`                     // Required
	AllowedDomains []string `json:"allowed_domains,omitempty"` // Optional array
	BlockedDomains []string `json:"blocked_domains,omitempty"` // Optional array
}
