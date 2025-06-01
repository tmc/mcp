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
	"time"

	"github.com/tmc/mcp/cmd/mcpspec/internal/command"
	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
)

// MessageType represents the type of a JSON-RPC message.
type MessageType int

const (
	RequestMessage MessageType = iota
	ResponseMessage
	NotificationMessage
	ErrorMessage
	UnknownMessage
)

// Message represents a JSON-RPC message with metadata.
type Message struct {
	Type        MessageType
	Data        *jsonrpc.Message
	RawData     []byte
	Timestamp   time.Time
	ElapsedTime time.Duration
	RequestID   interface{}
	Method      string
}

// SessionMetadata represents metadata for a session recording.
type SessionMetadata struct {
	Name        string
	Description string
	Date        time.Time
	Source      string
}

// TrafficStats tracks statistics about the observed traffic.
type TrafficStats struct {
	TotalMessages      int
	Requests           int
	Responses          int
	Notifications      int
	Errors             int
	Methods            map[string]int
	AverageRequestTime time.Duration
	TotalTime          time.Duration
}

// SpyCommand represents the mcp-spy command.
type SpyCommand struct {
	command.BaseCommand
	inputFile    string
	outputFile   string
	recordFile   string
	rawMode      bool
	colorMode    bool
	methodFilter string
	idFilter     string
	showTiming   bool
	hideHeaders  bool
	verbose      bool
	sessionName  string
	sessionDesc  string
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// NewSpyCommand creates a new SpyCommand.
func NewSpyCommand() *SpyCommand {
	return &SpyCommand{}
}

// Name returns the command name.
func (c *SpyCommand) Name() string {
	return "mcp-spy"
}

// Usage returns the command usage.
func (c *SpyCommand) Usage() string {
	return "Usage: mcp-spy [options]\n\n" +
		"Options:\n" +
		"  -i, --input <file>       Input file (default: stdin)\n" +
		"  -o, --output <file>      Output file (default: stdout)\n" +
		"  -r, --record <file>      Record session to file\n" +
		"  --raw                    Output raw JSON without formatting\n" +
		"  --color                  Colorize output\n" +
		"  -m, --method <method>    Filter by method\n" +
		"  --id <id>                Filter by ID\n" +
		"  -t, --timing             Show timing information\n" +
		"  --no-headers             Hide message headers\n" +
		"  -v, --verbose            Verbose output\n" +
		"  --name <name>            Session name for recording\n" +
		"  --description <desc>     Session description for recording\n"
}

// Execute runs the command.
func (c *SpyCommand) Execute(ctx context.Context, args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	fs.StringVar(&c.inputFile, "i", "", "Input file (default: stdin)")
	fs.StringVar(&c.inputFile, "input", "", "Input file (default: stdin)")
	fs.StringVar(&c.outputFile, "o", "", "Output file (default: stdout)")
	fs.StringVar(&c.outputFile, "output", "", "Output file (default: stdout)")
	fs.StringVar(&c.recordFile, "r", "", "Record session to file")
	fs.StringVar(&c.recordFile, "record", "", "Record session to file")
	fs.BoolVar(&c.rawMode, "raw", false, "Output raw JSON without formatting")
	fs.BoolVar(&c.colorMode, "color", false, "Colorize output")
	fs.StringVar(&c.methodFilter, "m", "", "Filter by method")
	fs.StringVar(&c.methodFilter, "method", "", "Filter by method")
	fs.StringVar(&c.idFilter, "id", "", "Filter by ID")
	fs.BoolVar(&c.showTiming, "t", false, "Show timing information")
	fs.BoolVar(&c.showTiming, "timing", false, "Show timing information")
	fs.BoolVar(&c.hideHeaders, "no-headers", false, "Hide message headers")
	fs.BoolVar(&c.verbose, "v", false, "Verbose output")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")
	fs.StringVar(&c.sessionName, "name", "", "Session name for recording")
	fs.StringVar(&c.sessionDesc, "description", "", "Session description for recording")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Open input file or use stdin
	var input io.Reader
	if c.inputFile != "" {
		file, err := os.Open(c.inputFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer file.Close()
		input = file
	} else {
		input = os.Stdin
	}

	// Open output file or use stdout
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

	// Create a session recorder if requested
	var recorder *SessionRecorder
	if c.recordFile != "" {
		metadata := SessionMetadata{
			Name:        c.sessionName,
			Description: c.sessionDesc,
			Date:        time.Now(),
			Source:      c.inputFile,
		}
		var err error
		recorder, err = NewSessionRecorder(c.recordFile, metadata)
		if err != nil {
			return fmt.Errorf("failed to create session recorder: %w", err)
		}
		defer recorder.Close()
	}

	// Process the input
	stats, err := c.processMessages(ctx, input, output, recorder)
	if err != nil {
		return err
	}

	// Print stats in verbose mode
	if c.verbose {
		c.printStats(output, stats)
	}

	return nil
}

// processMessages reads and processes JSON-RPC messages from the input.
func (c *SpyCommand) processMessages(ctx context.Context, input io.Reader, output io.Writer, recorder *SessionRecorder) (*TrafficStats, error) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024*10) // 10MB max size

	stats := &TrafficStats{
		Methods: make(map[string]int),
	}

	var pendingRequests = make(map[interface{}]*Message)
	startTime := time.Now()

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Parse the message
			msg, err := c.parseMessage(line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing message: %v\n", err)
				continue
			}

			// Apply filters
			if c.shouldFilter(msg) {
				continue
			}

			// Calculate elapsed time for responses
			if msg.Type == ResponseMessage || msg.Type == ErrorMessage {
				if req, ok := pendingRequests[msg.RequestID]; ok {
					msg.ElapsedTime = time.Since(req.Timestamp)
					delete(pendingRequests, msg.RequestID)
				}
			} else if msg.Type == RequestMessage {
				pendingRequests[msg.RequestID] = msg
			}

			// Update stats
			c.updateStats(stats, msg)

			// Display the message
			if !c.rawMode {
				c.displayFormattedMessage(output, msg)
			} else {
				fmt.Fprintln(output, line)
			}

			// Record the message if requested
			if recorder != nil {
				if err := recorder.RecordMessage(msg); err != nil {
					fmt.Fprintf(os.Stderr, "Error recording message: %v\n", err)
				}
			}
		}
	}

	stats.TotalTime = time.Since(startTime)

	if err := scanner.Err(); err != nil {
		return stats, fmt.Errorf("error reading input: %w", err)
	}

	return stats, nil
}

// parseMessage parses a JSON-RPC message.
func (c *SpyCommand) parseMessage(line string) (*Message, error) {
	var jsonrpcMsg jsonrpc.Message
	if err := json.Unmarshal([]byte(line), &jsonrpcMsg); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC message: %w", err)
	}

	msg := &Message{
		Type:      UnknownMessage,
		Data:      &jsonrpcMsg,
		RawData:   []byte(line),
		Timestamp: time.Now(),
		RequestID: jsonrpcMsg.ID,
		Method:    jsonrpcMsg.Method,
	}

	// Determine message type
	if jsonrpcMsg.Method != "" {
		if jsonrpcMsg.ID != nil {
			msg.Type = RequestMessage
		} else {
			msg.Type = NotificationMessage
		}
	} else if jsonrpcMsg.Error != nil {
		msg.Type = ErrorMessage
	} else {
		msg.Type = ResponseMessage
	}

	return msg, nil
}

// shouldFilter returns true if the message should be filtered out.
func (c *SpyCommand) shouldFilter(msg *Message) bool {
	// Filter by method
	if c.methodFilter != "" && msg.Method != c.methodFilter {
		return true
	}

	// Filter by ID
	if c.idFilter != "" {
		if msg.RequestID == nil {
			return true
		}

		var idStr string
		switch id := msg.RequestID.(type) {
		case string:
			idStr = id
		case float64:
			idStr = fmt.Sprintf("%g", id)
		case int:
			idStr = fmt.Sprintf("%d", id)
		default:
			idStr = fmt.Sprintf("%v", id)
		}

		if idStr != c.idFilter {
			return true
		}
	}

	return false
}

// updateStats updates the traffic statistics.
func (c *SpyCommand) updateStats(stats *TrafficStats, msg *Message) {
	stats.TotalMessages++

	switch msg.Type {
	case RequestMessage:
		stats.Requests++
		if msg.Method != "" {
			stats.Methods[msg.Method]++
		}
	case ResponseMessage:
		stats.Responses++
	case NotificationMessage:
		stats.Notifications++
		if msg.Method != "" {
			stats.Methods[msg.Method]++
		}
	case ErrorMessage:
		stats.Errors++
	}
}

// displayFormattedMessage displays a formatted message.
func (c *SpyCommand) displayFormattedMessage(output io.Writer, msg *Message) {
	var header string
	var colorStart, colorEnd string

	if c.colorMode {
		colorEnd = colorReset
	}

	switch msg.Type {
	case RequestMessage:
		header = "→ REQUEST"
		if c.colorMode {
			colorStart = colorGreen
		}
	case ResponseMessage:
		header = "← RESPONSE"
		if c.colorMode {
			colorStart = colorBlue
		}
	case NotificationMessage:
		header = "→ NOTIFICATION"
		if c.colorMode {
			colorStart = colorYellow
		}
	case ErrorMessage:
		header = "← ERROR"
		if c.colorMode {
			colorStart = colorRed
		}
	default:
		header = "? UNKNOWN"
		if c.colorMode {
			colorStart = colorPurple
		}
	}

	if !c.hideHeaders {
		if c.showTiming {
			fmt.Fprintf(output, "%s%s%s [Time: %s", colorStart, header, colorEnd, msg.Timestamp.Format("15:04:05.000"))
			if msg.ElapsedTime > 0 {
				fmt.Fprintf(output, ", Elapsed: %s", msg.ElapsedTime)
			}
			fmt.Fprintln(output, "]")
		} else {
			fmt.Fprintf(output, "%s%s%s\n", colorStart, header, colorEnd)
		}
	}

	// Format the message
	var jsonData []byte
	var err error

	if c.colorMode {
		// For colorized output, we need to format the JSON first
		var obj interface{}
		if err := json.Unmarshal(msg.RawData, &obj); err == nil {
			jsonData, err = json.MarshalIndent(obj, "", "  ")
			if err == nil {
				// Apply simple JSON highlighting (improved version possible)
				jsonStr := string(jsonData)
				jsonStr = strings.ReplaceAll(jsonStr, `"`, colorCyan+`"`+colorReset)
				jsonStr = strings.ReplaceAll(jsonStr, `{`, colorYellow+`{`+colorReset)
				jsonStr = strings.ReplaceAll(jsonStr, `}`, colorYellow+`}`+colorReset)
				jsonStr = strings.ReplaceAll(jsonStr, `[`, colorYellow+`[`+colorReset)
				jsonStr = strings.ReplaceAll(jsonStr, `]`, colorYellow+`]`+colorReset)
				fmt.Fprintln(output, jsonStr)
				return
			}
		}
	}

	// If color highlighting failed or is not enabled, use regular indentation
	var obj interface{}
	if err := json.Unmarshal(msg.RawData, &obj); err == nil {
		jsonData, err = json.MarshalIndent(obj, "", "  ")
		if err == nil {
			fmt.Fprintln(output, string(jsonData))
			return
		}
	}

	// If all else fails, just print the raw data
	fmt.Fprintln(output, string(msg.RawData))
}

// printStats prints the traffic statistics.
func (c *SpyCommand) printStats(output io.Writer, stats *TrafficStats) {
	fmt.Fprintln(output, "\nTraffic Statistics:")
	fmt.Fprintf(output, "  Total Messages: %d\n", stats.TotalMessages)
	fmt.Fprintf(output, "  Requests: %d\n", stats.Requests)
	fmt.Fprintf(output, "  Responses: %d\n", stats.Responses)
	fmt.Fprintf(output, "  Notifications: %d\n", stats.Notifications)
	fmt.Fprintf(output, "  Errors: %d\n", stats.Errors)
	fmt.Fprintf(output, "  Total Time: %s\n", stats.TotalTime)

	if len(stats.Methods) > 0 {
		fmt.Fprintln(output, "\nMethods:")
		for method, count := range stats.Methods {
			fmt.Fprintf(output, "  %s: %d\n", method, count)
		}
	}
}

// SessionRecorder records a session of JSON-RPC messages.
type SessionRecorder struct {
	file     *os.File
	writer   *bufio.Writer
	metadata SessionMetadata
}

// NewSessionRecorder creates a new session recorder.
func NewSessionRecorder(filePath string, metadata SessionMetadata) (*SessionRecorder, error) {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create session file: %w", err)
	}

	recorder := &SessionRecorder{
		file:     file,
		writer:   bufio.NewWriter(file),
		metadata: metadata,
	}

	// Write session metadata
	if err := recorder.writeMetadata(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	return recorder, nil
}

// writeMetadata writes the session metadata to the file.
func (r *SessionRecorder) writeMetadata() error {
	// Write metadata as comments
	fmt.Fprintln(r.writer, "# MCP Session Recording")
	fmt.Fprintln(r.writer, "#")
	if r.metadata.Name != "" {
		fmt.Fprintf(r.writer, "# Name: %s\n", r.metadata.Name)
	}
	if r.metadata.Description != "" {
		fmt.Fprintf(r.writer, "# Description: %s\n", r.metadata.Description)
	}
	fmt.Fprintf(r.writer, "# Date: %s\n", r.metadata.Date.Format(time.RFC3339))
	if r.metadata.Source != "" {
		fmt.Fprintf(r.writer, "# Source: %s\n", r.metadata.Source)
	}
	fmt.Fprintln(r.writer, "#")
	fmt.Fprintln(r.writer, "# -> indicates a request or notification")
	fmt.Fprintln(r.writer, "# <- indicates a response or error")
	fmt.Fprintln(r.writer, "#")

	return r.writer.Flush()
}

// RecordMessage records a message to the session file.
func (r *SessionRecorder) RecordMessage(msg *Message) error {
	var prefix string
	switch msg.Type {
	case RequestMessage, NotificationMessage:
		prefix = "-> "
	case ResponseMessage, ErrorMessage:
		prefix = "<- "
	default:
		prefix = "?? "
	}

	// Write the message with its direction prefix
	var obj interface{}
	if err := json.Unmarshal(msg.RawData, &obj); err == nil {
		// Format the JSON nicely
		jsonData, err := json.MarshalIndent(obj, "   ", "  ")
		if err == nil {
			fmt.Fprintf(r.writer, "%s%s\n", prefix, string(jsonData))
			return r.writer.Flush()
		}
	}

	// If formatting fails, use the raw data
	fmt.Fprintf(r.writer, "%s%s\n", prefix, string(msg.RawData))
	return r.writer.Flush()
}

// Close closes the session recorder.
func (r *SessionRecorder) Close() error {
	if r.writer != nil {
		r.writer.Flush()
	}
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

func main() {
	if err := NewSpyCommand().Execute(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
