package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/mcp"
)

func main() {
	// Configure logging to stderr
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("mcp-exec server starting...")

	// Define and parse command-line flags
	var (
		name    = flag.String("name", "exec-server", "name of the server")
		version = flag.String("version", "0.1.0", "version of the server")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Create server
	server := mcp.NewServer(*name, *version)
	log.Printf("Created server: name=%s version=%s", *name, *version)

	// Register the exec tool
	err := server.RegisterTool(NewExecTool())
	if err != nil {
		log.Fatalf("Failed to register exec tool: %v", err)
	}
	log.Printf("Registered exec tool")

	// Create transport
	transport := mcp.NewStdioTransport(context.Background())
	log.Printf("Created stdio transport")

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Send a ping notification
				ping := map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "$/ping",
					"params":  map[string]interface{}{},
				}
				pingJSON, _ := json.Marshal(ping)
				log.Printf("Sending ping...")
				_, err := transport.Write(append(pingJSON, '\n'))
				if err != nil {
					log.Printf("Ping error: %v", err)
				}
			}
		}
	}()

	// Handle messages
	for {
		msg := make([]byte, 4096)
		log.Printf("Waiting for message...")
		n, err := transport.Read(msg)
		if err != nil {
			if err == io.EOF {
				log.Printf("EOF received, waiting for more input...")
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log.Printf("Read error: %v", err)
			continue
		}

		if n == 0 {
			log.Printf("Empty message received, waiting for more input...")
			time.Sleep(100 * time.Millisecond)
			continue
		}

		log.Printf("Received message: %s", msg[:n])

		resp, err := server.Handle(context.Background(), msg[:n])
		if err != nil {
			log.Printf("Handle error: %v", err)
			continue
		}

		log.Printf("Sending response: %s", resp)
		_, err = transport.Write(append(resp, '\n'))
		if err != nil {
			log.Printf("Write error: %v", err)
			continue
		}
		log.Printf("Response sent successfully")
	}
}

// ExecTool implements a Tool that executes shell commands.
type ExecTool struct {
	name        string
	description string
}

// NewExecTool creates a new ExecTool instance.
func NewExecTool() mcp.Tool {
	return &ExecTool{
		name:        "exec",
		description: "Executes a shell command and returns the output.",
	}
}

// Name returns the name of the tool.
func (t *ExecTool) Name() string {
	return t.name
}

// Description returns the description of the tool.
func (t *ExecTool) Description() string {
	return t.description
}

// Handler executes the shell command and returns the output.
func (t *ExecTool) Handler(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
	// Handle base64-encoded JSON
	if len(args) > 0 && args[0] == '"' {
		var encodedArgs string
		if err := json.Unmarshal(args, &encodedArgs); err != nil {
			return nil, fmt.Errorf("unmarshal encoded args: %w", err)
		}
		decodedArgs, err := base64.StdEncoding.DecodeString(encodedArgs)
		if err != nil {
			return nil, fmt.Errorf("decode base64 args: %w", err)
		}
		args = decodedArgs
	}

	var params struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Timeout *float64 `json:"timeout"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error: invalid arguments: %v", err),
			}},
			IsError: true,
		}, nil
	}

	if params.Command == "" {
		return &mcp.ToolResult{
			Content: []mcp.Content{{
				Type: "text",
				Text: "Error: command is required",
			}},
			IsError: true,
		}, nil
	}

	timeout := 60 * time.Second
	if params.Timeout != nil {
		timeout = time.Duration(*params.Timeout * float64(time.Second))
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create a pipe for stderr
	stderrR, stderrW := io.Pipe()
	defer stderrR.Close()
	defer stderrW.Close()

	// Create command
	cmd := exec.CommandContext(ctx, params.Command, params.Args...)
	cmd.Stderr = stderrW

	// Start reading stderr in a goroutine
	var stderrBuf bytes.Buffer
	go func() {
		io.Copy(&stderrBuf, stderrR)
	}()

	// Run command and capture output
	output, err := cmd.Output()
	if err != nil {
		var errText string
		if exitErr, ok := err.(*exec.ExitError); ok {
			errText = stderrBuf.String()
			if errText == "" {
				errText = string(exitErr.Stderr)
			}
		} else {
			errText = err.Error()
		}
		if ctx.Err() == context.DeadlineExceeded {
			errText = "command timed out"
		}
		result := &mcp.ToolResult{
			Content: []mcp.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error: %s", errText),
			}},
			IsError: true,
		}
		// Log the result for debugging
		resultJSON, _ := json.Marshal(result)
		log.Printf("Error result: %s", resultJSON)
		return result, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{{
			Type: "text",
			Text: string(output),
		}},
	}, nil
}
