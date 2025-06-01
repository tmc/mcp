package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
	"github.com/tmc/mcp/cmd/mcpspec/internal/io"
	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
)

// RecvCommand represents the mcp-recv command.
type RecvCommand struct {
	command.BaseCommand
	inputFile   string
	outputFile  string
	rawMode     bool
	prettyPrint bool
	field       string
}

// NewRecvCommand creates a new RecvCommand.
func NewRecvCommand() *RecvCommand {
	return &RecvCommand{}
}

// Name returns the command name.
func (c *RecvCommand) Name() string {
	return "mcp-recv"
}

// Usage returns the command usage.
func (c *RecvCommand) Usage() string {
	return "Usage: mcp-recv [options]\n\n" +
		"Options:\n" +
		"  -i, --input <file>      Input file (default: stdin)\n" +
		"  -o, --output <file>     Output file (default: stdout)\n" +
		"  -r, --raw               Output raw JSON without formatting\n" +
		"  --pretty                Pretty-print JSON output\n" +
		"  -f, --field <path>      Extract a specific field (e.g., 'method' or 'params.foo')\n"
}

// Execute runs the command.
func (c *RecvCommand) Execute(ctx context.Context, args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	fs.StringVar(&c.inputFile, "i", "", "Input file (default: stdin)")
	fs.StringVar(&c.inputFile, "input", "", "Input file (default: stdin)")
	fs.StringVar(&c.outputFile, "o", "", "Output file (default: stdout)")
	fs.StringVar(&c.outputFile, "output", "", "Output file (default: stdout)")
	fs.BoolVar(&c.rawMode, "r", false, "Output raw JSON without formatting")
	fs.BoolVar(&c.rawMode, "raw", false, "Output raw JSON without formatting")
	fs.BoolVar(&c.prettyPrint, "pretty", false, "Pretty-print JSON output")
	fs.StringVar(&c.field, "f", "", "Extract a specific field (e.g., 'method' or 'params.foo')")
	fs.StringVar(&c.field, "field", "", "Extract a specific field (e.g., 'method' or 'params.foo')")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Read the JSON-RPC message from file or stdin
	reader, err := io.NewFileReader(c.inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input: %w", err)
	}
	defer reader.Close()

	// Parse the JSON-RPC message
	var msg jsonrpc.Message
	if err := json.NewDecoder(reader).Decode(&msg); err != nil {
		return fmt.Errorf("failed to parse JSON-RPC message: %w", err)
	}

	// Create a writer for output
	writer, err := io.NewFileWriter(c.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output writer: %w", err)
	}
	defer writer.Close()

	// Handle raw mode (just output the message as-is)
	if c.rawMode {
		encoder := json.NewEncoder(writer)
		if c.prettyPrint {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(msg); err != nil {
			return fmt.Errorf("failed to write JSON-RPC message: %w", err)
		}
		return nil
	}

	// Handle field extraction
	if c.field != "" {
		value, err := extractField(&msg, c.field)
		if err != nil {
			return err
		}

		if jsonValue, ok := value.(map[string]interface{}); ok {
			// If the value is a JSON object, format it nicely
			encoder := json.NewEncoder(writer)
			if c.prettyPrint {
				encoder.SetIndent("", "  ")
			}
			if err := encoder.Encode(jsonValue); err != nil {
				return fmt.Errorf("failed to write JSON value: %w", err)
			}
		} else {
			// Otherwise, just print it as a string
			fmt.Fprintln(writer, fmt.Sprintf("%v", value))
		}
		return nil
	}

	// Format the message for human-readable output
	if c.prettyPrint {
		output, err := formatMessagePretty(&msg)
		if err != nil {
			return fmt.Errorf("failed to format message: %w", err)
		}
		fmt.Fprintln(writer, output)
	} else {
		output, err := formatMessage(&msg)
		if err != nil {
			return fmt.Errorf("failed to format message: %w", err)
		}
		fmt.Fprintln(writer, output)
	}

	return nil
}

// extractField extracts a field from a JSON-RPC message given a path like "params.foo".
func extractField(msg *jsonrpc.Message, path string) (interface{}, error) {
	parts := strings.Split(path, ".")

	var current interface{}

	// Start with the top-level field
	switch parts[0] {
	case "jsonrpc":
		return msg.Version, nil
	case "method":
		if msg.Method == "" {
			return nil, fmt.Errorf("field 'method' not found in message")
		}
		return msg.Method, nil
	case "params":
		if msg.Params == nil {
			return nil, fmt.Errorf("field 'params' not found in message")
		}
		if len(parts) == 1 {
			// Return the entire params object
			var params interface{}
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				return nil, fmt.Errorf("failed to parse params: %w", err)
			}
			return params, nil
		}
		// Initialize current for nested lookup
		if err := json.Unmarshal(msg.Params, &current); err != nil {
			return nil, fmt.Errorf("failed to parse params: %w", err)
		}
		// Skip "params" and start from the next part
		parts = parts[1:]
	case "result":
		if msg.Result == nil {
			return nil, fmt.Errorf("field 'result' not found in message")
		}
		if len(parts) == 1 {
			// Return the entire result object
			var result interface{}
			if err := json.Unmarshal(msg.Result, &result); err != nil {
				return nil, fmt.Errorf("failed to parse result: %w", err)
			}
			return result, nil
		}
		// Initialize current for nested lookup
		if err := json.Unmarshal(msg.Result, &current); err != nil {
			return nil, fmt.Errorf("failed to parse result: %w", err)
		}
		// Skip "result" and start from the next part
		parts = parts[1:]
	case "error":
		if msg.Error == nil {
			return nil, fmt.Errorf("field 'error' not found in message")
		}
		if len(parts) == 1 {
			// Return the entire error object
			return msg.Error, nil
		}
		// For error, we use the object directly
		current = map[string]interface{}{
			"code":    msg.Error.Code,
			"message": msg.Error.Message,
			"data":    msg.Error.Data,
		}
		// Skip "error" and start from the next part
		parts = parts[1:]
	case "id":
		if msg.ID == nil {
			return nil, fmt.Errorf("field 'id' not found in message")
		}
		return msg.ID, nil
	default:
		return nil, fmt.Errorf("unknown field: %s", parts[0])
	}

	// Navigate through nested fields
	for _, part := range parts {
		if obj, ok := current.(map[string]interface{}); ok {
			value, exists := obj[part]
			if !exists {
				return nil, fmt.Errorf("field '%s' not found", part)
			}
			current = value
		} else {
			return nil, fmt.Errorf("cannot access '%s' in non-object field", part)
		}
	}

	return current, nil
}

// formatMessage formats a JSON-RPC message in a human-readable format.
func formatMessage(msg *jsonrpc.Message) (string, error) {
	var sb strings.Builder

	// Determine message type
	if msg.Method != "" {
		if msg.ID != nil {
			// It's a request
			sb.WriteString(fmt.Sprintf("Request: %s (id: %v)\n", msg.Method, msg.ID))
		} else {
			// It's a notification
			sb.WriteString(fmt.Sprintf("Notification: %s\n", msg.Method))
		}

		// Format params if present
		if msg.Params != nil && len(msg.Params) > 0 && string(msg.Params) != "null" {
			var params interface{}
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				return "", fmt.Errorf("failed to parse params: %w", err)
			}
			paramsBytes, err := json.Marshal(params)
			if err != nil {
				return "", fmt.Errorf("failed to format params: %w", err)
			}
			sb.WriteString(fmt.Sprintf("Params: %s\n", paramsBytes))
		}
	} else if msg.Result != nil {
		// It's a success response
		sb.WriteString(fmt.Sprintf("Response: (id: %v)\n", msg.ID))
		var result interface{}
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			return "", fmt.Errorf("failed to parse result: %w", err)
		}
		resultBytes, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to format result: %w", err)
		}
		sb.WriteString(fmt.Sprintf("Result: %s\n", resultBytes))
	} else if msg.Error != nil {
		// It's an error response
		sb.WriteString(fmt.Sprintf("Error: (id: %v)\n", msg.ID))
		sb.WriteString(fmt.Sprintf("  code: %d\n", msg.Error.Code))
		sb.WriteString(fmt.Sprintf("  message: %s\n", msg.Error.Message))
		if msg.Error.Data != nil {
			sb.WriteString(fmt.Sprintf("  data: %v\n", msg.Error.Data))
		}
	} else {
		return "", fmt.Errorf("invalid JSON-RPC message: missing method, result, and error")
	}

	return sb.String(), nil
}

// formatMessagePretty formats a JSON-RPC message with pretty-printed JSON.
func formatMessagePretty(msg *jsonrpc.Message) (string, error) {
	var sb strings.Builder

	// Determine message type
	if msg.Method != "" {
		if msg.ID != nil {
			// It's a request
			sb.WriteString(fmt.Sprintf("Request: %s (id: %v)\n", msg.Method, msg.ID))
		} else {
			// It's a notification
			sb.WriteString(fmt.Sprintf("Notification: %s\n", msg.Method))
		}

		// Format params if present
		if msg.Params != nil && len(msg.Params) > 0 && string(msg.Params) != "null" {
			var params interface{}
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				return "", fmt.Errorf("failed to parse params: %w", err)
			}
			paramsBytes, err := json.MarshalIndent(params, "  ", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to format params: %w", err)
			}
			sb.WriteString("Params:\n  ")
			sb.WriteString(strings.ReplaceAll(string(paramsBytes), "\n", "\n  "))
			sb.WriteString("\n")
		}
	} else if msg.Result != nil {
		// It's a success response
		sb.WriteString(fmt.Sprintf("Response: (id: %v)\n", msg.ID))
		var result interface{}
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			return "", fmt.Errorf("failed to parse result: %w", err)
		}
		resultBytes, err := json.MarshalIndent(result, "  ", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to format result: %w", err)
		}
		sb.WriteString("Result:\n  ")
		sb.WriteString(strings.ReplaceAll(string(resultBytes), "\n", "\n  "))
		sb.WriteString("\n")
	} else if msg.Error != nil {
		// It's an error response
		sb.WriteString(fmt.Sprintf("Error: (id: %v)\n", msg.ID))
		sb.WriteString(fmt.Sprintf("  code: %d\n", msg.Error.Code))
		sb.WriteString(fmt.Sprintf("  message: %s\n", msg.Error.Message))
		if msg.Error.Data != nil {
			dataBytes, err := json.MarshalIndent(msg.Error.Data, "    ", "  ")
			if err != nil {
				return "", fmt.Errorf("failed to format error data: %w", err)
			}
			sb.WriteString("  data:\n    ")
			sb.WriteString(strings.ReplaceAll(string(dataBytes), "\n", "\n    "))
			sb.WriteString("\n")
		}
	} else {
		return "", fmt.Errorf("invalid JSON-RPC message: missing method, result, and error")
	}

	return sb.String(), nil
}

func main() {
	if err := NewRecvCommand().Execute(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
