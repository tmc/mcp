// Package main implements an LSP server for mcpscripttest and mcptrace files
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Global logger
var logger *log.Logger

// JSON-RPC constants
const (
	jsonrpcVersion = "2.0"
)

// Error codes defined in the JSON-RPC spec
const (
	parseError      = -32700
	invalidRequest  = -32600
	methodNotFound  = -32601
	invalidParams   = -32602
	internalError   = -32603
	serverErrorBase = -32000
)

// LSPServer handles LSP requests for MCP trace and scripttest files
type LSPServer struct {
	rootPath       string
	documents      map[string]string
	documentsMutex sync.RWMutex
	initialized    bool
	stdin          *bufio.Reader
	stdout         io.Writer
}

// NewLSPServer creates a new LSP server instance
func NewLSPServer() *LSPServer {
	return &LSPServer{
		documents:      make(map[string]string),
		documentsMutex: sync.RWMutex{},
		stdin:          bufio.NewReader(os.Stdin),
		stdout:         os.Stdout,
	}
}

// JSONRPCMessage represents a JSON-RPC message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Run starts the LSP server
func (s *LSPServer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read message from stdin
			message, err := s.readMessage()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				logger.Printf("Error reading message: %v", err)
				continue
			}

			// Process message
			go s.handleMessage(ctx, message)
		}
	}
}

// readMessage reads a JSON-RPC message from stdin
func (s *LSPServer) readMessage() (JSONRPCMessage, error) {
	// Read content length header
	header, err := s.stdin.ReadString('\n')
	if err != nil {
		return JSONRPCMessage{}, err
	}

	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "Content-Length: ") {
		return JSONRPCMessage{}, fmt.Errorf("invalid header: %s", header)
	}

	// Parse content length
	contentLength := 0
	_, err = fmt.Sscanf(header, "Content-Length: %d", &contentLength)
	if err != nil {
		return JSONRPCMessage{}, err
	}

	// Read the empty line
	_, err = s.stdin.ReadString('\n')
	if err != nil {
		return JSONRPCMessage{}, err
	}

	// Read the message content
	content := make([]byte, contentLength)
	_, err = io.ReadFull(s.stdin, content)
	if err != nil {
		return JSONRPCMessage{}, err
	}

	// Parse the message
	var message JSONRPCMessage
	err = json.Unmarshal(content, &message)
	if err != nil {
		return JSONRPCMessage{}, err
	}

	return message, nil
}

// handleMessage processes an incoming JSON-RPC message
func (s *LSPServer) handleMessage(ctx context.Context, message JSONRPCMessage) {
	if message.Method == "initialize" {
		s.handleInitialize(ctx, message)
		return
	}

	if !s.initialized && message.Method != "exit" {
		s.sendErrorResponse(message.ID, invalidRequest, "Server not initialized")
		return
	}

	switch message.Method {
	case "initialized":
		s.handleInitialized(ctx, message)
	case "shutdown":
		s.handleShutdown(ctx, message)
	case "exit":
		s.handleExit(ctx, message)
	case "textDocument/didOpen":
		s.handleTextDocumentDidOpen(ctx, message)
	case "textDocument/didChange":
		s.handleTextDocumentDidChange(ctx, message)
	case "textDocument/didClose":
		s.handleTextDocumentDidClose(ctx, message)
	case "textDocument/completion":
		s.handleTextDocumentCompletion(ctx, message)
	case "textDocument/hover":
		s.handleTextDocumentHover(ctx, message)
	default:
		if message.ID != nil {
			s.sendErrorResponse(message.ID, methodNotFound, fmt.Sprintf("Method not supported: %s", message.Method))
		}
	}
}

// sendResponse sends a JSON-RPC response
func (s *LSPServer) sendResponse(id json.RawMessage, result interface{}) {
	response := JSONRPCMessage{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Result:  result,
	}

	s.sendMessage(response)
}

// sendErrorResponse sends a JSON-RPC error response
func (s *LSPServer) sendErrorResponse(id json.RawMessage, code int, message string) {
	response := JSONRPCMessage{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	s.sendMessage(response)
}

// sendNotification sends a JSON-RPC notification
func (s *LSPServer) sendNotification(method string, params interface{}) {
	notification := JSONRPCMessage{
		JSONRPC: jsonrpcVersion,
		Method:  method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			logger.Printf("Error marshaling notification params: %v", err)
			return
		}
		notification.Params = data
	}

	s.sendMessage(notification)
}

// sendMessage sends a JSON-RPC message to stdout
func (s *LSPServer) sendMessage(message JSONRPCMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		logger.Printf("Error marshaling message: %v", err)
		return
	}

	// Write the header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	fmt.Fprint(s.stdout, header)

	// Write the content
	_, err = s.stdout.Write(data)
	if err != nil {
		logger.Printf("Error writing message: %v", err)
	}
}

// handleInitialize handles the initialize request
func (s *LSPServer) handleInitialize(ctx context.Context, message JSONRPCMessage) {
	var params struct {
		RootPath     string   `json:"rootPath,omitempty"`
		RootURI      string   `json:"rootUri,omitempty"`
		Capabilities struct{} `json:"capabilities"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		s.sendErrorResponse(message.ID, parseError, fmt.Sprintf("Failed to parse initialize params: %v", err))
		return
	}

	s.rootPath = params.RootPath
	if s.rootPath == "" && params.RootURI != "" {
		s.rootPath = strings.TrimPrefix(params.RootURI, "file://")
	}

	// Define server capabilities
	result := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"textDocumentSync": 1, // Full document sync
			"completionProvider": map[string]interface{}{
				"resolveProvider":   false,
				"triggerCharacters": []string{">", ":"},
			},
			"hoverProvider": true,
		},
		"serverInfo": map[string]interface{}{
			"name":    "mcpspec-lsp",
			"version": "0.1.0",
		},
	}

	s.initialized = true
	s.sendResponse(message.ID, result)
	logger.Printf("Initialized LSP server with root path: %s", s.rootPath)
}

// handleInitialized handles the initialized notification
func (s *LSPServer) handleInitialized(ctx context.Context, message JSONRPCMessage) {
	logger.Printf("Client initialized")
}

// handleShutdown handles the shutdown request
func (s *LSPServer) handleShutdown(ctx context.Context, message JSONRPCMessage) {
	s.initialized = false
	s.sendResponse(message.ID, nil)
	logger.Printf("Server shutting down")
}

// handleExit handles the exit notification
func (s *LSPServer) handleExit(ctx context.Context, message JSONRPCMessage) {
	logger.Printf("Received exit notification")
	os.Exit(0)
}

// handleTextDocumentDidOpen handles textDocument/didOpen notifications
func (s *LSPServer) handleTextDocumentDidOpen(ctx context.Context, message JSONRPCMessage) {
	var params struct {
		TextDocument struct {
			URI        string `json:"uri"`
			LanguageID string `json:"languageId"`
			Version    int    `json:"version"`
			Text       string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		logger.Printf("Error parsing didOpen params: %v", err)
		return
	}

	uri := params.TextDocument.URI
	text := params.TextDocument.Text

	// Store document
	s.documentsMutex.Lock()
	s.documents[uri] = text
	s.documentsMutex.Unlock()

	// Send diagnostics
	s.processDiagnostics(uri, text)

	logger.Printf("Document opened: %s", uri)
}

// handleTextDocumentDidChange handles textDocument/didChange notifications
func (s *LSPServer) handleTextDocumentDidChange(ctx context.Context, message JSONRPCMessage) {
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		logger.Printf("Error parsing didChange params: %v", err)
		return
	}

	uri := params.TextDocument.URI
	// We're using full document sync so we should have exactly one change with the full content
	if len(params.ContentChanges) > 0 {
		text := params.ContentChanges[0].Text

		// Update document
		s.documentsMutex.Lock()
		s.documents[uri] = text
		s.documentsMutex.Unlock()

		// Send diagnostics
		s.processDiagnostics(uri, text)
	}

	logger.Printf("Document changed: %s", uri)
}

// handleTextDocumentDidClose handles textDocument/didClose notifications
func (s *LSPServer) handleTextDocumentDidClose(ctx context.Context, message JSONRPCMessage) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		logger.Printf("Error parsing didClose params: %v", err)
		return
	}

	uri := params.TextDocument.URI

	// Remove document
	s.documentsMutex.Lock()
	delete(s.documents, uri)
	s.documentsMutex.Unlock()

	logger.Printf("Document closed: %s", uri)
}

// handleTextDocumentCompletion handles textDocument/completion requests
func (s *LSPServer) handleTextDocumentCompletion(ctx context.Context, message JSONRPCMessage) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		s.sendErrorResponse(message.ID, parseError, fmt.Sprintf("Failed to parse completion params: %v", err))
		return
	}

	uri := params.TextDocument.URI

	// Get document content
	s.documentsMutex.RLock()
	content, ok := s.documents[uri]
	s.documentsMutex.RUnlock()

	if !ok {
		s.sendErrorResponse(message.ID, invalidParams, fmt.Sprintf("Document not found: %s", uri))
		return
	}

	// Generate completions based on document type and context
	var completionItems []map[string]interface{}

	if isMCPTraceFile(uri) {
		completionItems = getMCPTraceCompletions(content, params.Position.Line, params.Position.Character)
	} else if isMCPScriptFile(uri) {
		completionItems = getMCPScriptCompletions(content, params.Position.Line, params.Position.Character)
	}

	s.sendResponse(message.ID, completionItems)
}

// handleTextDocumentHover handles textDocument/hover requests
func (s *LSPServer) handleTextDocumentHover(ctx context.Context, message JSONRPCMessage) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
	}
	if err := json.Unmarshal(message.Params, &params); err != nil {
		s.sendErrorResponse(message.ID, parseError, fmt.Sprintf("Failed to parse hover params: %v", err))
		return
	}

	uri := params.TextDocument.URI

	// Get document content
	s.documentsMutex.RLock()
	content, ok := s.documents[uri]
	s.documentsMutex.RUnlock()

	if !ok {
		s.sendErrorResponse(message.ID, invalidParams, fmt.Sprintf("Document not found: %s", uri))
		return
	}

	// Generate hover response based on document type and context
	var hoverContent map[string]interface{}

	if isMCPTraceFile(uri) {
		hoverContent = getMCPTraceHover(content, params.Position.Line, params.Position.Character)
	} else if isMCPScriptFile(uri) {
		hoverContent = getMCPScriptHover(content, params.Position.Line, params.Position.Character)
	}

	s.sendResponse(message.ID, hoverContent)
}

// processDiagnostics generates and publishes diagnostics for a document
func (s *LSPServer) processDiagnostics(uri string, content string) {
	diagnostics := generateDiagnostics(uri, content)

	// Publish diagnostics
	params := map[string]interface{}{
		"uri":         uri,
		"diagnostics": diagnostics,
	}

	s.sendNotification("textDocument/publishDiagnostics", params)
}

// Helper functions

// isMCPTraceFile checks if a file is an MCP trace file
func isMCPTraceFile(uri string) bool {
	ext := filepath.Ext(uri)
	return ext == ".mcp" || ext == ".trace" || ext == ".mcptrace"
}

// isMCPScriptFile checks if a file is an MCP script file
func isMCPScriptFile(uri string) bool {
	ext := filepath.Ext(uri)
	return ext == ".txt" && (strings.Contains(uri, "scripttest") ||
		strings.Contains(uri, "testdata") ||
		strings.Contains(uri, "test.txt") ||
		strings.Contains(uri, "_test"))
}

// generateDiagnostics generates diagnostics for a document
func generateDiagnostics(uri string, content string) []map[string]interface{} {
	var diagnostics []map[string]interface{}

	if isMCPTraceFile(uri) {
		diagnostics = validateMCPTraceFile(content)
	} else if isMCPScriptFile(uri) {
		diagnostics = validateMCPScriptFile(content)
	}

	return diagnostics
}

// validateMCPTraceFile validates an MCP trace file and returns diagnostics
func validateMCPTraceFile(content string) []map[string]interface{} {
	var diagnostics []map[string]interface{}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if line follows mcp-send/mcp-recv format
		if !strings.HasPrefix(line, "mcp-send") && !strings.HasPrefix(line, "mcp-recv") {
			diagnostics = append(diagnostics, map[string]interface{}{
				"range": map[string]interface{}{
					"start": map[string]interface{}{
						"line":      i,
						"character": 0,
					},
					"end": map[string]interface{}{
						"line":      i,
						"character": len(line),
					},
				},
				"severity": 1, // Error
				"source":   "mcpspec-lsp",
				"message":  "MCP trace line must start with 'mcp-send' or 'mcp-recv'",
			})
			continue
		}

		// Extract the JSON part
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			diagnostics = append(diagnostics, map[string]interface{}{
				"range": map[string]interface{}{
					"start": map[string]interface{}{
						"line":      i,
						"character": 0,
					},
					"end": map[string]interface{}{
						"line":      i,
						"character": len(line),
					},
				},
				"severity": 1, // Error
				"source":   "mcpspec-lsp",
				"message":  "Invalid MCP trace format, expected 'mcp-send JSON' or 'mcp-recv JSON'",
			})
			continue
		}

		// Check if JSON is valid
		jsonText := parts[1]
		jsonStart := strings.IndexRune(line, '{')
		if jsonStart >= 0 {
			jsonText = line[jsonStart:]

			// Strip timestamp if present
			if hashIdx := strings.LastIndex(jsonText, "#"); hashIdx > 0 {
				jsonText = jsonText[:hashIdx]
			}

			var jsonObj interface{}
			if err := json.Unmarshal([]byte(jsonText), &jsonObj); err != nil {
				// Find exact position of JSON error
				errorPos := 0
				if strings.Contains(err.Error(), "offset") {
					fmt.Sscanf(err.Error(), "invalid character '%c' looking for beginning of object key string at offset %d", nil, &errorPos)
				}

				diagnostics = append(diagnostics, map[string]interface{}{
					"range": map[string]interface{}{
						"start": map[string]interface{}{
							"line":      i,
							"character": jsonStart + errorPos,
						},
						"end": map[string]interface{}{
							"line":      i,
							"character": jsonStart + errorPos + 1,
						},
					},
					"severity": 1, // Error
					"source":   "mcpspec-lsp",
					"message":  fmt.Sprintf("Invalid JSON: %v", err),
				})
			}
		}
	}

	return diagnostics
}

// validateMCPScriptFile validates an MCP script file and returns diagnostics
func validateMCPScriptFile(content string) []map[string]interface{} {
	var diagnostics []map[string]interface{}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for common script commands
		if strings.HasPrefix(line, ">") {
			// Command line, check for common errors
			cmd := strings.TrimSpace(line[1:])

			// Check for missing command
			if cmd == "" {
				diagnostics = append(diagnostics, map[string]interface{}{
					"range": map[string]interface{}{
						"start": map[string]interface{}{
							"line":      i,
							"character": 0,
						},
						"end": map[string]interface{}{
							"line":      i,
							"character": len(line),
						},
					},
					"severity": 1, // Error
					"source":   "mcpspec-lsp",
					"message":  "Empty command after '>'",
				})
			}
		}
	}

	return diagnostics
}

// getMCPTraceCompletions generates completions for MCP trace files
func getMCPTraceCompletions(content string, line int, character int) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"label":      "mcp-send",
			"kind":       14, // Keyword
			"detail":     "MCP send message",
			"insertText": "mcp-send {\"jsonrpc\":\"2.0\"}",
		},
		{
			"label":      "mcp-recv",
			"kind":       14, // Keyword
			"detail":     "MCP receive message",
			"insertText": "mcp-recv {\"jsonrpc\":\"2.0\"}",
		},
		{
			"label":      "jsonrpc request",
			"kind":       7, // Class
			"detail":     "JSON-RPC request template",
			"insertText": "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$1\",\"params\":{$2}}",
		},
		{
			"label":      "jsonrpc response",
			"kind":       7, // Class
			"detail":     "JSON-RPC response template",
			"insertText": "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{$1}}",
		},
		{
			"label":      "jsonrpc error",
			"kind":       7, // Class
			"detail":     "JSON-RPC error template",
			"insertText": "{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32000,\"message\":\"$1\"}}",
		},
	}
}

// getMCPScriptCompletions generates completions for MCP script files
func getMCPScriptCompletions(content string, line int, character int) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"label":      "> exec",
			"kind":       14, // Keyword
			"detail":     "Execute a command",
			"insertText": "> exec $1",
		},
		{
			"label":      "> cat",
			"kind":       14, // Keyword
			"detail":     "Display file content",
			"insertText": "> cat $1",
		},
		{
			"label":      "> cmp",
			"kind":       14, // Keyword
			"detail":     "Compare files",
			"insertText": "> cmp $1 $2",
		},
		{
			"label":      "> stdout",
			"kind":       14, // Keyword
			"detail":     "Check stdout for pattern",
			"insertText": "> stdout '$1'",
		},
		{
			"label":      "> stderr",
			"kind":       14, // Keyword
			"detail":     "Check stderr for pattern",
			"insertText": "> stderr '$1'",
		},
	}
}

// getMCPTraceHover generates hover info for MCP trace files
func getMCPTraceHover(content string, line int, character int) map[string]interface{} {
	lines := strings.Split(content, "\n")
	if line >= len(lines) {
		return nil
	}

	currentLine := lines[line]
	if strings.HasPrefix(currentLine, "mcp-send") {
		return map[string]interface{}{
			"contents": map[string]interface{}{
				"kind":  "markdown",
				"value": "**MCP Send Message**\n\nA message sent from the server to the client.",
			},
		}
	} else if strings.HasPrefix(currentLine, "mcp-recv") {
		return map[string]interface{}{
			"contents": map[string]interface{}{
				"kind":  "markdown",
				"value": "**MCP Receive Message**\n\nA message received by the server from the client.",
			},
		}
	}

	// Look for JSON content
	jsonStart := strings.IndexRune(currentLine, '{')
	if jsonStart >= 0 && character >= jsonStart {
		return map[string]interface{}{
			"contents": map[string]interface{}{
				"kind":  "markdown",
				"value": "**JSON-RPC Message**\n\nA JSON-RPC message for MCP protocol communication.",
			},
		}
	}

	return nil
}

// getMCPScriptHover generates hover info for MCP script files
func getMCPScriptHover(content string, line int, character int) map[string]interface{} {
	lines := strings.Split(content, "\n")
	if line >= len(lines) {
		return nil
	}

	currentLine := lines[line]
	if strings.HasPrefix(currentLine, ">") {
		cmd := strings.TrimSpace(currentLine[1:])

		if strings.HasPrefix(cmd, "exec") {
			return map[string]interface{}{
				"contents": map[string]interface{}{
					"kind":  "markdown",
					"value": "**Exec Command**\n\nExecutes a command and expects it to succeed (exit code 0).",
				},
			}
		} else if strings.HasPrefix(cmd, "cat") {
			return map[string]interface{}{
				"contents": map[string]interface{}{
					"kind":  "markdown",
					"value": "**Cat Command**\n\nPrints the contents of a file.",
				},
			}
		} else if strings.HasPrefix(cmd, "stdout") {
			return map[string]interface{}{
				"contents": map[string]interface{}{
					"kind":  "markdown",
					"value": "**Stdout Check**\n\nChecks if stdout contains the given pattern.",
				},
			}
		} else if strings.HasPrefix(cmd, "stderr") {
			return map[string]interface{}{
				"contents": map[string]interface{}{
					"kind":  "markdown",
					"value": "**Stderr Check**\n\nChecks if stderr contains the given pattern.",
				},
			}
		}
	}

	return nil
}

func main() {
	// Parse command line flags
	logFile := flag.String("log", "", "Path to log file")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	// Setup logging
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening log file: %v", err)
		}
		defer f.Close()
		logger = log.New(f, "", log.LstdFlags)
	} else {
		// If no log file specified, log to stderr
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	if *verbose {
		logger.Printf("Starting mcpspec-lsp server with verbose logging")
	}

	// Create and run the server
	server := NewLSPServer()
	if err := server.Run(context.Background()); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}
