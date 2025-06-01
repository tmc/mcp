package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
	"github.com/xeipuuv/gojsonschema"
)

// SpecMethod represents a method specification in the MCP spec.
type SpecMethod struct {
	Name    string                 `json:"name"`
	Params  map[string]interface{} `json:"params"`
	Result  map[string]interface{} `json:"result"`
	Error   map[string]interface{} `json:"error,omitempty"`
	Version string                 `json:"version,omitempty"`
}

// ServerSpec represents an MCP server specification.
type ServerSpec struct {
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description,omitempty"`
	Methods     []SpecMethod `json:"methods"`
}

// RecordingCommand represents a command in a recording.
type RecordingCommand struct {
	Direction   string // "->" for request, "<-" for response
	Message     *jsonrpc.Message
	LineNumber  int
	RawMessage  []byte
	MethodName  string
	ResponseFor string // ID of the request this response is for
}

// VerifyCommand represents the mcp-verify command.
type VerifyCommand struct {
	command.BaseCommand
	specFile         string
	recordingFile    string
	verbose          bool
	reportFormat     string
	outputFile       string
	strictValidation bool
	schemaValidation bool
	ignoreMethods    string
	requireMethods   string
	continueOnError  bool
}

// NewVerifyCommand creates a new VerifyCommand.
func NewVerifyCommand() *VerifyCommand {
	return &VerifyCommand{
		BaseCommand: command.BaseCommand{
			CommandName: "mcp-verify",
			Input:       os.Stdin,
			Output:      os.Stdout,
			Error:       os.Stderr,
		},
		reportFormat: "text",
	}
}

// Name returns the command name.
func (c *VerifyCommand) Name() string {
	return c.CommandName
}

// Usage returns the command usage.
func (c *VerifyCommand) Usage() string {
	return "Usage: mcp-verify [options]\n\n" +
		"Options:\n" +
		"  -s, --spec <file>          Specification file to verify against (required)\n" +
		"  -r, --recording <file>     Recording file or directory to verify (required)\n" +
		"  -v, --verbose              Verbose output\n" +
		"  -f, --format <format>      Report format (text, json)\n" +
		"  -o, --output <file>        Output file for verification report\n" +
		"  --strict                   Strict validation (no wildcards)\n" +
		"  --schema-validation        Use JSON Schema validation for method parameters and results\n" +
		"  --ignore-methods <list>    Comma-separated list of methods to ignore during verification\n" +
		"  --require-methods <list>   Comma-separated list of methods that must be present in recordings\n" +
		"  --continue-on-error        Continue verification on errors\n"
}

// VerificationError represents an error that occurred during verification.
type VerificationError struct {
	Method    string `json:"method,omitempty"`
	Message   string `json:"message"`
	LineNum   int    `json:"line_number,omitempty"`
	File      string `json:"file,omitempty"`
	ErrorType string `json:"error_type"`
	Details   string `json:"details,omitempty"`
}

// Report represents a verification report.
type Report struct {
	Success       bool                 `json:"success"`
	FileCount     int                  `json:"file_count,omitempty"`
	MethodCount   int                  `json:"method_count,omitempty"`
	Errors        []*VerificationError `json:"errors,omitempty"`
	FilesVerified []string             `json:"files_verified,omitempty"`
}

// Execute runs the command.
func (c *VerifyCommand) Execute(ctx context.Context, args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	fs.StringVar(&c.specFile, "s", "", "Specification file to verify against (required)")
	fs.StringVar(&c.specFile, "spec", "", "Specification file to verify against (required)")
	fs.StringVar(&c.recordingFile, "r", "", "Recording file or directory to verify (required)")
	fs.StringVar(&c.recordingFile, "recording", "", "Recording file or directory to verify (required)")
	fs.BoolVar(&c.verbose, "v", false, "Verbose output")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")
	fs.StringVar(&c.reportFormat, "f", "text", "Report format (text, json)")
	fs.StringVar(&c.reportFormat, "format", "text", "Report format (text, json)")
	fs.StringVar(&c.outputFile, "o", "", "Output file for verification report")
	fs.StringVar(&c.outputFile, "output", "", "Output file for verification report")
	fs.BoolVar(&c.strictValidation, "strict", false, "Strict validation (no wildcards)")
	fs.BoolVar(&c.schemaValidation, "schema-validation", false, "Use JSON Schema validation for method parameters and results")
	fs.StringVar(&c.ignoreMethods, "ignore-methods", "", "Comma-separated list of methods to ignore during verification")
	fs.StringVar(&c.requireMethods, "require-methods", "", "Comma-separated list of methods that must be present in recordings")
	fs.BoolVar(&c.continueOnError, "continue-on-error", false, "Continue verification on errors")

	// Display help if requested
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(c.Output, c.Usage())
		return nil
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate flags
	if c.specFile == "" {
		return fmt.Errorf("specification file is required")
	}
	if c.recordingFile == "" {
		return fmt.Errorf("recording file or directory is required")
	}

	// Parse spec file
	spec, err := c.parseSpecFile(c.specFile)
	if err != nil {
		return fmt.Errorf("failed to parse spec file: %w", err)
	}

	// Open output file if specified
	var output io.Writer
	if c.outputFile != "" {
		file, err := os.Create(c.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		output = file
	} else {
		output = c.Output
	}

	// Create a report to track verification results
	report := &Report{
		Success:       true,
		Errors:        make([]*VerificationError, 0),
		FilesVerified: make([]string, 0),
	}

	// Check if the recording is a directory
	fileInfo, err := os.Stat(c.recordingFile)
	if err != nil {
		return fmt.Errorf("failed to access recording file or directory: %w", err)
	}

	// Process recordings
	if fileInfo.IsDir() {
		// Get all recording files in the directory
		files, err := filepath.Glob(filepath.Join(c.recordingFile, "*.json"))
		if err != nil {
			return fmt.Errorf("failed to read recording directory: %w", err)
		}

		// Add .txt files as well
		txtFiles, err := filepath.Glob(filepath.Join(c.recordingFile, "*.txt"))
		if err != nil {
			return fmt.Errorf("failed to read recording directory: %w", err)
		}
		files = append(files, txtFiles...)

		report.FileCount = len(files)

		// Process each recording file
		var errors []error
		for _, file := range files {
			fmt.Fprintf(output, "Verifying recording file: %s\n", file)
			report.FilesVerified = append(report.FilesVerified, file)

			fileReport := &Report{
				Success: true,
				Errors:  make([]*VerificationError, 0),
			}

			if err := c.verifyRecording(spec, file, output, fileReport); err != nil {
				report.Success = false
				report.Errors = append(report.Errors, fileReport.Errors...)

				if c.continueOnError {
					errors = append(errors, fmt.Errorf("verification failed for %s: %w", file, err))
					fmt.Fprintf(c.Error, "Error in %s: %v\n", file, err)
				} else {
					// Generate the report before returning
					c.generateReport(report, output)
					return fmt.Errorf("verification failed for %s: %w", file, err)
				}
			} else {
				// Add method count from successful file verification
				report.MethodCount += fileReport.MethodCount
			}
		}

		// If there were errors and we continued, report them
		if len(errors) > 0 {
			report.Success = false
			// Generate the report
			c.generateReport(report, output)
			return fmt.Errorf("verification completed with %d errors", len(errors))
		}
	} else {
		// Process a single recording file
		report.FileCount = 1
		report.FilesVerified = append(report.FilesVerified, c.recordingFile)

		if err := c.verifyRecording(spec, c.recordingFile, output, report); err != nil {
			report.Success = false
			// Generate the report
			c.generateReport(report, output)
			return err
		}
	}

	// Generate the report
	c.generateReport(report, output)

	if report.Success {
		fmt.Fprintln(output, "Verification successful")
	}

	return nil
}

// generateReport outputs the verification report in the specified format.
func (c *VerifyCommand) generateReport(report *Report, output io.Writer) {
	if c.reportFormat == "json" {
		// Output JSON report
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(c.Error, "Failed to generate JSON report: %v\n", err)
			return
		}
		fmt.Fprintln(output, string(jsonData))
	} else {
		// Output text report
		if !report.Success {
			fmt.Fprintf(output, "\nVerification completed with %d errors\n", len(report.Errors))
			for i, err := range report.Errors {
				fmt.Fprintf(output, "Error %d: [%s] %s", i+1, err.ErrorType, err.Message)
				if err.Method != "" {
					fmt.Fprintf(output, " (Method: %s)", err.Method)
				}
				if err.File != "" {
					fmt.Fprintf(output, " (File: %s)", err.File)
				}
				if err.LineNum > 0 {
					fmt.Fprintf(output, " (Line: %d)", err.LineNum)
				}
				fmt.Fprintln(output)
				if err.Details != "" && c.verbose {
					fmt.Fprintf(output, "  Details: %s\n", err.Details)
				}
			}
		} else {
			fmt.Fprintf(output, "Verified %d files successfully\n", report.FileCount)
			if report.MethodCount > 0 {
				fmt.Fprintf(output, "Verified %d methods successfully\n", report.MethodCount)
			}
		}
	}
}

// parseSpecFile parses an MCP server specification file.
func (c *VerifyCommand) parseSpecFile(filePath string) (*ServerSpec, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	// Parse the JSON
	var spec ServerSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec file: %w", err)
	}

	return &spec, nil
}

// verifyRecording verifies a recording against a server specification.
func (c *VerifyCommand) verifyRecording(spec *ServerSpec, recordingFile string, output io.Writer, report *Report) error {
	// Parse the recording file
	commands, err := c.parseRecordingFile(recordingFile)
	if err != nil {
		verErr := &VerificationError{
			Message:   fmt.Sprintf("failed to parse recording file: %v", err),
			File:      recordingFile,
			ErrorType: "PARSE_ERROR",
		}
		report.Errors = append(report.Errors, verErr)
		return fmt.Errorf("failed to parse recording file: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(output, "Found %d commands in recording\n", len(commands))
	}

	// Collect methods that must be present
	requiredMethods := make(map[string]bool)
	if c.requireMethods != "" {
		for _, method := range strings.Split(c.requireMethods, ",") {
			requiredMethods[strings.TrimSpace(method)] = false
		}
	}

	// Collect methods to ignore
	ignoreMethods := make(map[string]bool)
	if c.ignoreMethods != "" {
		for _, method := range strings.Split(c.ignoreMethods, ",") {
			ignoreMethods[strings.TrimSpace(method)] = true
		}
	}

	// Create a map of methods by name for faster lookup
	methodMap := make(map[string]SpecMethod)
	for _, method := range spec.Methods {
		methodMap[method.Name] = method
	}

	// Group commands by request/response pairs
	pairs := c.groupCommandPairs(commands)

	// Track how many methods are verified
	methodCount := 0

	// Verify each pair
	var errors []error
	for _, pair := range pairs {
		req := pair[0]
		resp := pair[1]

		// Skip ignored methods
		if ignoreMethods[req.MethodName] {
			if c.verbose {
				fmt.Fprintf(output, "Skipping ignored method: %s\n", req.MethodName)
			}
			continue
		}

		// Mark method as seen for required methods
		if _, required := requiredMethods[req.MethodName]; required {
			requiredMethods[req.MethodName] = true
		}

		// Look up the method in the spec
		method, found := methodMap[req.MethodName]
		if !found {
			err := fmt.Errorf("method %s not found in specification", req.MethodName)
			verErr := &VerificationError{
				Message:   fmt.Sprintf("method %s not found in specification", req.MethodName),
				Method:    req.MethodName,
				File:      recordingFile,
				LineNum:   req.LineNumber,
				ErrorType: "METHOD_NOT_FOUND",
			}
			report.Errors = append(report.Errors, verErr)

			if c.continueOnError {
				errors = append(errors, err)
				fmt.Fprintf(c.Error, "Error: method %s not found in specification at line %d\n",
					req.MethodName, req.LineNumber)
				continue
			} else {
				return err
			}
		}

		methodCount++

		// Verify the method params and result
		if err := c.verifyMethod(req, resp, method, output, recordingFile, report); err != nil {
			if c.continueOnError {
				errors = append(errors, err)
			} else {
				return err
			}
		}
	}

	// Update method count in the report
	report.MethodCount = methodCount

	// Check if all required methods were seen
	var missingMethods []string
	for method, seen := range requiredMethods {
		if !seen {
			missingMethods = append(missingMethods, method)
		}
	}
	if len(missingMethods) > 0 {
		err := fmt.Errorf("required methods not found in recording: %s", strings.Join(missingMethods, ", "))
		verErr := &VerificationError{
			Message:   fmt.Sprintf("required methods not found in recording: %s", strings.Join(missingMethods, ", ")),
			File:      recordingFile,
			ErrorType: "MISSING_REQUIRED_METHODS",
			Details:   strings.Join(missingMethods, ", "),
		}
		report.Errors = append(report.Errors, verErr)

		if c.continueOnError {
			errors = append(errors, err)
			fmt.Fprintf(c.Error, "Error: required methods not found: %s\n",
				strings.Join(missingMethods, ", "))
		} else {
			return err
		}
	}

	// If there were errors, return an error
	if len(errors) > 0 {
		return fmt.Errorf("verification completed with %d errors", len(errors))
	}

	return nil
}

// parseRecordingFile parses a recording file.
func (c *VerifyCommand) parseRecordingFile(filePath string) ([]*RecordingCommand, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open recording file: %w", err)
	}
	defer file.Close()

	var commands []*RecordingCommand
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse command
		if strings.HasPrefix(line, "->") || strings.HasPrefix(line, "<-") {
			direction := line[:2]
			jsonStr := strings.TrimSpace(line[2:])

			var msg jsonrpc.Message
			if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
				return nil, fmt.Errorf("line %d: invalid JSON: %w", lineNumber, err)
			}

			// Store method name for requests
			methodName := ""
			responseFor := ""
			if direction == "->" {
				methodName = msg.Method
			} else if direction == "<-" && msg.ID != nil {
				// For responses, store the ID they're responding to
				responseFor = fmt.Sprintf("%v", msg.ID)
			}

			commands = append(commands, &RecordingCommand{
				Direction:   direction,
				Message:     &msg,
				LineNumber:  lineNumber,
				RawMessage:  []byte(jsonStr),
				MethodName:  methodName,
				ResponseFor: responseFor,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading recording file: %w", err)
	}

	return commands, nil
}

// groupCommandPairs groups commands into request/response pairs.
func (c *VerifyCommand) groupCommandPairs(commands []*RecordingCommand) [][2]*RecordingCommand {
	// Map to store requests by ID
	requestsById := make(map[string]*RecordingCommand)

	// Map to store responses by the ID they're responding to
	responsesById := make(map[string]*RecordingCommand)

	// First pass: collect all requests and responses
	for _, cmd := range commands {
		if cmd.Direction == "->" && cmd.Message.ID != nil {
			requestsById[fmt.Sprintf("%v", cmd.Message.ID)] = cmd
		} else if cmd.Direction == "<-" && cmd.Message.ID != nil {
			responsesById[fmt.Sprintf("%v", cmd.Message.ID)] = cmd
		}
	}

	// Second pass: match requests with responses
	var pairs [][2]*RecordingCommand
	for id, req := range requestsById {
		var resp *RecordingCommand
		if respCmd, found := responsesById[id]; found {
			resp = respCmd
		}
		pairs = append(pairs, [2]*RecordingCommand{req, resp})
	}

	return pairs
}

// verifyMethod verifies a method call against its specification.
func (c *VerifyCommand) verifyMethod(req, resp *RecordingCommand, method SpecMethod, output io.Writer, recordingFile string, report *Report) error {
	if c.verbose {
		fmt.Fprintf(output, "Verifying method: %s\n", method.Name)
	}

	// If using JSON Schema validation
	if c.schemaValidation {
		// Verify parameters
		if req.Message.Params != nil && len(method.Params) > 0 {
			if err := c.validateWithJSONSchema(req.Message.Params, method.Params, "params"); err != nil {
				errDetails := fmt.Sprintf("%v", err)
				verErr := &VerificationError{
					Message:   fmt.Sprintf("schema validation error for %s params", method.Name),
					Method:    method.Name,
					File:      recordingFile,
					LineNum:   req.LineNumber,
					ErrorType: "SCHEMA_VALIDATION_ERROR",
					Details:   errDetails,
				}
				report.Errors = append(report.Errors, verErr)
				fmt.Fprintf(c.Error, "Schema validation error for %s params: %v\n", method.Name, err)
				return fmt.Errorf("schema validation error for %s params", method.Name)
			}
		}

		// Verify result
		if resp != nil && resp.Message.Result != nil && len(method.Result) > 0 {
			if err := c.validateWithJSONSchema(resp.Message.Result, method.Result, "result"); err != nil {
				errDetails := fmt.Sprintf("%v", err)
				verErr := &VerificationError{
					Message:   fmt.Sprintf("schema validation error for %s result", method.Name),
					Method:    method.Name,
					File:      recordingFile,
					LineNum:   resp.LineNumber,
					ErrorType: "SCHEMA_VALIDATION_ERROR",
					Details:   errDetails,
				}
				report.Errors = append(report.Errors, verErr)
				fmt.Fprintf(c.Error, "Schema validation error for %s result: %v\n", method.Name, err)
				return fmt.Errorf("schema validation error for %s result", method.Name)
			}
		}

		// Verify error if present
		if resp != nil && resp.Message.Error != nil && len(method.Error) > 0 {
			errorBytes, err := json.Marshal(resp.Message.Error)
			if err != nil {
				verErr := &VerificationError{
					Message:   "failed to marshal error",
					Method:    method.Name,
					File:      recordingFile,
					LineNum:   resp.LineNumber,
					ErrorType: "MARSHAL_ERROR",
				}
				report.Errors = append(report.Errors, verErr)
				fmt.Fprintf(c.Error, "Failed to marshal error: %v\n", err)
				return fmt.Errorf("failed to marshal error")
			}

			if err := c.validateWithJSONSchema(errorBytes, method.Error, "error"); err != nil {
				errDetails := fmt.Sprintf("%v", err)
				verErr := &VerificationError{
					Message:   fmt.Sprintf("schema validation error for %s error", method.Name),
					Method:    method.Name,
					File:      recordingFile,
					LineNum:   resp.LineNumber,
					ErrorType: "SCHEMA_VALIDATION_ERROR",
					Details:   errDetails,
				}
				report.Errors = append(report.Errors, verErr)
				fmt.Fprintf(c.Error, "Schema validation error for %s error: %v\n", method.Name, err)
				return fmt.Errorf("schema validation error for %s error", method.Name)
			}
		}
	} else {
		// Verify using simple field validation

		// Check for required result fields
		if resp != nil && resp.Message.Result != nil {
			var resultObj map[string]interface{}
			if err := json.Unmarshal(resp.Message.Result, &resultObj); err != nil {
				verErr := &VerificationError{
					Message:   "failed to parse result",
					Method:    method.Name,
					File:      recordingFile,
					LineNum:   resp.LineNumber,
					ErrorType: "PARSE_ERROR",
				}
				report.Errors = append(report.Errors, verErr)
				fmt.Fprintf(c.Error, "Failed to parse result: %v\n", err)
				return fmt.Errorf("failed to parse result")
			}

			// Check required fields
			if requiredFields, ok := method.Result["required"].([]interface{}); ok {
				for _, field := range requiredFields {
					fieldName, ok := field.(string)
					if !ok {
						continue
					}
					if _, found := resultObj[fieldName]; !found {
						verErr := &VerificationError{
							Message:   fmt.Sprintf("validation error: required field '%s' missing in %s result", fieldName, method.Name),
							Method:    method.Name,
							File:      recordingFile,
							LineNum:   resp.LineNumber,
							ErrorType: "FIELD_VALIDATION_ERROR",
							Details:   fmt.Sprintf("Required field missing: %s", fieldName),
						}
						report.Errors = append(report.Errors, verErr)
						fmt.Fprintf(c.Error, "Validation error: required field '%s' missing in %s result at line %d\n",
							fieldName, method.Name, resp.LineNumber)
						return fmt.Errorf("validation error: required field missing in result")
					}
				}
			}
		}
	}

	if c.verbose {
		fmt.Fprintf(output, "Method %s validated successfully\n", method.Name)
	}
	return nil
}

// validateWithJSONSchema validates data against a JSON schema.
func (c *VerifyCommand) validateWithJSONSchema(data []byte, schema map[string]interface{}, context string) error {
	// Convert schema to a JSON document
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Load schema
	schemaLoader := gojsonschema.NewStringLoader(string(schemaBytes))

	// Load data - ensure it's always treated as a string
	dataLoader := gojsonschema.NewStringLoader(string(data))

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Check result
	if !result.Valid() {
		var errors []string
		for _, err := range result.Errors() {
			errors = append(errors, err.String())
		}
		return fmt.Errorf("schema validation errors:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func main() {
	if err := NewVerifyCommand().Execute(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
