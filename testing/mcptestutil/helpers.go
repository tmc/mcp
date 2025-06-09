// Package mcptestutil provides helper functions for testing MCP implementations.
package mcptestutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp"
)

// CreateTempFile creates a temporary file with the given content.
func CreateTempFile(t *testing.T, name, content string) string {
	t.Helper()
	
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, name)
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	
	return filePath
}

// CreateTempDir creates a temporary directory with optional subdirectories.
func CreateTempDir(t *testing.T, subdirs ...string) string {
	t.Helper()
	
	tempDir := t.TempDir()
	
	for _, subdir := range subdirs {
		dirPath := filepath.Join(tempDir, subdir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create subdir %s: %v", subdir, err)
		}
	}
	
	return tempDir
}

// WithTimeout creates a context with timeout for testing.
func WithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), timeout)
}

// WithDeadline creates a context with deadline for testing.
func WithDeadline(t *testing.T, deadline time.Time) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithDeadline(context.Background(), deadline)
}

// RandomPort returns a random available port for testing.
func RandomPort(t *testing.T) int {
	t.Helper()
	
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get random port: %v", err)
	}
	defer listener.Close()
	
	return listener.Addr().(*net.TCPAddr).Port
}

// CreateSampleTool creates a sample tool for testing.
func CreateSampleTool(name, description string) *mcp.Tool {
	return &mcp.Tool{
		Name:        name,
		Description: description,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"input": {
					"type": "string",
					"description": "The input parameter"
				}
			},
			"required": ["input"]
		}`),
	}
}

// CreateSampleResource creates a sample resource for testing.
func CreateSampleResource(uri, description, mimeType string) *mcp.Resource {
	return &mcp.Resource{
		URI:         uri,
		Description: description,
		MimeType:    mimeType,
	}
}

// CreateSamplePrompt creates a sample prompt for testing.
func CreateSamplePrompt(name, description string, args ...mcp.PromptArgument) *mcp.Prompt {
	return &mcp.Prompt{
		Name:        name,
		Description: description,
		Arguments:   args,
	}
}

// CreatePromptArgument creates a prompt argument for testing.
func CreatePromptArgument(name, description string, required bool) mcp.PromptArgument {
	return mcp.PromptArgument{
		Name:        name,
		Description: description,
		Required:    required,
	}
}

// CreateTextContent creates text content for testing.
func CreateTextContent(text string) mcp.Content {
	return mcp.TextContent{
		Type: "text",
		Text: text,
	}
}

// CreateImageContent creates image content for testing.
func CreateImageContent(data []byte, mimeType string) mcp.Content {
	return mcp.ImageContent{
		Type: "image",
		Data: data,
		MimeType: mimeType,
	}
}

// CreateResourceContent creates resource content for testing.
func CreateResourceContent(uri string) mcp.Content {
	// Note: MCP doesn't have a ResourceContent type in the current implementation
	// Return a TextContent with the URI as a placeholder
	return mcp.TextContent{
		Type: "text",
		Text: "Resource: " + uri,
	}
}

// CreateInitializeRequest creates an initialize request for testing.
func CreateInitializeRequest(clientName, clientVersion string) *mcp.InitializeRequest {
	return &mcp.InitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities: mcp.ClientCapabilities{
			Experimental: make(map[string]interface{}),
			Sampling:     &struct{}{},
		},
		ClientInfo: mcp.Implementation{
			Name:    clientName,
			Version: clientVersion,
		},
	}
}

// CreateInitializeResult creates an initialize result for testing.
func CreateInitializeResult(serverName, serverVersion string) *mcp.InitializeResult {
	return &mcp.InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: mcp.ServerCapabilities{
			Tools:     &struct{ListChanged bool `json:"listChanged,omitempty"`}{},
			Resources: &struct{Subscribe bool `json:"subscribe,omitempty"`; ListChanged bool `json:"listChanged,omitempty"`}{},
			Prompts:   &struct{ListChanged bool `json:"listChanged,omitempty"`}{},
		},
		ServerInfo: mcp.Implementation{
			Name:    serverName,
			Version: serverVersion,
		},
	}
}

// CreateCallToolRequest creates a tool call request for testing.
func CreateCallToolRequest(toolName string, arguments map[string]interface{}) *mcp.CallToolRequest {
	argBytes, _ := json.Marshal(arguments)
	return &mcp.CallToolRequest{
		Name:      toolName,
		Arguments: json.RawMessage(argBytes),
	}
}

// CreateCallToolResult creates a tool call result for testing.
func CreateCallToolResult(content ...mcp.Content) *mcp.CallToolResult {
	anyContent := make([]any, len(content))
	for i, c := range content {
		anyContent[i] = c
	}
	return &mcp.CallToolResult{
		Content: anyContent,
	}
}

// ParseJSON parses JSON string into interface{} for testing.
func ParseJSON(t *testing.T, jsonStr string) interface{} {
	t.Helper()
	
	var result interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	return result
}

// ToJSON converts interface{} to JSON string for testing.
func ToJSON(t *testing.T, v interface{}) string {
	t.Helper()
	
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}
	
	return string(data)
}

// PrettyJSON converts interface{} to pretty-printed JSON string for testing.
func PrettyJSON(t *testing.T, v interface{}) string {
	t.Helper()
	
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to pretty JSON: %v", err)
	}
	
	return string(data)
}

// SkipIfShort skips the test if -short flag is used.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// RequireEnv requires an environment variable to be set for the test.
func RequireEnv(t *testing.T, key string) string {
	t.Helper()
	
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Environment variable %s is required for this test", key)
	}
	
	return value
}

// SetEnv sets environment variables for the duration of the test.
func SetEnv(t *testing.T, envVars map[string]string) {
	t.Helper()
	
	// Store original values
	original := make(map[string]string)
	for key := range envVars {
		original[key] = os.Getenv(key)
	}
	
	// Set new values
	for key, value := range envVars {
		os.Setenv(key, value)
	}
	
	// Restore original values on cleanup
	t.Cleanup(func() {
		for key, value := range original {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	})
}

// WaitForCondition waits for a condition to be true or timeout.
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	t.Fatalf("Condition not met within %v: %s", timeout, message)
}

// RetryUntilSuccess retries a function until it succeeds or timeout is reached.
func RetryUntilSuccess(t *testing.T, fn func() error, timeout time.Duration, message string) {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	var lastErr error
	
	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	t.Fatalf("Function did not succeed within %v: %s (last error: %v)", 
		timeout, message, lastErr)
}

// CaptureOutput captures stdout and stderr for the duration of the test function.
func CaptureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	
	// This is a simplified version - in a real implementation you'd need
	// to properly redirect os.Stdout and os.Stderr
	fn()
	return "", ""
}

// LogWithTimestamp logs a message with timestamp for debugging tests.
func LogWithTimestamp(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	message := fmt.Sprintf(format, args...)
	t.Logf("[%s] %s", time.Now().Format("15:04:05.000"), message)
}

// TableTest runs table-driven tests with parallel execution.
func TableTest[T any](t *testing.T, tests []T, testFunc func(*testing.T, T)) {
	t.Helper()
	
	for i, test := range tests {
		test := test // Capture loop variable
		testName := fmt.Sprintf("test_%d", i)
		
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			testFunc(t, test)
		})
	}
}

// NamedTableTest runs table-driven tests with custom names.
func NamedTableTest[T any](t *testing.T, tests map[string]T, testFunc func(*testing.T, T)) {
	t.Helper()
	
	for name, test := range tests {
		test := test // Capture loop variable
		
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testFunc(t, test)
		})
	}
}

// CleanupFunc is a function that performs cleanup.
type CleanupFunc func()

// WithCleanup runs a function with automatic cleanup.
func WithCleanup(t *testing.T, setup func() CleanupFunc, testFunc func()) {
	t.Helper()
	
	cleanup := setup()
	if cleanup != nil {
		t.Cleanup(cleanup)
	}
	
	testFunc()
}

// NormalizeWhitespace normalizes whitespace in strings for comparison.
func NormalizeWhitespace(s string) string {
	// Replace multiple whitespace with single space and trim
	lines := strings.Fields(s)
	return strings.Join(lines, " ")
}