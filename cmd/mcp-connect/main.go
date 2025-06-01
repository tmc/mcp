// mcp-connect is a unified client for all MCP transport types (stdio, sse, streamableHttp)
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Transport types
const (
	TransportStdio          = "stdio"
	TransportSSE            = "sse"
	TransportStreamableHTTP = "http"
)

// JSON-RPC structures
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      interface{}      `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Transport interface
type Transport interface {
	Connect() error
	SendRequest(req JSONRPCRequest) (*JSONRPCResponse, error)
	Close() error
}

// StdioTransport for process-based communication
type StdioTransport struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	scanner *bufio.Scanner
}

func NewStdioTransport(command string, args ...string) *StdioTransport {
	return &StdioTransport{
		cmd: exec.Command(command, args...),
	}
}

func (t *StdioTransport) Connect() error {
	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	t.scanner = bufio.NewScanner(t.stdout)

	// Log stderr in background
	go func() {
		scanner := bufio.NewScanner(t.stderr)
		for scanner.Scan() {
			log.Printf("stderr: %s", scanner.Text())
		}
	}()

	return nil
}

func (t *StdioTransport) SendRequest(req JSONRPCRequest) (*JSONRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := t.stdin.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("no response received")
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(t.scanner.Text()), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

func (t *StdioTransport) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}
	return nil
}

// SSETransport for Server-Sent Events
type SSETransport struct {
	baseURL      string
	client       *http.Client
	sessionURL   string
	sseConn      io.ReadCloser
	responseChan chan JSONRPCResponse
	errorChan    chan error
	wg           sync.WaitGroup
}

func NewSSETransport(baseURL string) *SSETransport {
	return &SSETransport{
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 30 * time.Second},
		responseChan: make(chan JSONRPCResponse, 10),
		errorChan:    make(chan error, 1),
	}
}

func (t *SSETransport) Connect() error {
	// Connect to SSE endpoint
	sseURL := t.baseURL + "/sse"
	req, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE: %w", err)
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return fmt.Errorf("SSE connection failed: %s", resp.Status)
	}

	t.sseConn = resp.Body

	// Start goroutine to read SSE stream
	t.wg.Add(1)
	go t.readSSEStream()

	// Wait for session endpoint
	time.Sleep(500 * time.Millisecond)
	if t.sessionURL == "" {
		return fmt.Errorf("failed to get session endpoint")
	}

	return nil
}

func (t *SSETransport) readSSEStream() {
	defer t.wg.Done()
	scanner := bufio.NewScanner(t.sseConn)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "endpoint":
				t.sessionURL = t.baseURL + data
			case "message":
				var resp JSONRPCResponse
				if err := json.Unmarshal([]byte(data), &resp); err == nil {
					t.responseChan <- resp
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.errorChan <- err
	}
}

func (t *SSETransport) SendRequest(req JSONRPCRequest) (*JSONRPCResponse, error) {
	if t.sessionURL == "" {
		return nil, fmt.Errorf("no session URL available")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request to session endpoint
	httpReq, err := http.NewRequest("POST", t.sessionURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	resp.Body.Close()

	// Wait for response through SSE stream
	select {
	case jsonResp := <-t.responseChan:
		return &jsonResp, nil
	case err := <-t.errorChan:
		return nil, err
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (t *SSETransport) Close() error {
	if t.sseConn != nil {
		t.sseConn.Close()
	}
	t.wg.Wait()
	close(t.responseChan)
	close(t.errorChan)
	return nil
}

// StreamableHTTPTransport for HTTP streaming
type StreamableHTTPTransport struct {
	baseURL string
	client  *http.Client
}

func NewStreamableHTTPTransport(baseURL string) *StreamableHTTPTransport {
	return &StreamableHTTPTransport{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *StreamableHTTPTransport) Connect() error {
	// No persistent connection needed for streamable HTTP
	return nil
}

func (t *StreamableHTTPTransport) SendRequest(req JSONRPCRequest) (*JSONRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := t.baseURL + "/mcp"
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Required headers for streamableHttp
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %s - %s", resp.Status, string(body))
	}

	// Handle streaming response
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		// Read SSE response
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var jsonResp JSONRPCResponse
				if err := json.Unmarshal([]byte(data), &jsonResp); err == nil {
					return &jsonResp, nil
				}
			}
		}
		return nil, fmt.Errorf("no valid response in stream")
	} else {
		// Regular JSON response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var jsonResp JSONRPCResponse
		if err := json.Unmarshal(body, &jsonResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return &jsonResp, nil
	}
}

func (t *StreamableHTTPTransport) Close() error {
	// No cleanup needed
	return nil
}

func main() {
	var (
		transport = flag.String("transport", "stdio", "Transport type: stdio, sse, http")
		url       = flag.String("url", "", "Server URL (for sse and http transports)")
		command   = flag.String("cmd", "", "Command to run (for stdio transport)")
		script    = flag.String("script", "", "Script file with JSON-RPC requests")
		request   = flag.String("request", "", "Single JSON-RPC request")
		verbose   = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	// Create transport based on type
	var trans Transport
	switch *transport {
	case TransportStdio:
		if *command == "" {
			// Default to server-everything stdio
			args := []string{"@modelcontextprotocol/server-everything", "stdio"}
			trans = NewStdioTransport("npx", args...)
		} else {
			// Parse command
			parts := strings.Fields(*command)
			trans = NewStdioTransport(parts[0], parts[1:]...)
		}
	case TransportSSE:
		if *url == "" {
			*url = "http://localhost:3001"
		}
		trans = NewSSETransport(*url)
	case TransportStreamableHTTP:
		if *url == "" {
			*url = "http://localhost:3001"
		}
		trans = NewStreamableHTTPTransport(*url)
	default:
		log.Fatalf("Unknown transport: %s", *transport)
	}

	// Connect
	if *verbose {
		log.Printf("Connecting via %s transport...", *transport)
	}
	if err := trans.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer trans.Close()

	// Process requests
	if *request != "" {
		// Single request mode
		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(*request), &req); err != nil {
			log.Fatalf("Failed to parse request: %v", err)
		}

		resp, err := trans.SendRequest(req)
		if err != nil {
			log.Fatalf("Request failed: %v", err)
		}

		// Pretty print response
		output, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(output))
	} else if *script != "" {
		// Script mode - process multiple requests from file
		file, err := os.Open(*script)
		if err != nil {
			log.Fatalf("Failed to open script: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			var req JSONRPCRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				log.Printf("Failed to parse request: %v", err)
				continue
			}

			resp, err := trans.SendRequest(req)
			if err != nil {
				log.Printf("Request failed: %v", err)
				continue
			}

			output, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(output))
			fmt.Println("---")
		}
	} else {
		// Interactive mode - read from stdin
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Enter JSON-RPC requests (one per line):")

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var req JSONRPCRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				log.Printf("Failed to parse request: %v", err)
				continue
			}

			resp, err := trans.SendRequest(req)
			if err != nil {
				log.Printf("Request failed: %v", err)
				continue
			}

			output, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(output))
		}
	}
}
