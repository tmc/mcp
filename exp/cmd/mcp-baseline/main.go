package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"golang.org/x/exp/jsonrpc2"
)

var (
	// Common flags
	serverAddr   = flag.String("server", "localhost:8080", "MCP server address")
	baselineFile = flag.String("baseline", "", "Baseline file path")
	outputFile   = flag.String("output", "", "Output file path")
	methodsList  = flag.String("methods", "", "Comma-separated list of methods to test (or baseline files for merge)")
	testDataFile = flag.String("test-data", "", "Test data file path")
	ciMode       = flag.Bool("ci", false, "CI mode (exit with non-zero code on failure)")
	reportFormat = flag.String("report", "text", "Report format (text, json, html)")
	verbose      = flag.Bool("v", false, "Verbose output")
	showHelp     = flag.Bool("help", false, "Show help information")
)

// Metadata contains baseline metadata
type Metadata struct {
	ServerName      string    `json:"server_name"`
	ServerVersion   string    `json:"server_version"`
	RecordedAt      time.Time `json:"recorded_at"`
	ProtocolVersion string    `json:"protocol_version"`
}

// ValidationRule defines how to validate a response field
type ValidationRule struct {
	Path  string `json:"path"`
	Match string `json:"match"` // exact, exists, regex, numeric_equal, etc.
}

// RequestResponse is a test case with request and expected response
type RequestResponse struct {
	Name             string                 `json:"name"`
	Request          map[string]interface{} `json:"request"`
	ExpectedResponse map[string]interface{} `json:"expected_response"`
	ValidationRules  []ValidationRule       `json:"validation_rules,omitempty"`
	Setup            []map[string]interface{} `json:"setup,omitempty"`
	Cleanup          []map[string]interface{} `json:"cleanup,omitempty"`
}

// Baseline represents a complete test baseline
type Baseline struct {
	Metadata  Metadata          `json:"metadata"`
	TestCases []RequestResponse `json:"test_cases"`
}

// ValidationResult represents the result of a test case validation
type ValidationResult struct {
	Name      string   `json:"name"`
	Success   bool     `json:"success"`
	Errors    []string `json:"errors,omitempty"`
	Request   map[string]interface{} `json:"request,omitempty"`
	Expected  map[string]interface{} `json:"expected,omitempty"`
	Actual    map[string]interface{} `json:"actual,omitempty"`
}

// TestReport contains the complete test results
type TestReport struct {
	ServerAddress   string            `json:"server_address"`
	BaselineFile    string            `json:"baseline_file"`
	Timestamp       time.Time         `json:"timestamp"`
	OverallSuccess  bool              `json:"overall_success"`
	Results         []ValidationResult `json:"results"`
	SuccessCount    int               `json:"success_count"`
	FailureCount    int               `json:"failure_count"`
	ExecutionTimeMs int64             `json:"execution_time_ms"`
}

// printUsage prints the command usage information
func printUsage() {
	fmt.Println("Usage: mcp-baseline [command] [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  record   Record a baseline from a server")
	fmt.Println("  verify   Verify a server against a baseline")
	fmt.Println("  list     List methods in a baseline")
	fmt.Println("  extract  Extract a subset of methods from a baseline")
	fmt.Println("  merge    Merge multiple baselines")
	fmt.Println("  update   Update specific methods in a baseline")
	fmt.Println("\nCommon options:")
	fmt.Println("  --server      MCP server address (default: localhost:8080)")
	fmt.Println("  --baseline    Baseline file path")
	fmt.Println("  --output      Output file path")
	fmt.Println("  --methods     Comma-separated list of methods to test (or baseline files for merge)")
	fmt.Println("  --test-data   Test data file path")
	fmt.Println("  --ci          CI mode (exit with non-zero code on failure)")
	fmt.Println("  --report      Report format: text, json, html (default: text)")
	fmt.Println("  --v           Enable verbose output")
	fmt.Println("  --help        Show help information")
}

func main() {
	// Only parse arguments after checking for the command
	if len(os.Args) < 2 {
		fmt.Println("Missing command. Use 'record', 'verify', 'list', 'extract', 'merge', or 'update'")
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle --help flag before command
	if command == "--help" || command == "-h" {
		printUsage()
		os.Exit(0)
	}

	// Parse the remaining arguments after the command
	flag.CommandLine.Parse(os.Args[2:])

	// Handle --help flag after command
	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	switch command {
	case "record":
		if *outputFile == "" {
			fmt.Println("Error: --output flag required for record command")
			os.Exit(1)
		}
		recordBaseline()
	case "verify":
		if *baselineFile == "" {
			fmt.Println("Error: --baseline flag required for verify command")
			os.Exit(1)
		}
		verifyBaseline()
	case "list":
		if *baselineFile == "" {
			fmt.Println("Error: --baseline flag required for list command")
			os.Exit(1)
		}
		listBaseline()
	case "extract":
		if *baselineFile == "" || *outputFile == "" || *methodsList == "" {
			fmt.Println("Error: --baseline, --output, and --methods flags required for extract command")
			os.Exit(1)
		}
		extractBaseline()
	case "merge":
		if *outputFile == "" {
			fmt.Println("Error: --output flag required for merge command")
			os.Exit(1)
		}
		if *methodsList == "" {
			fmt.Println("Error: --methods flag required for merge command (comma-separated list of baseline files)")
			os.Exit(1)
		}
		mergeBaselines()
	case "update":
		if *baselineFile == "" || *methodsList == "" {
			fmt.Println("Error: --baseline and --methods flags required for update command")
			os.Exit(1)
		}
		updateBaseline()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

// recordBaseline records a new baseline from a server
func recordBaseline() {
	fmt.Printf("Recording baseline from server %s\n", *serverAddr)

	// Create MCP client
	client, err := NewMCPClient(*serverAddr)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Prepare initialize request using MCP types
	initParams := mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo: mcp.Implementation{
			Name:    "mcp-baseline",
			Version: "1.0.0",
		},
		Capabilities: mcp.ClientCapabilities{},
	}

	// Convert to map for consistency in the baseline
	initParamsMap := map[string]interface{}{
		"protocolVersion": initParams.ProtocolVersion,
		"clientInfo": map[string]interface{}{
			"name":    initParams.ClientInfo.Name,
			"version": initParams.ClientInfo.Version,
		},
		"capabilities": map[string]interface{}{},
	}

	// Create the raw request for storing in the baseline
	initRequest := map[string]interface{}{
		"jsonrpc": mcp.JSONRPC_VERSION,
		"id":      1,
		"method":  string(mcp.MethodInitialize),
		"params":  initParamsMap,
	}

	// Send the initialize request
	initResponse, err := client.SendRequest(string(mcp.MethodInitialize), initParams)
	if err != nil {
		fmt.Printf("Error initializing connection to server: %v\n", err)
		os.Exit(1)
	}

	// Extract server info
	serverName := "unknown"
	serverVersion := "unknown"
	protocolVersion := "unknown"

	if result, ok := initResponse["result"].(map[string]interface{}); ok {
		if serverInfo, ok := result["serverInfo"].(map[string]interface{}); ok {
			if name, ok := serverInfo["name"].(string); ok {
				serverName = name
			}
			if version, ok := serverInfo["version"].(string); ok {
				serverVersion = version
			}
		}
		if version, ok := result["protocolVersion"].(string); ok {
			protocolVersion = version
		}
	}

	// Create baseline metadata
	baseline := Baseline{
		Metadata: Metadata{
			ServerName:      serverName,
			ServerVersion:   serverVersion,
			RecordedAt:      time.Now(),
			ProtocolVersion: protocolVersion,
		},
		TestCases: []RequestResponse{
			{
				Name:             string(mcp.MethodInitialize),
				Request:          initRequest,
				ExpectedResponse: initResponse,
				ValidationRules: []ValidationRule{
					{Path: "result.protocolVersion", Match: "exact"},
					{Path: "result.serverInfo.name", Match: "exact"},
					{Path: "result.capabilities", Match: "exists"},
				},
			},
		},
	}
	
	// Get methods to test
	methods := []string{}
	if *methodsList != "" {
		methods = strings.Split(*methodsList, ",")
	} else {
		// Try to discover methods by checking capabilities
		// First, list the tools to discover available methods
		fmt.Println("Querying server for available tools...")

		toolsResponse, err := client.SendRequest(string(mcp.MethodToolsList), mcp.ListToolsRequest{})
		if err != nil {
			fmt.Printf("Warning: Could not list tools: %v\n", err)
			// Fall back to standard MCP methods
			defaultMethods := []string{
				string(mcp.MethodPing),
				string(mcp.MethodResourcesList),
				string(mcp.MethodResourcesTemplatesList),
				string(mcp.MethodResourcesRead),
				string(mcp.MethodPromptsList),
				string(mcp.MethodPromptsGet),
				string(mcp.MethodToolsList),
				string(mcp.MethodToolsCall),
			}
			methods = defaultMethods
		} else {
			// Extract tools from response
			if result, ok := toolsResponse["result"].(map[string]interface{}); ok {
				if toolsList, ok := result["tools"].([]interface{}); ok {
					for _, toolInterface := range toolsList {
						if tool, ok := toolInterface.(map[string]interface{}); ok {
							if name, ok := tool["name"].(string); ok {
								methods = append(methods, string(mcp.MethodToolsCall))
								fmt.Printf("Discovered tool: %s\n", name)
							}
						}
					}
				}
			}

			// Add standard methods
			standardMethods := []string{
				string(mcp.MethodPing),
				string(mcp.MethodResourcesList),
				string(mcp.MethodResourcesTemplatesList),
				string(mcp.MethodResourcesRead),
				string(mcp.MethodPromptsList),
				string(mcp.MethodPromptsGet),
			}
			for _, method := range standardMethods {
				if !contains(methods, method) {
					methods = append(methods, method)
				}
			}
		}
	}

	// Execute methods with test data
	testData := loadTestData()

	for _, method := range methods {
		if method == string(mcp.MethodInitialize) {
			continue // Already tested
		}

		fmt.Printf("Testing method: %s\n", method)

		// Get test parameters for this method
		var params interface{}
		if paramsMap, ok := testData[method].(map[string]interface{}); ok {
			params = paramsMap
		} else {
			// Use default parameters based on method type
			switch method {
			case string(mcp.MethodPing):
				params = map[string]interface{}{}
			case string(mcp.MethodResourcesList):
				params = mcp.ListResourcesRequest{}
			case string(mcp.MethodResourcesTemplatesList):
				params = mcp.ListResourceTemplatesRequest{}
			case string(mcp.MethodResourcesRead):
				params = mcp.ReadResourceRequest{URI: "example://resource"}
			case string(mcp.MethodPromptsList):
				params = mcp.ListPromptsRequest{}
			case string(mcp.MethodPromptsGet):
				params = mcp.GetPromptRequest{Name: "default"}
			case string(mcp.MethodToolsList):
				params = mcp.ListToolsRequest{}
			case string(mcp.MethodToolsCall):
				// Default to a simple echo tool call
				params = mcp.CallToolRequest{
					Name:      "echo",
					Arguments: json.RawMessage(`{"text": "Hello from mcp-baseline"}`),
				}
			default:
				// For unknown methods, use an empty parameter map
				params = map[string]interface{}{}
			}
		}

		// Create request object for the baseline
		request := map[string]interface{}{
			"jsonrpc": mcp.JSONRPC_VERSION,
			"id":      2,
			"method":  method,
			"params":  params,
		}

		// For simplicity, we're not handling setup/cleanup here
		// In a real implementation, you'd need to handle proper test state management

		response, err := client.SendRequest(method, params)
		if err != nil {
			fmt.Printf("Error testing method %s: %v\n", method, err)
			// Include error responses in the baseline too (they're valid responses)
			if response != nil {
				// Create test case with the error response
				testCase := RequestResponse{
					Name:             method,
					Request:          request,
					ExpectedResponse: response,
					ValidationRules: []ValidationRule{
						{Path: "error.code", Match: "exact"},
					},
				}
				baseline.TestCases = append(baseline.TestCases, testCase)
				fmt.Printf("Recorded error case for method: %s\n", method)
			}
			continue
		}

		// Create test case
		testCase := RequestResponse{
			Name:             method,
			Request:          request,
			ExpectedResponse: response,
		}

		// Add validation rules based on response structure and method
		if _, ok := response["result"].(map[string]interface{}); ok {
			rules := []ValidationRule{
				{Path: "result", Match: "exists"},
			}

			// Add method-specific validation rules
			switch method {
			case string(mcp.MethodPing):
				rules = append(rules, ValidationRule{Path: "result", Match: "exists"})
			case string(mcp.MethodResourcesList):
				rules = append(rules, ValidationRule{Path: "result.resources", Match: "exists"})
			case string(mcp.MethodResourcesTemplatesList):
				rules = append(rules, ValidationRule{Path: "result.templates", Match: "exists"})
			case string(mcp.MethodPromptsList):
				rules = append(rules, ValidationRule{Path: "result.prompts", Match: "exists"})
			case string(mcp.MethodToolsList):
				rules = append(rules, ValidationRule{Path: "result.tools", Match: "exists"})
			}

			testCase.ValidationRules = rules
		}

		baseline.TestCases = append(baseline.TestCases, testCase)
		fmt.Printf("Recorded test case for method: %s\n", method)
	}
	
	// Write baseline to file
	err = writeJSONFile(*outputFile, baseline)
	if err != nil {
		fmt.Printf("Error writing baseline file: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Baseline recorded successfully to %s\n", *outputFile)
}

// verifyBaseline verifies a server against a recorded baseline
func verifyBaseline() {
	fmt.Printf("Verifying server %s against baseline %s\n", *serverAddr, *baselineFile)
	
	// Load baseline
	baseline, err := loadBaseline(*baselineFile)
	if err != nil {
		fmt.Printf("Error loading baseline: %v\n", err)
		os.Exit(1)
	}
	
	// Create report
	report := TestReport{
		ServerAddress: *serverAddr,
		BaselineFile:  *baselineFile,
		Timestamp:     time.Now(),
		Results:       []ValidationResult{},
	}
	
	startTime := time.Now()
	
	// Execute test cases
	for _, testCase := range baseline.TestCases {
		fmt.Printf("Testing method: %s\n", testCase.Name)
		
		// Run setup steps
		for _, setup := range testCase.Setup {
			setupMethod, _ := setup["method"].(string)
			setupParams, _ := setup["params"].(map[string]interface{})
			
			setupRequest := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      99, // Use high ID for setup operations
				"method":  setupMethod,
				"params":  setupParams,
			}
			
			_, err := sendRequest(setupRequest)
			if err != nil {
				fmt.Printf("Error in setup for test case %s: %v\n", testCase.Name, err)
				// Continue anyway - the test may still work
			}
		}
		
		// Send test request
		actualResponse, err := sendRequest(testCase.Request)
		
		// Validate response
		result := validateResponse(testCase, actualResponse, err)
		report.Results = append(report.Results, result)
		
		if result.Success {
			report.SuccessCount++
			if *verbose {
				fmt.Printf("✓ %s: Success\n", testCase.Name)
			}
		} else {
			report.FailureCount++
			fmt.Printf("✗ %s: Failed\n", testCase.Name)
			for _, errMsg := range result.Errors {
				fmt.Printf("  - %s\n", errMsg)
			}
		}
		
		// Run cleanup steps
		for _, cleanup := range testCase.Cleanup {
			cleanupMethod, _ := cleanup["method"].(string)
			cleanupParams, _ := cleanup["params"].(map[string]interface{})
			
			cleanupRequest := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      999, // Use very high ID for cleanup operations
				"method":  cleanupMethod,
				"params":  cleanupParams,
			}
			
			_, err := sendRequest(cleanupRequest)
			if err != nil {
				fmt.Printf("Warning: Error in cleanup for test case %s: %v\n", testCase.Name, err)
				// Continue anyway - the test has already completed
			}
		}
	}
	
	// Finalize report
	report.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	report.OverallSuccess = report.FailureCount == 0
	
	// Output report
	generateReport(&report)
	
	// In CI mode, exit with error if verification failed
	if *ciMode && !report.OverallSuccess {
		os.Exit(1)
	}
}

// listBaseline lists the methods in a baseline
func listBaseline() {
	baseline, err := loadBaseline(*baselineFile)
	if err != nil {
		fmt.Printf("Error loading baseline: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Baseline %s contains %d test cases:\n", *baselineFile, len(baseline.TestCases))
	fmt.Printf("Server: %s %s\n", baseline.Metadata.ServerName, baseline.Metadata.ServerVersion)
	fmt.Printf("Recorded: %s\n", baseline.Metadata.RecordedAt.Format(time.RFC3339))
	fmt.Printf("Protocol: %s\n\n", baseline.Metadata.ProtocolVersion)
	
	fmt.Println("Test cases:")
	for i, testCase := range baseline.TestCases {
		fmt.Printf("%d. %s\n", i+1, testCase.Name)
		
		if *verbose {
			method, _ := testCase.Request["method"].(string)
			fmt.Printf("   Method: %s\n", method)
			
			// Print parameters in a readable format
			if params, ok := testCase.Request["params"].(map[string]interface{}); ok {
				fmt.Printf("   Parameters:\n")
				for k, v := range params {
					fmt.Printf("     %s: %v\n", k, v)
				}
			}
			
			// Print validation rules
			if len(testCase.ValidationRules) > 0 {
				fmt.Printf("   Validation Rules:\n")
				for _, rule := range testCase.ValidationRules {
					fmt.Printf("     %s: %s\n", rule.Path, rule.Match)
				}
			}
			
			fmt.Println()
		}
	}
}

// extractBaseline extracts a subset of methods into a new baseline
func extractBaseline() {
	baseline, err := loadBaseline(*baselineFile)
	if err != nil {
		fmt.Printf("Error loading baseline: %v\n", err)
		os.Exit(1)
	}
	
	methods := strings.Split(*methodsList, ",")
	methodMap := make(map[string]bool)
	for _, method := range methods {
		methodMap[method] = true
	}
	
	newBaseline := Baseline{
		Metadata: baseline.Metadata,
		TestCases: []RequestResponse{},
	}
	
	for _, testCase := range baseline.TestCases {
		method, _ := testCase.Request["method"].(string)
		if methodMap[testCase.Name] || methodMap[method] {
			newBaseline.TestCases = append(newBaseline.TestCases, testCase)
		}
	}
	
	if len(newBaseline.TestCases) == 0 {
		fmt.Println("Warning: No matching test cases found")
	}
	
	err = writeJSONFile(*outputFile, newBaseline)
	if err != nil {
		fmt.Printf("Error writing extracted baseline: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Extracted %d test cases to %s\n", len(newBaseline.TestCases), *outputFile)
}

// updateBaseline updates specific methods in a baseline
func updateBaseline() {
	baseline, err := loadBaseline(*baselineFile)
	if err != nil {
		fmt.Printf("Error loading baseline: %v\n", err)
		os.Exit(1)
	}
	
	methods := strings.Split(*methodsList, ",")
	methodMap := make(map[string]bool)
	for _, method := range methods {
		methodMap[method] = true
	}
	
	// Create a map of existing test cases by name for easy lookup
	testCaseMap := make(map[string]int)
	for i, testCase := range baseline.TestCases {
		testCaseMap[testCase.Name] = i
	}
	
	// Update each specified method
	for _, methodName := range methods {
		fmt.Printf("Updating method: %s\n", methodName)
		
		// Find existing test case
		index, exists := testCaseMap[methodName]
		if !exists {
			fmt.Printf("Method %s not found in baseline, skipping\n", methodName)
			continue
		}
		
		// Get existing test case
		testCase := baseline.TestCases[index]
		
		// Send request to server
		response, err := sendRequest(testCase.Request)
		if err != nil {
			fmt.Printf("Error testing method %s: %v\n", methodName, err)
			continue
		}
		
		// Update expected response
		testCase.ExpectedResponse = response
		baseline.TestCases[index] = testCase
		
		fmt.Printf("Updated test case for method: %s\n", methodName)
	}
	
	// Update metadata
	baseline.Metadata.RecordedAt = time.Now()
	
	// Write updated baseline
	output := *baselineFile
	if *outputFile != "" {
		output = *outputFile
	}
	
	err = writeJSONFile(output, baseline)
	if err != nil {
		fmt.Printf("Error writing updated baseline: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Baseline updated successfully to %s\n", output)
}

// validateResponse validates a response against expected values
func validateResponse(testCase RequestResponse, actualResponse map[string]interface{}, err error) ValidationResult {
	result := ValidationResult{
		Name:     testCase.Name,
		Success:  true,
		Errors:   []string{},
		Request:  testCase.Request,
		Expected: testCase.ExpectedResponse,
		Actual:   actualResponse,
	}
	
	// Check for errors in the request
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Request failed: %v", err))
		return result
	}
	
	// If JSON-RPC error is returned, check if it's expected
	if actualError, hasError := actualResponse["error"].(map[string]interface{}); hasError {
		expectedError, expectsError := testCase.ExpectedResponse["error"].(map[string]interface{})
		
		if !expectsError {
			result.Success = false
			errCode := actualError["code"]
			errMsg := actualError["message"]
			result.Errors = append(result.Errors, fmt.Sprintf("Unexpected error: %v - %v", errCode, errMsg))
			return result
		}
		
		// Compare error codes
		if actualError["code"] != expectedError["code"] {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Expected error code %v but got %v", 
				expectedError["code"], actualError["code"]))
		}
		
		return result
	}
	
	// Apply validation rules
	for _, rule := range testCase.ValidationRules {
		if !validateRule(rule, testCase.ExpectedResponse, actualResponse) {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Validation failed for rule: %s (%s)",
				rule.Path, rule.Match))
		}
	}
	
	// If no validation rules, fall back to exact comparison
	if len(testCase.ValidationRules) == 0 {
		if !reflect.DeepEqual(testCase.ExpectedResponse, actualResponse) {
			result.Success = false
			result.Errors = append(result.Errors, "Response does not match expected value")
		}
	}
	
	return result
}

// generateValidationRules creates validation rules for a specific MCP method response
func generateValidationRules(method string, response map[string]interface{}) []ValidationRule {
	// Start with basic validation rules
	rules := []ValidationRule{}

	// Check if we have a result or error
	if _, hasError := response["error"].(map[string]interface{}); hasError {
		// Validate error response
		rules = append(rules, ValidationRule{Path: "error.code", Match: "exact"})
		return rules
	}

	// Basic result validation
	rules = append(rules, ValidationRule{Path: "result", Match: "exists"})

	// Add method-specific validation
	if result, ok := response["result"].(map[string]interface{}); ok {
		switch method {
		case string(mcp.MethodInitialize):
			// For initialize, validate protocol version and server info
			rules = append(rules,
				ValidationRule{Path: "result.protocolVersion", Match: "exact"},
				ValidationRule{Path: "result.serverInfo", Match: "exists"},
				ValidationRule{Path: "result.serverInfo.name", Match: "exact"},
				ValidationRule{Path: "result.capabilities", Match: "exists"},
			)

		case string(mcp.MethodPing):
			// For ping, just validate that result exists (already done)

		case string(mcp.MethodResourcesList):
			// For resources/list, validate resources array exists
			if _, hasResources := result["resources"]; hasResources {
				rules = append(rules, ValidationRule{Path: "result.resources", Match: "exists"})
			}

		case string(mcp.MethodResourcesTemplatesList):
			// For resources/templates/list, validate templates array exists
			if _, hasTemplates := result["templates"]; hasTemplates {
				rules = append(rules, ValidationRule{Path: "result.templates", Match: "exists"})
			}

		case string(mcp.MethodResourcesRead):
			// For resources/read, validate contents exists
			if _, hasContents := result["contents"]; hasContents {
				rules = append(rules, ValidationRule{Path: "result.contents", Match: "exists"})
			}

		case string(mcp.MethodPromptsList):
			// For prompts/list, validate prompts array exists
			if _, hasPrompts := result["prompts"]; hasPrompts {
				rules = append(rules, ValidationRule{Path: "result.prompts", Match: "exists"})
			}

		case string(mcp.MethodPromptsGet):
			// For prompts/get, validate messages exists
			if _, hasMessages := result["messages"]; hasMessages {
				rules = append(rules, ValidationRule{Path: "result.messages", Match: "exists"})
			}

		case string(mcp.MethodToolsList):
			// For tools/list, validate tools array exists
			if _, hasTools := result["tools"]; hasTools {
				rules = append(rules, ValidationRule{Path: "result.tools", Match: "exists"})
			}

		case string(mcp.MethodToolsCall):
			// For tools/call, validate content exists
			if _, hasContent := result["content"]; hasContent {
				rules = append(rules, ValidationRule{Path: "result.content", Match: "exists"})
			}
		}
	}

	return rules
}

// validateRule validates a single rule against actual response
func validateRule(rule ValidationRule, expected, actual map[string]interface{}) bool {
	// Get expected value
	expectedValue := getValueByPath(expected, rule.Path)
	if expectedValue == nil {
		// If expected value doesn't exist, rule is invalid
		return false
	}

	// Get actual value
	actualValue := getValueByPath(actual, rule.Path)

	// Apply rule based on match type
	switch rule.Match {
	case "exists":
		return actualValue != nil
	case "exact":
		return reflect.DeepEqual(expectedValue, actualValue)
	case "regex":
		// A real implementation would use regex matching here
		// For now, we'll do a simple string contains check
		expectedStr, expectedOk := expectedValue.(string)
		actualStr, actualOk := actualValue.(string)
		if !expectedOk || !actualOk {
			return false
		}
		return strings.Contains(actualStr, expectedStr)
	case "numeric_equal":
		// Convert to float for numeric comparison
		expectedNum, expectedOk := toFloat64(expectedValue)
		actualNum, actualOk := toFloat64(actualValue)
		return expectedOk && actualOk && expectedNum == actualNum
	case "numeric_greater":
		expectedNum, expectedOk := toFloat64(expectedValue)
		actualNum, actualOk := toFloat64(actualValue)
		return expectedOk && actualOk && actualNum > expectedNum
	case "numeric_less":
		expectedNum, expectedOk := toFloat64(expectedValue)
		actualNum, actualOk := toFloat64(actualValue)
		return expectedOk && actualOk && actualNum < expectedNum
	case "type_check":
		// Check if the types match
		return reflect.TypeOf(expectedValue) == reflect.TypeOf(actualValue)
	default:
		// Unknown rule
		return false
	}
}

// toFloat64 attempts to convert various numeric types to float64
func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// getValueByPath gets a value from a nested map using a dot-separated path
func getValueByPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data
	
	for i, part := range parts {
		// Last part of the path
		if i == len(parts)-1 {
			return current[part]
		}
		
		// Get the next level
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return nil
		}
		
		current = next
	}
	
	return nil
}

// generateReport generates a test report in the specified format
func generateReport(report *TestReport) {
	switch *reportFormat {
	case "json":
		if *outputFile != "" {
			err := writeJSONFile(*outputFile, report)
			if err != nil {
				fmt.Printf("Error writing JSON report: %v\n", err)
			} else {
				fmt.Printf("JSON report written to %s\n", *outputFile)
			}
		} else {
			// Print to stdout
			jsonData, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(jsonData))
		}
	case "html":
		if *outputFile != "" {
			// Simple HTML template
			html := generateHTMLReport(report)
			err := os.WriteFile(*outputFile, []byte(html), 0644)
			if err != nil {
				fmt.Printf("Error writing HTML report: %v\n", err)
			} else {
				fmt.Printf("HTML report written to %s\n", *outputFile)
			}
		} else {
			fmt.Println("HTML report requires --output flag")
		}
	default: // text
		fmt.Printf("\n===== MCP Baseline Verification Report =====\n")
		fmt.Printf("Server: %s\n", report.ServerAddress)
		fmt.Printf("Baseline: %s\n", report.BaselineFile)
		fmt.Printf("Time: %s\n", report.Timestamp.Format(time.RFC3339))
		fmt.Printf("Duration: %dms\n\n", report.ExecutionTimeMs)
		
		fmt.Printf("Results: %d tests, %d passed, %d failed\n\n", 
			len(report.Results), report.SuccessCount, report.FailureCount)
		
		if report.FailureCount > 0 {
			fmt.Println("Failed tests:")
			for _, result := range report.Results {
				if !result.Success {
					fmt.Printf("- %s\n", result.Name)
					for _, err := range result.Errors {
						fmt.Printf("  * %s\n", err)
					}
				}
			}
			fmt.Println()
		}
		
		if report.OverallSuccess {
			fmt.Println("✓ Overall Result: SUCCESS")
		} else {
			fmt.Println("✗ Overall Result: FAILURE")
		}
		fmt.Println("=============================================")
	}
}

// generateHTMLReport generates a simple HTML report
func generateHTMLReport(report *TestReport) string {
	// This is a very basic HTML template - in a real implementation
	// you'd use a proper HTML template engine
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>MCP Baseline Verification Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 10px; border-radius: 5px; }
        .success { color: green; }
        .failure { color: red; }
        table { border-collapse: collapse; width: 100%%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
    </style>
</head>
<body>
    <div class="header">
        <h1>MCP Baseline Verification Report</h1>
        <p>Server: %s</p>
        <p>Baseline: %s</p>
        <p>Time: %s</p>
        <p>Duration: %dms</p>
    </div>
    
    <h2>Summary</h2>
    <p>%d tests, %d passed, %d failed</p>
    <p class="%s">Overall Result: %s</p>
    
    <h2>Test Results</h2>
    <table>
        <tr>
            <th>Test</th>
            <th>Result</th>
            <th>Details</th>
        </tr>`,
		report.ServerAddress,
		report.BaselineFile,
		report.Timestamp.Format(time.RFC3339),
		report.ExecutionTimeMs,
		len(report.Results), report.SuccessCount, report.FailureCount,
		func() string {
			if report.OverallSuccess {
				return "success"
			}
			return "failure"
		}(),
		func() string {
			if report.OverallSuccess {
				return "SUCCESS"
			}
			return "FAILURE"
		}())
	
	// Add test results
	for _, result := range report.Results {
		status := "✓"
		class := "success"
		details := "Test passed"
		
		if !result.Success {
			status = "✗"
			class = "failure"
			details = strings.Join(result.Errors, "<br>")
		}
		
		html += fmt.Sprintf(`
        <tr>
            <td>%s</td>
            <td class="%s">%s</td>
            <td>%s</td>
        </tr>`, result.Name, class, status, details)
	}
	
	// Close HTML
	html += `
    </table>
</body>
</html>`
	
	return html
}

// MCPClient represents a client for communicating with an MCP server
type MCPClient struct {
	conn    *jsonrpc2.Connection
	httpURL string
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewMCPClient creates a new client for communicating with an MCP server
func NewMCPClient(serverAddr string) (*MCPClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create HTTP URL
	httpURL := fmt.Sprintf("http://%s/mcp", serverAddr)

	// Return the client
	return &MCPClient{
		httpURL: httpURL,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Close closes the client connection
func (c *MCPClient) Close() error {
	if c.conn != nil {
		c.cancel()
		return c.conn.Close()
	}
	return nil
}

// SendRequest sends a JSON-RPC request to the server
func (c *MCPClient) SendRequest(method string, params interface{}) (map[string]interface{}, error) {
	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create JSON-RPC request
	request := struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1, // Using a fixed ID for simplicity, in a real client this should be generated
		Method:  method,
		Params:  params,
	}

	// Serialize request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error serializing request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.httpURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Check for JSON-RPC error
	if errObj, hasError := response["error"].(map[string]interface{}); hasError {
		code, _ := errObj["code"].(float64)
		message, _ := errObj["message"].(string)
		return response, fmt.Errorf("JSON-RPC error: code=%v message=%s", code, message)
	}

	return response, nil
}

// Legacy function for backward compatibility
func sendRequest(request map[string]interface{}) (map[string]interface{}, error) {
	// Create client
	client, err := NewMCPClient(*serverAddr)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}
	defer client.Close()

	// Extract method and params
	method, _ := request["method"].(string)
	params, _ := request["params"].(map[string]interface{})

	// Send request
	return client.SendRequest(method, params)
}

// loadBaseline loads a baseline from a file
func loadBaseline(filePath string) (Baseline, error) {
	var baseline Baseline
	
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return baseline, fmt.Errorf("error reading file: %w", err)
	}
	
	// Parse JSON
	err = json.Unmarshal(data, &baseline)
	if err != nil {
		return baseline, fmt.Errorf("error parsing JSON: %w", err)
	}
	
	return baseline, nil
}

// loadTestData loads test data from a file or uses defaults
func loadTestData() map[string]interface{} {
	// Default test data
	defaultData := map[string]interface{}{
		"read": map[string]interface{}{
			"path": "test.txt",
		},
		"write": map[string]interface{}{
			"path":    "test.txt",
			"content": "Test content from mcp-baseline",
		},
		"append": map[string]interface{}{
			"path":    "test.txt",
			"content": "\nAppended by mcp-baseline",
		},
		"delete": map[string]interface{}{
			"path": "test.txt",
		},
		"mkdir": map[string]interface{}{
			"path": "test-dir",
		},
		"list": map[string]interface{}{
			"path": ".",
		},
		"move": map[string]interface{}{
			"source":      "test.txt",
			"destination": "test-moved.txt",
		},
		"copy": map[string]interface{}{
			"source":      "test.txt",
			"destination": "test-copy.txt",
		},
		"stat": map[string]interface{}{
			"path": "test.txt",
		},
		"find": map[string]interface{}{
			"path":    ".",
			"pattern": "*.txt",
		},
	}
	
	// If test data file is specified, load it
	if *testDataFile != "" {
		data, err := os.ReadFile(*testDataFile)
		if err != nil {
			fmt.Printf("Warning: Error reading test data file: %v\n", err)
			fmt.Println("Using default test data")
			return defaultData
		}
		
		var testData map[string]interface{}
		err = json.Unmarshal(data, &testData)
		if err != nil {
			fmt.Printf("Warning: Error parsing test data file: %v\n", err)
			fmt.Println("Using default test data")
			return defaultData
		}
		
		return testData
	}
	
	return defaultData
}

// mergeBaselines merges multiple baseline files into a single baseline
func mergeBaselines() {
	fmt.Println("Merging baseline files...")

	// Interpret methodsList as a list of baseline files to merge
	baselineFiles := strings.Split(*methodsList, ",")
	if len(baselineFiles) < 2 {
		fmt.Println("Error: At least two baseline files must be specified for merge")
		os.Exit(1)
	}

	// Create a new merged baseline with empty test cases
	mergedBaseline := Baseline{
		Metadata: Metadata{
			ServerName:      "merged-baseline",
			ServerVersion:   "1.0.0",
			RecordedAt:      time.Now(),
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		},
		TestCases: []RequestResponse{},
	}

	// Maps to track unique test cases and avoid duplicates
	methodMap := make(map[string]bool)

	// Process each baseline file
	for _, baselineFile := range baselineFiles {
		fmt.Printf("Processing baseline file: %s\n", baselineFile)

		baseline, err := loadBaseline(baselineFile)
		if err != nil {
			fmt.Printf("Warning: Error loading baseline %s: %v\n", baselineFile, err)
			continue
		}

		// Update metadata if the first file
		if len(methodMap) == 0 {
			mergedBaseline.Metadata.ServerName = baseline.Metadata.ServerName
			mergedBaseline.Metadata.ServerVersion = baseline.Metadata.ServerVersion
			mergedBaseline.Metadata.ProtocolVersion = baseline.Metadata.ProtocolVersion
		}

		// Add test cases that don't already exist
		for _, testCase := range baseline.TestCases {
			method, _ := testCase.Request["method"].(string)
			if !methodMap[method] {
				methodMap[method] = true
				mergedBaseline.TestCases = append(mergedBaseline.TestCases, testCase)
				fmt.Printf("Added test case for method: %s\n", method)
			}
		}
	}

	// Write merged baseline to file
	err := writeJSONFile(*outputFile, mergedBaseline)
	if err != nil {
		fmt.Printf("Error writing merged baseline: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully merged %d baseline files with %d unique methods to %s\n",
		len(baselineFiles), len(mergedBaseline.TestCases), *outputFile)
}

// writeJSONFile writes a JSON-serializable value to a file
func writeJSONFile(filePath string, data interface{}) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("error creating directories: %w", err)
	}

	// Serialize data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing JSON: %w", err)
	}

	// Write file
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}