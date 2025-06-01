package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
)

// ScriptCommand represents a command in a script session.
type ScriptCommand struct {
	Direction   string // "->" for request, "<-" for expected response
	Message     *jsonrpc.Message
	LineNumber  int
	IsWildcard  bool
	RawMessage  []byte
	ExpectedMsg string
}

// SessionMetadata represents metadata for a session recording.
type SessionMetadata struct {
	Name        string
	Description string
	Date        time.Time
	Source      string
}

// ReplayCommand represents the mcp-replay command.
type ReplayCommand struct {
	command.BaseCommand
	scriptFile     string
	endpoint       string
	outputFile     string
	method         string
	responseFile   string
	delay          float64
	dryRun         bool
	validate       bool
	strict         bool
	verbose        bool
	range_         string
	interactive    bool
	outputFormat   string
	ignoreErrors   bool
	retries        int
	retryDelay     float64
	followupScript string
}

// NewReplayCommand creates a new ReplayCommand.
func NewReplayCommand() *ReplayCommand {
	return &ReplayCommand{}
}

// Name returns the command name.
func (c *ReplayCommand) Name() string {
	return "mcp-replay"
}

// Usage returns the command usage.
func (c *ReplayCommand) Usage() string {
	return "Usage: mcp-replay [options]\n\n" +
		"Options:\n" +
		"  -s, --script <file>      Script file to replay (required)\n" +
		"  -e, --endpoint <url>     Server endpoint to send requests to\n" +
		"  -o, --output <file>      Output file for server responses\n" +
		"  -m, --method <method>    Filter by method\n" +
		"  -r, --response <file>    File containing example responses (for validation)\n" +
		"  -d, --delay <seconds>    Delay between requests (default: 0)\n" +
		"  --dry-run                Parse script but don't send requests\n" +
		"  --validate               Validate responses against expected responses\n" +
		"  --strict                 Strict validation (no wildcards)\n" +
		"  -v, --verbose            Verbose output\n" +
		"  --range <start-end>      Play only commands in the specified range\n" +
		"  -i, --interactive        Interactive mode (prompt before each request)\n" +
		"  --format <format>        Output format (json, text, raw)\n" +
		"  --ignore-errors          Continue on errors\n" +
		"  --retries <n>            Number of retries for failed requests\n" +
		"  --retry-delay <seconds>  Delay between retries\n" +
		"  --followup <file>        Script to run after this one completes\n"
}

// Execute runs the command.
func (c *ReplayCommand) Execute(ctx context.Context, args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	fs.StringVar(&c.scriptFile, "s", "", "Script file to replay (required)")
	fs.StringVar(&c.scriptFile, "script", "", "Script file to replay (required)")
	fs.StringVar(&c.endpoint, "e", "", "Server endpoint to send requests to")
	fs.StringVar(&c.endpoint, "endpoint", "", "Server endpoint to send requests to")
	fs.StringVar(&c.outputFile, "o", "", "Output file for server responses")
	fs.StringVar(&c.outputFile, "output", "", "Output file for server responses")
	fs.StringVar(&c.method, "m", "", "Filter by method")
	fs.StringVar(&c.method, "method", "", "Filter by method")
	fs.StringVar(&c.responseFile, "r", "", "File containing example responses (for validation)")
	fs.StringVar(&c.responseFile, "response", "", "File containing example responses (for validation)")
	fs.StringVar(&c.responseFile, "response-file", "", "File containing example responses (for validation)")
	fs.Float64Var(&c.delay, "d", 0, "Delay between requests (default: 0)")
	fs.Float64Var(&c.delay, "delay", 0, "Delay between requests (default: 0)")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Parse script but don't send requests")
	fs.BoolVar(&c.validate, "validate", false, "Validate responses against expected responses")
	fs.BoolVar(&c.strict, "strict", false, "Strict validation (no wildcards)")
	fs.BoolVar(&c.verbose, "v", false, "Verbose output")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")
	fs.StringVar(&c.range_, "range", "", "Play only commands in the specified range")
	fs.BoolVar(&c.interactive, "i", false, "Interactive mode (prompt before each request)")
	fs.BoolVar(&c.interactive, "interactive", false, "Interactive mode (prompt before each request)")
	fs.StringVar(&c.outputFormat, "format", "text", "Output format (json, text, raw)")
	fs.BoolVar(&c.ignoreErrors, "ignore-errors", false, "Continue on errors")
	fs.IntVar(&c.retries, "retries", 0, "Number of retries for failed requests")
	fs.Float64Var(&c.retryDelay, "retry-delay", 1.0, "Delay between retries")
	fs.StringVar(&c.followupScript, "followup", "", "Script to run after this one completes")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate flags
	if c.scriptFile == "" {
		return fmt.Errorf("script file is required")
	}

	// For validation, we need a response file
	if c.validate && c.responseFile == "" && !c.dryRun {
		return fmt.Errorf("response file is required for validation")
	}

	// Parse the script
	commands, metadata, err := c.parseScript(c.scriptFile)
	if err != nil {
		return err
	}

	// Apply the method filter
	if c.method != "" {
		var filtered []*ScriptCommand
		for _, cmd := range commands {
			if cmd.Direction == "->" && cmd.Message.Method == c.method {
				filtered = append(filtered, cmd)
			} else if cmd.Direction == "<-" && len(filtered) > 0 {
				// Include the expected response for the last filtered request
				filtered = append(filtered, cmd)
			}
		}
		commands = filtered
	}

	// Apply the range filter
	if c.range_ != "" {
		rangeRe := regexp.MustCompile(`^(\d+)-(\d+)$`)
		matches := rangeRe.FindStringSubmatch(c.range_)
		if len(matches) != 3 {
			return fmt.Errorf("invalid range format, expected <start>-<end>")
		}

		start, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("invalid start index: %w", err)
		}

		end, err := strconv.Atoi(matches[2])
		if err != nil {
			return fmt.Errorf("invalid end index: %w", err)
		}

		if start < 1 || end > len(commands) || start > end {
			return fmt.Errorf("range out of bounds, valid range is 1-%d", len(commands))
		}

		// Convert from 1-based to 0-based indexing
		commands = commands[start-1 : end]
	}

	// If this is a dry run, just print the script information
	if c.dryRun {
		return c.printScriptInfo(commands, metadata)
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
		output = os.Stdout
	}

	// Load response data for validation if specified
	var responseData []byte
	if c.validate && c.responseFile != "" {
		responseData, err = os.ReadFile(c.responseFile)
		if err != nil {
			return fmt.Errorf("failed to read response file: %w", err)
		}
	}

	// Execute the script
	if err := c.executeScript(ctx, commands, output, responseData); err != nil {
		return err
	}

	// Execute followup script if specified
	if c.followupScript != "" {
		fmt.Println("Executing followup script:", c.followupScript)
		followupCmd := NewReplayCommand()
		// Copy relevant flags to the followup command
		followupCmd.endpoint = c.endpoint
		followupCmd.outputFile = c.outputFile
		followupCmd.method = c.method
		followupCmd.delay = c.delay
		followupCmd.dryRun = c.dryRun
		followupCmd.validate = c.validate
		followupCmd.strict = c.strict
		followupCmd.verbose = c.verbose
		followupCmd.interactive = c.interactive
		followupCmd.outputFormat = c.outputFormat
		followupCmd.ignoreErrors = c.ignoreErrors
		followupCmd.retries = c.retries
		followupCmd.retryDelay = c.retryDelay
		// Set the script file to the followup script
		followupCmd.scriptFile = c.followupScript

		// Execute the followup script
		if err := followupCmd.Execute(ctx, []string{}); err != nil {
			return fmt.Errorf("failed to execute followup script: %w", err)
		}
	}

	return nil
}

// parseScript parses a script file.
func (c *ReplayCommand) parseScript(filePath string) ([]*ScriptCommand, *SessionMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open script file: %w", err)
	}
	defer file.Close()

	var commands []*ScriptCommand
	metadata := &SessionMetadata{}

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse metadata from comments
		if strings.HasPrefix(line, "#") {
			c.parseMetadataFromComment(line, metadata)
			continue
		}

		// Parse command
		if strings.HasPrefix(line, "->") || strings.HasPrefix(line, "<-") {
			direction := line[:2]
			jsonStr := strings.TrimSpace(line[2:])

			var msg jsonrpc.Message
			isWildcard := strings.Contains(jsonStr, "*")

			if !isWildcard {
				if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
					return nil, nil, fmt.Errorf("line %d: invalid JSON: %w", lineNumber, err)
				}
			}

			commands = append(commands, &ScriptCommand{
				Direction:   direction,
				Message:     &msg,
				LineNumber:  lineNumber,
				IsWildcard:  isWildcard,
				RawMessage:  []byte(jsonStr),
				ExpectedMsg: jsonStr,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading script file: %w", err)
	}

	return commands, metadata, nil
}

// parseMetadataFromComment parses metadata from a comment line.
func (c *ReplayCommand) parseMetadataFromComment(line string, metadata *SessionMetadata) {
	line = strings.TrimPrefix(line, "#")
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "Name:") {
		metadata.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
	} else if strings.HasPrefix(line, "Description:") {
		metadata.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
	} else if strings.HasPrefix(line, "Date:") {
		dateStr := strings.TrimSpace(strings.TrimPrefix(line, "Date:"))
		t, err := time.Parse(time.RFC3339, dateStr)
		if err == nil {
			metadata.Date = t
		}
	} else if strings.HasPrefix(line, "Source:") {
		metadata.Source = strings.TrimSpace(strings.TrimPrefix(line, "Source:"))
	}
}

// printScriptInfo prints information about the script without executing it.
func (c *ReplayCommand) printScriptInfo(commands []*ScriptCommand, metadata *SessionMetadata) error {
	fmt.Println("Dry run mode enabled. Commands will not be executed.")

	// Print metadata
	fmt.Println("\nScript file:", c.scriptFile)
	if metadata.Name != "" {
		fmt.Println("Name:", metadata.Name)
	}
	if metadata.Description != "" {
		fmt.Println("Description:", metadata.Description)
	}
	if !metadata.Date.IsZero() {
		fmt.Println("Date:", metadata.Date.Format(time.RFC3339))
	}
	if metadata.Source != "" {
		fmt.Println("Source:", metadata.Source)
	}

	// Print configuration
	if c.endpoint != "" {
		fmt.Println("\nEndpoint:", c.endpoint)
	}
	if c.outputFile != "" {
		fmt.Println("Output:", c.outputFile)
	}
	if c.method != "" {
		fmt.Println("Method filter:", c.method)
	}
	if c.delay > 0 {
		fmt.Println("Delay:", c.delay, "seconds")
	}
	if c.range_ != "" {
		fmt.Println("Playing commands", c.range_)
	}

	// Print validation settings
	if c.validate {
		fmt.Println("\nValidation mode enabled")
		if c.strict {
			fmt.Println("Strict validation (no wildcards)")
		}
		if c.responseFile != "" {
			fmt.Println("Response file:", c.responseFile)
		}
	}

	// Print commands
	fmt.Printf("\nCommands (%d):\n", len(commands))
	for i, cmd := range commands {
		dirSymbol := "→"
		if cmd.Direction == "<-" {
			dirSymbol = "←"
		}

		if c.verbose {
			fmt.Printf("%d: %s Line %d: ", i+1, dirSymbol, cmd.LineNumber)
			if cmd.Direction == "->" && cmd.Message.Method != "" {
				fmt.Printf("Method: %s", cmd.Message.Method)
				if cmd.Message.ID != nil {
					fmt.Printf(", ID: %v", cmd.Message.ID)
				}
			} else if cmd.Direction == "<-" {
				if cmd.Message.Error != nil {
					fmt.Printf("Error response")
				} else {
					fmt.Printf("Response")
				}
				if cmd.Message.ID != nil {
					fmt.Printf(", ID: %v", cmd.Message.ID)
				}
			}
			fmt.Println()
		} else {
			methodName := "<unknown>"
			if cmd.Direction == "->" && cmd.Message.Method != "" {
				methodName = cmd.Message.Method
			} else if cmd.Direction == "<-" {
				if cmd.Message.Error != nil {
					methodName = "error"
				} else {
					methodName = "response"
				}
			}
			fmt.Printf("%d: %s %s\n", i+1, dirSymbol, methodName)
		}
	}

	return nil
}

// executeScript executes the script commands.
func (c *ReplayCommand) executeScript(ctx context.Context, commands []*ScriptCommand, output io.Writer, responseData []byte) error {
	fmt.Printf("Executing script with %d commands\n", len(commands))

	if c.validate {
		fmt.Println("Validation mode enabled")
	}

	// Create a new HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// If this is validation mode with response data, we don't need a real client
	var responseScanner *bufio.Scanner
	if c.validate && len(responseData) > 0 {
		responseScanner = bufio.NewScanner(strings.NewReader(string(responseData)))
	}

	// Find all request/response pairs
	var pairs [][]*ScriptCommand
	for i := 0; i < len(commands); {
		if i+1 < len(commands) && commands[i].Direction == "->" && commands[i+1].Direction == "<-" {
			// Add request/response pair
			pairs = append(pairs, []*ScriptCommand{commands[i], commands[i+1]})
			i += 2
		} else if commands[i].Direction == "->" {
			// Add request without expected response
			pairs = append(pairs, []*ScriptCommand{commands[i], nil})
			i++
		} else {
			// Skip lone responses
			i++
		}
	}

	// Execute each pair
	for i, pair := range pairs {
		req := pair[0]
		expectedResp := pair[1]

		// Print request information
		fmt.Printf("Command %d/%d: ", i+1, len(pairs))
		if req.Message.Method != "" {
			fmt.Printf("%s", req.Message.Method)
			if req.Message.ID != nil {
				fmt.Printf(" (id: %v)", req.Message.ID)
			}
		} else {
			fmt.Printf("Unknown request")
		}
		fmt.Println()

		// Prompt in interactive mode
		if c.interactive {
			fmt.Print("Press Enter to continue (or type 'q' to quit): ")
			var input string
			fmt.Scanln(&input)
			if input == "q" || input == "quit" {
				fmt.Println("Execution aborted by user.")
				return nil
			}
		}

		// Add delay if specified
		if c.delay > 0 && i > 0 {
			time.Sleep(time.Duration(c.delay * float64(time.Second)))
		}

		// If we're doing validation with supplied responses, use those
		if c.validate && responseScanner != nil {
			if !responseScanner.Scan() {
				return fmt.Errorf("not enough responses in response file")
			}

			responseLine := responseScanner.Text()
			if err := c.validateResponse(expectedResp, []byte(responseLine)); err != nil {
				if c.ignoreErrors {
					fmt.Printf("Validation error: %v (ignored)\n", err)
				} else {
					return fmt.Errorf("validation failed: %w", err)
				}
			} else {
				fmt.Println("Response validation successful")
			}

			// Output the response
			if c.outputFormat == "json" {
				// Pretty-print JSON
				var obj interface{}
				if err := json.Unmarshal([]byte(responseLine), &obj); err == nil {
					jsonData, err := json.MarshalIndent(obj, "", "  ")
					if err == nil {
						fmt.Fprintln(output, string(jsonData))
					} else {
						fmt.Fprintln(output, responseLine)
					}
				} else {
					fmt.Fprintln(output, responseLine)
				}
			} else if c.outputFormat == "raw" {
				fmt.Fprintln(output, responseLine)
			} else {
				// Default to text format
				fmt.Fprint(output, "Response: ")
				if strings.HasPrefix(responseLine, "{") {
					var msg jsonrpc.Message
					if err := json.Unmarshal([]byte(responseLine), &msg); err == nil {
						if msg.Result != nil {
							fmt.Fprintln(output, "Success")
						} else if msg.Error != nil {
							fmt.Fprintf(output, "Error: %s (code: %d)\n", msg.Error.Message, msg.Error.Code)
						} else {
							fmt.Fprintln(output, "Empty response")
						}
					} else {
						fmt.Fprintln(output, "Invalid JSON")
					}
				} else {
					fmt.Fprintln(output, responseLine)
				}
			}

			continue
		}

		// Skip actual execution in validation mode or if no endpoint is provided
		if c.validate || c.endpoint == "" {
			fmt.Println("Skipping execution (validation mode or no endpoint)")
			continue
		}

		// Send the request to the server
		resp, err := c.sendRequest(client, req)
		if err != nil {
			if c.ignoreErrors {
				fmt.Printf("Request error: %v (ignored)\n", err)
				continue
			} else {
				return fmt.Errorf("request failed: %w", err)
			}
		}

		// Validate the response if expected
		if expectedResp != nil {
			if err := c.validateResponse(expectedResp, resp); err != nil {
				if c.ignoreErrors {
					fmt.Printf("Validation error: %v (ignored)\n", err)
				} else {
					return fmt.Errorf("validation failed: %w", err)
				}
			} else {
				fmt.Println("Response validation successful")
			}
		}

		// Output the response
		if c.outputFormat == "json" {
			// Pretty-print JSON
			var obj interface{}
			if err := json.Unmarshal(resp, &obj); err == nil {
				jsonData, err := json.MarshalIndent(obj, "", "  ")
				if err == nil {
					fmt.Fprintln(output, string(jsonData))
				} else {
					fmt.Fprintln(output, string(resp))
				}
			} else {
				fmt.Fprintln(output, string(resp))
			}
		} else if c.outputFormat == "raw" {
			fmt.Fprintln(output, string(resp))
		} else {
			// Default to text format
			fmt.Fprint(output, "Response: ")
			if len(resp) > 0 && resp[0] == '{' {
				var msg jsonrpc.Message
				if err := json.Unmarshal(resp, &msg); err == nil {
					if msg.Result != nil {
						fmt.Fprintln(output, "Success")
					} else if msg.Error != nil {
						fmt.Fprintf(output, "Error: %s (code: %d)\n", msg.Error.Message, msg.Error.Code)
					} else {
						fmt.Fprintln(output, "Empty response")
					}
				} else {
					fmt.Fprintln(output, "Invalid JSON")
				}
			} else {
				fmt.Fprintln(output, string(resp))
			}
		}
	}

	return nil
}

// sendRequest sends a request to the server.
func (c *ReplayCommand) sendRequest(client *http.Client, req *ScriptCommand) ([]byte, error) {
	// Create the HTTP request
	httpReq, err := http.NewRequest("POST", c.endpoint, strings.NewReader(string(req.RawMessage)))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send the request with retries
	var httpResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retry %d/%d after %g seconds...\n", attempt, c.retries, c.retryDelay)
			time.Sleep(time.Duration(c.retryDelay * float64(time.Second)))
		}

		httpResp, lastErr = client.Do(httpReq)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("HTTP request failed after %d retries: %w", c.retries, lastErr)
	}
	defer httpResp.Body.Close()

	// Read the response
	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, nil
}

// validateResponse validates a response against an expected response.
func (c *ReplayCommand) validateResponse(expected *ScriptCommand, actual []byte) error {
	if expected == nil {
		return nil
	}

	// If strict validation is enabled, compare the actual response directly
	if c.strict {
		// Compare normalized JSON
		var expectedObj, actualObj interface{}
		if err := json.Unmarshal([]byte(expected.ExpectedMsg), &expectedObj); err != nil {
			return fmt.Errorf("failed to parse expected response: %w", err)
		}
		if err := json.Unmarshal(actual, &actualObj); err != nil {
			return fmt.Errorf("failed to parse actual response: %w", err)
		}

		expectedJSON, _ := json.Marshal(expectedObj)
		actualJSON, _ := json.Marshal(actualObj)

		if string(expectedJSON) != string(actualJSON) {
			return fmt.Errorf("response mismatch:\nExpected: %s\nActual: %s", expected.ExpectedMsg, string(actual))
		}

		return nil
	}

	// If the expected response contains wildcards, use pattern matching
	if expected.IsWildcard {
		return c.validateWildcardResponse(expected.ExpectedMsg, string(actual))
	}

	// Parse both messages
	var expectedMsg, actualMsg jsonrpc.Message
	if err := json.Unmarshal([]byte(expected.ExpectedMsg), &expectedMsg); err != nil {
		return fmt.Errorf("failed to parse expected response: %w", err)
	}
	if err := json.Unmarshal(actual, &actualMsg); err != nil {
		return fmt.Errorf("failed to parse actual response: %w", err)
	}

	// Compare IDs
	if !compareValues(expectedMsg.ID, actualMsg.ID) {
		return fmt.Errorf("ID mismatch: expected %v, got %v", expectedMsg.ID, actualMsg.ID)
	}

	// Compare error responses
	if expectedMsg.Error != nil {
		if actualMsg.Error == nil {
			return fmt.Errorf("expected error response, got success")
		}

		if expectedMsg.Error.Code != actualMsg.Error.Code {
			return fmt.Errorf("error code mismatch: expected %d, got %d", expectedMsg.Error.Code, actualMsg.Error.Code)
		}
	}

	// Compare success responses
	if expectedMsg.Result != nil {
		if actualMsg.Result == nil {
			return fmt.Errorf("expected success response, got error or empty response")
		}

		// Compare result objects (simple case)
		var expectedResult, actualResult interface{}
		if err := json.Unmarshal(expectedMsg.Result, &expectedResult); err != nil {
			return fmt.Errorf("failed to parse expected result: %w", err)
		}
		if err := json.Unmarshal(actualMsg.Result, &actualResult); err != nil {
			return fmt.Errorf("failed to parse actual result: %w", err)
		}

		if !compareObjects(expectedResult, actualResult) {
			return fmt.Errorf("result mismatch:\nExpected: %s\nActual: %s", string(expectedMsg.Result), string(actualMsg.Result))
		}
	}

	return nil
}

// validateWildcardResponse validates a response against an expected pattern with wildcards.
func (c *ReplayCommand) validateWildcardResponse(pattern, actual string) error {
	// Parse actual response
	var actualObj interface{}
	if err := json.Unmarshal([]byte(actual), &actualObj); err != nil {
		return fmt.Errorf("failed to parse actual response: %w", err)
	}

	// Convert pattern to a regexp
	regexpPattern := pattern

	// Escape special regexp characters
	regexpPattern = regexp.QuoteMeta(regexpPattern)

	// Replace wildcard asterisks with regexp pattern
	regexpPattern = strings.ReplaceAll(regexpPattern, "\\*", ".*")

	// Create regexp object
	re, err := regexp.Compile(regexpPattern)
	if err != nil {
		return fmt.Errorf("failed to compile pattern: %w", err)
	}

	// Match against the actual response
	if !re.MatchString(actual) {
		return fmt.Errorf("response doesn't match pattern:\nPattern: %s\nActual: %s", pattern, actual)
	}

	return nil
}

// compareValues compares two values with special handling for wildcard matching.
func compareValues(expected, actual interface{}) bool {
	// If expected is nil, actual can be anything
	if expected == nil {
		return true
	}

	// If expected is "*", actual can be anything
	if expectedStr, ok := expected.(string); ok && expectedStr == "*" {
		return true
	}

	// Otherwise, compare the values directly
	return fmt.Sprintf("%v", expected) == fmt.Sprintf("%v", actual)
}

// compareObjects compares two objects with special handling for wildcard matching.
func compareObjects(expected, actual interface{}) bool {
	// If expected is nil, actual can be anything
	if expected == nil {
		return true
	}

	// If expected is a string "*", actual can be anything
	if expectedStr, ok := expected.(string); ok && expectedStr == "*" {
		return true
	}

	// If types are different, they're not equal
	if fmt.Sprintf("%T", expected) != fmt.Sprintf("%T", actual) {
		return false
	}

	// Handle object types
	switch expectedVal := expected.(type) {
	case map[string]interface{}:
		actualVal, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}

		// All expected fields must be in actual
		for k, v := range expectedVal {
			actualV, ok := actualVal[k]
			if !ok {
				return false
			}

			if !compareObjects(v, actualV) {
				return false
			}
		}

		return true

	case []interface{}:
		actualVal, ok := actual.([]interface{})
		if !ok {
			return false
		}

		// If expected has more elements than actual, they're not equal
		if len(expectedVal) > len(actualVal) {
			return false
		}

		// Special case: if expected array has one element and it's "*", accept any array
		if len(expectedVal) == 1 {
			if strVal, ok := expectedVal[0].(string); ok && strVal == "*" {
				return true
			}
		}

		// Otherwise, compare element by element
		for i, v := range expectedVal {
			if i >= len(actualVal) {
				return false
			}

			if !compareObjects(v, actualVal[i]) {
				return false
			}
		}

		return true

	default:
		// For primitive types, compare as strings
		return fmt.Sprintf("%v", expected) == fmt.Sprintf("%v", actual)
	}
}

func main() {
	if err := NewReplayCommand().Execute(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
