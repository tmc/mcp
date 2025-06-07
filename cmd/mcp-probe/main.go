// mcp-probe is a diagnostic tool for testing MCP servers
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tmc/mcp/jsonrpc2"
)

var (
	timeout         = flag.Duration("timeout", 5*time.Second, "Timeout for individual operations")
	totalTimeout    = flag.Duration("total-timeout", 10*time.Second, "Total timeout for all operations")
	verbose         = flag.Bool("v", false, "Verbose output")
	httpURL         = flag.String("http", "", "HTTP endpoint for HTTP transport")
	sseURL          = flag.String("sse", "", "SSE endpoint for SSE transport")
	testTool        = flag.String("test-tool", "", "Tool to test (if server supports tools)")
	tool            = flag.String("tool", "", "Tool to test (alias for -test-tool)")
	protocolVersion = flag.String("protocol-version", "2025-03-26", "MCP protocol version to use")
)

type Transport interface {
	Send(ctx context.Context, req *jsonrpc2.Request) error
	Receive(ctx context.Context) (*jsonrpc2.Response, error)
	Close() error
}

type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
}

func NewStdioTransport(args []string) (*StdioTransport, error) {
	if len(args) == 0 {
		// Use stdin/stdout directly
		return &StdioTransport{
			stdin:  os.Stdin,
			stdout: os.Stdout,
			reader: bufio.NewReader(os.Stdin),
		}, nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		reader: bufio.NewReader(stdout),
	}, nil
}

func (t *StdioTransport) Send(ctx context.Context, req *jsonrpc2.Request) error {
	// Create a proper JSON-RPC 2.0 message
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"method":  req.Method,
		"params":  req.Params,
	}
	
	// Omit ID for notifications
	if !req.ID.IsValid() {
		delete(msg, "id")
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if *verbose {
		log.Printf("Sending: %s", data)
	}

	framedMsg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	_, err = t.stdin.Write([]byte(framedMsg))
	return err
}

func (t *StdioTransport) Receive(ctx context.Context) (*jsonrpc2.Response, error) {
	// Read headers
	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			// We found the content length header, continue
			continue
		}
	}

	// Read the JSON message
	var resp jsonrpc2.Response
	decoder := json.NewDecoder(t.reader)
	if err := decoder.Decode(&resp); err != nil {
		return nil, err
	}

	if *verbose {
		data, _ := json.Marshal(resp)
		log.Printf("Received: %s", data)
	}

	return &resp, nil
}

func (t *StdioTransport) Close() error {
	if t.cmd != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}

type HTTPTransport struct {
	url    string
	client *http.Client
}

func NewHTTPTransport(url string) *HTTPTransport {
	return &HTTPTransport{
		url:    url,
		client: &http.Client{},
	}
}

func (t *HTTPTransport) Send(ctx context.Context, req *jsonrpc2.Request) error {
	// Create a proper JSON-RPC 2.0 message
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"method":  req.Method,
		"params":  req.Params,
	}
	
	// Omit ID for notifications
	if !req.ID.IsValid() {
		delete(msg, "id")
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if *verbose {
		log.Printf("Sending: %s", data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rpcResp jsonrpc2.Response
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return err
	}

	if *verbose {
		log.Printf("Received: %s", body)
	}

	// Store response for Receive()
	// This is a simplified implementation
	return nil
}

func (t *HTTPTransport) Receive(ctx context.Context) (*jsonrpc2.Response, error) {
	// In a real implementation, we'd need to queue responses
	return nil, nil
}

func (t *HTTPTransport) Close() error {
	return nil
}

func main() {
	flag.Parse()

	// Handle -tool as alias for -test-tool
	if *tool != "" && *testTool == "" {
		*testTool = *tool
	}

	// If no arguments and no transport specified, just print sample requests
	if flag.NArg() == 0 && *httpURL == "" && *sseURL == "" {
		printSampleRequests()
		return
	}

	// Create context with total timeout
	ctx, cancel := context.WithTimeout(context.Background(), *totalTimeout)
	defer cancel()

	var transport Transport
	var err error

	// Choose transport based on flags
	if *httpURL != "" {
		transport = NewHTTPTransport(*httpURL)
	} else if *sseURL != "" {
		log.Fatal("SSE transport not yet implemented")
	} else {
		transport, err = NewStdioTransport(flag.Args())
		if err != nil {
			log.Fatalf("Failed to create stdio transport: %v", err)
		}
	}
	defer transport.Close()

	// Send initialize request
	initParams := fmt.Sprintf(`{
		"protocolVersion": "%s",
		"clientInfo": {
			"name": "mcp-probe",
			"version": "0.1.0"
		},
		"capabilities": {}
	}`, *protocolVersion)

	initReq := &jsonrpc2.Request{
		ID:     jsonrpc2.Int64ID(1),
		Method: "initialize",
		Params: json.RawMessage(initParams),
	}

	// Create context for individual operation
	sendCtx, sendCancel := context.WithTimeout(ctx, *timeout)
	defer sendCancel()

	if err := transport.Send(sendCtx, initReq); err != nil {
		log.Fatalf("Failed to send initialize: %v", err)
	}

	recvCtx, recvCancel := context.WithTimeout(ctx, *timeout)
	defer recvCancel()

	resp, err := transport.Receive(recvCtx)
	if err != nil {
		log.Fatalf("Failed to receive response: %v", err)
	}

	fmt.Printf("Server initialized successfully\n")
	if resp.Result != nil {
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Result, &result); err == nil {
			if info, ok := result["serverInfo"].(map[string]interface{}); ok {
				fmt.Printf("Server: %s %s\n", info["name"], info["version"])
			}
			if caps, ok := result["capabilities"].(map[string]interface{}); ok {
				fmt.Printf("Capabilities: %+v\n", caps)
			}
		}
	}

	// If a tool was specified, test it
	if *testTool != "" {
		toolReq := &jsonrpc2.Request{
			ID:     jsonrpc2.Int64ID(2),
			Method: "tools/call",
			Params: json.RawMessage(fmt.Sprintf(`{
				"name": "%s",
				"arguments": {}
			}`, *testTool)),
		}

		toolSendCtx, toolSendCancel := context.WithTimeout(ctx, *timeout)
		defer toolSendCancel()

		if err := transport.Send(toolSendCtx, toolReq); err != nil {
			log.Fatalf("Failed to send tool request: %v", err)
		}

		toolRecvCtx, toolRecvCancel := context.WithTimeout(ctx, *timeout)
		defer toolRecvCancel()

		resp, err := transport.Receive(toolRecvCtx)
		if err != nil {
			log.Fatalf("Failed to receive tool response: %v", err)
		}

		if resp.Error != nil {
			fmt.Printf("Tool error: %v\n", resp.Error)
		} else {
			fmt.Printf("Tool response: %s\n", resp.Result)
		}
	}

	// Send notifications test
	notifyReq := &jsonrpc2.Request{
		Method: "notifications/initialized",
		Params: json.RawMessage(`{}`),
	}

	notifyCtx, notifyCancel := context.WithTimeout(ctx, *timeout)
	defer notifyCancel()

	if err := transport.Send(notifyCtx, notifyReq); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}

	fmt.Println("Probe completed successfully")
}

func printSampleRequests() {
	// Helper function to print a properly framed JSON-RPC message
	printFramedMessage := func(req *jsonrpc2.Request) {
		// Create a JSON-RPC 2.0 message with the jsonrpc field
		msg := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"method":  req.Method,
			"params":  req.Params,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			return
		}

		// Add Content-Length framing like the transport does
		framedMsg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s\r\n", len(data), data)
		fmt.Print(framedMsg)
	}

	// Print initialize request
	initParams := fmt.Sprintf(`{
		"protocolVersion": "%s",
		"clientInfo": {
			"name": "mcp-probe",
			"version": "0.1.0"
		},
		"capabilities": {}
	}`, *protocolVersion)

	initReq := &jsonrpc2.Request{
		ID:     jsonrpc2.Int64ID(1),
		Method: "initialize",
		Params: json.RawMessage(initParams),
	}

	printFramedMessage(initReq)

	// Print sample tool call
	toolReq := &jsonrpc2.Request{
		ID:     jsonrpc2.Int64ID(2),
		Method: "tools/call",
		Params: json.RawMessage(`{
			"name": "echo",
			"arguments": {
				"message": "Hello, MCP!"
			}
		}`),
	}

	printFramedMessage(toolReq)
}
