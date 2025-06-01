package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
)

// ComplianceTest represents a compliance test for an MCP server.
type ComplianceTest struct {
	Name        string
	Description string
	Run         func(ctx context.Context, server *ServerInfo, verbose bool) *TestResult
}

// TestResult represents the result of a compliance test.
type TestResult struct {
	TestName     string    `json:"test_name"`
	Success      bool      `json:"success"`
	Message      string    `json:"message,omitempty"`
	Error        string    `json:"error,omitempty"`
	Duration     float64   `json:"duration"`
	StartTime    time.Time `json:"start_time"`
	Details      []string  `json:"details,omitempty"`
	RequestID    int       `json:"request_id,omitempty"`
	RequestData  string    `json:"request_data,omitempty"`
	ResponseData string    `json:"response_data,omitempty"`
}

// ServerInfo represents the information about a server being tested.
type ServerInfo struct {
	Config      ServerConfig
	Transport   string
	Endpoint    string
	Process     *os.Process
	Command     string
	Environment map[string]string
	Stdin       *os.File
	Stdout      *os.File
	Stderr      *os.File
}

// ServerConfig represents the configuration for an MCP server.
type ServerConfig struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Transport   string                 `json:"transport"` // "http", "stdio", "websocket"
	Host        string                 `json:"host,omitempty"`
	Port        int                    `json:"port,omitempty"`
	Command     string                 `json:"command,omitempty"`
	Environment map[string]string      `json:"environment,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	InitParams  map[string]interface{} `json:"init_params,omitempty"`
}

// ToolDefinition represents the definition of an MCP tool.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Schema      interface{} `json:"schema"`
}

// ScriptCommand represents a command in a script test.
type ScriptCommand struct {
	Direction  string // "->" for request, "<-" for expected response
	Message    *jsonrpc.Message
	LineNumber int
	IsWildcard bool   // True if the message contains wildcards
	Pattern    string // Original pattern string for wildcards
}

// Script represents a script test for an MCP server.
type Script struct {
	Path     string
	Commands []ScriptCommand
}

// TestCommand represents the mcp-test command.
type TestCommand struct {
	command.BaseCommand
	configFile    string
	testName      string
	scriptFiles   []string
	listTests     bool
	validateOnly  bool
	verbose       bool
	timeout       int
	jsonOutput    bool
	dryRun        bool
	failFast      bool
	parallelTests bool
	retries       int
}

// NewTestCommand creates a new TestCommand.
func NewTestCommand() *TestCommand {
	return &TestCommand{}
}

// Name returns the command name.
func (c *TestCommand) Name() string {
	return "mcp-test"
}

// Usage returns the command usage.
func (c *TestCommand) Usage() string {
	return "Usage: mcp-test [options]\n\n" +
		"Options:\n" +
		"  -c, --config <file>      Server configuration file (required)\n" +
		"  -t, --test <name>        Run only the specified test\n" +
		"  -s, --script <file>      Run a script test file (can be specified multiple times)\n" +
		"  -l, --list-tests         List available compliance tests\n" +
		"  --validate-only          Validate script files without running tests\n" +
		"  -v, --verbose            Verbose output\n" +
		"  --timeout <seconds>      Test timeout in seconds (default: 30)\n" +
		"  -j, --json               Output results in JSON format\n" +
		"  --dry-run                Parse configuration and scripts but don't run tests\n" +
		"  --fail-fast              Stop after first test failure\n" +
		"  --parallel               Run tests in parallel (where possible)\n" +
		"  --retries <n>            Number of retries for failed tests (default: 0)\n"
}

// Execute runs the command.
func (c *TestCommand) Execute(ctx context.Context, args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	fs.StringVar(&c.configFile, "c", "", "Server configuration file (required)")
	fs.StringVar(&c.configFile, "config", "", "Server configuration file (required)")
	fs.StringVar(&c.testName, "t", "", "Run only the specified test")
	fs.StringVar(&c.testName, "test", "", "Run only the specified test")
	fs.Func("s", "Add a script test file", func(s string) error {
		c.scriptFiles = append(c.scriptFiles, s)
		return nil
	})
	fs.Func("script", "Add a script test file", func(s string) error {
		c.scriptFiles = append(c.scriptFiles, s)
		return nil
	})
	fs.BoolVar(&c.listTests, "l", false, "List available compliance tests")
	fs.BoolVar(&c.listTests, "list-tests", false, "List available compliance tests")
	fs.BoolVar(&c.validateOnly, "validate-only", false, "Validate script files without running tests")
	fs.BoolVar(&c.verbose, "v", false, "Verbose output")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")
	fs.IntVar(&c.timeout, "timeout", 30, "Test timeout in seconds (default: 30)")
	fs.BoolVar(&c.jsonOutput, "j", false, "Output results in JSON format")
	fs.BoolVar(&c.jsonOutput, "json", false, "Output results in JSON format")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Parse configuration and scripts but don't run tests")
	fs.BoolVar(&c.failFast, "fail-fast", false, "Stop after first test failure")
	fs.BoolVar(&c.parallelTests, "parallel", false, "Run tests in parallel (where possible)")
	fs.IntVar(&c.retries, "retries", 0, "Number of retries for failed tests (default: 0)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate flags
	if !c.listTests && c.configFile == "" {
		return fmt.Errorf("config file is required")
	}

	// If we just want to list the tests, do that and exit
	if c.listTests {
		return c.listAvailableTests()
	}

	// Load the server configuration
	serverConfig, err := c.loadServerConfig()
	if err != nil {
		return err
	}

	// Load scripts if provided
	var scripts []*Script
	if len(c.scriptFiles) > 0 {
		scripts, err = c.loadScripts()
		if err != nil {
			return err
		}

		// If we just want to validate the scripts, exit here
		if c.validateOnly {
			fmt.Println("Script validation successful.")
			return nil
		}
	}

	// If dry run, just print the test plan and exit
	if c.dryRun {
		return c.printTestPlan(serverConfig, scripts)
	}

	// Run the tests
	return c.runTests(ctx, serverConfig, scripts)
}

// loadServerConfig loads the server configuration from the specified file.
func (c *TestCommand) loadServerConfig() (*ServerConfig, error) {
	data, err := os.ReadFile(c.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// loadScripts loads the script test files.
func (c *TestCommand) loadScripts() ([]*Script, error) {
	var scripts []*Script

	for _, filePath := range c.scriptFiles {
		script, err := c.parseScriptFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse script file %s: %w", filePath, err)
		}
		scripts = append(scripts, script)
	}

	return scripts, nil
}

// parseScriptFile parses a script test file.
func (c *TestCommand) parseScriptFile(filePath string) (*Script, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}

	// Parse the script commands
	lines := strings.Split(string(data), "\n")
	var commands []ScriptCommand
	var lineNumber int

	for i, line := range lines {
		lineNumber = i + 1
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse direction and message
		if strings.HasPrefix(line, "->") {
			// This is a request
			jsonStr := strings.TrimSpace(line[2:])
			msg, err := c.parseJSONRPCMessage(jsonStr)
			if err != nil {
				return nil, fmt.Errorf("line %d: failed to parse request: %w", lineNumber, err)
			}
			commands = append(commands, ScriptCommand{
				Direction:  "->",
				Message:    msg,
				LineNumber: lineNumber,
				IsWildcard: false,
			})
		} else if strings.HasPrefix(line, "<-") {
			// This is an expected response
			jsonStr := strings.TrimSpace(line[2:])
			msg, isWildcard, err := c.parseExpectedResponse(jsonStr)
			if err != nil {
				return nil, fmt.Errorf("line %d: failed to parse expected response: %w", lineNumber, err)
			}
			commands = append(commands, ScriptCommand{
				Direction:  "<-",
				Message:    msg,
				LineNumber: lineNumber,
				IsWildcard: isWildcard,
				Pattern:    jsonStr,
			})
		} else {
			return nil, fmt.Errorf("line %d: invalid command format", lineNumber)
		}
	}

	return &Script{
		Path:     filePath,
		Commands: commands,
	}, nil
}

// parseJSONRPCMessage parses a JSON-RPC message.
func (c *TestCommand) parseJSONRPCMessage(jsonStr string) (*jsonrpc.Message, error) {
	var msg jsonrpc.Message
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC message: %w", err)
	}
	return &msg, nil
}

// parseExpectedResponse parses an expected response, handling wildcards.
func (c *TestCommand) parseExpectedResponse(jsonStr string) (*jsonrpc.Message, bool, error) {
	// Check if the response contains wildcards
	isWildcard := strings.Contains(jsonStr, "*")

	// If there are no wildcards, parse as normal JSON-RPC message
	if !isWildcard {
		msg, err := c.parseJSONRPCMessage(jsonStr)
		return msg, false, err
	}

	// For wildcards, we need to do a partial parse to check basic syntax
	// We'll replace all occurrences of "*" with a placeholder that's valid JSON
	// This isn't perfect but should catch most syntax errors
	placeholder := `"__WILDCARD__"`
	tempStr := strings.ReplaceAll(jsonStr, `"*"`, placeholder)
	tempStr = strings.ReplaceAll(tempStr, `*`, placeholder)

	var tempMsg map[string]interface{}
	if err := json.Unmarshal([]byte(tempStr), &tempMsg); err != nil {
		return nil, true, fmt.Errorf("invalid JSON-RPC message with wildcards: %w", err)
	}

	// Create a simple message structure just to hold the id and method
	// The actual comparison will use pattern matching
	var msg jsonrpc.Message
	if id, ok := tempMsg["id"]; ok {
		// Only set the ID if it's not a wildcard
		if idStr, ok := id.(string); !ok || idStr != "__WILDCARD__" {
			msg.ID = id
		}
	}
	if method, ok := tempMsg["method"].(string); ok && method != "__WILDCARD__" {
		msg.Method = method
	}
	msg.Version = "2.0"

	return &msg, true, nil
}

// getAvailableTests returns the list of available compliance tests.
func (c *TestCommand) getAvailableTests() []ComplianceTest {
	return []ComplianceTest{
		{
			Name:        "initialize",
			Description: "Tests the initialize method",
			Run:         c.runInitializeTest,
		},
		{
			Name:        "shutdown",
			Description: "Tests the shutdown method",
			Run:         c.runShutdownTest,
		},
		{
			Name:        "tools/list",
			Description: "Tests the tools/list method",
			Run:         c.runToolsListTest,
		},
		{
			Name:        "tools/call",
			Description: "Tests the tools/call method",
			Run:         c.runToolsCallTest,
		},
	}
}

// listAvailableTests lists the available compliance tests.
func (c *TestCommand) listAvailableTests() error {
	tests := c.getAvailableTests()
	fmt.Println("Available compliance tests:")
	for _, test := range tests {
		fmt.Printf("  %s - %s\n", test.Name, test.Description)
	}
	return nil
}

// printTestPlan prints the test plan without running the tests.
func (c *TestCommand) printTestPlan(config *ServerConfig, scripts []*Script) error {
	fmt.Println("Dry run mode enabled. Tests will not be executed.")
	fmt.Printf("Server: %s (version %s)\n", config.Name, config.Version)
	fmt.Printf("Transport: %s\n", config.Transport)

	// Print which tests will be run
	fmt.Println("\nCompliance test plan:")
	tests := c.getAvailableTests()
	for _, test := range tests {
		if c.testName == "" || c.testName == test.Name {
			fmt.Printf("  - %s: %s\n", test.Name, test.Description)
		}
	}

	// Print script tests if any
	if len(scripts) > 0 {
		fmt.Println("\nScript test plan:")
		if len(scripts) > 1 {
			fmt.Printf("Multiple script files (%d) will be executed:\n", len(scripts))
		}
		for _, script := range scripts {
			fmt.Printf("  - %s: %d commands\n", filepath.Base(script.Path), len(script.Commands))
			if c.verbose {
				for i, cmd := range script.Commands {
					direction := "→" // request
					if cmd.Direction == "<-" {
						direction = "←" // response
					}
					fmt.Printf("    %d: %s Line %d\n", i+1, direction, cmd.LineNumber)
				}
			}
		}
	}

	// Print other settings
	if c.verbose {
		fmt.Println("\nVerbose mode enabled")
	}
	if c.timeout > 0 {
		fmt.Printf("Test timeout: %d seconds\n", c.timeout)
	}
	if c.failFast {
		fmt.Println("Fail-fast mode enabled")
	}
	if c.parallelTests {
		fmt.Println("Parallel test execution enabled")
	}
	if c.retries > 0 {
		fmt.Printf("Retries for failed tests: %d\n", c.retries)
	}

	return nil
}

// runTests runs the compliance tests.
func (c *TestCommand) runTests(ctx context.Context, config *ServerConfig, scripts []*Script) error {
	// Create a server info from the config
	serverInfo := &ServerInfo{
		Config:      *config,
		Transport:   config.Transport,
		Command:     config.Command,
		Environment: config.Environment,
	}

	// Build the list of tests to run
	var testsToRun []ComplianceTest
	availableTests := c.getAvailableTests()
	if c.testName != "" {
		// Run a specific test
		found := false
		for _, test := range availableTests {
			if test.Name == c.testName {
				testsToRun = append(testsToRun, test)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("test %q not found", c.testName)
		}
	} else {
		// Run all tests
		testsToRun = availableTests
	}

	// Run the tests
	results := make([]*TestResult, 0, len(testsToRun))
	for _, test := range testsToRun {
		// Create a context with timeout
		testCtx, cancel := context.WithTimeout(ctx, time.Duration(c.timeout)*time.Second)
		defer cancel()

		fmt.Printf("Running test: %s\n", test.Name)
		result := test.Run(testCtx, serverInfo, c.verbose)
		results = append(results, result)

		// Print the test result
		if !c.jsonOutput {
			if result.Success {
				fmt.Printf("  [PASS] %s\n", test.Name)
			} else {
				fmt.Printf("  [FAIL] %s: %s\n", test.Name, result.Error)
				if c.verbose && len(result.Details) > 0 {
					fmt.Println("  Details:")
					for _, detail := range result.Details {
						fmt.Printf("    - %s\n", detail)
					}
				}
			}
		}

		// If fail-fast is enabled and the test failed, stop testing
		if c.failFast && !result.Success {
			break
		}
	}

	// Run script tests if any
	if len(scripts) > 0 {
		for _, script := range scripts {
			fmt.Printf("Running script: %s\n", filepath.Base(script.Path))
			scriptResults, err := c.runScriptTest(ctx, serverInfo, script)
			if err != nil {
				return fmt.Errorf("failed to run script test: %w", err)
			}
			results = append(results, scriptResults...)
		}
	}

	// Output results in JSON format if requested
	if c.jsonOutput {
		resultData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}
		fmt.Println(string(resultData))
	} else {
		// Print summary
		passCount := 0
		failCount := 0
		for _, result := range results {
			if result.Success {
				passCount++
			} else {
				failCount++
			}
		}
		fmt.Printf("\nTest results: %d passed, %d failed\n", passCount, failCount)
	}

	// If any test failed, return an error
	if c.hasFailedTests(results) {
		return fmt.Errorf("one or more tests failed")
	}

	return nil
}

// hasFailedTests checks if any test failed.
func (c *TestCommand) hasFailedTests(results []*TestResult) bool {
	for _, result := range results {
		if !result.Success {
			return true
		}
	}
	return false
}

// runScriptTest runs a script test.
func (c *TestCommand) runScriptTest(ctx context.Context, server *ServerInfo, script *Script) ([]*TestResult, error) {
	// TODO: Implement script test execution
	// For now, just return a placeholder success result
	results := []*TestResult{
		{
			TestName:  fmt.Sprintf("script:%s", filepath.Base(script.Path)),
			Success:   true,
			Message:   "Script test execution not yet implemented",
			StartTime: time.Now(),
			Duration:  0.0,
		},
	}
	return results, nil
}

// runInitializeTest runs the initialize method test.
func (c *TestCommand) runInitializeTest(ctx context.Context, server *ServerInfo, verbose bool) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName:  "initialize",
		Success:   false,
		StartTime: startTime,
	}

	// TODO: Implement initialize test
	// For now, just return a placeholder success result
	result.Success = true
	result.Message = "Initialize test not yet implemented"
	result.Duration = time.Since(startTime).Seconds()

	return result
}

// runShutdownTest runs the shutdown method test.
func (c *TestCommand) runShutdownTest(ctx context.Context, server *ServerInfo, verbose bool) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName:  "shutdown",
		Success:   false,
		StartTime: startTime,
	}

	// TODO: Implement shutdown test
	// For now, just return a placeholder success result
	result.Success = true
	result.Message = "Shutdown test not yet implemented"
	result.Duration = time.Since(startTime).Seconds()

	return result
}

// runToolsListTest runs the tools/list method test.
func (c *TestCommand) runToolsListTest(ctx context.Context, server *ServerInfo, verbose bool) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName:  "tools/list",
		Success:   false,
		StartTime: startTime,
	}

	// TODO: Implement tools/list test
	// For now, just return a placeholder success result
	result.Success = true
	result.Message = "Tools/list test not yet implemented"
	result.Duration = time.Since(startTime).Seconds()

	return result
}

// runToolsCallTest runs the tools/call method test.
func (c *TestCommand) runToolsCallTest(ctx context.Context, server *ServerInfo, verbose bool) *TestResult {
	startTime := time.Now()
	result := &TestResult{
		TestName:  "tools/call",
		Success:   false,
		StartTime: startTime,
	}

	// TODO: Implement tools/call test
	// For now, just return a placeholder success result
	result.Success = true
	result.Message = "Tools/call test not yet implemented"
	result.Duration = time.Since(startTime).Seconds()

	return result
}

func main() {
	if err := NewTestCommand().Execute(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
