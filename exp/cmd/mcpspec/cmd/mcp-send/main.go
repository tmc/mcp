package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
	"github.com/tmc/mcp/cmd/mcpspec/internal/io"
	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
)

// SendCommand represents the mcp-send command.
type SendCommand struct {
	command.BaseCommand
	method       string
	paramsFile   string
	id           int
	notification bool
	outputFile   string
	prettyPrint  bool
}

// NewSendCommand creates a new SendCommand.
func NewSendCommand() *SendCommand {
	return &SendCommand{}
}

// Name returns the command name.
func (c *SendCommand) Name() string {
	return "mcp-send"
}

// Usage returns the command usage.
func (c *SendCommand) Usage() string {
	return "Usage: mcp-send [options] <method>\n\n" +
		"Options:\n" +
		"  -p, --params <file>     JSON params file (default: stdin)\n" +
		"  -i, --id <id>           Request ID (default: 1, ignored for notifications)\n" +
		"  -n, --notification      Send as notification (no response expected)\n" +
		"  -o, --output <file>     Output file (default: stdout)\n" +
		"  --pretty                Pretty-print JSON output\n"
}

// Execute runs the command.
func (c *SendCommand) Execute(ctx context.Context, args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	fs.StringVar(&c.paramsFile, "p", "", "JSON params file (default: stdin)")
	fs.StringVar(&c.paramsFile, "params", "", "JSON params file (default: stdin)")
	fs.IntVar(&c.id, "i", 1, "Request ID (default: 1, ignored for notifications)")
	fs.IntVar(&c.id, "id", 1, "Request ID (default: 1, ignored for notifications)")
	fs.BoolVar(&c.notification, "n", false, "Send as notification (no response expected)")
	fs.BoolVar(&c.notification, "notification", false, "Send as notification (no response expected)")
	fs.StringVar(&c.outputFile, "o", "", "Output file (default: stdout)")
	fs.StringVar(&c.outputFile, "output", "", "Output file (default: stdout)")
	fs.BoolVar(&c.prettyPrint, "pretty", false, "Pretty-print JSON output")

	// Parse flags and extract the method
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Get the method argument
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: method is required")
		fmt.Fprintln(os.Stderr, c.Usage())
		return fmt.Errorf("method is required")
	}
	c.method = fs.Arg(0)

	// Read params from file or stdin
	var params interface{}
	if c.paramsFile != "" {
		// Read from file
		reader, err := io.NewFileReader(c.paramsFile)
		if err != nil {
			return fmt.Errorf("failed to open params file: %w", err)
		}
		defer reader.Close()

		if err := json.NewDecoder(reader).Decode(&params); err != nil {
			return fmt.Errorf("failed to parse params JSON: %w", err)
		}
	} else {
		// Read from stdin
		reader, err := io.NewFileReader("-")
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		defer reader.Close()

		if err := json.NewDecoder(reader).Decode(&params); err != nil {
			if err.Error() == "EOF" {
				// Empty input, use null params
				params = nil
			} else {
				return fmt.Errorf("failed to parse params JSON from stdin: %w", err)
			}
		}
	}

	// Create the JSON-RPC message
	var msg *jsonrpc.Message
	var err error
	if c.notification {
		msg, err = jsonrpc.NewNotification(c.method, params)
	} else {
		msg, err = jsonrpc.NewRequest(c.method, params, c.id)
	}
	if err != nil {
		return fmt.Errorf("failed to create JSON-RPC message: %w", err)
	}

	// Create a writer for output
	writer, err := io.NewFileWriter(c.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output writer: %w", err)
	}
	defer writer.Close()

	// Write the message to the output
	if c.prettyPrint {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(msg); err != nil {
			return fmt.Errorf("failed to write JSON-RPC message: %w", err)
		}
	} else {
		if err := json.NewEncoder(writer).Encode(msg); err != nil {
			return fmt.Errorf("failed to write JSON-RPC message: %w", err)
		}
	}

	return nil
}

func main() {
	if err := NewSendCommand().Execute(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
