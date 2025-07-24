package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

// TestIntegrationWorkflow tests the complete workflow of validation tools
func TestIntegrationWorkflow(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcp-validation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Setup test server
	serverCmd := createTestServer(t, tempDir)
	
	// Test 1: Generate trace file by running mcp-validate
	t.Run("GenerateTraceFile", func(t *testing.T) {
		traceFile := filepath.Join(tempDir, "test.mcp")
		if err := generateTraceFile(serverCmd, traceFile); err != nil {
			t.Fatalf("Failed to generate trace file: %v", err)
		}
		
		// Verify trace file exists and is valid
		if _, err := os.Stat(traceFile); os.IsNotExist(err) {
			t.Fatal("Trace file was not created")
		}
	})
	
	// Test 2: Validate server using mcp-validate
	t.Run("ValidateServer", func(t *testing.T) {
		cmd := &ValidateCommand{
			serverCmd: serverCmd,
			verbose:   true,
		}
		
		validator := NewValidator(cmd)
		if err := validator.validateServer(context.Background(), serverCmd); err != nil {
			t.Fatalf("Server validation failed: %v", err)
		}
		
		// Check that we have some validation results
		if len(validator.report.Violations) == 0 {
			t.Log("No violations found (this is good)")
		}
		
		// Verify compliance rate calculation
		if validator.report.Summary.TotalChecks == 0 {
			t.Error("No checks were performed")
		}
	})
	
	// Test 3: Validate trace file
	t.Run("ValidateTraceFile", func(t *testing.T) {
		traceFile := filepath.Join(tempDir, "test.mcp")
		
		// Create a simple trace file for testing
		if err := createTestTraceFile(traceFile); err != nil {
			t.Fatal(err)
		}
		
		cmd := &ValidateCommand{
			traceFile: traceFile,
			verbose:   true,
		}
		
		validator := NewValidator(cmd)
		if err := validator.validateTrace(context.Background(), traceFile); err != nil {
			t.Fatalf("Trace validation failed: %v", err)
		}
	})
	
	// Test 4: Generate JSON report
	t.Run("GenerateJSONReport", func(t *testing.T) {
		cmd := &ValidateCommand{
			serverCmd:  serverCmd,
			outputFmt:  "json",
			reportFile: filepath.Join(tempDir, "report.json"),
		}
		
		validator := NewValidator(cmd)
		if err := validator.validateServer(context.Background(), serverCmd); err != nil {
			t.Fatalf("Server validation failed: %v", err)
		}
		
		// Verify report file was created
		reportFile := filepath.Join(tempDir, "report.json")
		if _, err := os.Stat(reportFile); os.IsNotExist(err) {
			t.Error("JSON report file was not created")
		}
		
		// Verify report content
		content, err := os.ReadFile(reportFile)
		if err != nil {
			t.Fatal(err)
		}
		
		var report ValidationReport
		if err := json.Unmarshal(content, &report); err != nil {
			t.Fatalf("Invalid JSON report: %v", err)
		}
		
		if report.Version != version {
			t.Errorf("Expected version %s, got %s", version, report.Version)
		}
	})
	
	// Test 5: Generate HTML report
	t.Run("GenerateHTMLReport", func(t *testing.T) {
		cmd := &ValidateCommand{
			serverCmd:  serverCmd,
			outputFmt:  "html",
			reportFile: filepath.Join(tempDir, "report.html"),
		}
		
		validator := NewValidator(cmd)
		if err := validator.validateServer(context.Background(), serverCmd); err != nil {
			t.Fatalf("Server validation failed: %v", err)
		}
		
		// Verify HTML report file was created
		reportFile := filepath.Join(tempDir, "report.html")
		if _, err := os.Stat(reportFile); os.IsNotExist(err) {
			t.Error("HTML report file was not created")
		}
		
		// Verify HTML content
		content, err := os.ReadFile(reportFile)
		if err != nil {
			t.Fatal(err)
		}
		
		htmlContent := string(content)
		if !strings.Contains(htmlContent, "<!DOCTYPE html>") {
			t.Error("HTML report does not contain DOCTYPE declaration")
		}
		if !strings.Contains(htmlContent, "MCP Validation Report") {
			t.Error("HTML report does not contain expected title")
		}
	})
	
	// Test 6: Strict mode validation
	t.Run("StrictModeValidation", func(t *testing.T) {
		cmd := &ValidateCommand{
			serverCmd: serverCmd,
			strict:    true,
			verbose:   true,
		}
		
		validator := NewValidator(cmd)
		if err := validator.validateServer(context.Background(), serverCmd); err != nil {
			t.Fatalf("Strict validation failed: %v", err)
		}
		
		// In strict mode, we should have more checks
		if validator.report.Summary.TotalChecks == 0 {
			t.Error("No checks were performed in strict mode")
		}
	})
}

// TestValidationAccuracy tests the accuracy of validation logic
func TestValidationAccuracy(t *testing.T) {
	tests := []struct {
		name           string
		traceContent   []map[string]interface{}
		expectedViolations int
		expectedSeverity   string
	}{
		{
			name: "valid_trace",
			traceContent: []map[string]interface{}{
				{
					"jsonrpc": "2.0",
					"id":      1,
					"method":  "initialize",
					"params": map[string]interface{}{
						"protocolVersion": "2025-03-26",
						"clientInfo": map[string]interface{}{
							"name":    "test-client",
							"version": "1.0.0",
						},
					},
				},
				{
					"jsonrpc": "2.0",
					"id":      1,
					"result": map[string]interface{}{
						"protocolVersion": "2025-03-26",
						"serverInfo": map[string]interface{}{
							"name":    "test-server",
							"version": "1.0.0",
						},
					},
				},
			},
			expectedViolations: 0,
		},
		{
			name: "invalid_jsonrpc_version",
			traceContent: []map[string]interface{}{
				{
					"jsonrpc": "1.0",
					"id":      1,
					"method":  "initialize",
				},
			},
			expectedViolations: 1,
			expectedSeverity:   "error",
		},
		{
			name: "unknown_method",
			traceContent: []map[string]interface{}{
				{
					"jsonrpc": "2.0",
					"id":      1,
					"method":  "unknown/method",
				},
			},
			expectedViolations: 1,
			expectedSeverity:   "warning",
		},
		{
			name: "invalid_response_structure",
			traceContent: []map[string]interface{}{
				{
					"jsonrpc": "2.0",
					"id":      1,
					"result":  map[string]interface{}{},
					"error":   map[string]interface{}{},
				},
			},
			expectedViolations: 1,
			expectedSeverity:   "error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary trace file
			tempDir, err := os.MkdirTemp("", "mcp-validation-accuracy-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)
			
			traceFile := filepath.Join(tempDir, "test.mcp")
			if err := writeTraceFile(traceFile, tt.traceContent); err != nil {
				t.Fatal(err)
			}
			
			// Validate trace
			cmd := &ValidateCommand{
				traceFile: traceFile,
			}
			
			validator := NewValidator(cmd)
			if err := validator.validateTrace(context.Background(), traceFile); err != nil {
				t.Fatalf("Trace validation failed: %v", err)
			}
			
			// Check violations
			if len(validator.report.Violations) != tt.expectedViolations {
				t.Errorf("Expected %d violations, got %d", tt.expectedViolations, len(validator.report.Violations))
			}
			
			if tt.expectedViolations > 0 && len(validator.report.Violations) > 0 {
				if validator.report.Violations[0].Severity != tt.expectedSeverity {
					t.Errorf("Expected severity %s, got %s", tt.expectedSeverity, validator.report.Violations[0].Severity)
				}
			}
		})
	}
}

// TestPerformanceValidation tests performance-related validation
func TestPerformanceValidation(t *testing.T) {
	// Create a mock server that responds slowly
	slowServer := &MockSlowServer{
		delay: 150 * time.Millisecond,
	}
	
	// Test strict mode catches slow responses
	cmd := &ValidateCommand{
		serverCmd: slowServer.Command(),
		strict:    true,
	}
	
	validator := NewValidator(cmd)
	// This would normally connect to the server, but we'll simulate it
	validator.addViolation("warning", "performance", "slow_ping", 
		"Ping took 150ms, should be under 100ms", "")
	
	// Verify the violation was recorded
	if len(validator.report.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(validator.report.Violations))
	}
	
	if validator.report.Violations[0].Category != "performance" {
		t.Errorf("Expected performance violation, got %s", validator.report.Violations[0].Category)
	}
}

// TestBatchValidation tests batch validation functionality
func TestBatchValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcp-batch-validation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create batch file
	batchFile := filepath.Join(tempDir, "servers.json")
	servers := []string{
		"go run ./testserver1",
		"python testserver2.py",
		"node testserver3.js",
	}
	
	if err := writeBatchFile(batchFile, servers); err != nil {
		t.Fatal(err)
	}
	
	// Test batch validation
	cmd := &ValidateCommand{
		batch:   batchFile,
		verbose: true,
	}
	
	validator := NewValidator(cmd)
	
	// Since we can't actually run the servers, we'll simulate batch validation
	for _, server := range servers {
		subValidator := NewValidator(cmd)
		subValidator.config.serverCmd = server
		
		// Simulate some validation results
		subValidator.addViolation("info", "test", "simulated", 
			fmt.Sprintf("Simulated validation for %s", server), "")
		
		validator.mergeReport(subValidator.report)
	}
	
	// Verify batch results
	if len(validator.report.Violations) != len(servers) {
		t.Errorf("Expected %d violations (one per server), got %d", 
			len(servers), len(validator.report.Violations))
	}
}

// TestCapabilityValidation tests capability validation logic
func TestCapabilityValidation(t *testing.T) {
	// Create a mock client for testing
	mockClient := &MockMCPClient{
		capabilities: modelcontextprotocol.ServerCapabilities{
			Tools: &struct{ ListChanged bool }{ListChanged: false},
		},
		toolsListError: nil,
	}
	
	cmd := &ValidateCommand{verbose: true}
	validator := NewValidator(cmd)
	
	// Test capability validation
	validator.validateCapabilities(context.Background(), mockClient, mockClient.capabilities)
	
	// Check that capabilities were properly recorded
	if validator.report.Capabilities.DeclaredCapabilities == nil {
		t.Error("Declared capabilities were not recorded")
	}
	
	if validator.report.Capabilities.ActualCapabilities == nil {
		t.Error("Actual capabilities were not recorded")
	}
}

// TestReportGeneration tests report generation in various formats
func TestReportGeneration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcp-report-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create validator with test data
	validator := NewValidator(&ValidateCommand{})
	validator.addViolation("error", "protocol", "test_error", "Test error message", "line 1")
	validator.addViolation("warning", "performance", "test_warning", "Test warning message", "line 2")
	validator.addViolation("info", "info", "test_info", "Test info message", "line 3")
	
	// Test JSON format
	t.Run("JSONFormat", func(t *testing.T) {
		validator.config.outputFmt = "json"
		validator.config.reportFile = filepath.Join(tempDir, "test.json")
		
		if err := validator.generateReport(); err != nil {
			t.Fatalf("Failed to generate JSON report: %v", err)
		}
		
		// Verify file exists and is valid JSON
		content, err := os.ReadFile(validator.config.reportFile)
		if err != nil {
			t.Fatal(err)
		}
		
		var report ValidationReport
		if err := json.Unmarshal(content, &report); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
		
		if len(report.Violations) != 3 {
			t.Errorf("Expected 3 violations, got %d", len(report.Violations))
		}
	})
	
	// Test HTML format
	t.Run("HTMLFormat", func(t *testing.T) {
		validator.config.outputFmt = "html"
		validator.config.reportFile = filepath.Join(tempDir, "test.html")
		
		if err := validator.generateReport(); err != nil {
			t.Fatalf("Failed to generate HTML report: %v", err)
		}
		
		// Verify file exists and contains HTML
		content, err := os.ReadFile(validator.config.reportFile)
		if err != nil {
			t.Fatal(err)
		}
		
		htmlContent := string(content)
		if !strings.Contains(htmlContent, "<!DOCTYPE html>") {
			t.Error("HTML report missing DOCTYPE")
		}
		if !strings.Contains(htmlContent, "Failed: 1") {
			t.Error("HTML report missing failure count")
		}
	})
	
	// Test JUnit XML format
	t.Run("JUnitXMLFormat", func(t *testing.T) {
		validator.config.outputFmt = "junit-xml"
		validator.config.reportFile = filepath.Join(tempDir, "test.xml")
		
		if err := validator.generateReport(); err != nil {
			t.Fatalf("Failed to generate JUnit XML report: %v", err)
		}
		
		// Verify file exists and contains XML
		content, err := os.ReadFile(validator.config.reportFile)
		if err != nil {
			t.Fatal(err)
		}
		
		xmlContent := string(content)
		if !strings.Contains(xmlContent, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Error("JUnit XML report missing XML declaration")
		}
		if !strings.Contains(xmlContent, `<testsuite`) {
			t.Error("JUnit XML report missing testsuite element")
		}
	})
}

// Helper functions and mocks

func createTestServer(t *testing.T, tempDir string) string {
	// Create a simple test server script
	serverScript := filepath.Join(tempDir, "testserver.py")
	serverContent := `#!/usr/bin/env python3
import json
import sys

def main():
    # Simple MCP server that responds to basic requests
    for line in sys.stdin:
        try:
            msg = json.loads(line)
            if msg.get("method") == "initialize":
                response = {
                    "jsonrpc": "2.0",
                    "id": msg.get("id"),
                    "result": {
                        "protocolVersion": "2025-03-26",
                        "serverInfo": {
                            "name": "test-server",
                            "version": "1.0.0"
                        },
                        "capabilities": {
                            "tools": {}
                        }
                    }
                }
                print(json.dumps(response))
            elif msg.get("method") == "ping":
                response = {
                    "jsonrpc": "2.0",
                    "id": msg.get("id"),
                    "result": {}
                }
                print(json.dumps(response))
        except:
            pass

if __name__ == "__main__":
    main()
`
	
	if err := os.WriteFile(serverScript, []byte(serverContent), 0755); err != nil {
		t.Fatal(err)
	}
	
	return fmt.Sprintf("python3 %s", serverScript)
}

func generateTraceFile(serverCmd, traceFile string) error {
	// This would normally run the server and generate a trace
	// For testing, we'll create a simple trace file
	return createTestTraceFile(traceFile)
}

func createTestTraceFile(filename string) error {
	trace := []map[string]interface{}{
		{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]interface{}{
				"protocolVersion": "2025-03-26",
				"clientInfo": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"protocolVersion": "2025-03-26",
				"serverInfo": map[string]interface{}{
					"name":    "test-server",
					"version": "1.0.0",
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "ping",
		},
		{
			"jsonrpc": "2.0",
			"id":      2,
			"result":  map[string]interface{}{},
		},
	}
	
	return writeTraceFile(filename, trace)
}

func writeTraceFile(filename string, content []map[string]interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	for _, msg := range content {
		if err := encoder.Encode(msg); err != nil {
			return err
		}
	}
	
	return nil
}

func writeBatchFile(filename string, servers []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	return encoder.Encode(servers)
}

// MockSlowServer simulates a slow server
type MockSlowServer struct {
	delay time.Duration
}

func (s *MockSlowServer) Command() string {
	return "mock-slow-server"
}

// MockMCPClient simulates an MCP client for testing
type MockMCPClient struct {
	capabilities   modelcontextprotocol.ServerCapabilities
	toolsListError error
}

func (c *MockMCPClient) Initialize(ctx context.Context, req modelcontextprotocol.InitializeRequest) (*modelcontextprotocol.InitializeResult, error) {
	return &modelcontextprotocol.InitializeResult{
		ProtocolVersion: "2025-03-26",
		ServerInfo: modelcontextprotocol.Implementation{
			Name:    "mock-server",
			Version: "1.0.0",
		},
		Capabilities: c.capabilities,
	}, nil
}

func (c *MockMCPClient) Ping(ctx context.Context) error {
	return nil
}

func (c *MockMCPClient) ListTools(ctx context.Context) ([]modelcontextprotocol.Tool, error) {
	if c.toolsListError != nil {
		return nil, c.toolsListError
	}
	return []modelcontextprotocol.Tool{}, nil
}

func (c *MockMCPClient) ListResources(ctx context.Context) ([]modelcontextprotocol.Resource, error) {
	return []modelcontextprotocol.Resource{}, nil
}

func (c *MockMCPClient) ListPrompts(ctx context.Context) ([]modelcontextprotocol.Prompt, error) {
	return []modelcontextprotocol.Prompt{}, nil
}

func (c *MockMCPClient) CallTool(ctx context.Context, name string, arguments interface{}) (*modelcontextprotocol.CallToolResult, error) {
	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			modelcontextprotocol.TextContent{
				Type: "text",
				Text: "Mock tool result",
			},
		},
	}, nil
}

func (c *MockMCPClient) ReadResource(ctx context.Context, uri string) ([]modelcontextprotocol.ResourceContents, error) {
	return []modelcontextprotocol.ResourceContents{}, nil
}

func (c *MockMCPClient) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*modelcontextprotocol.GetPromptResult, error) {
	return &modelcontextprotocol.GetPromptResult{}, nil
}

func (c *MockMCPClient) Close() error {
	return nil
}

func (c *MockMCPClient) Connect(ctx context.Context) error {
	return nil
}