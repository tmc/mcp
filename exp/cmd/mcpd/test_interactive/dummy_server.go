//go:build ignore
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// A simple long-running server that reads from stdin, processes commands,
// and doesn't exit immediately. Can be used to test mcpd's interactive mode.
func main() {
	// Print startup message to stderr
	fmt.Fprintln(os.Stderr, "Dummy server started")
	
	// Create a scanner to read lines from stdin
	scanner := bufio.NewScanner(os.Stdin)
	
	// Keep running until told to exit
	for {
		// Check if there's input available
		if scanner.Scan() {
			line := scanner.Text()
			
			// Process the line (simple echo)
			processLine(line)
		} else {
			// Check for errors
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
				break
			}
			
			// No more input (EOF)
			break
		}
	}
	
	fmt.Fprintln(os.Stderr, "Dummy server shutting down")
}

// processLine handles a line of input
func processLine(line string) {
	line = strings.TrimSpace(line)
	
	// Handle special commands
	if strings.HasPrefix(line, "sleep") {
		// Sleep command (e.g., "sleep 5")
		parts := strings.Fields(line)
		if len(parts) > 1 {
			seconds := 1 // Default to 1 second
			fmt.Sscanf(parts[1], "%d", &seconds)
			fmt.Fprintf(os.Stderr, "Sleeping for %d seconds...\n", seconds)
			time.Sleep(time.Duration(seconds) * time.Second)
			fmt.Println("{\"result\": \"done sleeping\"}")
			return
		}
	} else if line == "prompt" {
		// Send an interactive prompt request
		promptRequest := `{"jsonrpc":"2.0","method":"interactive/promptUser","params":{"prompt_message":"Please enter your name","input_id":"name_prompt_1","timeout_seconds":30}}`
		fmt.Println(promptRequest)
		return
	} else if line == "password" {
		// Send a password prompt request
		promptRequest := `{"jsonrpc":"2.0","method":"interactive/promptUser","params":{"prompt_message":"Enter your password","input_id":"password_prompt_1","input_type":"password","timeout_seconds":30}}`
		fmt.Println(promptRequest)
		return
	} else if strings.HasPrefix(line, "echo") {
		// Echo command (e.g., "echo hello")
		parts := strings.SplitN(line, " ", 2)
		if len(parts) > 1 {
			message := parts[1]
			fmt.Printf("{\"result\": \"%s\"}\n", message)
			return
		}
	} else if line == "exit" || line == "quit" {
		fmt.Println("{\"result\": \"goodbye\"}")
		os.Exit(0)
	} else if strings.HasPrefix(line, "{") {
		// Handle JSON input (look for interactive/userInput)
		if strings.Contains(line, "interactive/userInput") {
			// Extract the user input
			start := strings.Index(line, "\"value\":")
			if start > 0 {
				start += 9 // Move past "\"value\":" and the quote
				end := strings.Index(line[start:], "\"")
				if end > 0 {
					value := line[start : start+end]
					fmt.Printf("{\"result\": \"received input: %s\"}\n", value)
					return
				}
			}
		}
	}
	
	// Default echo
	fmt.Printf("{\"result\": \"unknown command: %s\"}\n", line)
}