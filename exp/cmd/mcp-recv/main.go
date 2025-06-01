package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/jsonrpc2"
)

func main() {
	log.SetPrefix("mcp-recv: ")

	var (
		input      = flag.String("input", "", "Input file (default: stdin)")
		output     = flag.String("output", "", "Output file (default: stdout)")
		format     = flag.Bool("format", false, "Pretty-print the JSON output")
		extract    = flag.String("extract", "", "Extract a specific field from the result (e.g., 'result.tools')")
		typeOnly   = flag.Bool("type", false, "Show only the message type (request, response, or notification)")
		methodOnly = flag.Bool("method", false, "Show only the method name (for requests and notifications)")
		verbose    = flag.Bool("v", false, "Verbose mode")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\nReceives an MCP message from stdin or input file and processes it\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  mcp-send -method tools/list | mcp-recv\n")
		fmt.Fprintf(os.Stderr, "  mcp-recv -input response.json -format\n")
		fmt.Fprintf(os.Stderr, "  mcp-recv -extract result.content\n")
		fmt.Fprintf(os.Stderr, "  mcp-recv -type\n")
	}

	flag.Parse()

	if err := run(*input, *output, *format, *extract, *typeOnly, *methodOnly, *verbose); err != nil {
		log.Fatal(err)
	}
}

func run(inputPath, outputPath string, format bool, extract string, typeOnly, methodOnly, verbose bool) error {
	// Set up input and output
	var in io.Reader
	if inputPath != "" {
		file, err := os.Open(inputPath)
		if err != nil {
			return fmt.Errorf("error opening input file: %w", err)
		}
		defer file.Close()
		in = file
	} else {
		in = os.Stdin
	}

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

	// Read and parse the JSON-RPC message
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return fmt.Errorf("no input received")
	}

	line := scanner.Bytes()
	if verbose {
		log.Printf("Received: %s", string(line))
	}

	// Since jsonrpc2.Message is an interface, we'll work with the raw JSON
	// and decode it as needed

	// Check if valid JSON
	var rawMsg map[string]json.RawMessage
	if err := json.Unmarshal(line, &rawMsg); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	// Process based on message type
	if typeOnly {
		return processTypeOnly(out, rawMsg)
	}

	if methodOnly {
		return processMethodOnly(out, rawMsg)
	}

	// Process the full message
	return processMessage(out, line, rawMsg, format, extract, verbose)
}

func processTypeOnly(out io.Writer, rawMsg map[string]json.RawMessage) error {
	var msgType string

	// Check for ID to determine if it's a request/response or notification
	_, hasID := rawMsg["id"]
	_, hasMethod := rawMsg["method"]

	if hasID {
		if hasMethod {
			msgType = "request"
		} else {
			msgType = "response"
		}
	} else if hasMethod {
		msgType = "notification"
	} else {
		msgType = "unknown"
	}

	_, err := fmt.Fprintln(out, msgType)
	return err
}

func processMethodOnly(out io.Writer, rawMsg map[string]json.RawMessage) error {
	methodBytes, hasMethod := rawMsg["method"]
	if !hasMethod {
		// Check if it's a response, and if so, try to extract the request method from context
		if _, hasID := rawMsg["id"]; hasID {
			return fmt.Errorf("cannot extract method: message is a response")
		}
		return fmt.Errorf("message has no method")
	}

	var method string
	if err := json.Unmarshal(methodBytes, &method); err != nil {
		return fmt.Errorf("error parsing method: %w", err)
	}

	_, err := fmt.Fprintln(out, method)
	return err
}

func processMessage(out io.Writer, data []byte, rawMsg map[string]json.RawMessage, format bool, extract string, verbose bool) error {
	// Check for ID and method to determine message type
	_, hasID := rawMsg["id"]
	_, hasMethod := rawMsg["method"]

	switch {
	case hasID:
		if hasMethod {
			// It's a request
			var req jsonrpc2.Request
			if err := json.Unmarshal(data, &req); err != nil {
				return fmt.Errorf("error parsing JSON-RPC request: %w", err)
			}
			return processRequest(out, &req, format, extract)
		} else {
			// It's a response
			var resp jsonrpc2.Response
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("error parsing JSON-RPC response: %w", err)
			}
			return processResponse(out, &resp, format, extract, verbose)
		}

	case hasMethod:
		// It's a notification
		type Notification struct {
			Method string           `json:"method"`
			Params *json.RawMessage `json:"params,omitempty"`
		}
		var notif Notification
		if err := json.Unmarshal(data, &notif); err != nil {
			return fmt.Errorf("error parsing JSON-RPC notification: %w", err)
		}
		return processNotification(out, &notif, format, extract)

	default:
		return fmt.Errorf("unrecognized message format")
	}
}

func processRequest(out io.Writer, req *jsonrpc2.Request, format bool, extract string) error {
	if extract != "" {
		if req.Params == nil {
			return fmt.Errorf("request has no params to extract from")
		}

		value, err := extractField(*req.Params, extract)
		if err != nil {
			return err
		}

		return printJSON(out, value, format)
	}

	// Print request details
	fmt.Fprintf(out, "Request ID: %v\n", req.ID())
	fmt.Fprintf(out, "Method: %s\n", req.Method())

	if req.Params() != nil {
		fmt.Fprintln(out, "Params:")
		return printJSON(out, req.Params(), format)
	}

	return nil
}

func processResponse(out io.Writer, resp *jsonrpc2.Response, format bool, extract string, verbose bool) error {
	// Handle error responses
	if resp.Error != nil {
		fmt.Fprintf(out, "Error: code=%d message=%s\n", resp.Error.Code, resp.Error.Message)
		if verbose && resp.Error.Data != nil {
			fmt.Fprintln(out, "Error data:")
			return printJSON(out, resp.Error.Data, format)
		}
		return nil
	}

	// Handle successful responses
	if resp.Result != nil {
		// Extract specific field if requested
		if extract != "" {
			value, err := extractField(*resp.Result, extract)
			if err != nil {
				return err
			}
			return printJSON(out, value, format)
		}

		fmt.Fprintf(out, "Response ID: %v\n", resp.ID)
		fmt.Fprintln(out, "Result:")
		return printJSON(out, resp.Result, format)
	}

	return nil
}

func processNotification(out io.Writer, notif *struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
}, format bool, extract string) error {
	if extract != "" {
		if notif.Params == nil {
			return fmt.Errorf("notification has no params to extract from")
		}

		value, err := extractField(*notif.Params, extract)
		if err != nil {
			return err
		}

		return printJSON(out, value, format)
	}

	// Print notification details
	fmt.Fprintf(out, "Notification Method: %s\n", notif.Method)

	if notif.Params != nil {
		fmt.Fprintln(out, "Params:")
		return printJSON(out, notif.Params, format)
	}

	return nil
}

// printJSON prints either formatted or raw JSON
func printJSON(out io.Writer, data interface{}, format bool) error {
	var bytes []byte
	var err error

	if format {
		bytes, err = json.MarshalIndent(data, "", "  ")
	} else {
		bytes, err = json.Marshal(data)
	}

	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	_, err = fmt.Fprintln(out, string(bytes))
	return err
}

// extractField extracts a nested field from a JSON object using dot notation
func extractField(data json.RawMessage, path string) (interface{}, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	parts := strings.Split(path, ".")
	return extractNestedField(obj, parts)
}

func extractNestedField(data interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return data, nil
	}

	current := path[0]
	remaining := path[1:]

	// Handle map (object) access
	if m, ok := data.(map[string]interface{}); ok {
		val, exists := m[current]
		if !exists {
			return nil, fmt.Errorf("field '%s' not found", current)
		}

		if len(remaining) == 0 {
			return val, nil
		}
		return extractNestedField(val, remaining)
	}

	// Handle array access with numeric index
	if a, ok := data.([]interface{}); ok {
		// Try to parse the path component as an array index
		var index int
		if _, err := fmt.Sscanf(current, "%d", &index); err != nil {
			return nil, fmt.Errorf("expected numeric index for array, got '%s'", current)
		}

		if index < 0 || index >= len(a) {
			return nil, fmt.Errorf("array index %d out of bounds (0-%d)", index, len(a)-1)
		}

		if len(remaining) == 0 {
			return a[index], nil
		}
		return extractNestedField(a[index], remaining)
	}

	return nil, fmt.Errorf("cannot extract '%s' from %T", current, data)
}
