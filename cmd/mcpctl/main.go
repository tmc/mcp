package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/mcp/internal/mcpclient"
)

var (
	debug   = flag.Bool("debug", false, "enable debug logging")
	timeout = flag.Duration("timeout", 5*time.Second, "timeout for operations")
)

type pipeTransport struct {
	ctx    context.Context
	reader io.Reader
	writer io.Writer
}

func (t *pipeTransport) Read(p []byte) (n int, err error)  { return t.reader.Read(p) }
func (t *pipeTransport) Write(p []byte) (n int, err error) { return t.writer.Write(p) }
func (t *pipeTransport) Close() error                      { return nil }
func (t *pipeTransport) Context() context.Context          { return t.ctx }

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] server-cmd [command]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  list     - List available tools\n")
		fmt.Fprintf(os.Stderr, "  info     - Show initialization info\n")
		fmt.Fprintf(os.Stderr, "  call     - Call a tool (e.g. call exec 'ls -l')\n")
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	serverCmd := flag.Arg(0)
	command := "list"
	if flag.NArg() > 1 {
		command = flag.Arg(1)
	}

	if *debug {
		log.Printf("Server command: %s", serverCmd)
		log.Printf("Command: %s", command)
	}

	// Start the server process
	cmd := exec.Command("sh", "-c", serverCmd)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Create transport with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	transport := &pipeTransport{
		ctx:    ctx,
		reader: stdout,
		writer: stdin,
	}

	// Initialize client
	client := mcpclient.NewClient("mcpctl", "0.1.0", transport)
	reply, err := client.Initialize(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Execute requested command
	switch command {
	case "list":
		tools, err := client.ListTools(ctx)
		if err != nil {
			log.Fatalf("Failed to list tools: %v", err)
		}
		fmt.Println("Available tools:")
		for _, tool := range tools.Tools {
			fmt.Printf("  %s - %s\n", tool.Name, tool.Description)
		}

	case "info":
		prettyJSON, _ := json.MarshalIndent(reply, "", "  ")
		fmt.Printf("Server Info:\n%s\n", prettyJSON)

	case "call":
		if flag.NArg() < 4 {
			log.Fatal("Usage: call <tool> <args>")
		}
		toolName := flag.Arg(2)
		toolArgs := flag.Arg(3)

		args := map[string]interface{}{
			"command": toolArgs,
		}
		data, err := json.Marshal(args)
		if err != nil {
			log.Fatalf("Failed to marshal args: %v", err)
		}

		result, err := client.CallTool(ctx, toolName, data)
		if err != nil {
			log.Fatalf("Tool call failed: %v", err)
		}

		for _, c := range result.Content {
			if c.Text != "" {
				fmt.Print(c.Text)
			}
		}

	default:
		log.Fatalf("Unknown command: %s", command)
	}

	if err := cmd.Wait(); err != nil {
		if *debug {
			log.Printf("Server exited with error: %v", err)
		}
	}
}
