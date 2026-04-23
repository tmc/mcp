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
	"time"
)

// JSON-RPC request structure
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSON-RPC response structure
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

func main() {
	var (
		url       = flag.String("url", "http://localhost:3001", "Server URL")
		transport = flag.String("transport", "sse", "Transport type (sse, http)")
		method    = flag.String("method", "initialize", "JSON-RPC method")
		params    = flag.String("params", `{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"HTTPTest","version":"1.0.0"}}`, "JSON-RPC params")
	)
	flag.Parse()

	switch *transport {
	case "sse":
		testSSE(*url, *method, *params)
	case "http":
		testHTTP(*url, *method, *params)
	default:
		log.Fatalf("Unknown transport: %s", *transport)
	}
}

func testSSE(baseURL, method, params string) {
	fmt.Printf("Testing SSE transport at %s\n", baseURL)

	// Try to connect to SSE endpoint
	sseURL := baseURL + "/sse"
	fmt.Printf("Connecting to SSE endpoint: %s\n", sseURL)

	client := &http.Client{Timeout: 30 * time.Second}

	// First, try to initialize via POST
	initURL := baseURL + "/mcp"
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  json.RawMessage(params),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := client.Post(initURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Try alternative endpoints
		fmt.Printf("Failed to POST to %s: %v\n", initURL, err)
		fmt.Println("Trying alternative endpoint...")

		// Try root endpoint
		resp, err = client.Post(baseURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Failed to POST to %s: %v\n", baseURL, err)
		}
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Response Status: %s\n", resp.Status)
		fmt.Printf("Response Headers: %v\n", resp.Header)

		// Try to parse as JSON-RPC response
		var jsonResp JSONRPCResponse
		if err := json.Unmarshal(body, &jsonResp); err == nil {
			fmt.Println("Received JSON-RPC response:")
			pretty, _ := json.MarshalIndent(jsonResp, "", "  ")
			fmt.Println(string(pretty))
		} else {
			fmt.Printf("Response Body: %s\n", string(body))
		}
	}

	// Now connect to SSE stream
	sseReq, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		log.Fatalf("Failed to create SSE request: %v", err)
	}
	sseReq.Header.Set("Accept", "text/event-stream")

	sseResp, err := client.Do(sseReq)
	if err != nil {
		fmt.Printf("Failed to connect to SSE: %v\n", err)
		return
	}
	defer sseResp.Body.Close()

	fmt.Printf("SSE Connection Status: %s\n", sseResp.Status)

	// Read SSE stream
	scanner := bufio.NewScanner(sseResp.Body)
	eventCount := 0
	timeout := time.After(5 * time.Second)

	fmt.Println("Reading SSE events...")
	for {
		select {
		case <-timeout:
			fmt.Printf("Timeout after %d events\n", eventCount)
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if line != "" {
					fmt.Printf("SSE: %s\n", line)
					eventCount++
				}
			} else {
				if err := scanner.Err(); err != nil {
					fmt.Printf("SSE Error: %v\n", err)
				}
				return
			}
		}
	}
}

func testHTTP(baseURL, method, params string) {
	fmt.Printf("Testing HTTP transport at %s\n", baseURL)

	client := &http.Client{Timeout: 30 * time.Second}

	// Create JSON-RPC request
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  json.RawMessage(params),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
	}

	// Try different endpoints
	endpoints := []string{
		baseURL + "/mcp",
		baseURL + "/rpc",
		baseURL + "/api",
		baseURL,
	}

	for _, endpoint := range endpoints {
		fmt.Printf("\nTrying endpoint: %s\n", endpoint)

		resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Status: %s\n", resp.Status)
		fmt.Printf("Headers: %v\n", resp.Header)

		// Try to parse as JSON-RPC
		var jsonResp JSONRPCResponse
		if err := json.Unmarshal(body, &jsonResp); err == nil {
			fmt.Println("JSON-RPC Response:")
			pretty, _ := json.MarshalIndent(jsonResp, "", "  ")
			fmt.Println(string(pretty))
		} else {
			// Show first 500 chars of response
			if len(body) > 500 {
				fmt.Printf("Response (first 500 chars): %s...\n", string(body[:500]))
			} else {
				fmt.Printf("Response: %s\n", string(body))
			}
		}
	}
}
