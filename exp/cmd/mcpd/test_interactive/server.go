//go:build ignore

// A simple test server that sends interactive prompts
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Interactive server is a simple test server that sends prompts
func main() {
	if len(os.Args) > 1 && os.Args[1] == "interactive_test" {
		runInteractiveServer()
		return
	}
}

func runInteractiveServer() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Fprintf(os.Stderr, "Interactive test server started\n")

	// Look for incoming requests
	for scanner.Scan() {
		line := scanner.Text()

		// Try to parse as JSON-RPC
		var request map[string]interface{}
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing request: %v\n", err)
			continue
		}

		// Extract method and ID
		method, _ := request["method"].(string)
		id, _ := request["id"].(interface{})

		fmt.Fprintf(os.Stderr, "Received request: method=%s, id=%v\n", method, id)

		// Basic echo method
		if method == "echo" {
			params, _ := request["params"].(map[string]interface{})
			message, _ := params["message"].(string)

			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  message,
			}

			jsonResponse, _ := json.Marshal(response)
			fmt.Println(string(jsonResponse))
		} else if method == "interactive/userInput" {
			// Handle response from a prompt
			params, _ := request["params"].(map[string]interface{})
			originalID, _ := params["original_input_id"].(string)
			value, _ := params["value"].(string)
			status, _ := params["status"].(string)

			fmt.Fprintf(os.Stderr, "Received user input: status=%s, value=%s for prompt=%s\n",
				status, value, originalID)

			// Send a response acknowledging the input
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  fmt.Sprintf("Input received: %s (status: %s)", value, status),
			}

			jsonResponse, _ := json.Marshal(response)
			fmt.Println(string(jsonResponse))

			// For "prompt-chain" ID, send another prompt after a short delay
			if strings.Contains(originalID, "prompt-chain") {
				time.Sleep(500 * time.Millisecond)

				parts := strings.Split(originalID, "-")
				var counter int
				fmt.Sscanf(parts[len(parts)-1], "%d", &counter)

				// Only chain up to 3 prompts
				if counter < 3 {
					sendPrompt(fmt.Sprintf("prompt-chain-%d", counter+1),
						fmt.Sprintf("This is follow-up prompt %d. Continue?", counter+1),
						"text")
				}
			}
		} else if method == "prompt" {
			// Send an interactive prompt
			params, _ := request["params"].(map[string]interface{})
			promptID, _ := params["prompt_id"].(string)
			message, _ := params["message"].(string)

			if promptID == "" {
				promptID = fmt.Sprintf("prompt_%d", time.Now().Unix())
			}

			if message == "" {
				message = "Please enter a value:"
			}

			sendPrompt(promptID, message, "text")

			// Immediately respond to the original request
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  "Prompt sent",
			}

			jsonResponse, _ := json.Marshal(response)
			fmt.Println(string(jsonResponse))
		} else if method == "password_prompt" {
			// Send a password prompt
			promptID := fmt.Sprintf("password_prompt_%d", time.Now().Unix())
			message := "Please enter your password:"

			sendPrompt(promptID, message, "password")

			// Immediately respond to the original request
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  "Password prompt sent",
			}

			jsonResponse, _ := json.Marshal(response)
			fmt.Println(string(jsonResponse))
		} else if method == "prompt_chain" {
			// Start a chain of prompts
			sendPrompt("prompt-chain-1", "This is the first prompt in a chain. Continue?", "text")

			// Immediately respond to the original request
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  "Prompt chain started",
			}

			jsonResponse, _ := json.Marshal(response)
			fmt.Println(string(jsonResponse))
		} else if method == "signal_test" {
			// Method to test signal handling during a long operation
			params, _ := request["params"].(map[string]interface{})
			signal, _ := params["signal"].(string)
			delay, _ := params["delay"].(float64)

			// Respond immediately that we're starting the operation
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  fmt.Sprintf("Signal test started: %s with delay %f", signal, delay),
			}

			jsonResponse, _ := json.Marshal(response)
			fmt.Println(string(jsonResponse))

			// Simulate a long operation
			time.Sleep(time.Duration(delay) * time.Second)

			// Send a notification that the operation is complete
			notification := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "signal_test_complete",
				"params": map[string]interface{}{
					"signal": signal,
					"time":   time.Now().String(),
				},
			}

			jsonNotification, _ := json.Marshal(notification)
			fmt.Println(string(jsonNotification))
		} else {
			// Unknown method
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "Method not found",
					"data":    fmt.Sprintf("Method '%s' not implemented", method),
				},
			}

			jsonResponse, _ := json.Marshal(response)
			fmt.Println(string(jsonResponse))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

// sendPrompt sends an interactive prompt request
func sendPrompt(promptID, message, inputType string) {
	promptRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "interactive/promptUser",
		"params": map[string]interface{}{
			"prompt_message":  message,
			"input_id":        promptID,
			"input_type":      inputType,
			"timeout_seconds": 60,
		},
	}

	jsonPrompt, _ := json.Marshal(promptRequest)
	fmt.Println(string(jsonPrompt))
	fmt.Fprintf(os.Stderr, "Sent prompt: %s (type: %s)\n", promptID, inputType)
}
