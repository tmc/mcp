//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// A simple server that stays running indefinitely, processing commands
// from stdin when provided.
func main() {
	fmt.Fprintln(os.Stderr, "Long-running server started and waiting for input...")

	// Print periodic heartbeat messages to stderr so we know it's still alive
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Fprintln(os.Stderr, "Server is still running...")
			}
		}
	}()

	// Read from stdin in a loop
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		processCommand(line)
	}

	close(done)
	fmt.Fprintln(os.Stderr, "Server shutting down")
}

// processCommand handles input commands
func processCommand(line string) {
	line = strings.TrimSpace(line)

	// Skip empty lines
	if line == "" {
		return
	}

	fmt.Fprintf(os.Stderr, "Received command: %s\n", line)

	// Check for JSON-RPC command
	if strings.HasPrefix(line, "{") && strings.Contains(line, "\"method\"") {
		handleJSONRPC(line)
		return
	}

	// Handle simple text commands
	if line == "help" {
		fmt.Println("{\"result\": \"Available commands: help, echo, prompt, password, time, exit\"}")
	} else if strings.HasPrefix(line, "echo ") {
		message := strings.TrimPrefix(line, "echo ")
		fmt.Printf("{\"result\": \"%s\"}\n", message)
	} else if line == "prompt" {
		// Send an interactive prompt request
		promptRequest := `{"jsonrpc":"2.0","method":"interactive/promptUser","params":{"prompt_message":"Please enter your name","input_id":"name_prompt_1","timeout_seconds":30}}`
		fmt.Println(promptRequest)
	} else if line == "password" {
		// Send a password prompt request
		promptRequest := `{"jsonrpc":"2.0","method":"interactive/promptUser","params":{"prompt_message":"Enter your password","input_id":"password_prompt_1","input_type":"password","timeout_seconds":30}}`
		fmt.Println(promptRequest)
	} else if line == "time" {
		fmt.Printf("{\"result\": \"%s\"}\n", time.Now().Format(time.RFC3339))
	} else if line == "exit" || line == "quit" {
		fmt.Println("{\"result\": \"goodbye\"}")
		os.Exit(0)
	} else {
		fmt.Printf("{\"result\": \"unknown command: %s\"}\n", line)
	}
}

// handleJSONRPC handles JSON-RPC formatted requests
func handleJSONRPC(jsonStr string) {
	// Very simple parser just to handle interactive/userInput responses
	if strings.Contains(jsonStr, "\"method\":\"interactive/userInput\"") {
		// Extract the input value
		if strings.Contains(jsonStr, "\"value\":") {
			valueStart := strings.Index(jsonStr, "\"value\":") + 8
			if valueStart >= 8 {
				// Find the next quote after "value":"
				quoteStart := strings.Index(jsonStr[valueStart:], "\"")
				if quoteStart >= 0 {
					valueStart += quoteStart + 1
					valueEnd := strings.Index(jsonStr[valueStart:], "\"")
					if valueEnd >= 0 {
						value := jsonStr[valueStart : valueStart+valueEnd]
						fmt.Printf("{\"result\": \"Thank you for your input: %s\"}\n", value)
						return
					}
				}
			}
		}
	}

	// Echo back the JSON as the result if we couldn't parse it
	fmt.Printf("{\"result\": \"Received JSON: %s\"}\n", jsonStr)
}
