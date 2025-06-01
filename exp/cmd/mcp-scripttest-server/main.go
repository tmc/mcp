package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/txtar"
)

var (
	scriptFile  = flag.String("script", "", "Path to scripttest file")
	workDir     = flag.String("workdir", "", "Working directory for file operations (defaults to temp dir)")
	port        = flag.String("port", "8080", "Port to listen on")
	verboseMode = flag.Bool("v", false, "Verbose mode")
)

// ScriptCommand represents a single command in the script file
type ScriptCommand struct {
	Command   string                 // expect-recv, send, exec, etc.
	Data      string                 // Raw command data
	Pattern   map[string]interface{} // Parsed JSON pattern for expect-recv
	Response  map[string]interface{} // Response to send
	Variables map[string]string      // Variables for this command
}

// ServerState holds the current state of the server
type ServerState struct {
	Variables      map[string]string
	Counters       map[string]int
	RequestHistory []map[string]interface{}
	CurrentDir     string
	mutex          sync.RWMutex
}

// Create a new server state
func NewServerState() *ServerState {
	return &ServerState{
		Variables:      make(map[string]string),
		Counters:       make(map[string]int),
		RequestHistory: []map[string]interface{}{},
		CurrentDir:     "",
	}
}

// Set a variable in the server state
func (s *ServerState) SetVariable(name, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Variables[name] = value
}

// Get a variable from the server state
func (s *ServerState) GetVariable(name string) (string, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	val, ok := s.Variables[name]
	return val, ok
}

// Increment a counter in the server state
func (s *ServerState) IncrementCounter(name string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Counters[name]++
	return s.Counters[name]
}

// Get a counter value from the server state
func (s *ServerState) GetCounter(name string) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Counters[name]
}

// Add a request to the history
func (s *ServerState) AddRequest(request map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.RequestHistory = append(s.RequestHistory, request)
}

// Global server state
var serverState *ServerState

// Script holds the parsed and processed script file
type Script struct {
	Commands  []ScriptCommand
	FileState map[string][]byte
}

func main() {
	flag.Parse()

	if *scriptFile == "" {
		log.Fatal("Script file is required")
	}

	// Create working directory if not specified
	if *workDir == "" {
		tempDir, err := os.MkdirTemp("", "mcp-scripttest-server")
		if err != nil {
			log.Fatalf("Failed to create temp directory: %v", err)
		}
		*workDir = tempDir
		defer os.RemoveAll(tempDir)
	} else {
		// Ensure working directory exists
		err := os.MkdirAll(*workDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create working directory: %v", err)
		}
	}

	// Parse script file
	script, err := parseScriptFile(*scriptFile)
	if err != nil {
		log.Fatalf("Failed to parse script file: %v", err)
	}

	// Initialize server state
	serverState = NewServerState()
	serverState.CurrentDir = *workDir
	serverState.SetVariable("WORKDIR", *workDir)

	// Set up file state from script
	setupFileState(script)

	// Setup and start server
	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		handleMCPRequest(w, r, script)
	})

	log.Printf("Starting mcp-scripttest-server on port %s", *port)
	log.Printf("Working directory: %s", *workDir)
	log.Printf("Using script file: %s", *scriptFile)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *port), nil))
}

// Parse the script file
func parseScriptFile(filePath string) (*Script, error) {
	// Read the script file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}

	// Parse as txtar
	archive := txtar.Parse(data)

	// Process script commands
	script := &Script{
		Commands:  []ScriptCommand{},
		FileState: make(map[string][]byte),
	}

	// Process commands from the script section
	lines := strings.Split(string(archive.Comment), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if *verboseMode {
			log.Printf("Processing script line %d: %s", i+1, line)
		}

		// Parse command
		parts := strings.SplitN(line, " ", 2)
		command := parts[0]
		var data string
		if len(parts) > 1 {
			data = parts[1]
		}

		scriptCmd := ScriptCommand{
			Command:   command,
			Data:      data,
			Variables: make(map[string]string),
		}

		// Parse JSON patterns or responses
		if command == "expect-recv" {
			var pattern map[string]interface{}
			err := json.Unmarshal([]byte(data), &pattern)
			if err != nil {
				return nil, fmt.Errorf("error parsing JSON pattern on line %d: %w", i+1, err)
			}
			scriptCmd.Pattern = pattern
		} else if command == "send" {
			var response map[string]interface{}
			err := json.Unmarshal([]byte(data), &response)
			if err != nil {
				return nil, fmt.Errorf("error parsing JSON response on line %d: %w", i+1, err)
			}
			scriptCmd.Response = response
		}

		script.Commands = append(script.Commands, scriptCmd)
	}

	// Process file state from archive files
	for _, file := range archive.Files {
		script.FileState[file.Name] = file.Data
	}

	return script, nil
}

// Setup file state from script
func setupFileState(script *Script) {
	for fileName, content := range script.FileState {
		// Get absolute file path
		filePath := filepath.Join(serverState.CurrentDir, fileName)

		// Create directory if it doesn't exist
		dir := filepath.Dir(filePath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("Failed to create directory %s: %v", dir, err)
			continue
		}

		// Write file
		err = os.WriteFile(filePath, content, 0644)
		if err != nil {
			log.Printf("Failed to write file %s: %v", filePath, err)
			continue
		}

		if *verboseMode {
			log.Printf("Created file: %s (%d bytes)", filePath, len(content))
		}
	}
}

// Handle an incoming MCP request
func handleMCPRequest(w http.ResponseWriter, r *http.Request, script *Script) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusInternalServerError)
		return
	}

	// Parse request
	var request map[string]interface{}
	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log request
	if *verboseMode {
		log.Printf("Received request: %s", string(body))
	}

	// Add request to history
	serverState.AddRequest(request)

	// Process request
	responseData, err := processRequest(request, script)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error processing request: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseData)
}

// Process a request according to the script
func processRequest(request map[string]interface{}, script *Script) ([]byte, error) {
	// Find matching expect-recv command
	matchedCommand := -1
	var matchedPattern map[string]interface{}

	for i, cmd := range script.Commands {
		if cmd.Command != "expect-recv" {
			continue
		}

		if matchesPattern(request, cmd.Pattern) {
			matchedCommand = i
			matchedPattern = cmd.Pattern
			break
		}
	}
	_ = matchedPattern

	if matchedCommand < 0 {
		// No matching command found, return default error response
		return json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"error": map[string]interface{}{
				"code":    -32603,
				"message": "Unexpected request: no matching pattern in script",
			},
		})
	}

	if *verboseMode {
		log.Printf("Matched command %d: %s %s", matchedCommand, script.Commands[matchedCommand].Command, script.Commands[matchedCommand].Data)
	}

	// Execute all commands from the matched expect-recv until the next expect-recv
	response, err := executeCommandBlock(matchedCommand, request, script)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Execute commands from a matched expect-recv until the next expect-recv
func executeCommandBlock(startIdx int, request map[string]interface{}, script *Script) ([]byte, error) {
	var response []byte
	var err error

	// Skip the matched expect-recv command
	i := startIdx + 1

	// Execute all commands until the next expect-recv
	for i < len(script.Commands) && script.Commands[i].Command != "expect-recv" {
		cmd := script.Commands[i]

		if *verboseMode {
			log.Printf("Executing command: %s %s", cmd.Command, cmd.Data)
		}

		switch cmd.Command {
		case "send":
			// Process variables in the response
			processedResponse := processVariables(cmd.Response, request)
			response, err = json.Marshal(processedResponse)
			if err != nil {
				return nil, fmt.Errorf("error marshaling response: %w", err)
			}

			if *verboseMode {
				log.Printf("Sending response: %s", string(response))
			}

		case "exec":
			// Execute shell command
			err = executeShellCommand(cmd.Data)
			if err != nil {
				log.Printf("Error executing shell command: %v", err)
				// Continue with script even if command fails
			}

		case "set-var":
			// Set a variable
			varParts := strings.SplitN(cmd.Data, "=", 2)
			if len(varParts) != 2 {
				return nil, fmt.Errorf("invalid set-var format: %s", cmd.Data)
			}
			varName := strings.TrimSpace(varParts[0])
			varValue := strings.TrimSpace(varParts[1])

			// Handle shell command expansion
			if strings.HasPrefix(varValue, "$(") && strings.HasSuffix(varValue, ")") {
				cmdStr := varValue[2 : len(varValue)-1]
				output, err := executeShellCommandWithOutput(cmdStr)
				if err != nil {
					log.Printf("Error executing command for variable: %v", err)
					varValue = ""
				} else {
					varValue = strings.TrimSpace(output)
				}
			}

			serverState.SetVariable(varName, varValue)
			if *verboseMode {
				log.Printf("Set variable %s = %s", varName, varValue)
			}

		case "counter":
			// Increment or check counter
			parts := strings.Split(cmd.Data, " ")
			counterName := parts[0]
			if len(parts) == 1 {
				// Just increment the counter
				newVal := serverState.IncrementCounter(counterName)
				if *verboseMode {
					log.Printf("Incremented counter %s to %d", counterName, newVal)
				}
			}

		case "delay":
			// Delay execution
			delayMs, err := strconv.Atoi(cmd.Data)
			if err != nil {
				log.Printf("Invalid delay value: %s", cmd.Data)
			} else {
				if *verboseMode {
					log.Printf("Delaying for %d ms", delayMs)
				}
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}

		case "if", "else", "endif":
			// Basic flow control
			// NOTE: This is a simplified implementation that only supports basic counter checking
			if cmd.Command == "if" {
				parts := strings.Split(cmd.Data, " ")
				if len(parts) >= 3 {
					counterName := parts[0]
					operator := parts[1]
					valueStr := parts[2]

					counterValue := serverState.GetCounter(counterName)
					value, err := strconv.Atoi(valueStr)

					if err != nil {
						log.Printf("Invalid value in if condition: %s", valueStr)
						// Skip to endif
						for i < len(script.Commands) && script.Commands[i].Command != "endif" {
							i++
						}
					} else {
						conditionMet := false
						switch operator {
						case ">":
							conditionMet = counterValue > value
						case ">=":
							conditionMet = counterValue >= value
						case "<":
							conditionMet = counterValue < value
						case "<=":
							conditionMet = counterValue <= value
						case "==":
							conditionMet = counterValue == value
						case "!=":
							conditionMet = counterValue != value
						}

						if !conditionMet {
							// Skip to else or endif
							for i < len(script.Commands) {
								i++
								if i >= len(script.Commands) {
									break
								}
								if script.Commands[i].Command == "else" || script.Commands[i].Command == "endif" {
									break
								}
							}

							if i < len(script.Commands) && script.Commands[i].Command == "else" {
								// Move past the else
								i++
							}
						}
					}
				}
			} else if cmd.Command == "else" {
				// Skip to endif
				for i < len(script.Commands) && script.Commands[i].Command != "endif" {
					i++
				}
			}
		}

		i++
	}

	return response, nil
}

// Check if a request matches a pattern
func matchesPattern(request, pattern map[string]interface{}) bool {
	for key, expectedValue := range pattern {
		actualValue, exists := request[key]
		if !exists {
			return false
		}

		// Check for type match
		switch expected := expectedValue.(type) {
		case string:
			// For string values, support wildcards
			if actual, ok := actualValue.(string); ok {
				if strings.Contains(expected, "*") {
					// Convert wildcard pattern to regex
					regexPattern := "^" + strings.Replace(regexp.QuoteMeta(expected), "\\*", ".*", -1) + "$"
					matched, err := regexp.MatchString(regexPattern, actual)
					if err != nil || !matched {
						return false
					}
				} else if expected != actual {
					return false
				}
			} else {
				return false
			}
		case map[string]interface{}:
			// For nested maps, recursively check
			if actualMap, ok := actualValue.(map[string]interface{}); ok {
				if !matchesPattern(actualMap, expected) {
					return false
				}
			} else {
				return false
			}
		default:
			// For other types, check exact equality
			if !reflect.DeepEqual(expectedValue, actualValue) {
				return false
			}
		}
	}

	return true
}

// Process variables in a response
func processVariables(response map[string]interface{}, request map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy and process each field
	for key, value := range response {
		processedValue := processValueVariables(value, request)
		result[key] = processedValue
	}

	return result
}

// Process variables in a value (recursively)
func processValueVariables(value interface{}, request map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Replace ${VAR} with variable values
		return replaceVariables(v, request)
	case map[string]interface{}:
		// Process each field in the map
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = processValueVariables(val, request)
		}
		return result
	case []interface{}:
		// Process each item in the array
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = processValueVariables(val, request)
		}
		return result
	default:
		// Return other values as-is
		return v
	}
}

// Replace ${VAR} with variable values
func replaceVariables(text string, request map[string]interface{}) string {
	// Replace built-in variables
	result := text

	// Replace variables from state
	re := regexp.MustCompile(`\${([^}]+)}`)
	matches := re.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		varName := match[1]
		replacement := ""

		// Special handling for request fields
		if strings.HasPrefix(varName, "REQUEST.") {
			field := strings.TrimPrefix(varName, "REQUEST.")
			replacement = getRequestField(request, field)
		} else if varName == "ID" {
			// Extract ID from request
			if id, ok := request["id"]; ok {
				replacement = fmt.Sprintf("%v", id)
			}
		} else {
			// Regular variable from state
			if val, ok := serverState.GetVariable(varName); ok {
				replacement = val
			}
		}

		result = strings.Replace(result, match[0], replacement, -1)
	}

	return result
}

// Get a field from the request (supports dot notation for nested fields)
func getRequestField(request map[string]interface{}, field string) string {
	parts := strings.Split(field, ".")
	current := request

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return as string
			if val, ok := current[part]; ok {
				return fmt.Sprintf("%v", val)
			}
			return ""
		}

		// Navigate to nested field
		if nested, ok := current[part].(map[string]interface{}); ok {
			current = nested
		} else {
			return ""
		}
	}

	return ""
}

// Execute a shell command in the working directory
func executeShellCommand(command string) error {
	// Replace variables in command
	command = replaceVariables(command, nil)

	// Execute command
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = serverState.CurrentDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("WORKDIR=%s", serverState.CurrentDir))

	// Capture output for logging
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if *verboseMode {
		if stdout.Len() > 0 {
			log.Printf("Command stdout: %s", stdout.String())
		}
		if stderr.Len() > 0 {
			log.Printf("Command stderr: %s", stderr.String())
		}
	}

	return err
}

// Execute a shell command and return its output
func executeShellCommandWithOutput(command string) (string, error) {
	// Replace variables in command
	command = replaceVariables(command, nil)

	// Execute command
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = serverState.CurrentDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("WORKDIR=%s", serverState.CurrentDir))

	output, err := cmd.Output()
	return string(output), err
}

// reflect is used for deep equals implementation
