//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	fmt.Fprintf(os.Stderr, "Minimal server starting...\n")
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading line: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			length, err := strconv.Atoi(lengthStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing Content-Length: %v\n", err)
				continue
			}

			// Read empty line
			reader.ReadString('\n')

			// Read the JSON body
			body := make([]byte, length)
			n, err := io.ReadFull(reader, body)
			if err != nil || n != length {
				fmt.Fprintf(os.Stderr, "Error reading body: got %d bytes, expected %d: %v\n", n, length, err)
				continue
			}

			var req Request
			if err := json.Unmarshal(body, &req); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
				continue
			}

			// Handle the request
			fmt.Fprintf(os.Stderr, "Received request: %+v\n", req)
			switch req.Method {
			case "initialize":
				resp := Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: json.RawMessage(`{
						"protocolVersion": "2025-03-26",
						"serverInfo": {
							"name": "minimal-server",
							"version": "1.0.0"
						},
						"capabilities": {}
					}`),
				}
				sendResponse(resp)
			default:
				fmt.Fprintf(os.Stderr, "Unknown method: %s\n", req.Method)
			}
		}
	}
}

func sendResponse(resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
		return
	}

	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	fmt.Fprint(os.Stdout, msg)
	os.Stdout.Sync()
}
