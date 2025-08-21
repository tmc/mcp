package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

const (
	programName = "mcp-validate"
	version     = "0.1.0"
)

// Command represents a CLI command
type Command interface {
	Name() string
	Usage() string
	Execute(ctx context.Context, args []string) error
}

// ValidateCommand handles protocol validation
type ValidateCommand struct {
	serverCmd  string
	traceFile  string
	schemaDir  string
	strict     bool
	batch      string
	outputFmt  string
	reportFile string
	live       bool
	target     string
	verbose    bool
}

func (c *ValidateCommand) Name() string {
	return "validate"
}

func (c *ValidateCommand) Usage() string {
	return "Validate MCP protocol compliance"
}

func (c *ValidateCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)

	fs.StringVar(&c.serverCmd, "server", "", "Server command to validate")
	fs.StringVar(&c.traceFile, "trace", "", "Trace file to validate")
	fs.StringVar(&c.schemaDir, "schema-dir", "", "Directory containing JSON schemas")
	fs.BoolVar(&c.strict, "strict", false, "Enable strict validation mode")
	fs.StringVar(&c.batch, "batch", "", "Batch validate servers from file")
	fs.StringVar(&c.outputFmt, "output-format", "json", "Output format (json, junit-xml, html)")
	fs.StringVar(&c.reportFile, "report", "", "Output report file")
	fs.BoolVar(&c.live, "live", false, "Live validation mode")
	fs.StringVar(&c.target, "target", "", "Target server for live validation")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	validator := NewValidator(c)
	return validator.Run(ctx)
}

// Validator performs MCP protocol validation
type Validator struct {
	config *ValidateCommand
	report *ValidationReport
}

// ValidationReport holds validation results
type ValidationReport struct {
	Version      string            `json:"version"`
	Timestamp    time.Time         `json:"timestamp"`
	Summary      ValidationSummary `json:"summary"`
	Violations   []Violation       `json:"violations"`
	Capabilities CapabilityReport  `json:"capabilities"`
}

// ValidationSummary provides high-level statistics
type ValidationSummary struct {
	TotalChecks    int     `json:"totalChecks"`
	PassedChecks   int     `json:"passedChecks"`
	FailedChecks   int     `json:"failedChecks"`
	WarningCount   int     `json:"warningCount"`
	ComplianceRate float64 `json:"complianceRate"`
	Status         string  `json:"status"`
}

// Violation represents a protocol compliance violation
type Violation struct {
	Severity   string          `json:"severity"`
	Category   string          `json:"category"`
	Rule       string          `json:"rule"`
	Message    string          `json:"message"`
	Location   string          `json:"location"`
	Details    json.RawMessage `json:"details,omitempty"`
	Suggestion string          `json:"suggestion,omitempty"`
}

// CapabilityReport shows capability compliance
type CapabilityReport struct {
	DeclaredCapabilities map[string]interface{} `json:"declared"`
	ActualCapabilities   map[string]interface{} `json:"actual"`
	Mismatches           []CapabilityMismatch   `json:"mismatches"`
}

// CapabilityMismatch represents capability declaration vs implementation mismatch
type CapabilityMismatch struct {
	Capability  string `json:"capability"`
	Declared    bool   `json:"declared"`
	Implemented bool   `json:"implemented"`
	Issue       string `json:"issue"`
}

// NewValidator creates a new validator instance
func NewValidator(config *ValidateCommand) *Validator {
	return &Validator{
		config: config,
		report: &ValidationReport{
			Version:    version,
			Timestamp:  time.Now(),
			Violations: []Violation{},
		},
	}
}

// Run executes the validation
func (v *Validator) Run(ctx context.Context) error {
	if v.config.verbose {
		log.Printf("Starting MCP validation...")
	}

	// Determine validation mode
	switch {
	case v.config.serverCmd != "":
		return v.validateServer(ctx, v.config.serverCmd)
	case v.config.traceFile != "":
		return v.validateTrace(ctx, v.config.traceFile)
	case v.config.batch != "":
		return v.validateBatch(ctx, v.config.batch)
	case v.config.live && v.config.target != "":
		return v.validateLive(ctx, v.config.target)
	default:
		return fmt.Errorf("no validation target specified")
	}
}

// validateServer validates a server by launching it
func (v *Validator) validateServer(ctx context.Context, serverCmd string) error {
	if v.config.verbose {
		log.Printf("Validating server: %s", serverCmd)
	}

	// Create a test client
	transport := mcp.NewCommandTransport(serverCmd)
	client := mcp.NewClient(transport)

	// Connect and initialize
	if err := client.Connect(ctx); err != nil {
		v.addViolation("error", "connection", "server_connect",
			fmt.Sprintf("Failed to connect to server: %v", err), "")
		return v.generateReport()
	}
	defer client.Close()

	// Initialize connection
	initReq := modelcontextprotocol.InitializeRequest{
		ProtocolVersion: modelcontextprotocol.LATEST_PROTOCOL_VERSION,
		ClientInfo: modelcontextprotocol.Implementation{
			Name:    programName,
			Version: version,
		},
		Capabilities: modelcontextprotocol.ClientCapabilities{},
	}

	initResult, err := client.Initialize(ctx, initReq)
	if err != nil {
		v.addViolation("error", "protocol", "initialization",
			fmt.Sprintf("Initialization failed: %v", err), "")
		return v.generateReport()
	}

	// Validate protocol version
	v.validateProtocolVersion(initResult.ProtocolVersion)

	// Validate server info
	v.validateServerInfo(initResult.ServerInfo)

	// Validate capabilities
	v.validateCapabilities(ctx, client, initResult.Capabilities)

	// Run protocol compliance tests
	v.runProtocolTests(ctx, client)

	return v.generateReport()
}

// validateProtocolVersion checks protocol version compliance
func (v *Validator) validateProtocolVersion(version string) {
	if version != modelcontextprotocol.LATEST_PROTOCOL_VERSION {
		v.addViolation("warning", "protocol", "version_mismatch",
			fmt.Sprintf("Server uses protocol version %s, expected %s",
				version, modelcontextprotocol.LATEST_PROTOCOL_VERSION),
			"",
		)
	}
}

// validateServerInfo validates server implementation info
func (v *Validator) validateServerInfo(info modelcontextprotocol.Implementation) {
	if strings.TrimSpace(info.Name) == "" {
		v.addViolation("error", "protocol", "server_info",
			"Server name is required but empty", "",
		)
	}

	if strings.TrimSpace(info.Version) == "" {
		v.addViolation("error", "protocol", "server_info",
			"Server version is required but empty", "",
		)
	}
}

// validateCapabilities validates declared vs actual capabilities
func (v *Validator) validateCapabilities(ctx context.Context, client *mcp.Client, caps modelcontextprotocol.ServerCapabilities) {
	v.report.Capabilities.DeclaredCapabilities = structToMap(caps)
	v.report.Capabilities.ActualCapabilities = make(map[string]interface{})

	// Test tools capability
	if caps.Tools != nil {
		if _, err := client.ListTools(ctx); err != nil {
			v.report.Capabilities.Mismatches = append(v.report.Capabilities.Mismatches,
				CapabilityMismatch{
					Capability:  "tools",
					Declared:    true,
					Implemented: false,
					Issue:       fmt.Sprintf("Tools capability declared but ListTools failed: %v", err),
				},
			)
		} else {
			v.report.Capabilities.ActualCapabilities["tools"] = true
		}
	}

	// Test resources capability
	if caps.Resources != nil {
		if _, err := client.ListResources(ctx); err != nil {
			v.report.Capabilities.Mismatches = append(v.report.Capabilities.Mismatches,
				CapabilityMismatch{
					Capability:  "resources",
					Declared:    true,
					Implemented: false,
					Issue:       fmt.Sprintf("Resources capability declared but ListResources failed: %v", err),
				},
			)
		} else {
			v.report.Capabilities.ActualCapabilities["resources"] = true
		}
	}

	// Test prompts capability
	if caps.Prompts != nil {
		if _, err := client.ListPrompts(ctx); err != nil {
			v.report.Capabilities.Mismatches = append(v.report.Capabilities.Mismatches,
				CapabilityMismatch{
					Capability:  "prompts",
					Declared:    true,
					Implemented: false,
					Issue:       fmt.Sprintf("Prompts capability declared but ListPrompts failed: %v", err),
				},
			)
		} else {
			v.report.Capabilities.ActualCapabilities["prompts"] = true
		}
	}
}

// runProtocolTests runs comprehensive protocol compliance tests
func (v *Validator) runProtocolTests(ctx context.Context, client *mcp.Client) {
	// Test required endpoints
	v.testPing(ctx, client)

	// Test error handling
	v.testErrorHandling(ctx, client)

	// Test parameter validation
	v.testParameterValidation(ctx, client)

	if v.config.strict {
		// Run strict mode tests
		v.runStrictTests(ctx, client)
	}
}

// testPing validates ping functionality
func (v *Validator) testPing(ctx context.Context, client *mcp.Client) {
	if err := client.Ping(ctx); err != nil {
		v.addViolation("error", "protocol", "ping",
			fmt.Sprintf("Ping failed: %v", err),
			"",
		)
	}
}

// testErrorHandling validates error response formats
func (v *Validator) testErrorHandling(ctx context.Context, client *mcp.Client) {
	// Test with invalid tool name
	_, err := client.CallTool(ctx, "nonexistent-tool-12345", nil)
	if err == nil {
		v.addViolation("error", "error_handling", "missing_error",
			"Server should return error for non-existent tool",
			"",
		)
	} else {
		// Validate error format
		if !isValidMCPError(err) {
			v.addViolation("error", "error_handling", "invalid_error_format",
				fmt.Sprintf("Invalid error format: %v", err),
				"",
			)
		}
	}
}

// testParameterValidation tests parameter validation
func (v *Validator) testParameterValidation(ctx context.Context, client *mcp.Client) {
	// This would test various parameter validation scenarios
	// Implementation depends on available tools
}

// runStrictTests runs additional strict mode tests
func (v *Validator) runStrictTests(ctx context.Context, client *mcp.Client) {
	// Test response time requirements
	start := time.Now()
	if err := client.Ping(ctx); err == nil {
		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			v.addViolation("warning", "performance", "slow_ping",
				fmt.Sprintf("Ping took %v, should be under 100ms", elapsed),
				"",
			)
		}
	}
}

// validateTrace validates a trace file
func (v *Validator) validateTrace(ctx context.Context, traceFile string) error {
	file, err := os.Open(traceFile)
	if err != nil {
		return fmt.Errorf("failed to open trace file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	lineNum := 0

	for {
		var msg json.RawMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			v.addViolation("error", "trace", "invalid_json",
				fmt.Sprintf("Invalid JSON at line %d: %v", lineNum, err),
				fmt.Sprintf("line %d", lineNum),
			)
			continue
		}
		lineNum++

		// Validate message structure
		v.validateMessage(msg, lineNum)
	}

	return v.generateReport()
}

// validateMessage validates a single protocol message
func (v *Validator) validateMessage(msg json.RawMessage, lineNum int) {
	var baseMsg struct {
		Jsonrpc string      `json:"jsonrpc"`
		Method  string      `json:"method,omitempty"`
		ID      interface{} `json:"id,omitempty"`
		Result  interface{} `json:"result,omitempty"`
		Error   interface{} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(msg, &baseMsg); err != nil {
		v.addViolation("error", "message", "invalid_structure",
			fmt.Sprintf("Invalid message structure: %v", err),
			fmt.Sprintf("line %d", lineNum),
		)
		return
	}

	// Validate JSON-RPC version
	if baseMsg.Jsonrpc != "2.0" {
		v.addViolation("error", "protocol", "jsonrpc_version",
			fmt.Sprintf("Invalid JSON-RPC version: %s", baseMsg.Jsonrpc),
			fmt.Sprintf("line %d", lineNum),
		)
	}

	// Validate message type
	if baseMsg.Method != "" {
		// Request or notification
		if baseMsg.Result != nil || baseMsg.Error != nil {
			v.addViolation("error", "message", "invalid_request",
				"Request cannot have result or error fields",
				fmt.Sprintf("line %d", lineNum),
			)
		}

		// Validate method name
		if !isValidMethod(baseMsg.Method) {
			v.addViolation("warning", "protocol", "unknown_method",
				fmt.Sprintf("Unknown method: %s", baseMsg.Method),
				fmt.Sprintf("line %d", lineNum),
			)
		}
	} else if baseMsg.ID != nil {
		// Response
		if baseMsg.Result == nil && baseMsg.Error == nil {
			v.addViolation("error", "message", "invalid_response",
				"Response must have either result or error",
				fmt.Sprintf("line %d", lineNum),
			)
		}
		if baseMsg.Result != nil && baseMsg.Error != nil {
			v.addViolation("error", "message", "invalid_response",
				"Response cannot have both result and error",
				fmt.Sprintf("line %d", lineNum),
			)
		}
	} else {
		// Invalid message
		v.addViolation("error", "message", "invalid_type",
			"Message must be request, notification, or response",
			fmt.Sprintf("line %d", lineNum),
		)
	}
}

// validateBatch validates multiple servers from a file
func (v *Validator) validateBatch(ctx context.Context, batchFile string) error {
	file, err := os.Open(batchFile)
	if err != nil {
		return fmt.Errorf("failed to open batch file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var servers []string
	if err := decoder.Decode(&servers); err != nil {
		return fmt.Errorf("failed to parse batch file: %w", err)
	}

	for _, server := range servers {
		if v.config.verbose {
			log.Printf("Validating server: %s", server)
		}

		// Create a new validator for each server
		subValidator := NewValidator(v.config)
		subValidator.config.serverCmd = server

		if err := subValidator.validateServer(ctx, server); err != nil {
			log.Printf("Failed to validate %s: %v", server, err)
		}

		// Merge results
		v.mergeReport(subValidator.report)
	}

	return v.generateReport()
}

// validateLive performs live validation of a running server
func (v *Validator) validateLive(ctx context.Context, target string) error {
	// This would connect to a running server and perform continuous validation
	// Implementation would be similar to validateServer but with continuous monitoring
	return fmt.Errorf("live validation not yet implemented")
}

// addViolation adds a violation to the report
func (v *Validator) addViolation(severity, category, rule, message, location string) {
	violation := Violation{
		Severity: severity,
		Category: category,
		Rule:     rule,
		Message:  message,
		Location: location,
	}

	// Add suggestions based on common issues
	violation.Suggestion = getSuggestion(category, rule)

	v.report.Violations = append(v.report.Violations, violation)

	// Update summary
	v.report.Summary.TotalChecks++
	if severity == "error" {
		v.report.Summary.FailedChecks++
	} else if severity == "warning" {
		v.report.Summary.WarningCount++
	} else {
		v.report.Summary.PassedChecks++
	}
}

// generateReport generates the final validation report
func (v *Validator) generateReport() error {
	// Calculate compliance rate
	if v.report.Summary.TotalChecks > 0 {
		v.report.Summary.ComplianceRate = float64(v.report.Summary.PassedChecks) /
			float64(v.report.Summary.TotalChecks) * 100
	}

	// Set status
	if v.report.Summary.FailedChecks > 0 {
		v.report.Summary.Status = "failed"
	} else if v.report.Summary.WarningCount > 0 {
		v.report.Summary.Status = "passed_with_warnings"
	} else {
		v.report.Summary.Status = "passed"
	}

	// Output report
	var output io.Writer = os.Stdout
	if v.config.reportFile != "" {
		file, err := os.Create(v.config.reportFile)
		if err != nil {
			return fmt.Errorf("failed to create report file: %w", err)
		}
		defer file.Close()
		output = file
	}

	switch v.config.outputFmt {
	case "json":
		return v.outputJSON(output)
	case "junit-xml":
		return v.outputJUnitXML(output)
	case "html":
		return v.outputHTML(output)
	default:
		return fmt.Errorf("unsupported output format: %s", v.config.outputFmt)
	}
}

// outputJSON outputs report in JSON format
func (v *Validator) outputJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v.report)
}

// outputJUnitXML outputs report in JUnit XML format
func (v *Validator) outputJUnitXML(w io.Writer) error {
	// Convert to JUnit XML format
	fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(w, `<testsuite name="mcp-validate" tests="%d" failures="%d" errors="%d" time="0">`,
		v.report.Summary.TotalChecks,
		v.report.Summary.FailedChecks,
		0,
	)

	for _, violation := range v.report.Violations {
		testName := fmt.Sprintf("%s.%s", violation.Category, violation.Rule)
		if violation.Severity == "error" {
			fmt.Fprintf(w, `<testcase name="%s" classname="%s">`, testName, violation.Category)
			fmt.Fprintf(w, `<failure message="%s">%s</failure>`,
				escapeXML(violation.Message),
				escapeXML(violation.Suggestion))
			fmt.Fprintln(w, `</testcase>`)
		} else {
			fmt.Fprintf(w, `<testcase name="%s" classname="%s"/>`, testName, violation.Category)
		}
	}

	fmt.Fprintln(w, `</testsuite>`)
	return nil
}

// outputHTML outputs report in HTML format
func (v *Validator) outputHTML(w io.Writer) error {
	// Generate HTML report
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>MCP Validation Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .summary { background: #f0f0f0; padding: 15px; border-radius: 5px; }
        .passed { color: green; }
        .failed { color: red; }
        .warning { color: orange; }
        table { border-collapse: collapse; width: 100%%; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        tr:nth-child(even) { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>MCP Validation Report</h1>
    <div class="summary">
        <h2>Summary</h2>
        <p>Generated: %s</p>
        <p>Total Checks: %d</p>
        <p class="passed">Passed: %d</p>
        <p class="failed">Failed: %d</p>
        <p class="warning">Warnings: %d</p>
        <p>Compliance Rate: %.1f%%</p>
        <p>Status: <span class="%s">%s</span></p>
    </div>`,
		v.report.Timestamp.Format(time.RFC3339),
		v.report.Summary.TotalChecks,
		v.report.Summary.PassedChecks,
		v.report.Summary.FailedChecks,
		v.report.Summary.WarningCount,
		v.report.Summary.ComplianceRate,
		v.report.Summary.Status,
		strings.ToUpper(v.report.Summary.Status),
	)

	if len(v.report.Violations) > 0 {
		html += `
    <h2>Violations</h2>
    <table>
        <tr>
            <th>Severity</th>
            <th>Category</th>
            <th>Rule</th>
            <th>Message</th>
            <th>Location</th>
            <th>Suggestion</th>
        </tr>`

		for _, violation := range v.report.Violations {
			html += fmt.Sprintf(`
        <tr>
            <td class="%s">%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
        </tr>`,
				violation.Severity,
				violation.Severity,
				violation.Category,
				violation.Rule,
				violation.Message,
				violation.Location,
				violation.Suggestion,
			)
		}
		html += `
    </table>`
	}

	html += `
</body>
</html>`

	_, err := fmt.Fprint(w, html)
	return err
}

// mergeReport merges another report into this one
func (v *Validator) mergeReport(other *ValidationReport) {
	v.report.Violations = append(v.report.Violations, other.Violations...)
	v.report.Summary.TotalChecks += other.Summary.TotalChecks
	v.report.Summary.PassedChecks += other.Summary.PassedChecks
	v.report.Summary.FailedChecks += other.Summary.FailedChecks
	v.report.Summary.WarningCount += other.Summary.WarningCount
}

// Helper functions

func isValidMethod(method string) bool {
	validMethods := []string{
		"initialize",
		"ping",
		"resources/list",
		"resources/templates/list",
		"resources/read",
		"prompts/list",
		"prompts/get",
		"tools/list",
		"tools/call",
		"notifications/cancelled",
		"notifications/progress",
		"notifications/message",
		"notifications/resources/list_changed",
		"notifications/prompts/list_changed",
		"notifications/tools/list_changed",
	}

	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}
	return false
}

func isValidMCPError(err error) bool {
	// Check if error follows MCP error format
	// This is a simplified check
	return err != nil && strings.Contains(err.Error(), "mcp:")
}

func getSuggestion(category, rule string) string {
	suggestions := map[string]map[string]string{
		"protocol": {
			"version_mismatch": "Update server to use the latest protocol version",
			"server_info":      "Ensure server provides complete implementation info",
			"jsonrpc_version":  "Use JSON-RPC 2.0 for all messages",
			"unknown_method":   "Check method name spelling and protocol documentation",
		},
		"connection": {
			"server_connect": "Check server is running and accessible",
		},
		"error_handling": {
			"missing_error":        "Server should return appropriate errors for invalid requests",
			"invalid_error_format": "Use standard MCP error format with code and message",
		},
		"message": {
			"invalid_structure": "Ensure message follows JSON-RPC 2.0 structure",
			"invalid_request":   "Requests should not have result or error fields",
			"invalid_response":  "Responses must have either result or error, not both",
			"invalid_type":      "Message must be a valid request, notification, or response",
		},
		"performance": {
			"slow_ping": "Optimize server response time for basic operations",
		},
	}

	if catSuggestions, ok := suggestions[category]; ok {
		if suggestion, ok := catSuggestions[rule]; ok {
			return suggestion
		}
	}
	return ""
}

func structToMap(v interface{}) map[string]interface{} {
	// Convert struct to map for reporting
	b, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n", programName)
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  validate    Validate MCP protocol compliance\n")
		fmt.Fprintf(os.Stderr, "  version     Show version information\n")
		os.Exit(1)
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "validate":
		cmd := &ValidateCommand{}
		if err := cmd.Execute(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("%s version %s\n", programName, version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
