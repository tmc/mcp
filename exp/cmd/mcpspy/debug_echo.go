// Additional debugging functionality for mcpspy

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// Enhanced JSON message structure for debugging
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSON-RPC error object
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Deep debug logs parsed JSON-RPC messages for analysis
func deepDebugMessage(data []byte) {
	if !*veryVerbose {
		return
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("DEBUG: Not a valid JSON-RPC message: %v", err)
		return
	}

	logFile, err := os.OpenFile("mcp_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("DEBUG: Could not open debug log: %v", err)
		return
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	// Format the message for logging
	var buf bytes.Buffer
	prettyJSON := json.NewEncoder(&buf)
	prettyJSON.SetIndent("", "  ")
	if err := prettyJSON.Encode(msg); err != nil {
		logger.Printf("DEBUG: Error formatting message: %v", err)
		return
	}

	// Log detailed message info
	logger.Printf("======= JSON-RPC Message =======\n")
	logger.Printf("JSONRPC: %s\n", msg.JSONRPC)
	logger.Printf("ID: %v\n", msg.ID)

	if msg.Method != "" {
		logger.Printf("Method: %s\n", msg.Method)
		logger.Printf("Params: %s\n", string(msg.Params))
	}

	if msg.Result != nil {
		logger.Printf("Result: %+v\n", msg.Result)
	}

	if msg.Error != nil {
		logger.Printf("Error: Code=%d, Message=%s\n", msg.Error.Code, msg.Error.Message)
		if msg.Error.Data != nil {
			logger.Printf("Error Data: %+v\n", msg.Error.Data)
		}
	}

	logger.Printf("Full message:\n%s\n", buf.String())
	logger.Printf("==============================\n\n")
}

// TestEchoTool specifically tests the echo tool
func TestEchoTool() {
	// Open debug log
	logFile, err := os.OpenFile("mcp_echo_test.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening debug log: %v\n", err)
		return
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)

	// Also log to stderr for visibility
	consoleLogger := log.New(os.Stderr, "ECHO TEST: ", log.LstdFlags)
	consoleLogger.Println("Starting echo tool test - logging details to mcp_echo_test.log")

	logger.Println("Starting Echo Tool Test")

	// Connect to server with timeout
	dialer := net.Dialer{Timeout: 5 * time.Second}
	consoleLogger.Println("Connecting to server at localhost:7000...")
	conn, err := dialer.Dial("tcp", "localhost:7000")
	if err != nil {
		errorMsg := fmt.Sprintf("Error connecting to server: %v\n", err)
		logger.Println(errorMsg)
		consoleLogger.Println(errorMsg)
		return
	}
	defer conn.Close()
	consoleLogger.Println("Connected successfully")

	// Send initialize request
	initReq := `{"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"roots":{}},"clientInfo":{"name":"echo-test","version":"0.1.0"}},"jsonrpc":"2.0","id":1}`
	logger.Printf("Sending initialize request: %s\n", initReq)
	_, err = fmt.Fprintln(conn, initReq)
	if err != nil {
		logger.Printf("Error sending initialize request: %v\n", err)
		return
	}

	// Read initialize response
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	consoleLogger.Println("Waiting for initialize response...")
	n, err := conn.Read(buf)
	if err != nil {
		errorMsg := fmt.Sprintf("Error reading initialize response: %v\n", err)
		logger.Println(errorMsg)
		consoleLogger.Println(errorMsg)
		return
	}
	initResp := string(buf[:n])
	logger.Printf("Received initialize response: %s\n", initResp)
	consoleLogger.Println("Received initialize response successfully")

	// Send echo tool request with various message types to test
	messages := []string{
		`{"method":"tools/call","params":{"name":"echo","arguments":{"message":"Simple message"}},"jsonrpc":"2.0","id":2}`,
		`{"method":"tools/call","params":{"name":"echo","arguments":{"message":""}},"jsonrpc":"2.0","id":3}`,
		`{"method":"tools/call","params":{"name":"echo","arguments":{"message":123}},"jsonrpc":"2.0","id":4}`,
		`{"method":"tools/call","params":{"name":"echo","arguments":{"message":null}},"jsonrpc":"2.0","id":5}`,
		`{"method":"tools/call","params":{"name":"echo","arguments":{"message":true}},"jsonrpc":"2.0","id":6}`,
	}

	for i, msg := range messages {
		testNum := i + 1
		logger.Printf("Test %d: Sending echo request: %s\n", testNum, msg)
		consoleLogger.Printf("Test %d: Sending echo request...\n", testNum)

		_, err = fmt.Fprintln(conn, msg)
		if err != nil {
			errorMsg := fmt.Sprintf("Error sending echo request: %v\n", err)
			logger.Printf("Test %d: %s", testNum, errorMsg)
			consoleLogger.Printf("Test %d: %s", testNum, errorMsg)
			continue
		}

		// Read echo response with 5 second timeout
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		consoleLogger.Printf("Test %d: Waiting for response (5s timeout)...\n", testNum)

		n, err := conn.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				errorMsg := "TIMEOUT - No response received within 5 seconds\n"
				logger.Printf("Test %d: %s", testNum, errorMsg)
				consoleLogger.Printf("Test %d: %s", testNum, errorMsg)
			} else {
				errorMsg := fmt.Sprintf("Error reading echo response: %v\n", err)
				logger.Printf("Test %d: %s", testNum, errorMsg)
				consoleLogger.Printf("Test %d: %s", testNum, errorMsg)
			}
			continue
		}

		echoResp := string(buf[:n])
		logger.Printf("Test %d: Received echo response: %s\n", testNum, echoResp)
		consoleLogger.Printf("Test %d: Received response (%d bytes)\n", testNum, n)

		// Try to parse and analyze the response
		var respObj map[string]interface{}
		if err := json.Unmarshal([]byte(echoResp), &respObj); err != nil {
			logger.Printf("Test %d: Could not parse response as JSON: %v\n", testNum, err)
		} else {
			// Check if it's an error or result
			if errObj, hasError := respObj["error"]; hasError {
				consoleLogger.Printf("Test %d: ERROR RESPONSE: %v\n", testNum, errObj)
			} else if _, hasResult := respObj["result"]; hasResult {
				consoleLogger.Printf("Test %d: Success response received\n", testNum)
			} else {
				consoleLogger.Printf("Test %d: Unexpected response format\n", testNum)
			}
		}
	}

	logger.Println("Echo Tool Test Completed")
	consoleLogger.Println("Echo Tool Test Completed - see mcp_echo_test.log for details")
}
