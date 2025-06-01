package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetPrefix("mcp-send: ")

	var (
		method  = flag.String("method", "", "JSON-RPC method name (e.g., tools/call)")
		params  = flag.String("params", "", "JSON-RPC params as a string")
		file    = flag.String("file", "", "Path to a file containing the parameters JSON")
		output  = flag.String("output", "", "Output file (default: stdout)")
		id      = flag.Int("id", 1, "Request ID to use")
		notify  = flag.Bool("notify", false, "Send as notification (no ID)")
		verbose = flag.Bool("v", false, "Verbose mode")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\nSends an MCP message to stdout or specified output file\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  mcp-send -method tools/list\n")
		fmt.Fprintf(os.Stderr, "  mcp-send -method tools/call -params '{\"name\":\"echo\",\"arguments\":{\"message\":\"hello\"}}'\n")
		fmt.Fprintf(os.Stderr, "  mcp-send -method tools/call -file params.json\n")
		fmt.Fprintf(os.Stderr, "  mcp-send -method tools/list -notify\n")
	}

	flag.Parse()

	if *method == "" {
		log.Fatal("method is required")
	}

	if err := run(*method, *params, *file, *output, *id, *notify, *verbose); err != nil {
		log.Fatal(err)
	}
}

func run(method, paramsStr, filePath, outputPath string, id int, notify, verbose bool) error {
	var paramsJSON json.RawMessage

	// Check if params is provided via file or command line
	if filePath != "" {
		if paramsStr != "" {
			return fmt.Errorf("cannot specify both -params and -file")
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("error reading params file: %w", err)
		}
		paramsJSON = json.RawMessage(data)
	} else if paramsStr != "" {
		// Validate that the params string is valid JSON
		var obj interface{}
		if err := json.Unmarshal([]byte(paramsStr), &obj); err != nil {
			return fmt.Errorf("params is not valid JSON: %w", err)
		}
		paramsJSON = json.RawMessage(paramsStr)
	}

	// Validate the method is a known MCP method
	if !isValidMethod(method) {
		log.Printf("Warning: '%s' is not a standard MCP method", method)
	}

	// Create the message
	var msgBytes []byte
	var err error

	if notify {
		// Create a notification (no ID)
		type Notification struct {
			JSONRPC string           `json:"jsonrpc"`
			Method  string           `json:"method"`
			Params  *json.RawMessage `json:"params,omitempty"`
		}
		notification := &Notification{
			JSONRPC: "2.0",
			Method:  method,
		}
		if len(paramsJSON) > 0 {
			notification.Params = &paramsJSON
		}
		msgBytes, err = json.Marshal(notification)
	} else {
		// Create a request with ID
		type Request struct {
			JSONRPC string           `json:"jsonrpc"`
			ID      int              `json:"id"`
			Method  string           `json:"method"`
			Params  *json.RawMessage `json:"params,omitempty"`
		}
		request := &Request{
			JSONRPC: "2.0",
			ID:      id,
			Method:  method,
		}
		if len(paramsJSON) > 0 {
			request.Params = &paramsJSON
		}
		msgBytes, err = json.Marshal(request)
	}

	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	if verbose {
		log.Printf("Sending: %s", string(msgBytes))
	}

	// Write to file or stdout
	var out io.Writer
	if outputPath != "" {
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("error creating output file: %w", err)
		}
		defer file.Close()
		out = file
	} else {
		out = os.Stdout
	}

	// Write the JSON bytes directly without adding a newline
	if _, err := out.Write(msgBytes); err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}

	return nil
}

// isValidMethod checks if the method name is a standard MCP method
func isValidMethod(method string) bool {
	standardMethods := []string{
		"initialize",
		"ping",
		"resources/list",
		"resources/templates/list",
		"resources/read",
		"prompts/list",
		"prompts/get",
		"tools/list",
		"tools/call",
	}

	for _, m := range standardMethods {
		if m == method {
			return true
		}
	}

	return false
}
