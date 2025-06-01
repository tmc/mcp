// mcp-trace-codegen: Real-time Go code generation from MCP traces
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tmc/mcp/exp/trace"
	"github.com/tmc/mcp/exp/codegen"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type TraceCodeGenerator struct {
	analyzer      *trace.Analyzer
	generator     *codegen.Generator
	buffer        *RollingCodeBuffer
	lastUpdate    time.Time
	opts          Options
}

type Options struct {
	realtime     bool
	showProgress bool
	clearScreen  bool
	packageName  string
	outputFile   string
}

func main() {
	var opts Options
	flag.BoolVar(&opts.realtime, "realtime", true, "Enable real-time display")
	flag.BoolVar(&opts.showProgress, "progress", true, "Show progress indicators")
	flag.BoolVar(&opts.clearScreen, "clear", true, "Clear screen on updates")
	flag.StringVar(&opts.packageName, "package", "generated", "Package name for generated code")
	flag.StringVar(&opts.outputFile, "output", "", "Output file (default: stdout)")
	flag.Parse()

	tcg := NewTraceCodeGenerator(opts)
	
	reader := bufio.NewReader(os.Stdin)
	if err := tcg.ProcessStream(reader); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewTraceCodeGenerator(opts Options) *TraceCodeGenerator {
	return &TraceCodeGenerator{
		analyzer:  trace.NewAnalyzer(),
		generator: codegen.NewGenerator(opts.packageName),
		buffer:    NewRollingCodeBuffer(),
		opts:      opts,
	}
}

func (tcg *TraceCodeGenerator) ProcessStream(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	
	for scanner.Scan() {
		line := scanner.Text()
		if err := tcg.ProcessLine(line); err != nil {
			// Log error but continue processing
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
	
	return scanner.Err()
}

func (tcg *TraceCodeGenerator) ProcessLine(line string) error {
	// Parse the trace line
	entry, err := tcg.parseTraceLine(line)
	if err != nil {
		return fmt.Errorf("parsing line: %w", err)
	}
	
	// Update analyzer state
	tcg.analyzer.ProcessEntry(entry)
	
	// Generate/update code based on new information
	code := tcg.generateCode()
	
	// Update display
	if tcg.opts.realtime {
		tcg.displayUpdate(code)
	}
	
	return nil
}

func (tcg *TraceCodeGenerator) parseTraceLine(line string) (*trace.Entry, error) {
	// MCP trace format: timestamp direction method params/result
	parts := strings.SplitN(line, " ", 4)
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid trace line format")
	}
	
	entry := &trace.Entry{
		Timestamp: parts[0],
		Direction: parts[1],
		Method:    parts[2],
	}
	
	// Parse JSON payload
	if err := json.Unmarshal([]byte(parts[3]), &entry.Payload); err != nil {
		return nil, fmt.Errorf("parsing payload: %w", err)
	}
	
	return entry, nil
}

func (tcg *TraceCodeGenerator) generateCode() string {
	state := tcg.analyzer.GetState()
	
	var code strings.Builder
	
	// Package and imports
	code.WriteString(tcg.generator.GeneratePackage())
	code.WriteString("\n\n")
	code.WriteString(tcg.generator.GenerateImports(state))
	code.WriteString("\n\n")
	
	// Server type
	if state.HasServer {
		code.WriteString(tcg.generator.GenerateServerType(state))
		code.WriteString("\n\n")
	}
	
	// Client type
	if state.HasClient {
		code.WriteString(tcg.generator.GenerateClientType(state))
		code.WriteString("\n\n")
	}
	
	// Tools
	for _, tool := range state.Tools {
		code.WriteString(tcg.generator.GenerateTool(tool))
		code.WriteString("\n\n")
	}
	
	// Handlers
	for _, handler := range state.Handlers {
		code.WriteString(tcg.generator.GenerateHandler(handler))
		code.WriteString("\n\n")
	}
	
	// Main function if applicable
	if state.IsExecutable {
		code.WriteString(tcg.generator.GenerateMain(state))
		code.WriteString("\n")
	}
	
	return code.String()
}

func (tcg *TraceCodeGenerator) displayUpdate(code string) {
	// Clear screen if requested
	if tcg.opts.clearScreen {
		fmt.Print("\033[H\033[2J")
	}
	
	// Show progress
	if tcg.opts.showProgress {
		state := tcg.analyzer.GetState()
		fmt.Printf("=== MCP Trace Code Generator ===\n")
		fmt.Printf("Messages: %d | Tools: %d | Handlers: %d\n",
			state.MessageCount, len(state.Tools), len(state.Handlers))
		fmt.Printf("Last update: %s\n", time.Now().Format("15:04:05"))
		fmt.Println(strings.Repeat("=", 30))
		fmt.Println()
	}
	
	// Display the code
	fmt.Print(code)
	
	// If output file specified, also write there
	if tcg.opts.outputFile != "" {
		if err := os.WriteFile(tcg.opts.outputFile, []byte(code), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		}
	}
	
	tcg.lastUpdate = time.Now()
}

// RollingCodeBuffer maintains a window of generated code for smooth updates
type RollingCodeBuffer struct {
	current  string
	previous string
}

func NewRollingCodeBuffer() *RollingCodeBuffer {
	return &RollingCodeBuffer{}
}

func (b *RollingCodeBuffer) Update(code string) (string, bool) {
	if code == b.current {
		return "", false
	}
	
	b.previous = b.current
	b.current = code
	return code, true
}