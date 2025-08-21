package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		setup   func(t *testing.T) (cleanup func())
	}{
		{
			name:    "no target specified",
			args:    []string{},
			wantErr: true,
		},
		{
			name: "validate trace file",
			args: []string{"--trace", "test.mcp"},
			setup: func(t *testing.T) func() {
				// Create test trace file
				trace := []interface{}{
					map[string]interface{}{
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
					map[string]interface{}{
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
				}

				file, err := os.Create("test.mcp")
				if err != nil {
					t.Fatal(err)
				}

				encoder := json.NewEncoder(file)
				for _, msg := range trace {
					if err := encoder.Encode(msg); err != nil {
						t.Fatal(err)
					}
				}
				file.Close()

				return func() {
					os.Remove("test.mcp")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			if tt.setup != nil {
				cleanup = tt.setup(t)
				defer cleanup()
			}

			cmd := &ValidateCommand{}
			err := cmd.Execute(context.Background(), tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_validateMessage(t *testing.T) {
	tests := []struct {
		name           string
		message        map[string]interface{}
		wantViolations int
		wantSeverity   string
	}{
		{
			name: "valid request",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "initialize",
				"params":  map[string]interface{}{},
			},
			wantViolations: 0,
		},
		{
			name: "invalid jsonrpc version",
			message: map[string]interface{}{
				"jsonrpc": "1.0",
				"id":      1,
				"method":  "initialize",
			},
			wantViolations: 1,
			wantSeverity:   "error",
		},
		{
			name: "request with result field",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "initialize",
				"result":  map[string]interface{}{},
			},
			wantViolations: 1,
			wantSeverity:   "error",
		},
		{
			name: "response without result or error",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
			},
			wantViolations: 1,
			wantSeverity:   "error",
		},
		{
			name: "response with both result and error",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  map[string]interface{}{},
				"error":   map[string]interface{}{},
			},
			wantViolations: 1,
			wantSeverity:   "error",
		},
		{
			name: "unknown method",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "unknown/method",
			},
			wantViolations: 1,
			wantSeverity:   "warning",
		},
		{
			name: "valid notification",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "notifications/progress",
				"params":  map[string]interface{}{},
			},
			wantViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(&ValidateCommand{})

			msgBytes, err := json.Marshal(tt.message)
			if err != nil {
				t.Fatal(err)
			}

			v.validateMessage(msgBytes, 1)

			if len(v.report.Violations) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(v.report.Violations), tt.wantViolations)
			}

			if tt.wantViolations > 0 && len(v.report.Violations) > 0 {
				if v.report.Violations[0].Severity != tt.wantSeverity {
					t.Errorf("got severity %s, want %s", v.report.Violations[0].Severity, tt.wantSeverity)
				}
			}
		})
	}
}

func TestValidator_validateProtocolVersion(t *testing.T) {
	v := NewValidator(&ValidateCommand{})

	// Test matching version
	v.validateProtocolVersion("2025-03-26")
	if len(v.report.Violations) != 0 {
		t.Errorf("expected no violations for matching version")
	}

	// Test mismatched version
	v.validateProtocolVersion("2024-01-01")
	if len(v.report.Violations) != 1 {
		t.Errorf("expected 1 violation for mismatched version")
	}
	if v.report.Violations[0].Severity != "warning" {
		t.Errorf("expected warning severity for version mismatch")
	}
}

func TestValidator_validateServerInfo(t *testing.T) {
	tests := []struct {
		name           string
		serverInfo     modelcontextprotocol.Implementation
		wantViolations int
	}{
		{
			name: "valid server info",
			serverInfo: modelcontextprotocol.Implementation{
				Name:    "test-server",
				Version: "1.0.0",
			},
			wantViolations: 0,
		},
		{
			name: "empty server name",
			serverInfo: modelcontextprotocol.Implementation{
				Name:    "",
				Version: "1.0.0",
			},
			wantViolations: 1,
		},
		{
			name: "empty server version",
			serverInfo: modelcontextprotocol.Implementation{
				Name:    "test-server",
				Version: "",
			},
			wantViolations: 1,
		},
		{
			name: "whitespace only name",
			serverInfo: modelcontextprotocol.Implementation{
				Name:    "   ",
				Version: "1.0.0",
			},
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(&ValidateCommand{})
			v.validateServerInfo(tt.serverInfo)

			if len(v.report.Violations) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(v.report.Violations), tt.wantViolations)
			}
		})
	}
}

func TestValidator_outputFormats(t *testing.T) {
	v := NewValidator(&ValidateCommand{})

	// Add some test violations
	v.addViolation("error", "test", "test_rule", "Test error message", "line 1")
	v.addViolation("warning", "test", "test_warning", "Test warning message", "line 2")

	t.Run("JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		if err := v.outputJSON(&buf); err != nil {
			t.Fatal(err)
		}

		var report ValidationReport
		if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}

		if len(report.Violations) != 2 {
			t.Errorf("expected 2 violations in JSON output")
		}
	})

	t.Run("JUnit XML output", func(t *testing.T) {
		var buf bytes.Buffer
		if err := v.outputJUnitXML(&buf); err != nil {
			t.Fatal(err)
		}

		output := buf.String()
		if !strings.Contains(output, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Error("missing XML declaration")
		}
		if !strings.Contains(output, `<testsuite`) {
			t.Error("missing testsuite element")
		}
		if !strings.Contains(output, `failures="1"`) {
			t.Error("incorrect failure count")
		}
	})

	t.Run("HTML output", func(t *testing.T) {
		var buf bytes.Buffer
		if err := v.outputHTML(&buf); err != nil {
			t.Fatal(err)
		}

		output := buf.String()
		if !strings.Contains(output, `<!DOCTYPE html>`) {
			t.Error("missing HTML doctype")
		}
		if !strings.Contains(output, `<title>MCP Validation Report</title>`) {
			t.Error("missing title")
		}
		if !strings.Contains(output, `Total Checks: 2`) {
			t.Error("incorrect total checks")
		}
		if !strings.Contains(output, `Failed: 1`) {
			t.Error("incorrect failed count")
		}
	})
}

func TestValidator_suggestions(t *testing.T) {
	tests := []struct {
		category       string
		rule           string
		wantSuggestion string
	}{
		{
			category:       "protocol",
			rule:           "version_mismatch",
			wantSuggestion: "Update server to use the latest protocol version",
		},
		{
			category:       "connection",
			rule:           "server_connect",
			wantSuggestion: "Check server is running and accessible",
		},
		{
			category:       "unknown",
			rule:           "unknown",
			wantSuggestion: "",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.category, tt.rule), func(t *testing.T) {
			got := getSuggestion(tt.category, tt.rule)
			if got != tt.wantSuggestion {
				t.Errorf("getSuggestion() = %v, want %v", got, tt.wantSuggestion)
			}
		})
	}
}

func TestValidator_isValidMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{"initialize", true},
		{"ping", true},
		{"tools/list", true},
		{"resources/read", true},
		{"notifications/progress", true},
		{"unknown/method", false},
		{"", false},
		{"tools/unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := isValidMethod(tt.method); got != tt.want {
				t.Errorf("isValidMethod(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestValidator_reportGeneration(t *testing.T) {
	v := NewValidator(&ValidateCommand{})

	// Add various violations
	v.addViolation("error", "protocol", "test1", "Error 1", "")
	v.addViolation("error", "protocol", "test2", "Error 2", "")
	v.addViolation("warning", "performance", "test3", "Warning 1", "")
	v.addViolation("info", "info", "test4", "Info 1", "")

	// Generate report to update summary
	var buf bytes.Buffer
	v.config.outputFmt = "json"
	if err := v.generateReport(); err != nil {
		t.Fatal(err)
	}

	// Check summary calculations
	if v.report.Summary.TotalChecks != 4 {
		t.Errorf("expected 4 total checks, got %d", v.report.Summary.TotalChecks)
	}
	if v.report.Summary.FailedChecks != 2 {
		t.Errorf("expected 2 failed checks, got %d", v.report.Summary.FailedChecks)
	}
	if v.report.Summary.WarningCount != 1 {
		t.Errorf("expected 1 warning, got %d", v.report.Summary.WarningCount)
	}
	if v.report.Summary.PassedChecks != 1 {
		t.Errorf("expected 1 passed check, got %d", v.report.Summary.PassedChecks)
	}
	if v.report.Summary.ComplianceRate != 25.0 {
		t.Errorf("expected 25%% compliance rate, got %.1f%%", v.report.Summary.ComplianceRate)
	}
	if v.report.Summary.Status != "failed" {
		t.Errorf("expected 'failed' status, got %s", v.report.Summary.Status)
	}
}

func TestValidator_mergeReport(t *testing.T) {
	v1 := NewValidator(&ValidateCommand{})
	v1.addViolation("error", "test", "test1", "Error 1", "")
	v1.report.Summary.TotalChecks = 2
	v1.report.Summary.PassedChecks = 1
	v1.report.Summary.FailedChecks = 1

	v2 := NewValidator(&ValidateCommand{})
	v2.addViolation("warning", "test", "test2", "Warning 1", "")
	v2.report.Summary.TotalChecks = 3
	v2.report.Summary.PassedChecks = 2
	v2.report.Summary.WarningCount = 1

	v1.mergeReport(v2.report)

	if len(v1.report.Violations) != 2 {
		t.Errorf("expected 2 violations after merge, got %d", len(v1.report.Violations))
	}
	if v1.report.Summary.TotalChecks != 5 {
		t.Errorf("expected 5 total checks after merge, got %d", v1.report.Summary.TotalChecks)
	}
	if v1.report.Summary.PassedChecks != 3 {
		t.Errorf("expected 3 passed checks after merge, got %d", v1.report.Summary.PassedChecks)
	}
	if v1.report.Summary.WarningCount != 1 {
		t.Errorf("expected 1 warning after merge, got %d", v1.report.Summary.WarningCount)
	}
}

// MockServer for testing server validation
type MockMCPServer struct {
	*httptest.Server
	capabilities modelcontextprotocol.ServerCapabilities
	failPing     bool
	slowPing     bool
}

func NewMockMCPServer(t *testing.T) *MockMCPServer {
	mock := &MockMCPServer{
		capabilities: modelcontextprotocol.ServerCapabilities{
			Tools: &struct{ ListChanged bool }{},
			Resources: &struct {
				Subscribe   bool
				ListChanged bool
			}{},
			Prompts: &struct{ ListChanged bool }{},
		},
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Logf("Failed to decode request: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		method, _ := req["method"].(string)
		id := req["id"]

		var response interface{}

		switch method {
		case "initialize":
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]interface{}{
					"protocolVersion": "2025-03-26",
					"serverInfo": map[string]interface{}{
						"name":    "mock-server",
						"version": "1.0.0",
					},
					"capabilities": mock.capabilities,
				},
			}
		case "ping":
			if mock.failPing {
				response = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"error": map[string]interface{}{
						"code":    -32603,
						"message": "Internal error",
					},
				}
			} else {
				if mock.slowPing {
					time.Sleep(200 * time.Millisecond)
				}
				response = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      id,
					"result":  map[string]interface{}{},
				}
			}
		case "tools/list":
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]interface{}{
					"tools": []interface{}{},
				},
			}
		case "tools/call":
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]interface{}{
					"code":    -32602,
					"message": "mcp: tool not found",
				},
			}
		default:
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "Method not found",
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	return mock
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"<tag>", "&lt;tag&gt;"},
		{"A & B", "A &amp; B"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"'apostrophe'", "&apos;apostrophe&apos;"},
		{"<>&\"'", "&lt;&gt;&amp;&quot;&apos;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeXML(tt.input)
			if got != tt.expected {
				t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
