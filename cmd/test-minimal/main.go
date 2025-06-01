package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/mcp"
)

type loggingTransport struct {
	base io.ReadWriteCloser
}

func (t *loggingTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return t, nil
}

func (t *loggingTransport) Read(p []byte) (n int, err error) {
	n, err = t.base.Read(p)
	if n > 0 {
		fmt.Fprintf(os.Stderr, "READ: %q\n", p[:n])
	}
	return
}

func (t *loggingTransport) Write(p []byte) (n int, err error) {
	fmt.Fprintf(os.Stderr, "WRITE: %q\n", p)
	return t.base.Write(p)
}

func (t *loggingTransport) Close() error {
	return t.base.Close()
}

func main() {
	// Test against time server over stdio
	cmd := exec.Command("go", "run", "./examples/servers/mcp-time-server")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	// Show server stderr
	go io.Copy(os.Stderr, stderr)

	// Start server
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Create transport
	transport := &loggingTransport{
		base: struct {
			io.Reader
			io.Writer
			io.Closer
		}{
			Reader: stdout,
			Writer: stdin,
			Closer: io.NopCloser(nil),
		},
	}

	// Create client
	client, err := mcp.NewClient(transport)
	if err != nil {
		log.Fatal(err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Initialize
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Initializing...")
	result, err := client.Initialize(ctx, mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo: mcp.Implementation{
			Name:    "test",
			Version: "1.0",
		},
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Success: %v\n", result.ServerInfo)
	}
}
